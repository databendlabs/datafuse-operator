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

type ComputeInstanceState string

const (
	ComputeInstanceReadyState    ComputeInstanceState = "READY"
	ComputeInstanceLivenessState ComputeInstanceState = "LIVE"
	ComputeInstancePendingState  ComputeInstanceState = "Pending"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatafuseComputeInstanceSpec defines the desired state of DatafuseComputeInstance
type DatafuseComputeInstanceSpec struct {
	// Name is the specific name of current instance
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Optional
	Name *string `json:"name"`
	// Num of cpus for the instance,
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	Cores *int32 `json:"cores"`
	// CoreLimit specifies a hard limit on CPU cores for the instance.
	// +kubebuilder:validation:Type=string
	// +kubebuilder:default="1200m"
	// +kubebuilder:validation:Optional
	CoreLimit *string `json:"coreLimit"`
	// Memory is the amount of memory to request for the instance.
	// +kubebuilder:validation:Type=string
	// +kubebuilder:default="512m"
	// +kubebuilder:validation:Optional
	Memory *string `json:"memory"`
	// MemoryLimit is the amount of memory limit for the instance. in MiB
	// +kubebuilder:validation:Type=string
	// +kubebuilder:default="512m"
	// +kubebuilder:validation:Optional
	MemoryLimit *string `json:"memorylimit"`
	// Labels are the Kubernetes labels to be added to the pod.
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels"`
	// Image is the container image to use. Overrides Spec.Image if set.
	// +kubebuilder:validation:Type=string
	// +kubebuilder:default="datafuselabs/fuse-query:latest"
	// +kubebuilder:validation:Optional
	Image *string `json:"image"`
	// ImagePullPolicy is the image pull policy for the driver, executor, and init-container.
	// +kubebuilder:validation:Type=string
	// +kubebuilder:default=Always
	// +kubebuilder:validation:Optional
	ImagePullPolicy *string `json:"imagePullPolicy"`
	// Priority range from 1 - 10 inclusive, higher priority means more workload will be distributed to the instance
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:validation:Optional
	Priority *int32 `json:"priority"`
	// Port open to Mysql Client connection
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:default=3307
	// +kubebuilder:validation:Optional
	MysqlPort *int32 `json:"mysqlPort"`
	// Port open to Clickhouse Client connection
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:default=9000
	// +kubebuilder:validation:Optional
	ClickhousePort *int32 `json:"clickhousePort"`
	// Port for warp HTTP connection, can get cluster infomation and support to add/remove port
	// We also use HTTP port for health check and readiness check
	//TODO(zhihanz) docs on readiness check difference between leaders and workers
	// example: https://github.com/datafuselabs/datafuse/blob/master/fusequery/example/cluster.sh
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:default=8080
	// +kubebuilder:validation:Optional
	HTTPPort *int32 `json:"httpPort"`
	// Port for gRPC communication
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:default=9090
	// +kubebuilder:validation:Optional
	RPCPort *int32 `json:"rpcPort"`
	// Port for metrics exporter
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:default=7070
	// +kubebuilder:validation:Optional
	MetricsPort *int32 `json:"metricsPort"`
}

// DatafuseComputeInstanceStatus defines the observed state of DatafuseComputeInstance
type DatafuseComputeInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status ComputeInstanceState `json:"status,omitempty"`
}
