package e2e

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	operatorFramework "datafuselabs.io/datafuse-operator/tests/framework"
	"datafuselabs.io/datafuse-operator/tests/utils/retry"
	sql_utils "datafuselabs.io/datafuse-operator/tests/utils/sql"
	"datafuselabs.io/datafuse-operator/utils"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/util/podutils"
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
	// t.Run("simple", testSimpleDeploy)
	t.Run("multi-worker", testMultiDeploy)
}

func testMultiDeploy(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup(t)
	ns := ctx.CreateNamespace(t, framework.KubeClient)
	t.Logf("current deploy namespace %s", ns)
	op, err := operatorFramework.MakeDatafuseOperatorFromYaml("./testfiles/leader_with_2workers.yaml")
	assert.NoError(t, err)
	op.Namespace = ns
	op.Spec.ComputeGroups[0].Namespace = ns
	err = operatorFramework.CreateDatafuseOperator(framework.Client, ns, op)
	assert.NoError(t, err)
	retry.UntilSuccessOrFail(t, func() error {
		_, err = framework.KubeClient.CoreV1().Services(ns).Get(context.TODO(), "multi-worker", v1.GetOptions{})
		if err != nil {
			return err
		}
		_, err := framework.KubeClient.AppsV1().Deployments(ns).Get(context.TODO(), "multi-worker-leader", v1.GetOptions{})
		if err != nil {
			return err
		}
		_, err = framework.KubeClient.AppsV1().Deployments(ns).Get(context.TODO(), "t1", v1.GetOptions{})
		if err != nil {
			return err
		}
		_, err = framework.KubeClient.AppsV1().Deployments(ns).Get(context.TODO(), "t2", v1.GetOptions{})
		if err != nil {
			return err
		}
		return nil
	}, *retry.NewConfig().SetCoverage(1).SetInterval(1 * time.Second).SetTimeOut(5 * time.Minute))

	// test on use different clients to query on current service
	url := fmt.Sprintf("multi-worker.%s.svc.cluster.local", ns)
	t.Logf("Test on mysql client integration")
	testClientIntegration(t, ns, "MySQL", url, "3306", "./sqlfiles", "./sqlfiles/mysql")
	t.Logf("Test on clickhouse client integration")
	testClientIntegration(t, ns, "ClickHouse", url, "9000", "./sqlfiles", "./sqlfiles/clickhouse")
	validateQueries(t, "MySQL", url, "3306", "mysql-client", "mysql-client", ns, "./validate", "./", false)
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
		_, err = framework.KubeClient.CoreV1().Services(ns).Get(context.TODO(), "group1", v1.GetOptions{})
		if err != nil {
			return err
		}
		_, err := framework.KubeClient.AppsV1().Deployments(ns).Get(context.TODO(), "group1-leader", v1.GetOptions{})
		if err != nil {
			return err
		}
		return nil
	}, *retry.NewConfig().SetCoverage(1).SetInterval(1 * time.Second).SetTimeOut(5 * time.Minute))

	// test on use different clients to query on current service
	url := fmt.Sprintf("group1.%s.svc.cluster.local", ns)
	t.Logf("Test on mysql client integration")
	testClientIntegration(t, ns, "MySQL", url, "3306", "./sqlfiles", "./sqlfiles/mysql")
	t.Logf("Test on clickhouse client integration")
	testClientIntegration(t, ns, "ClickHouse", url, "9000", "./sqlfiles", "./sqlfiles/clickhouse")
}

// will query on url:port given namespace and desired client
func testClientIntegration(t *testing.T, ns, client, url, port, queryDir, resultDir string) {
	switch client {
	case "MySQL":
		err := createClient(t, "./clients/mysql-client.yaml", "mysql-client", "mysql-client", ns)
		assert.NoError(t, err)
		err = validateQueries(t, "MySQL", url, port, "mysql-client", "mysql-client", ns, queryDir, resultDir, true)
		assert.NoError(t, err)
	case "ClickHouse":
		err := createClient(t, "./clients/clickhouse-client.yaml", "clickhouse-client", "clickhouse-client", ns)
		assert.NoError(t, err)
		err = validateQueries(t, "ClickHouse", url, port, "clickhouse-client", "clickhouse-client", ns, queryDir, resultDir, true)
		assert.NoError(t, err)
	default:
		t.Fatal("Unimplemented client interface")
	}
}

func createClient(t *testing.T, pathToYaml, containerName, podName, ns string) error {
	mysql, err := operatorFramework.MakePod(pathToYaml)
	assert.NoError(t, err)
	err = operatorFramework.CreatePod(framework.KubeClient, ns, mysql)
	assert.NoError(t, err)
	retry.UntilSuccessOrFail(t, func() error {
		pod, err := framework.KubeClient.CoreV1().Pods(ns).Get(context.TODO(), podName, v1.GetOptions{})
		if err != nil {
			return err
		}
		if !podutils.IsPodReady(pod) {
			return fmt.Errorf("pod %s in %s is not ready", pod.Name, pod.Namespace)
		}
		return nil
	}, *retry.NewConfig().SetCoverage(1).SetInterval(1 * time.Second).SetTimeOut(5 * time.Minute))
	return nil
}

func validateQueries(t *testing.T, client, url, port, containerName, podName, ns, queryDir, resultDir string, validate bool) error {
	var queryCommand func(query string) []string
	switch client {
	case "MySQL":
		queryCommand = func(query string) []string {
			return []string{"mysql", "-h", url, "-P", port, "-e", query}
		}
	case "ClickHouse":
		queryCommand = func(query string) []string {
			return []string{"clickhouse-client", "-h", url, "--port", port, "--query", query}
		}
	}
	pairs, err := sql_utils.BuildSQLPair(`%s`, queryDir, resultDir)
	assert.NoError(t, err)
	for key, pair := range pairs {
		t.Logf("Testing on query file %s", key)
		execOptions := framework.MakeExecOptions(containerName, podName, ns,
			queryCommand(pair.Query))
		stdOut, stdErr, err := framework.ExecWithOptions(execOptions)
		assert.NoError(t, err)
		t.Logf("test in %s with std error %s", key, stdErr)
		if validate {
			assert.Equal(t, pair.Result, stdOut)
		} else {
			t.Logf("test in %s has result %s", key, stdOut)
		}
	}
	return nil
}
