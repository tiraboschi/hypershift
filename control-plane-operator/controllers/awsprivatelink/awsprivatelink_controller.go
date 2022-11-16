package awsprivatelink

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	hyperv1 "github.com/openshift/hypershift/api/v1alpha1"
	awsutil "github.com/openshift/hypershift/cmd/infra/aws/util"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/manifests"
	"github.com/openshift/hypershift/support/upsert"
	"github.com/openshift/hypershift/support/util"
)

const (
	defaultResync = 10 * time.Hour
)

// PrivateServiceObserver watches a given Service type LB and reconciles
// an awsEndpointService CR representation for it.
type PrivateServiceObserver struct {
	client.Client

	clientset *kubeclient.Clientset
	log       logr.Logger

	ControllerName   string
	ServiceNamespace string
	ServiceName      string
	HCPNamespace     string
	upsert.CreateOrUpdateProvider
}

func nameMapper(names []string) handler.MapFunc {
	nameSet := sets.NewString(names...)
	return func(obj client.Object) []reconcile.Request {
		if !nameSet.Has(obj.GetName()) {
			return nil
		}
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: obj.GetNamespace(),
					Name:      obj.GetName(),
				},
			},
		}
	}
}

func namedResourceHandler(names ...string) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(nameMapper(names))
}

func ControllerName(name string) string {
	return fmt.Sprintf("%s-observer", name)
}

func (r *PrivateServiceObserver) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	r.log = ctrl.Log.WithName(r.ControllerName).WithValues("name", r.ServiceName, "namespace", r.ServiceNamespace)
	var err error
	r.clientset, err = kubeclient.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(r.clientset, defaultResync, informers.WithNamespace(r.ServiceNamespace))
	services := informerFactory.Core().V1().Services()
	c, err := controller.New(r.ControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	if err := c.Watch(&source.Informer{Informer: services.Informer()}, namedResourceHandler(r.ServiceName)); err != nil {
		return err
	}
	mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		informerFactory.Start(ctx.Done())
		return nil
	}))
	return nil
}

func (r *PrivateServiceObserver) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log.Info("reconciling")

	// Fetch the Service
	svc, err := r.clientset.CoreV1().Services(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.log.Info("service not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the HostedControlPlane
	hcpList := &hyperv1.HostedControlPlaneList{}
	if err := r.List(ctx, hcpList, &client.ListOptions{Namespace: r.HCPNamespace}); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get resource: %w", err)
	}
	if len(hcpList.Items) == 0 {
		// Return early if HostedControlPlane is deleted
		return ctrl.Result{}, nil
	}
	if len(hcpList.Items) > 1 {
		return ctrl.Result{}, fmt.Errorf("unexpected number of HostedControlPlanes in namespace, expected: 1, actual: %d", len(hcpList.Items))
	}

	hcp := hcpList.Items[0]

	// Return early if HostedControlPlane is deleted
	if !hcp.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		r.log.Info("load balancer not provisioned yet")
		return ctrl.Result{}, nil
	}
	awsEndpointService := &hyperv1.AWSEndpointService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ServiceName,
			Namespace: r.HCPNamespace,
		},
	}
	lbName := strings.Split(strings.Split(svc.Status.LoadBalancer.Ingress[0].Hostname, ".")[0], "-")[0]
	if _, err := r.CreateOrUpdate(ctx, r, awsEndpointService, func() error {
		awsEndpointService.Spec.NetworkLoadBalancerName = lbName
		if hcp.Spec.Platform.AWS != nil {
			awsEndpointService.Spec.ResourceTags = hcp.Spec.Platform.AWS.ResourceTags
		}
		return nil
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile AWSEndpointService: %w", err)
	}
	r.log.Info("reconcile complete", "request", req)
	return ctrl.Result{}, nil
}

const (
	finalizer                              = "hypershift.openshift.io/control-plane-operator-finalizer"
	endpointServiceDeletionRequeueDuration = 5 * time.Second
	hypershiftLocalZone                    = "hypershift.local"
	routerDomain                           = "apps"
)

