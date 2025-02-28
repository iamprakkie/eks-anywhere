/*
Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License").
You may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

//nolint
package snow

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AWSSnowIPPoolSpec defines the desired state of AWSSnowIPPool
type AWSSnowIPPoolSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make generate" to regenerate code after modifying this file

	// IPPools is the configuration of static ips used for machine's ip.
	IPPools []IPPool `json:"pools,omitempty"`
}

// IPPool is the configuration of static ips used for machine's ip.
type IPPool struct {
	// IPStart is the start of an ip range
	IPStart *string `json:"ipStart,omitempty"`
	// IPEnd is the end of an ip range
	IPEnd *string `json:"ipEnd,omitempty"`
	// Subnet is customers' network subnet, we can use it to determine if two ip addresses are in the same subnet.
	Subnet *string `json:"subnet,omitempty"`
	// Gateway is the gateway of this subnet. Used for routing purpose
	Gateway *string `json:"gateway,omitempty"`
}

// AWSSnowIPPoolStatus defines the observed state of AWSSnowIPPool
type AWSSnowIPPoolStatus struct { // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make generate" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AWSSnowIPPool is the Schema for the awssnowippools API
type AWSSnowIPPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSSnowIPPoolSpec   `json:"spec,omitempty"`
	Status AWSSnowIPPoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AWSSnowIPPoolList contains a list of AWSSnowIPPool
type AWSSnowIPPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSSnowIPPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AWSSnowIPPool{}, &AWSSnowIPPoolList{})
}
