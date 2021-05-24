package utils

import (
	"fmt"
	"testing"

	controller "datafuselabs.io/datafuse-operator/pkg/controllers"
	"datafuselabs.io/datafuse-operator/tests/framework"
	"datafuselabs.io/datafuse-operator/tests/utils/convert"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestSampleDeployment(t *testing.T) {
	op, err := framework.MakeDatafuseOperatorFromYaml("../../../tests/e2e/testfiles/default_operator.yaml")
	assert.NoError(t, err)
	opKey := fmt.Sprintf("%s-%s", op.Namespace, op.Name)
	groupKey := fmt.Sprintf("%s-%s", op.Spec.ComputeGroups[0].Namespace, op.Spec.ComputeGroups[0].Name)
	deploy := MakeDeployment(op.Spec.ComputeGroups[0].ComputeLeaders,
		"group1-leader-1", op.Spec.ComputeGroups[0].Namespace, opKey, groupKey, true)
	desiredLabels := map[string]string{
		controller.OperatorLabel:         opKey,
		controller.ComputeGroupLabel:     groupKey,
		controller.ComputeGroupRoleLabel: controller.ComputeGroupRoleLeader,
	}
	desiredEnvs := []v1.EnvVar{
		{
			Name:  controller.FUSE_QUERY_MYSQL_HANDLER_HOST,
			Value: "0.0.0.0",
		},
		{
			Name:  controller.FUSE_QUERY_MYSQL_HANDLER_PORT,
			Value: "3306",
		},
		{
			Name:  controller.FUSE_QUERY_CLICKHOUSE_HANDLER_HOST,
			Value: "0.0.0.0",
		},
		{
			Name:  controller.FUSE_QUERY_CLICKHOUSE_HANDLER_PORT,
			Value: "9000",
		},
		{
			Name:  controller.FUSE_QUERY_HTTP_API_ADDRESS,
			Value: "0.0.0.0:8080",
		},
		{
			Name:  controller.FUSE_QUERY_METRIC_API_ADDRESS,
			Value: "0.0.0.0:9098",
		},
		{
			Name:  controller.FUSE_QUERY_RPC_API_ADDRESS,
			Value: "0.0.0.0:9091",
		},
		{
			Name:  controller.FUSE_QUERY_PRIORITY,
			Value: "1",
		},
	}

	desiredResourceRequirements := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(1300, resource.DecimalSI),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1300m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}
	desiredLivenessProbe := &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/v1/hello",
				Port:   intstr.Parse(controller.ContainerHTTPPort),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		PeriodSeconds:    10,
		SuccessThreshold: 1,
		FailureThreshold: 3,
		TimeoutSeconds:   1,
	}
	desiredReadinessProbe := &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/v1/configs",
				Port:   intstr.Parse(controller.ContainerHTTPPort),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		PeriodSeconds:    10,
		SuccessThreshold: 1,
		FailureThreshold: 3,
		TimeoutSeconds:   1,
	}
	desiredPorts := []v1.ContainerPort{
		{
			Name:          controller.ContainerHTTPPort,
			ContainerPort: 8080,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          controller.ContainerMysqlPort,
			ContainerPort: 3306,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          controller.ContainerClickhousePort,
			ContainerPort: 9000,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          controller.ContainerRPCPort,
			ContainerPort: 9091,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          controller.ContainerMetricsPort,
			ContainerPort: 9098,
			Protocol:      corev1.ProtocolTCP,
		},
	}
	assert.Equal(t, deploy.Name, "group1-leader-1")
	assert.Equal(t, deploy.Namespace, op.Spec.ComputeGroups[0].Namespace)
	assert.Equal(t, deploy.Labels, desiredLabels)
	assert.Equal(t, deploy.Spec.Replicas, convert.Int32ToPtr(1))
	assert.Equal(t, deploy.Spec.Selector, &metav1.LabelSelector{MatchLabels: desiredLabels})
	assert.Equal(t, deploy.Spec.Template.Labels, desiredLabels)
	assert.Equal(t, deploy.Spec.Template.Spec.Containers[0].Env, desiredEnvs)
	assert.Equal(t, deploy.Spec.Template.Spec.Containers[0].Resources, desiredResourceRequirements)
	assert.Equal(t, deploy.Spec.Template.Spec.Containers[0].LivenessProbe, desiredLivenessProbe)
	assert.Equal(t, deploy.Spec.Template.Spec.Containers[0].ReadinessProbe, desiredReadinessProbe)
	assert.Equal(t, deploy.Spec.Template.Spec.Containers[0].Ports, desiredPorts)
}

