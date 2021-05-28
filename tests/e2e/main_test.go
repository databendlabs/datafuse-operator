package e2e

import (
	"flag"
	"log"
	"os"
	"testing"

	operatorFramework "datafuselabs.io/datafuse-operator/tests/framework"
	"datafuselabs.io/datafuse-operator/utils"
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

// func TestAllNS(t *testing.T) {
// 	ctx := framework.NewTestCtx(t)
// 	defer ctx.Cleanup(t)
// 	ns := ctx.CreateNamespace(t, framework.KubeClient)
// 	err := framework.Setup(ns, *opImage, "Always")
// 	assert.NoError(t, err)
// }