// AWSEndpointServiceReconciler watches AWSEndpointService resources and reconciles
// the existence of AWS Endpoints for it in the guest cluster infrastructure.
type AWSEndpointServiceReconciler struct {
	client.Client
	ec2Client     ec2iface.EC2API
	route53Client route53iface.Route53API
}

func (r *AWSEndpointServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	_, err := ctrl.NewControllerManagedBy(mgr).
		For(&hyperv1.AWSEndpointService{}).
		WithOptions(controller.Options{
			RateLimiter:             workqueue.NewItemExponentialFailureRateLimiter(3*time.Second, 30*time.Second),
			MaxConcurrentReconciles: 10,
		}).
		Build(r)
	if err != nil {
		return fmt.Errorf("failed setting up with a controller manager: %w", err)
	}

	r.Client = mgr.GetClient()

	// AWS_SHARED_CREDENTIALS_FILE and AWS_REGION envvar should be set in operator deployment
	awsSession := awsutil.NewSession("control-plane-operator", "", "", "", "")
	awsConfig := aws.NewConfig()
	r.ec2Client = ec2.New(awsSession, awsConfig)
	route53Config := aws.NewConfig()
	// Hardcode region for route53 config
	route53Config.Region = aws.String("us-east-1")
	r.route53Client = route53.New(awsSession, route53Config)

	return nil
}

