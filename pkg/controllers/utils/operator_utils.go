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

package utils

import (
	"fmt"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	client "k8s.io/client-go/kubernetes"

	"datafuselabs.io/datafuse-operator/pkg/apis/datafuse/v1alpha1"
	crdclientset "datafuselabs.io/datafuse-operator/pkg/client/clientset/versioned"
	controller "datafuselabs.io/datafuse-operator/pkg/controllers"
)

type OperatorSetter struct {
	K8sClient client.Interface
	Client    crdclientset.Interface
	AllNS     bool
	Namspaces []string
}

func IsOperatorCreated(op *v1alpha1.DatafuseOperator) bool {
	if op.Status.Status == "" {
		log.Debug().Msgf("defaulting operator status to Created")
		op.Status.Status = v1alpha1.OperatorCreated
	}
	return op.Status.Status == v1alpha1.OperatorCreated
}

func IsComputeGroupCreated(op *v1alpha1.DatafuseComputeGroup) bool {
	if op.Status.Status == "" {
		log.Debug().Msgf("defaulting operator status to Created")
		op.Status.Status = v1alpha1.ComputeGroupCreated
	}
	return op.Status.Status == v1alpha1.ComputeGroupCreated
}

func makePorts(instanceSpec v1alpha1.DatafuseComputeInstanceSpec) []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			Name:          controller.ContainerHTTPPort,
			ContainerPort: *instanceSpec.HTTPPort,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          controller.ContainerMysqlPort,
			ContainerPort: *instanceSpec.MysqlPort,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          controller.ContainerClickhousePort,
			ContainerPort: *instanceSpec.ClickhousePort,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          controller.ContainerRPCPort,
			ContainerPort: *instanceSpec.RPCPort,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          controller.ContainerMetricsPort,
			ContainerPort: *instanceSpec.MetricsPort,
			Protocol:      corev1.ProtocolTCP,
		},
	}
}

func makeServicePorts(d *appsv1.Deployment) []corev1.ServicePort {
	ports := []corev1.ServicePort{}
	for _, port := range d.Spec.Template.Spec.Containers[0].Ports {
		switch port.Name {
		case controller.ContainerMetricsPort,
			controller.ContainerHTTPPort,
			controller.ContainerRPCPort,
			controller.ContainerMysqlPort,
			controller.ContainerClickhousePort:
			ports = append(ports, corev1.ServicePort{Name: port.Name,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(int(port.ContainerPort)),
				Port:       port.ContainerPort})
		default:
			continue
		}
	}
	return ports
}

func makeEnvs(instanceSpec v1alpha1.DatafuseComputeInstanceSpec) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  controller.FUSE_QUERY_MYSQL_HANDLER_HOST,
			Value: "0.0.0.0",
		},
		{
			Name:  controller.FUSE_QUERY_MYSQL_HANDLER_PORT,
			Value: fmt.Sprintf("%d", *instanceSpec.MysqlPort),
		},
		{
			Name:  controller.FUSE_QUERY_CLICKHOUSE_HANDLER_HOST,
			Value: "0.0.0.0",
		},
		{
			Name:  controller.FUSE_QUERY_CLICKHOUSE_HANDLER_PORT,
			Value: fmt.Sprintf("%d", *instanceSpec.ClickhousePort),
		},
		{
			Name:  controller.FUSE_QUERY_HTTP_API_ADDRESS,
			Value: fmt.Sprintf("0.0.0.0:%d", *instanceSpec.HTTPPort),
		},
		{
			Name:  controller.FUSE_QUERY_METRIC_API_ADDRESS,
			Value: fmt.Sprintf("0.0.0.0:%d", *instanceSpec.MetricsPort),
		},
		{
			Name:  controller.FUSE_QUERY_RPC_API_ADDRESS,
			Value: fmt.Sprintf("0.0.0.0:%d", *instanceSpec.RPCPort),
		},
		{
			Name:  controller.FUSE_QUERY_PRIORITY,
			Value: fmt.Sprintf("%d", *instanceSpec.Priority),
		},
	}
}

func makeProbe(path, portName string) *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   path,
				Port:   intstr.Parse(portName),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		PeriodSeconds:    10,
		SuccessThreshold: 1,
		FailureThreshold: 3,
		TimeoutSeconds:   1,
	}
}

func getMilliCPU(cores *int32) int64 {
	if cores == nil {
		// default is to allocate one cpu
		return 1200
	}
	return int64(*cores)*1000 + 300 //allocate some additional resources to avoid cpu race condition
}

