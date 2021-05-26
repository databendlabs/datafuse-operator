// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package framework

import (
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
	"k8s.io/client-go/kubernetes"
)

// Framework contains all components needed to run e2e tests
// Framework contains all components required to run the test framework.
type Framework struct {
	KubeClient      kubernetes.Interface
	Client          crdclientset.Interface
	Namespace       *v1.Namespace
	OperatorPod     *v1.Pod
	APIServerClient apiclient.Interface
	DefaultTimeout  time.Duration
	MasterHost      string
}

func New(ns, sparkNs, kubeconfig, opImage string, opImagePullPolicy string) (*Framework, error) {
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
	namespace, err := CreateNamespace(k8sCli, ns)
	if err != nil {
		fmt.Println(nil, err, namespace)
	}

	f := &Framework{
		MasterHost:      config.Host,
		KubeClient:      k8sCli,
		Client:          client,
		Namespace:       namespace,
		APIServerClient: apiCli,
		DefaultTimeout:  5 * time.Minute,
	}

	return f, nil
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
