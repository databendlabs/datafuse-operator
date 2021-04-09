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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ComputeGroupState string
const (
	ComputeGroupDeployed ComputeGroupStatus = "Ready"
	ComputeGroupPending	 ComputeGroupStatus = "Pending"
)
// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatafuseComputeGroupSpec defines the desired state of DatafuseComputeGroup
type DatafuseComputeGroupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ComputeLeaders will incorporate all workers to form a cluster, designed for HA purpose
	// For performance consideration, suggest to set at most 3 to 5 leaders
	ComputeLeaders DatafuseComputeSet`json:"leaders,omitempty"`
	// Number of workers per cluster, workers are identical
	// +optional
	ComputeWorkers DatafuseComputeSet `json:"workers,omitempty"`
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Default=latest
	// +optional
	Version *string `json:"version,omitempty"`
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Default=default
	// +kubebuilder:validation:Optional
	Namespace *string	`json:"namespace,omitempty"`
}

// DatafuseComputeGroupStatus defines the observed state of DatafuseComputeGroup
type DatafuseComputeGroupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ReadyComputeLeaders map[string]ComputeInstanceState `json:"readyleaders,omitempty"`
	ReadyComputeWorkers	map[string]ComputeInstanceState `json:"readyworkers,omitempty"`
	status	ComputeGroupState							`json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatafuseComputeGroup is the Schema for the datafusecomputegroups API
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