func makeResourceRequirements(instanceSpec v1alpha1.DatafuseComputeInstanceSpec) corev1.ResourceRequirements {
	fillblank := func(input *string, quant resource.Quantity) resource.Quantity {
		if input == nil {
			return quant
		}
		return resource.MustParse(*input)
	}
	rq := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(getMilliCPU(instanceSpec.Cores), resource.DecimalSI),
			corev1.ResourceMemory: resource.MustParse(*instanceSpec.Memory),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    fillblank(instanceSpec.CoreLimit, *resource.NewMilliQuantity(getMilliCPU(instanceSpec.Cores), resource.DecimalSI)),
			corev1.ResourceMemory: resource.MustParse(*instanceSpec.MemoryLimit),
		},
	}
	return rq
}

func convertInstanceSpecToPodSpec(instanceSpec v1alpha1.DatafuseComputeInstanceSpec) corev1.PodSpec {
	return corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            controller.InstanceContainerName,
				Image:           *instanceSpec.Image,
				ImagePullPolicy: corev1.PullPolicy(*instanceSpec.ImagePullPolicy),
				Env:             makeEnvs(instanceSpec),
				Ports:           makePorts(instanceSpec),
				LivenessProbe:   makeProbe("/v1/hello", controller.ContainerHTTPPort),
				ReadinessProbe:  makeProbe("/v1/configs", controller.ContainerHTTPPort),
				Resources:       makeResourceRequirements(instanceSpec),
				Args:            []string{},
			},
		},
	}
}

func MakeFuseQueryPod(computeSet *v1alpha1.DatafuseComputeSetSpec, name, namespace, groupKey string, isLeader bool) *corev1.Pod {
	leaderString := func(isLeader bool) string {
		if isLeader {
			return controller.ComputeGroupRoleLeader
		} else {
			return controller.ComputeGroupRoleFollower
		}
	}
	labels := map[string]string{
		controller.ComputeGroupLabel:     groupKey,
		controller.ComputeGroupRoleLabel: leaderString(isLeader),
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: convertInstanceSpecToPodSpec(computeSet.DatafuseComputeInstanceSpec),
	}
}

func MakeService(name string, deployment *appsv1.Deployment) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       deployment.Namespace,
			Labels:          deployment.Labels,
			OwnerReferences: deployment.OwnerReferences,
		},
		Spec: corev1.ServiceSpec{
			Selector: deployment.Labels,
			Ports:    makeServicePorts(deployment),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

// make a deployment based
func MakeDeployment(computeSet *v1alpha1.DatafuseComputeSetSpec, name, namespace, operatorKey, groupKey string, isLeader bool) *appsv1.Deployment {
	leaderString := func(isLeader bool) string {
		if isLeader {
			return controller.ComputeGroupRoleLeader
		} else {
			return controller.ComputeGroupRoleFollower
		}
	}
	labels := map[string]string{
		controller.OperatorLabel:         operatorKey,
		controller.ComputeGroupLabel:     groupKey,
		controller.ComputeGroupRoleLabel: leaderString(isLeader),
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: computeSet.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: convertInstanceSpecToPodSpec(computeSet.DatafuseComputeInstanceSpec),
			},
		},
	}
}

func AddDeployOwnership(deploy *v1.Deployment, owner *v1alpha1.DatafuseComputeGroup) {
	if deploy.OwnerReferences == nil {
		deploy.OwnerReferences = []metav1.OwnerReference{}
	}
	if metav1.IsControlledBy(deploy, owner) {
		return
	}
	deploy.OwnerReferences = append(deploy.OwnerReferences, *metav1.NewControllerRef(owner, v1alpha1.SchemeGroupVersion.WithKind("DatafuseComputeGroup")))
}

// make a deployment based
func MakeComputeGroup(owner *v1alpha1.DatafuseOperator, groupSpec v1alpha1.DatafuseComputeGroupSpec) *v1alpha1.DatafuseComputeGroup {
	group := &v1alpha1.DatafuseComputeGroup{}
	group.SetName(groupSpec.Name)
	group.SetNamespace(groupSpec.Namespace)
	group.Name = groupSpec.Name
	group.Namespace = groupSpec.Namespace
	opKey := GetOperatorKey(owner.ObjectMeta)
	group.Labels = make(map[string]string)
	group.Labels[controller.OperatorLabel] = opKey
	v1alpha1.SetDatafuseComputeGroupDefaults(owner, group)
	group.Spec = groupSpec
	return group
}

func GetOperatorKey(owner metav1.ObjectMeta) string {
	return fmt.Sprintf("%s-%s", owner.Namespace, owner.Name)
}
