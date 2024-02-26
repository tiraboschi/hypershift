/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

// HostedControlPlaneStatusApplyConfiguration represents an declarative configuration of the HostedControlPlaneStatus type for use
// with apply.
type HostedControlPlaneStatusApplyConfiguration struct {
	Ready                          *bool                                   `json:"ready,omitempty"`
	Initialized                    *bool                                   `json:"initialized,omitempty"`
	ExternalManagedControlPlane    *bool                                   `json:"externalManagedControlPlane,omitempty"`
	ControlPlaneEndpoint           *APIEndpointApplyConfiguration          `json:"controlPlaneEndpoint,omitempty"`
	OAuthCallbackURLTemplate       *string                                 `json:"oauthCallbackURLTemplate,omitempty"`
	VersionStatus                  *ClusterVersionStatusApplyConfiguration `json:"versionStatus,omitempty"`
	Version                        *string                                 `json:"version,omitempty"`
	ReleaseImage                   *string                                 `json:"releaseImage,omitempty"`
	LastReleaseImageTransitionTime *v1.Time                                `json:"lastReleaseImageTransitionTime,omitempty"`
	KubeConfig                     *KubeconfigSecretRefApplyConfiguration  `json:"kubeConfig,omitempty"`
	KubeadminPassword              *corev1.LocalObjectReference            `json:"kubeadminPassword,omitempty"`
	Conditions                     []metav1.ConditionApplyConfiguration    `json:"conditions,omitempty"`
	Platform                       *PlatformStatusApplyConfiguration       `json:"platform,omitempty"`
	NodeCount                      *int                                    `json:"nodeCount,omitempty"`
}

// HostedControlPlaneStatusApplyConfiguration constructs an declarative configuration of the HostedControlPlaneStatus type for use with
// apply.
func HostedControlPlaneStatus() *HostedControlPlaneStatusApplyConfiguration {
	return &HostedControlPlaneStatusApplyConfiguration{}
}

// WithReady sets the Ready field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Ready field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithReady(value bool) *HostedControlPlaneStatusApplyConfiguration {
	b.Ready = &value
	return b
}

// WithInitialized sets the Initialized field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Initialized field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithInitialized(value bool) *HostedControlPlaneStatusApplyConfiguration {
	b.Initialized = &value
	return b
}

// WithExternalManagedControlPlane sets the ExternalManagedControlPlane field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ExternalManagedControlPlane field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithExternalManagedControlPlane(value bool) *HostedControlPlaneStatusApplyConfiguration {
	b.ExternalManagedControlPlane = &value
	return b
}

// WithControlPlaneEndpoint sets the ControlPlaneEndpoint field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ControlPlaneEndpoint field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithControlPlaneEndpoint(value *APIEndpointApplyConfiguration) *HostedControlPlaneStatusApplyConfiguration {
	b.ControlPlaneEndpoint = value
	return b
}

// WithOAuthCallbackURLTemplate sets the OAuthCallbackURLTemplate field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the OAuthCallbackURLTemplate field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithOAuthCallbackURLTemplate(value string) *HostedControlPlaneStatusApplyConfiguration {
	b.OAuthCallbackURLTemplate = &value
	return b
}

// WithVersionStatus sets the VersionStatus field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the VersionStatus field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithVersionStatus(value *ClusterVersionStatusApplyConfiguration) *HostedControlPlaneStatusApplyConfiguration {
	b.VersionStatus = value
	return b
}

// WithVersion sets the Version field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Version field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithVersion(value string) *HostedControlPlaneStatusApplyConfiguration {
	b.Version = &value
	return b
}

// WithReleaseImage sets the ReleaseImage field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ReleaseImage field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithReleaseImage(value string) *HostedControlPlaneStatusApplyConfiguration {
	b.ReleaseImage = &value
	return b
}

// WithLastReleaseImageTransitionTime sets the LastReleaseImageTransitionTime field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LastReleaseImageTransitionTime field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithLastReleaseImageTransitionTime(value v1.Time) *HostedControlPlaneStatusApplyConfiguration {
	b.LastReleaseImageTransitionTime = &value
	return b
}

// WithKubeConfig sets the KubeConfig field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the KubeConfig field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithKubeConfig(value *KubeconfigSecretRefApplyConfiguration) *HostedControlPlaneStatusApplyConfiguration {
	b.KubeConfig = value
	return b
}

// WithKubeadminPassword sets the KubeadminPassword field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the KubeadminPassword field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithKubeadminPassword(value corev1.LocalObjectReference) *HostedControlPlaneStatusApplyConfiguration {
	b.KubeadminPassword = &value
	return b
}

// WithConditions adds the given value to the Conditions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Conditions field.
func (b *HostedControlPlaneStatusApplyConfiguration) WithConditions(values ...*metav1.ConditionApplyConfiguration) *HostedControlPlaneStatusApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithConditions")
		}
		b.Conditions = append(b.Conditions, *values[i])
	}
	return b
}

// WithPlatform sets the Platform field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Platform field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithPlatform(value *PlatformStatusApplyConfiguration) *HostedControlPlaneStatusApplyConfiguration {
	b.Platform = value
	return b
}

// WithNodeCount sets the NodeCount field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the NodeCount field is set to the value of the last call.
func (b *HostedControlPlaneStatusApplyConfiguration) WithNodeCount(value int) *HostedControlPlaneStatusApplyConfiguration {
	b.NodeCount = &value
	return b
}
