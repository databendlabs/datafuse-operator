// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package framework

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	crdclientset "datafuselabs.io/datafuse-operator/pkg/client/clientset/versioned"
	datafuseutils "datafuselabs.io/datafuse-operator/utils"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	apiclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

// Framework contains all components needed to run e2e tests
// Framework contains all components required to run the test framework.
type Framework struct {
	KubeClient      kubernetes.Interface
	Client          crdclientset.Interface
	OperatorPod     *v1.Pod
	APIServerClient apiclient.Interface
	DefaultTimeout  time.Duration
	MasterHost      string
}

func New(kubeconfig, opImage string) (*Framework, error) {
	datafuseutils.KubeConfig = kubeconfig
	config, err := datafuseutils.GetK8sConfig()
	if err != nil {
		return nil, errors.Wrap(err, "cannot fetch kube-config")
	}
	k8sCli, err := datafuseutils.GetK8sClient()
	if err != nil {
		return nil, errors.Wrap(err, "creating new kube-client failed")
	}
	apiCli, err := apiclient.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating new kube-client failed")
	}

	client, err := crdclientset.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating new datafuse client failed")
	}

	f := &Framework{
		MasterHost:      config.Host,
		KubeClient:      k8sCli,
		Client:          client,
		APIServerClient: apiCli,
		DefaultTimeout:  5 * time.Minute,
	}

	return f, nil
}

func (f *Framework) CreateDatafuseCRDs() error {
	op, err := f.MakeCRD("../../manifests/crds/datafuse.datafuselabs.io_datafuseoperators.yaml")
	if err != nil {
		return fmt.Errorf("cannot retrieve operator CRD, %+v", err)
	}
	cg, err := f.MakeCRD("../../manifests/crds/datafuse.datafuselabs.io_datafusecomputegroups.yaml")
	if err != nil {
		return fmt.Errorf("cannot retrieve compute group CRD, %+v", err)
	}
	err = f.CreateCRD(op)
	if err != nil {
		return fmt.Errorf("cannot create operator CRD, %+v", err)
	}
	err = WaitForCRDReady(func(opts metav1.ListOptions) (runtime.Object, error) {
		return f.Client.DatafuseV1alpha1().DatafuseOperators(v1.NamespaceAll).List(context.TODO(), opts)
	})
	if err != nil {
		return err
	}

	err = f.CreateCRD(cg)
	if err != nil {
		return fmt.Errorf("cannot create compute group CRD, %+v", err)
	}
	err = WaitForCRDReady(func(opts metav1.ListOptions) (runtime.Object, error) {
		return f.Client.DatafuseV1alpha1().DatafuseComputeGroups(v1.NamespaceAll).List(context.TODO(), opts)
	})
	if err != nil {
		return err
	}

	return nil
}

func (f *Framework) Setup(namespace, opImage string, opImagePullPolicy string) error {
	if err := f.CreateDatafuseCRDs(); err != nil {
		return err
	}
	if err := f.setupOperator(namespace, opImage, opImagePullPolicy); err != nil {
		return errors.Wrap(err, "setup operator failed")
	}

	return nil
}

func (f *Framework) setupOperator(namespace, opImage string, opImagePullPolicy string) error {
	if _, err := CreateServiceAccount(f.KubeClient, namespace, "../../manifests/datafuse-rbac.yaml"); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create service account")
	}
	if err := CreateClusterRole(f.KubeClient, "../../manifests/datafuse-rbac.yaml"); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create cluster role")
	}

	if _, err := CreateClusterRoleBinding(f.KubeClient, namespace, "../../manifests/datafuse-rbac.yaml"); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create cluster role binding")
	}

	if err := CreateRole(f.KubeClient, namespace, "../../manifests/datafuse-rbac.yaml"); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create role")
	}

	if _, err := CreateRoleBinding(f.KubeClient, namespace, "../../manifests/datafuse-rbac.yaml"); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create role binding")
	}

	deploy, err := MakeDeployment("../../manifests/datafuse-operator.yaml")
	if err != nil {
		return err
	}
	deploy.Namespace = namespace
	if opImage != "" {
		// Override operator image used, if specified when running tests.
		deploy.Spec.Template.Spec.Containers[0].Image = opImage
	}

	for _, container := range deploy.Spec.Template.Spec.Containers {
		container.ImagePullPolicy = v1.PullPolicy(opImagePullPolicy)
	}

	err = CreateDeployment(f.KubeClient, namespace, deploy)
	if err != nil {
		return err
	}

	opts := metav1.ListOptions{LabelSelector: fields.SelectorFromSet(fields.Set(deploy.Spec.Template.ObjectMeta.Labels)).String()}
	err = WaitForPodsReady(f.KubeClient, namespace, f.DefaultTimeout, 1, opts)
	if err != nil {
		return errors.Wrap(err, "failed to wait for operator to become ready")
	}

	pl, err := f.KubeClient.CoreV1().Pods(namespace).List(context.TODO(), opts)
	if err != nil {
		return err
	}
	f.OperatorPod = &pl.Items[0]
	return nil
}

// Teardown tears down a previously initialized test environment.
func (f *Framework) Teardown(ns string) error {
	if err := DeleteClusterRole(f.KubeClient, "../../manifest/spark-operator-rbac.yaml"); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to delete operator cluster role")
	}

	if err := DeleteClusterRoleBinding(f.KubeClient, "../../manifest/spark-operator-rbac.yaml"); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to delete operator cluster role binding")
	}

	if err := f.KubeClient.AppsV1().Deployments(ns).Delete(context.TODO(), "datafuse-controller-manager", metav1.DeleteOptions{}); err != nil {
		return err
	}

	if err := DeleteNamespace(f.KubeClient, ns); err != nil {
		return err
	}

	return nil
}

type FinalizerFn func() error

type TestCtx struct {
	ID         string
	cleanUpFns []FinalizerFn
}

func (f *Framework) NewTestCtx(t *testing.T) TestCtx {
	return TestCtx{ID: GenerateTestID(time.Now().Unix(), t)}
}

func GenerateTestID(timestamp int64, t *testing.T) string {
	prefix := strings.TrimPrefix(
		strings.Replace(
			strings.ToLower(t.Name()),
			"/",
			"-",
			-1,
		),
		"test",
	)
	return prefix + "-" + strconv.FormatInt(int64(timestamp), 36)
}

// GetObjID returns an ascending ID based on the length of cleanUpFns. It is
// based on the premise that every new object also appends a new finalizerFn on
// cleanUpFns. This can e.g. be used to create multiple namespaces in the same
// test context.
func (ctx *TestCtx) GetObjID() string {
	return ctx.ID + "-" + strconv.Itoa(len(ctx.cleanUpFns))
}

func (ctx *TestCtx) Cleanup(t *testing.T) {
	var eg errgroup.Group

	for i := len(ctx.cleanUpFns) - 1; i >= 0; i-- {
		eg.Go(ctx.cleanUpFns[i])
	}

	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}
}

func (ctx *TestCtx) AddFinalizerFn(fn FinalizerFn) {
	ctx.cleanUpFns = append(ctx.cleanUpFns, fn)
}
