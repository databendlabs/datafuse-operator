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
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
type ComputeInstanceState string
const (
	ComputeInstanceReadyState	ComputeInstanceState = "READY"
	ComputeInstanceLivenessState	ComputeInstanceState = "LIVE"
	ComputeInstancePendingState	ComputeInstanceState = "Pending"
)
// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatafuseComputeInstanceSpec defines the desired state of DatafuseComputeInstance
type DatafuseComputeInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Version *string `json:"version,omitempty"`
	// Num of cpus for the instance
	// +kubebuilder:validation:Minimum=1
	Cores *int32 `json:"cores,omitempty"`
	// CoreLimit specifies a hard limit on CPU cores for the pod.
	// +Optional
	CoreLimit *string `json:"coreLimit,omitempty"`
	// Memory is the amount of memory to request for the pod.
	// +Optional.
	Memory *string `json:"memory,omitempty"`
	// EnvFrom is a list of sources to populate environment variables in the container.
	// +optional
	EnvFrom []apiv1.EnvFromSource `json:"envFrom,omitempty"`
	// Labels are the Kubernetes labels to be added to the pod.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// Image is the container image to use. Overrides Spec.Image if set.
	// +optional
	Image *string `json:"image,omitempty"`
	// ImagePullPolicy is the image pull policy for the driver, executor, and init-container.
	// +Optional.
	ImagePullPolicy *string `json:"imagePullPolicy,omitempty"`
	// Priority range from 1 - 10 inclusive, higher priority means more workload will be distributed to the instance
	// +Optional.
	Priority *string `json:"priority,omitempty"`
	Namespace *string	`json:"namespace,omitempty"`
}

// DatafuseComputeInstanceStatus defines the observed state of DatafuseComputeInstance
type DatafuseComputeInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status	ComputeInstanceState	`json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatafuseComputeInstance is the Schema for the datafusecomputeinstances API
type DatafuseComputeInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatafuseComputeInstanceSpec   `json:"spec,omitempty"`
	Status DatafuseComputeInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatafuseComputeInstanceList contains a list of DatafuseComputeInstance
type DatafuseComputeInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatafuseComputeInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatafuseComputeInstance{}, &DatafuseComputeInstanceList{})
}
