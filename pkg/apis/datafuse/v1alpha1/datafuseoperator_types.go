/*
Copyright 2021.

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

// +kubebuilder:rbac:groups=datafuse.datafuselabs.io,resources=datafuseoperators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=datafuse.datafuselabs.io,resources=datafuseoperators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",resources=deployments/status,verbs=get
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OperatorState string

const (
	OperatorReady   OperatorState = "Ready"
	OperatorPending OperatorState = "Pending"
	OperatorCreated OperatorState = "Created"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatafuseOperatorSpec defines the desired state of DatafuseOperator
type DatafuseOperatorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Define a set of compute groups belongs to the cluster
	// +kubebuilder:validation:Required
	ComputeGroups []*DatafuseComputeGroupSpec `json:"computeGroups,omitempty"`
	// Fuse Query and Fuse Store will share the same version
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Default=latest
	// +kubebuilder:validation:Optional
	Version *string `json:"version,omitempty"`
}

// DatafuseOperatorStatus defines the observed state of DatafuseOperator
type DatafuseOperatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// state of each compute group. map from compute group name to their status
	ComputeGroupStates map[string]ComputeGroupState `json:"computeGroupStates,omitempty"`
	// +kubebuilder:validation:Enum=Created;Pending;Ready
	// +kubebuilder:validation:Default=Created
	// +kubebuilder:validation:Optional
	Status OperatorState `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatafuseOperator is the Schema for the datafuseclusters API
//+k8s:openapi-gen=true
//+genclient
//+kubebuilder:resource:shortName=op-df
type DatafuseOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatafuseOperatorSpec   `json:"spec,omitempty"`
	Status DatafuseOperatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatafuseOperatorList contains a list of DatafuseCluster
type DatafuseOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatafuseOperator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatafuseOperator{}, &DatafuseOperatorList{})
}