func (r *AWSEndpointServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log, err := logr.FromContext(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("logger not found: %w", err)
	}

	log.Info("reconciling")

	// Fetch the AWSEndpointService
	obj := &hyperv1.AWSEndpointService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get resource: %w", err)
	}

	// Don't change the cached object
	awsEndpointService := obj.DeepCopy()

	// Return early if deleted
	if !awsEndpointService.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(awsEndpointService, finalizer) {
			// If we previously removed our finalizer, don't delete again and return early
			return ctrl.Result{}, nil
		}
		completed, err := r.delete(ctx, awsEndpointService, r.ec2Client, r.route53Client)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete resource: %w", err)
		}
		if !completed {
			return ctrl.Result{RequeueAfter: endpointServiceDeletionRequeueDuration}, nil
		}
		if controllerutil.ContainsFinalizer(awsEndpointService, finalizer) {
			controllerutil.RemoveFinalizer(awsEndpointService, finalizer)
			if err := r.Update(ctx, awsEndpointService); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}

	// Ensure the awsEndpointService has a finalizer for cleanup
	if !controllerutil.ContainsFinalizer(awsEndpointService, finalizer) {
		controllerutil.AddFinalizer(awsEndpointService, finalizer)
		if err := r.Update(ctx, awsEndpointService); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	serviceName := awsEndpointService.Status.EndpointServiceName
	if serviceName == "" {
		// Service Endpoint is not yet set, wait for hypershift-operator to populate
		// Likely observing our own Create
		return ctrl.Result{}, nil
	}

	// Fetch the HostedControlPlane
	hcpList := &hyperv1.HostedControlPlaneList{}
	if err := r.List(ctx, hcpList, &client.ListOptions{Namespace: req.Namespace}); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get resource: %w", err)
	}
	if len(hcpList.Items) == 0 {
		// Return early if HostedControlPlane is deleted
		return ctrl.Result{}, nil
	}
	if len(hcpList.Items) > 1 {
		return ctrl.Result{}, fmt.Errorf("unexpected number of HostedControlPlanes in namespace, expected: 1, actual: %d", len(hcpList.Items))
	}
	hcp := &hcpList.Items[0]

	// Reconcile the AWSEndpointService
	oldStatus := awsEndpointService.Status.DeepCopy()
	if err := reconcileAWSEndpointService(ctx, awsEndpointService, hcp, r.ec2Client, r.route53Client); err != nil {
		meta.SetStatusCondition(&awsEndpointService.Status.Conditions, metav1.Condition{
			Type:    string(hyperv1.AWSEndpointAvailable),
			Status:  metav1.ConditionFalse,
			Reason:  hyperv1.AWSErrorReason,
			Message: err.Error(),
		})
		if !equality.Semantic.DeepEqual(*oldStatus, awsEndpointService.Status) {
			if err := r.Status().Update(ctx, awsEndpointService); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	meta.SetStatusCondition(&awsEndpointService.Status.Conditions, metav1.Condition{
		Type:    string(hyperv1.AWSEndpointAvailable),
		Status:  metav1.ConditionTrue,
		Reason:  hyperv1.AWSSuccessReason,
		Message: "",
	})

	if !equality.Semantic.DeepEqual(*oldStatus, awsEndpointService.Status) {
		if err := r.Status().Update(ctx, awsEndpointService); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("reconcilation complete")
	return ctrl.Result{}, nil
}

func hasAWSConfig(platform *hyperv1.PlatformSpec) bool {
	return platform.Type == hyperv1.AWSPlatform && platform.AWS != nil && platform.AWS.CloudProviderConfig != nil &&
		platform.AWS.CloudProviderConfig.Subnet != nil && platform.AWS.CloudProviderConfig.Subnet.ID != nil
}

func diffSubnetIDs(desired []string, existing []*string) (added, removed []*string) {
	var found bool
	for i, desiredID := range desired {
		found = false
		for _, existingID := range existing {
			if desiredID == *existingID {
				found = true
				break
			}
		}
		if !found {
			added = append(added, &desired[i])
		}
	}
	for _, existingID := range existing {
		found = false
		for _, desiredID := range desired {
			if desiredID == *existingID {
				found = true
				break
			}
		}
		if !found {
			removed = append(removed, existingID)
		}
	}
	return
}

func reconcileAWSEndpointService(ctx context.Context, awsEndpointService *hyperv1.AWSEndpointService, hcp *hyperv1.HostedControlPlane, ec2Client ec2iface.EC2API, route53Client route53iface.Route53API) error {
	log, err := logr.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("logger not found: %w", err)
	}

	if len(awsEndpointService.Status.EndpointServiceName) == 0 {
		log.Info("endpoint service name is not set, ignoring", "name", awsEndpointService.Name)
		return nil
	}

	endpointID := awsEndpointService.Status.EndpointID
	var endpointDNSEntries []*ec2.DnsEntry
	if endpointID != "" {
		// check if Endpoint exists in AWS
		output, err := ec2Client.DescribeVpcEndpointsWithContext(ctx, &ec2.DescribeVpcEndpointsInput{
			VpcEndpointIds: []*string{aws.String(endpointID)},
		})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "InvalidVpcEndpointId.NotFound" {
					// clear the EndpointID so a new Endpoint is created on the requeue
					awsEndpointService.Status.EndpointID = ""
					return fmt.Errorf("endpoint with id %s not found, resetting status", endpointID)
				} else {
					return errors.New(awsErr.Code())
				}
			}
			return err
		}
		if len(output.VpcEndpoints) == 0 {
			// This should not happen but just in case
			// clear the EndpointID so a new Endpoint is created on the requeue
			awsEndpointService.Status.EndpointID = ""
			return fmt.Errorf("endpoint with id %s not found, resetting status", endpointID)
		}
		log.Info("endpoint exists", "endpointID", endpointID)
		endpointDNSEntries = output.VpcEndpoints[0].DnsEntries

		// ensure endpoint has the right subnets
		added, removed := diffSubnetIDs(awsEndpointService.Spec.SubnetIDs, output.VpcEndpoints[0].SubnetIds)
		if added != nil || removed != nil {
			log.Info("endpoint subnets have changed")
			_, err := ec2Client.ModifyVpcEndpointWithContext(ctx, &ec2.ModifyVpcEndpointInput{
				VpcEndpointId:   aws.String(endpointID),
				AddSubnetIds:    added,
				RemoveSubnetIds: removed,
			})
			if err != nil {
				msg := err.Error()
				if awsErr, ok := err.(awserr.Error); ok {
					msg = awsErr.Code()
				}
				log.Error(err, "failed to modify vpc endpoint")
				return fmt.Errorf("failed to modify vpc endpoint: %s", msg)
			}
			log.Info("endpoint subnets updated")
		} else {
			log.Info("endpoint subnets are unchanged")
		}
	} else {
		if !hasAWSConfig(&hcp.Spec.Platform) {
			return fmt.Errorf("AWS platform information not provided in HostedControlPlane")
		}

		// Verify there is not already an Endpoint that we can adopt
		// This can happen if we have a stale status on AWSEndpointService or encoutered
		// an error updating the AWSEndpointService on the previous reconcile
		output, err := ec2Client.DescribeVpcEndpointsWithContext(ctx, &ec2.DescribeVpcEndpointsInput{
			Filters: apiTagToEC2Filter(awsEndpointService.Name, hcp.Spec.Platform.AWS.ResourceTags),
		})
		if err != nil {
			msg := err.Error()
			if awsErr, ok := err.(awserr.Error); ok {
				msg = awsErr.Code()
			}
			log.Error(err, "failed to describe vpc endpoints")
			return fmt.Errorf("failed to describe vpc endpoints: %s", msg)
		}
		if len(output.VpcEndpoints) != 0 {
			endpointID = *output.VpcEndpoints[0].VpcEndpointId
			log.Info("endpoint already exists, adopting", "endpointID", endpointID)
			awsEndpointService.Status.EndpointID = endpointID
			endpointDNSEntries = output.VpcEndpoints[0].DnsEntries
		} else {
			log.Info("endpoint does not already exist")
			// Create the Endpoint
			subnetIDs := []*string{}
			for i := range awsEndpointService.Spec.SubnetIDs {
				subnetIDs = append(subnetIDs, &awsEndpointService.Spec.SubnetIDs[i])
			}
			output, err := ec2Client.CreateVpcEndpointWithContext(ctx, &ec2.CreateVpcEndpointInput{
				ServiceName:     aws.String(awsEndpointService.Status.EndpointServiceName),
				VpcId:           aws.String(hcp.Spec.Platform.AWS.CloudProviderConfig.VPC),
				VpcEndpointType: aws.String(ec2.VpcEndpointTypeInterface),
				SubnetIds:       subnetIDs,
				TagSpecifications: []*ec2.TagSpecification{{
					ResourceType: aws.String("vpc-endpoint"),
					Tags:         apiTagToEC2Tag(awsEndpointService.Name, hcp.Spec.Platform.AWS.ResourceTags),
				}},
			})
			if err != nil {
				msg := err.Error()
				if awsErr, ok := err.(awserr.Error); ok {
					msg = awsErr.Code()
				}
				log.Error(err, "failed to create vpc endpoint")
				return fmt.Errorf("failed to create vpc endpoint: %s", msg)
			}
			if output == nil || output.VpcEndpoint == nil {
				return fmt.Errorf("CreateVpcEndpointWithContext output is nil")
			}

			endpointID = *output.VpcEndpoint.VpcEndpointId
			log.Info("endpoint created", "endpointID", endpointID)
			awsEndpointService.Status.EndpointID = endpointID
			endpointDNSEntries = output.VpcEndpoint.DnsEntries
		}
	}

	if len(endpointDNSEntries) == 0 {
		log.Info("endpoint has no DNS entries, skipping DNS record creation", "endpointID", endpointID)
		return nil
	}

	recordNames := recordsForService(awsEndpointService, hcp)
	if len(recordNames) == 0 {
		log.Info("WARNING: no mapping from AWSEndpointService to DNS")
		return nil
	}

	zoneName := zoneName(hcp.Name)
	zoneID, err := lookupZoneID(ctx, route53Client, zoneName)
	if err != nil {
		return err
	}

	var fqdns []string
	for _, recordName := range recordNames {
		fqdn := fmt.Sprintf("%s.%s", recordName, zoneName)
		fqdns = append(fqdns, fqdn)
		err = createRecord(ctx, route53Client, zoneID, fqdn, *(endpointDNSEntries[0].DnsName))
		if err != nil {
			return err
		}
		log.Info("DNS record created", "fqdn", fqdn)
	}

	//lint:ignore SA1019 we reset the deprecated field precicely
	// because it is deprecated.
	awsEndpointService.Status.DNSName = ""
	awsEndpointService.Status.DNSNames = fqdns
	awsEndpointService.Status.DNSZoneID = zoneID

	return nil
}