func TestMakeComputeGroup(t *testing.T) {
	op, err := framework.MakeDatafuseOperatorFromYaml("../../../tests/e2e/testfiles/default_operator.yaml")
	assert.NoError(t, err)
	opKey := fmt.Sprintf("%s-%s", op.Namespace, op.Name)

	groupSpec := op.Spec.ComputeGroups[0]
	assert.NotNil(t, groupSpec)
	group := MakeComputeGroup(op, *groupSpec)
	assert.Equal(t, group.Name, groupSpec.Name)
	assert.Equal(t, group.Namespace, groupSpec.Namespace)
	assert.Equal(t, group.Labels, map[string]string{controller.OperatorLabel: opKey})
	assert.Equal(t, metav1.IsControlledBy(group, op), true)
	assert.Equal(t, group.Spec, *groupSpec)
}

func TestMakeFuseQueryPod(t *testing.T) {
	op, err := framework.MakeDatafuseOperatorFromYaml("../../../tests/e2e/testfiles/default_operator.yaml")
	assert.NoError(t, err)
	groupKey := fmt.Sprintf("%s-%s", op.Spec.ComputeGroups[0].Namespace, op.Spec.ComputeGroups[0].Name)
	pod := MakeFuseQueryPod(op.Spec.ComputeGroups[0].ComputeLeaders, "group1-leader-1-123",
		op.Spec.ComputeGroups[0].Namespace, groupKey, true)
	desiredLabels := map[string]string{
		controller.ComputeGroupLabel:     groupKey,
		controller.ComputeGroupRoleLabel: controller.ComputeGroupRoleLeader,
	}
	desiredEnvs := []v1.EnvVar{
		{
			Name:  controller.FUSE_QUERY_MYSQL_HANDLER_HOST,
			Value: "0.0.0.0",
		},
		{
			Name:  controller.FUSE_QUERY_MYSQL_HANDLER_PORT,
			Value: "3306",
		},
		{
			Name:  controller.FUSE_QUERY_CLICKHOUSE_HANDLER_HOST,
			Value: "0.0.0.0",
		},
		{
			Name:  controller.FUSE_QUERY_CLICKHOUSE_HANDLER_PORT,
			Value: "9000",
		},
		{
			Name:  controller.FUSE_QUERY_HTTP_API_ADDRESS,
			Value: "0.0.0.0:8080",
		},
		{
			Name:  controller.FUSE_QUERY_METRIC_API_ADDRESS,
			Value: "0.0.0.0:9098",
		},
		{
			Name:  controller.FUSE_QUERY_RPC_API_ADDRESS,
			Value: "0.0.0.0:9091",
		},
		{
			Name:  controller.FUSE_QUERY_PRIORITY,
			Value: "1",
		},
	}

	desiredResourceRequirements := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(1300, resource.DecimalSI),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1300m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}
	desiredLivenessProbe := &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/v1/hello",
				Port:   intstr.Parse(controller.ContainerHTTPPort),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		PeriodSeconds:    10,
		SuccessThreshold: 1,
		FailureThreshold: 3,
		TimeoutSeconds:   1,
	}
	desiredReadinessProbe := &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/v1/configs",
				Port:   intstr.Parse(controller.ContainerHTTPPort),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		PeriodSeconds:    10,
		SuccessThreshold: 1,
		FailureThreshold: 3,
		TimeoutSeconds:   1,
	}
	desiredPorts := []v1.ContainerPort{
		{
			Name:          controller.ContainerHTTPPort,
			ContainerPort: 8080,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          controller.ContainerMysqlPort,
			ContainerPort: 3306,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          controller.ContainerClickhousePort,
			ContainerPort: 9000,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          controller.ContainerRPCPort,
			ContainerPort: 9091,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          controller.ContainerMetricsPort,
			ContainerPort: 9098,
			Protocol:      corev1.ProtocolTCP,
		},
	}
	assert.Equal(t, pod.Name, "group1-leader-1-123")
	assert.Equal(t, pod.Namespace, op.Spec.ComputeGroups[0].Namespace)
	assert.Equal(t, pod.Labels, desiredLabels)
	assert.Equal(t, pod.Labels, desiredLabels)
	assert.Equal(t, pod.Spec.Containers[0].Env, desiredEnvs)
	assert.Equal(t, pod.Spec.Containers[0].Resources, desiredResourceRequirements)
	assert.Equal(t, pod.Spec.Containers[0].LivenessProbe, desiredLivenessProbe)
	assert.Equal(t, pod.Spec.Containers[0].ReadinessProbe, desiredReadinessProbe)
	assert.Equal(t, pod.Spec.Containers[0].Ports, desiredPorts)
}
