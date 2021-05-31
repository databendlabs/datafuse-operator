package e2e

import (
	"context"
	"flag"
	"log"
	"os"
	"testing"
	"time"

	operatorFramework "datafuselabs.io/datafuse-operator/tests/framework"
	"datafuselabs.io/datafuse-operator/tests/utils/retry"
	"datafuselabs.io/datafuse-operator/utils"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	framework *operatorFramework.Framework
	opImage   *string
)

func stringPtr(s string) *string {
	return &s
}

func TestMain(m *testing.M) {
	kubeconfig := flag.String(
		"kubeconfig",
		"",
		"kube config path, e.g. $HOME/.kube/config",
	)
	opImage = flag.String(
		"operator-image",
		"",
		"operator image, e.g. docker.io/datafuselabs/datafuse-operator:latest",
	)
	flag.Parse()

	var (
		err      error
		exitCode int
	)

	if kubeconfig == nil {
		kubeconfig = stringPtr(utils.GetKubeConfigLocation())
	}
	if opImage == nil {
		opImage = stringPtr("docker.io/datafuselabs//datafuse-operator:latest")
	}

	if framework, err = operatorFramework.New(*kubeconfig, *opImage); err != nil {
		log.Printf("failed to setup framework: %v\n", err)
		os.Exit(1)
	}

	exitCode = m.Run()

	os.Exit(exitCode)
}

func TestAllNS(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup(t)
	ns := ctx.CreateNamespace(t, framework.KubeClient)
	err := framework.Setup(ns, *opImage, "Always")
	assert.NoError(t, err)
	t.Run("simple", testSimpleDeploy)
}

func testSimpleDeploy(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup(t)
	ns := ctx.CreateNamespace(t, framework.KubeClient)
	t.Logf("current deploy namespace %s", ns)
	op, err := operatorFramework.MakeDatafuseOperatorFromYaml("./testfiles/default_operator.yaml")
	assert.NoError(t, err)
	op.Namespace = ns
	op.Spec.ComputeGroups[0].Namespace = ns
	err = operatorFramework.CreateDatafuseOperator(framework.Client, ns, op)
	assert.NoError(t, err)
	retry.UntilSuccessOrFail(t, func() error {
		_, err := framework.KubeClient.AppsV1().Deployments(ns).Get(context.TODO(), "group1-leader", v1.GetOptions{})
		if err != nil {
			return err
		}
		return nil
	}, *retry.NewConfig().SetCoverage(1).SetInterval(1 * time.Second).SetTimeOut(5 * time.Minute))

}