func zoneName(hcpName string) string {
	return fmt.Sprintf("%s.%s", hcpName, hypershiftLocalZone)
}

func RouterZoneName(hcpName string) string {
	return routerDomain + "." + zoneName(hcpName)
}

func recordsForService(awsEndpointService *hyperv1.AWSEndpointService, hcp *hyperv1.HostedControlPlane) []string {
	if awsEndpointService.Name == manifests.KubeAPIServerPrivateService("").Name {
		return []string{"api"}

	}
	if awsEndpointService.Name != manifests.PrivateRouterService("").Name {
		return nil
	}

	// If the kas is exposed through a route, the router needs to have DNS entries for both
	// the kas and the apps domain
	if m := util.ServicePublishingStrategyByTypeForHCP(hcp, hyperv1.APIServer); m != nil && m.Type == hyperv1.Route {
		return []string{"api", "*." + routerDomain}
	}

	return []string{"*.apps"}

}

func apiTagToEC2Tag(name string, in []hyperv1.AWSResourceTag) []*ec2.Tag {
	result := make([]*ec2.Tag, len(in))
	for _, val := range in {
		result = append(result, &ec2.Tag{Key: aws.String(val.Key), Value: aws.String(val.Value)})
	}
	result = append(result, &ec2.Tag{Key: aws.String("AWSEndpointService"), Value: aws.String(name)})

	return result
}

