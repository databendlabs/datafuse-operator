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
package v1alpha1

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatafuseComputeSetSpec defines the desired state of DatafuseComputeSet
type DatafuseComputeSetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Number of compute instances
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:validation:Default=1
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Optional
	Replicas *int32 `json:"replicas,omitempty"`
	// Template for each idempotent instances
	DatafuseComputeInstanceSpec `json:",inline"`
}

// DatafuseComputeSetStatus defines the observed state of DatafuseComputeSet
type DatafuseComputeSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Replicas        int32                           `json:"replicas,omitempty"`
	Selector        string                          `json:"selector,omitempty"`       // this must be the string form of the selector
	InstancesStatus map[string]ComputeInstanceState `json:"instancestatus,omitempty"` // map from compute instance pod name to state
}
