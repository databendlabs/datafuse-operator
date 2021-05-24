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
// +kubebuilder:rbac:groups=datafuse.datafuselabs.io,resources=datafusecomputegroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=datafuse.datafuselabs.io,resources=datafusecomputegroups/status,verbs=get;update;patch
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ComputeGroupState string

const (
	ComputeGroupDeployed ComputeGroupState = "Ready"
	ComputeGroupPending  ComputeGroupState = "Pending"
	ComputeGroupCreated  ComputeGroupState = "Created"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatafuseComputeGroupSpec defines the desired state of DatafuseComputeGroup
type DatafuseComputeGroupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ComputeLeaders will incorporate all workers to form a cluster, designed for HA purpose
	// For performance consideration, suggest to set 3 to 5 leaders
	ComputeLeaders *DatafuseComputeSetSpec `json:"leaders,omitempty"`
	// Number of workers per cluster, workers are identical
	ComputeWorkers []*DatafuseComputeSetSpec `json:"workers,omitempty"`
	// Define the name of exposed service
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Define the namespace where group should exists
	// +kubebuilder:validation:Type=string
	// +kubebuilder:default=default
	Namespace string `json:"namespace,omitempty"`
}

// DatafuseComputeGroupStatus defines the observed state of DatafuseComputeGroup
type DatafuseComputeGroupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ReadyComputeLeaders map[string]ComputeInstanceState `json:"readyleaders,omitempty"`
	ReadyComputeWorkers map[string]ComputeInstanceState `json:"readyworkers,omitempty"`
	// Define the current status of group
	// +kubebuilder:validation:Type=string
	// +kubebuilder:default=Created
	Status             ComputeGroupState         `json:"status,omitempty"`
	GroupServiceStatus ComputeGroupServiceStatus `json:"service,omitempty"`
}

// defines the service which the group binded with
type ComputeGroupServiceStatus struct {
	// the name of the serivce for this group
	Name string `json:"name,omitempty"`
}

// +kubebuilder:object:root=true

// DatafuseComputeGroup is the Schema for the datafusecomputegroups API
//+k8s:openapi-gen=true
//+genclient
//+kubebuilder:resource:shortName=cg-df
type DatafuseComputeGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatafuseComputeGroupSpec   `json:"spec,omitempty"`
	Status DatafuseComputeGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatafuseComputeGroupList contains a list of DatafuseComputeGroup
type DatafuseComputeGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatafuseComputeGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatafuseComputeGroup{}, &DatafuseComputeGroupList{})
}