func apiTagToEC2Filter(name string, in []hyperv1.AWSResourceTag) []*ec2.Filter {
	result := make([]*ec2.Filter, len(in))
	for _, val := range in {
		result = append(result, &ec2.Filter{Name: aws.String("tag:" + val.Key), Values: aws.StringSlice([]string{val.Value})})
	}
	result = append(result, &ec2.Filter{Name: aws.String("tag:AWSEndpointService"), Values: aws.StringSlice([]string{name})})

	return result
}

func (r *AWSEndpointServiceReconciler) delete(ctx context.Context, awsEndpointService *hyperv1.AWSEndpointService, ec2Client ec2iface.EC2API, route53Client route53iface.Route53API) (bool, error) {
	log, err := logr.FromContext(ctx)
	if err != nil {
		return false, fmt.Errorf("logger not found: %w", err)
	}

	endpointID := awsEndpointService.Status.EndpointID
	if endpointID != "" {
		if _, err := ec2Client.DeleteVpcEndpointsWithContext(ctx, &ec2.DeleteVpcEndpointsInput{
			VpcEndpointIds: []*string{aws.String(endpointID)},
		}); err != nil {
			return false, err
		}

		// check if Endpoint exists in AWS
		output, err := ec2Client.DescribeVpcEndpointsWithContext(ctx, &ec2.DescribeVpcEndpointsInput{
			VpcEndpointIds: []*string{aws.String(endpointID)},
		})
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok {
				if awsErr.Code() != "InvalidVpcEndpointId.NotFound" {
					return false, err
				}
			} else {
				return false, err
			}

		}

		if output != nil && len(output.VpcEndpoints) != 0 {
			return false, fmt.Errorf("resource requested for deletion but still present")
		}

		log.Info("endpoint deleted", "endpointID", endpointID)
	}

	zoneID := awsEndpointService.Status.DNSZoneID
	if err != nil {
		return false, err
	}

	for _, fqdn := range awsEndpointService.Status.DNSNames {
		if fqdn != "" && zoneID != "" {
			record, err := findRecord(ctx, route53Client, zoneID, fqdn)
			if err != nil {
				return false, err
			}
			if record != nil {
				err = deleteRecord(ctx, route53Client, zoneID, record)
				if err != nil {
					return false, err
				}
				log.Info("DNS record deleted", "fqdn", fqdn)
			} else {
				log.Info("no DNS record found", "fqdn", fqdn)
			}
		}
	}

	return true, nil
}
