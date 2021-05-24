package e2e

import (
	"testing"
)

var (
	crdLocation string = "../../config/crd/bases"
	crdNames           = []string{"datafusecomputeinstances.datafuse.datafuselabs.io", "datafusecomputegroups.datafuse.datafuselabs.io", "datafuseoperators.datafuse.datafuselabs.io", "datafusecomputesets.datafuse.datafuselabs.io"}
)

// func TestCRDInstall(t *testing.T) {
// 	err := framework.InstallCRDs()
// 	assert.NoError(t, err)
// 	// kubeConfig := ctrl.GetConfigOrDie()
// 	// externClient := apiextensionsclient.NewForConfigOrDie(kubeConfig)
// 	// for _, crdName := range crdNames {
// 	// 	if _, err = externClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), crdName, metav1.GetOptions{}); err != nil {
// 	// 		t.Fatalf("resource %s not deployed, %v", crdName, err)
// 	// 	}
// 	// }

// }

func TestMakeDatafuseOperator(t *testing.T) {
	// kubeConfig := ctrl.GetConfigOrDie()
	// k8sClient := kubernetes.NewForConfigOrDie(kubeConfig)
	// Client := crdclientset.NewForConfigOrDie(kubeConfig)
	// op, err := framework.MakeDatafuseOperatorFromYaml("./testfiles/default_operator.yaml")
	// assert.NoError(t, err)
	// assert.Equal(t, op.Name, "datafusecluster-default")
	// assert.Equal(t, op.Namespace, "default")
	// assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.Image, convert.StringToPtr("zhihanz/fuse-query:latest"))
	// assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.ImagePullPolicy, convert.StringToPtr("Always"))
	// assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.ClickhousePort, convert.Int32ToPtr(9000))
	// assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.MysqlPort, convert.Int32ToPtr(3306))
	// assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.RPCPort, convert.Int32ToPtr(9091))
	// assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.HTTPPort, convert.Int32ToPtr(8081))
	// assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.Cores, convert.Int32ToPtr(1))
	// assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.CoreLimit, convert.Int32ToPtr(2))
	// assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.Memory, convert.Int32ToPtr(512))
	// assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.MemoryLimit, convert.Int32ToPtr(1024))
	// assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.Replicas, convert.Int32ToPtr(1))
}

// create a sample deployment and deploy it into local cluster
func TestSampleDeployment(t *testing.T) {
	// kubeConfig := ctrl.GetConfigOrDie()
	// k8sClient := kubernetes.NewForConfigOrDie(kubeConfig)
	// _ = crdclientset.NewForConfigOrDie(kubeConfig)
	// op, err := framework.MakeDatafuseOperatorFromYaml("./testfiles/default_operator.yaml")
	// // deployment, err := framework.MakeDeployment("./testfiles/default_generated_deployment.yaml")
	// assert.NoError(t, err)
	// opKey := fmt.Sprintf("%s-%s", op.Namespace, op.Name)
	// groupKey := fmt.Sprintf("%s-%s", op.Spec.ComputeGroups[0].Namespace, op.Spec.ComputeGroups[0].Name)
	// deploy := controller.MakeDeployment(op, op.Spec.ComputeGroups[0].ComputeLeaders,
	// 	"group1-leader-1", op.Namespace, opKey, groupKey, true)
	// _, err = k8sClient.AppsV1().Deployments("default").Create(context.TODO(), deploy, metav1.CreateOptions{})
	// // _, err = k8sClient.AppsV1().Deployments("default").Create(context.TODO(), deployment, metav1.CreateOptions{})
	// assert.NoError(t, err)
	// t.Fatalf("")
}
