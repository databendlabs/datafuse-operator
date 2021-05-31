// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package operator

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/diff"

	"datafuselabs.io/datafuse-operator/pkg/apis/datafuse/v1alpha1"
	crdclientset "datafuselabs.io/datafuse-operator/pkg/client/clientset/versioned"
	crdfake "datafuselabs.io/datafuse-operator/pkg/client/clientset/versioned/fake"
	crdinformers "datafuselabs.io/datafuse-operator/pkg/client/informers/externalversions"
	controller "datafuselabs.io/datafuse-operator/pkg/controllers"
	utils "datafuselabs.io/datafuse-operator/pkg/controllers/utils"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t *testing.T

	client     *crdfake.Clientset
	kubeclient *k8sfake.Clientset
	// Objects to put in the store.
	groupLister    []*v1alpha1.DatafuseComputeGroup
	operatorLister []*v1alpha1.DatafuseOperator

	servicesLister   []*corev1.Service
	deploymentLister []*appsv1.Deployment
	// Actions expected to happen on the client.
	kubeactions []core.Action
	actions     []core.Action
	// Objects from here preloaded into NewSimpleFake.
	kubeobjects []runtime.Object
	objects     []runtime.Object
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.objects = []runtime.Object{}
	f.kubeobjects = []runtime.Object{}
	return f
}

func fakeK8sClienset(deployments []*appsv1.Deployment, services []corev1.Service) (client kubernetes.Interface) {
	var csObjs []runtime.Object
	for _, op := range deployments {
		csObjs = append(csObjs, op.DeepCopy())
	}

	for _, svc := range services {
		csObjs = append(csObjs, svc.DeepCopy())
	}

	client = k8sfake.NewSimpleClientset(csObjs...)
	return client
}

func fakeClientset(operators []*v1alpha1.DatafuseOperator, computeGroups []*v1alpha1.DatafuseComputeGroup) (cs crdclientset.Interface) {
	var csObjs []runtime.Object
	for _, op := range operators {
		csObjs = append(csObjs, op.DeepCopy())
	}
	for _, g := range computeGroups {
		csObjs = append(csObjs, g.DeepCopy())
	}
	cs = crdfake.NewSimpleClientset(csObjs...)
	return cs
}

func newOperator(name string, computeGroups []*v1alpha1.DatafuseComputeGroupSpec) *v1alpha1.DatafuseOperator {
	return &v1alpha1.DatafuseOperator{
		TypeMeta: metav1.TypeMeta{APIVersion: v1alpha1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: v1alpha1.DatafuseOperatorSpec{
			ComputeGroups: computeGroups,
		},
	}
}

type workerSettings struct {
	replicas *int32
	priority *int32
}

func newComputeGroupSpec(name, namespace string, numOfLeaders *int32, workers []*workerSettings) *v1alpha1.DatafuseComputeGroupSpec {
	followers := []*v1alpha1.DatafuseComputeSetSpec{}
	for _, worker := range workers {
		followers = append(followers, &v1alpha1.DatafuseComputeSetSpec{
			Replicas:                    worker.replicas,
			DatafuseComputeInstanceSpec: defaultInstanceSpec(worker.priority),
		})
	}
	return &v1alpha1.DatafuseComputeGroupSpec{
		ComputeLeaders: &v1alpha1.DatafuseComputeSetSpec{
			Replicas:                    numOfLeaders,
			DatafuseComputeInstanceSpec: defaultInstanceSpec(int32Ptr(1)),
		},
		ComputeWorkers: followers,
		Name:           name,
		Namespace:      namespace,
	}
}

func newComputeGroup(name, namespace string, numOfLeaders *int32, workers []*workerSettings) *v1alpha1.DatafuseComputeGroup {
	followers := []*v1alpha1.DatafuseComputeSetSpec{}
	for _, worker := range workers {
		followers = append(followers, &v1alpha1.DatafuseComputeSetSpec{
			Replicas:                    worker.replicas,
			DatafuseComputeInstanceSpec: defaultInstanceSpec(worker.priority),
		})
	}
	return &v1alpha1.DatafuseComputeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.DatafuseComputeGroupSpec{
			ComputeLeaders: &v1alpha1.DatafuseComputeSetSpec{
				Replicas:                    numOfLeaders,
				DatafuseComputeInstanceSpec: defaultInstanceSpec(int32Ptr(1)),
			},
			ComputeWorkers: followers,
			Name:           name,
			Namespace:      namespace,
		},
	}
}

func defaultInstanceSpec(priority *int32) v1alpha1.DatafuseComputeInstanceSpec {
	return v1alpha1.DatafuseComputeInstanceSpec{
		Cores:           int32Ptr(1),
		CoreLimit:       strPtr("1300m"),
		Memory:          strPtr("512m"),
		MemoryLimit:     strPtr("512m"),
		Image:           strPtr("zhihanz/fuse-query:latest"),
		ImagePullPolicy: strPtr("Always"),
		HTTPPort:        int32Ptr(8080),
		ClickhousePort:  int32Ptr(9000),
		MysqlPort:       int32Ptr(3306),
		RPCPort:         int32Ptr(9091),
		MetricsPort:     int32Ptr(9098),
		Priority:        int32Ptr(1),
	}
}

func (f *fixture) newController() (*OperatorController, kubeinformers.SharedInformerFactory, crdinformers.SharedInformerFactory) {
	f.client = crdfake.NewSimpleClientset(f.objects...)
	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)

	i := crdinformers.NewSharedInformerFactory(f.client, noResyncPeriodFunc())
	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())
	setter := utils.OperatorSetter{AllNS: true, K8sClient: f.kubeclient, Client: f.client}
	c := NewController(&setter,
		k8sI.Apps().V1().Deployments(), k8sI.Core().V1().Services(), i.Datafuse().V1alpha1().DatafuseComputeGroups(), i.Datafuse().V1alpha1().DatafuseOperators())
	c.operatorSynced = alwaysReady
	c.deploymentsSynced = alwaysReady
	c.serviceSynced = alwaysReady
	c.groupSynced = alwaysReady
	c.recorder = &record.FakeRecorder{}
	for _, g := range f.groupLister {
		i.Datafuse().V1alpha1().DatafuseComputeGroups().Informer().GetIndexer().Add(g)
	}
	for _, d := range f.deploymentLister {
		k8sI.Apps().V1().Deployments().Informer().GetIndexer().Add(d)
	}
	for _, s := range f.servicesLister {
		k8sI.Core().V1().Services().Informer().GetIndexer().Add(s)
	}
	return c, k8sI, i
}

func (f *fixture) runGroup(cg *v1alpha1.DatafuseComputeGroup) {
	f.runGroupController(cg, true, false)
}

func (f *fixture) runOperator(op *v1alpha1.DatafuseOperator) {
	f.runOperatorController(op, true, false)
}

func (f *fixture) runOperatorController(op *v1alpha1.DatafuseOperator, startInformers bool, expectError bool) {
	c, k8sI, i := f.newController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		i.Start(stopCh)
		k8sI.Start(stopCh)
	}

	// error in update part has some issue
	_ = c.processNewOperator(op)

	actions := filterInformerActions(f.client.Actions())
	for i, action := range actions {
		if len(f.actions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.actions), actions[i:])
			break
		}

		expectedAction := f.actions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.actions) > len(actions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.actions)-len(actions), f.actions[len(actions):])
	}

	k8sActions := filterInformerActions(f.kubeclient.Actions())
	for i, action := range k8sActions {
		if len(f.kubeactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(k8sActions)-len(f.kubeactions), k8sActions[i:])
			break
		}

		expectedAction := f.kubeactions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.kubeactions) > len(k8sActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.kubeactions)-len(k8sActions), f.kubeactions[len(k8sActions):])
	}
}

func (f *fixture) runGroupController(cg *v1alpha1.DatafuseComputeGroup, startInformers bool, expectError bool) {
	c, k8sI, i := f.newController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		i.Start(stopCh)
		k8sI.Start(stopCh)
	}

	// error in update part has some issue
	_ = c.processNewComputeGroup(cg)

	actions := filterInformerActions(f.client.Actions())
	for i, action := range actions {
		if len(f.actions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.actions), actions[i:])
			break
		}

		expectedAction := f.actions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.actions) > len(actions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.actions)-len(actions), f.actions[len(actions):])
	}

	k8sActions := filterInformerActions(f.kubeclient.Actions())
	for i, action := range k8sActions {
		if len(f.kubeactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(k8sActions)-len(f.kubeactions), k8sActions[i:])
			break
		}

		expectedAction := f.kubeactions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.kubeactions) > len(k8sActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.kubeactions)-len(k8sActions), f.kubeactions[len(k8sActions):])
	}
}

// checkAction verifies that expected and actual actions are equal and both have
// same attached resources
func checkAction(expected, actual core.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
		t.Errorf("Action has wrong type. Expected: %t. Got: %t", expected, actual)
		return
	}

	switch a := actual.(type) {
	case core.CreateActionImpl:
		e, _ := expected.(core.CreateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.UpdateActionImpl:
		e, _ := expected.(core.UpdateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.PatchActionImpl:
		e, _ := expected.(core.PatchActionImpl)
		expPatch := e.GetPatch()
		patch := a.GetPatch()

		if !reflect.DeepEqual(expPatch, patch) {
			t.Errorf("Action %s %s has wrong patch\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expPatch, patch))
		}
	default:
		t.Errorf("Uncaptured Action %s %s, you should explicitly add a case to capture it",
			actual.GetVerb(), actual.GetResource().Resource)
	}
}

func (f *fixture) expectCreateDeploymentAction(d *appsv1.Deployment) {
	f.kubeactions = append(f.kubeactions, core.NewCreateAction(schema.GroupVersionResource{Resource: "deployments"}, d.Namespace, d))
}

func (f *fixture) expectCreateServiceAction(svc *corev1.Service) {
	f.kubeactions = append(f.kubeactions, core.NewCreateAction(schema.GroupVersionResource{Resource: "services"}, svc.Namespace, svc))
}

func (f *fixture) expectCreateGroupAction(d *v1alpha1.DatafuseComputeGroup) {
	f.actions = append(f.kubeactions, core.NewCreateAction(schema.GroupVersionResource{Resource: "datafusecomputegroups"}, d.Namespace, d))
}

func (f *fixture) expectUpdateDeploymentAction(d *appsv1.Deployment) {
	f.kubeactions = append(f.kubeactions, core.NewUpdateAction(schema.GroupVersionResource{Resource: "deployments"}, d.Namespace, d))
}

func (f *fixture) expectUpdateGroupAction(group *v1alpha1.DatafuseComputeGroup) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "datafusecomputegroups"}, group.Namespace, group)
	// TODO: Until #38113 is merged, we can't use Subresource
	//action.Subresource = "status"
	f.actions = append(f.actions, action)
}

func (f *fixture) expectUpdateOperatorAction(group *v1alpha1.DatafuseOperator) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "datafuseoperators"}, group.Namespace, group)
	// TODO: Until #38113 is merged, we can't use Subresource
	//action.Subresource = "status"
	f.actions = append(f.actions, action)
}

// filterInformerActions filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// nose level in our tests.
func filterInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "datafusecomputegroups") ||
				action.Matches("watch", "datafusecomputegroups") ||
				action.Matches("list", "deployments") ||
				action.Matches("list", "services") ||
				action.Matches("watch", "services") ||
				action.Matches("watch", "datafuseoperators") ||
				action.Matches("list", "datafuseoperators") ||
				action.Matches("watch", "deployments")) {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}

func TestCreatesDeployment(t *testing.T) {
	f := newFixture(t)
	foo := newComputeGroup("groupp1", "default", int32Ptr(1), []*workerSettings{})
	f.groupLister = append(f.groupLister, foo)
	f.objects = append(f.objects, foo)
	d := utils.MakeDeployment(foo.Spec.ComputeLeaders, "groupp1-leader", "default", "", fmt.Sprintf("default-groupp1"), true)
	updatedFoo := foo.DeepCopy()
	updatedFoo.Status.ReadyComputeLeaders = make(map[string]v1alpha1.ComputeInstanceState)
	utils.AddDeployOwnership(d, foo)
	svc := utils.MakeService("groupp1", d)
	updatedFoo.Status.ReadyComputeLeaders["default/groupp1-leader"] = v1alpha1.ComputeInstanceReadyState
	updatedFoo.Status.Status = v1alpha1.ComputeGroupDeployed
	f.expectCreateDeploymentAction(d)
	f.expectCreateServiceAction(svc)
	f.expectUpdateGroupAction(updatedFoo)
	f.runGroup(foo)
}

func TestCreatesComputeGroup(t *testing.T) {
	f := newFixture(t)
	foo := newComputeGroup("groupp1", "default", int32Ptr(1), []*workerSettings{})
	foo.Labels = make(map[string]string)
	fooSpec := newComputeGroupSpec("groupp1", "default", int32Ptr(1), []*workerSettings{})
	op := newOperator("operator1", []*v1alpha1.DatafuseComputeGroupSpec{fooSpec})
	f.operatorLister = append(f.operatorLister, op)
	f.objects = append(f.objects, op)
	v1alpha1.SetDatafuseComputeGroupDefaults(op, foo)
	foo.Labels["datafuse-operator"] = "default-operator1"
	updatedOperator := op.DeepCopy()
	updatedOperator.Status.ComputeGroupStates = make(map[string]v1alpha1.ComputeGroupState)
	updatedOperator.Status.ComputeGroupStates["default/groupp1"] = v1alpha1.ComputeGroupDeployed
	updatedOperator.Status.Status = v1alpha1.OperatorReady
	f.expectCreateGroupAction(foo)
	f.expectUpdateOperatorAction(updatedOperator)
	f.runOperator(op)
}

func TestDoNothingGroup(t *testing.T) {
	f := newFixture(t)
	foo := newComputeGroup("groupp1", "default", int32Ptr(1), []*workerSettings{})
	f.groupLister = append(f.groupLister, foo)
	f.objects = append(f.objects, foo)
	d := utils.MakeDeployment(foo.Spec.ComputeLeaders, "groupp1-leader", "default", "", fmt.Sprintf("default-groupp1"), true)
	updatedFoo := foo.DeepCopy()
	updatedFoo.Status.ReadyComputeLeaders = make(map[string]v1alpha1.ComputeInstanceState)
	utils.AddDeployOwnership(d, foo)
	svc := utils.MakeService("groupp1", d)
	f.deploymentLister = append(f.deploymentLister, d)
	f.servicesLister = append(f.servicesLister, svc)
	updatedFoo.Status.ReadyComputeLeaders["default/groupp1-leader"] = v1alpha1.ComputeInstanceReadyState
	updatedFoo.Status.Status = v1alpha1.ComputeGroupDeployed

	f.expectUpdateGroupAction(updatedFoo)
	f.runGroup(foo)
}

func TestDoNothingOperator(t *testing.T) {
	f := newFixture(t)
	foo := newComputeGroup("groupp1", "default", int32Ptr(1), []*workerSettings{})
	foo.Labels = make(map[string]string)
	fooSpec := newComputeGroupSpec("groupp1", "default", int32Ptr(1), []*workerSettings{})
	op := newOperator("operator1", []*v1alpha1.DatafuseComputeGroupSpec{fooSpec})
	v1alpha1.SetDatafuseComputeGroupDefaults(op, foo)
	foo.Labels["datafuse-operator"] = "default-operator1"
	f.operatorLister = append(f.operatorLister, op)
	f.groupLister = append(f.groupLister, foo)
	f.objects = append(f.objects, op, foo)

	updatedOperator := op.DeepCopy()
	updatedOperator.Status.ComputeGroupStates = make(map[string]v1alpha1.ComputeGroupState)
	updatedOperator.Status.ComputeGroupStates["default/groupp1"] = v1alpha1.ComputeGroupDeployed
	updatedOperator.Status.Status = v1alpha1.OperatorReady
	f.expectUpdateOperatorAction(updatedOperator)
	f.runOperator(op)
}

func TestUpdateDeployment(t *testing.T) {
	f := newFixture(t)
	foo := newComputeGroup("groupp1", "default", int32Ptr(2), []*workerSettings{})
	f.groupLister = append(f.groupLister, foo)
	f.objects = append(f.objects, foo)
	d := utils.MakeDeployment(foo.Spec.ComputeLeaders, "groupp1-leader", "default", "", fmt.Sprintf("default-groupp1"), true)
	d.Spec.Replicas = int32Ptr(1)
	updatedFoo := foo.DeepCopy()
	updatedFoo.Status.ReadyComputeLeaders = make(map[string]v1alpha1.ComputeInstanceState)
	utils.AddDeployOwnership(d, foo)
	svc := utils.MakeService("groupp1", d)
	f.servicesLister = append(f.servicesLister, svc)
	f.deploymentLister = append(f.deploymentLister, d)
	updatedFoo.Status.ReadyComputeLeaders["default/groupp1-leader"] = v1alpha1.ComputeInstanceReadyState
	updatedFoo.Status.Status = v1alpha1.ComputeGroupDeployed

	updatedD := d.DeepCopy()
	updatedD.Spec.Replicas = int32Ptr(2)
	f.expectUpdateDeploymentAction(updatedD)
	f.runGroup(foo)
}

func TestCreateWorkers(t *testing.T) {
	f := newFixture(t)
	foo := newComputeGroup("groupp1", "default", int32Ptr(1), []*workerSettings{{int32Ptr(1), int32Ptr(2)}, {int32Ptr(2), int32Ptr(3)}})
	f.groupLister = append(f.groupLister, foo)
	f.objects = append(f.objects, foo)
	d := utils.MakeDeployment(foo.Spec.ComputeLeaders, "groupp1-leader", "default", "", fmt.Sprintf("default-groupp1"), true)
	updatedFoo := foo.DeepCopy()
	updatedFoo.Status.ReadyComputeLeaders = make(map[string]v1alpha1.ComputeInstanceState)
	utils.AddDeployOwnership(d, foo)
	svc := utils.MakeService("groupp1", d)
	f.servicesLister = append(f.servicesLister, svc)
	updatedFoo.Status.ReadyComputeLeaders["default/groupp1-leader"] = v1alpha1.ComputeInstanceReadyState
	updatedFoo.Status.Status = v1alpha1.ComputeGroupDeployed
	w1 := utils.MakeDeployment(foo.Spec.ComputeLeaders, "groupp1-worker0", "default", "", fmt.Sprintf("default-groupp1"), false)
	w1.Spec.Replicas = int32Ptr(1)
	for _, env := range w1.Spec.Template.Spec.Containers[0].Env {
		if env.Name == controller.FUSE_QUERY_PRIORITY {
			env.Value = "2"
		}
	}
	utils.AddDeployOwnership(w1, foo)
	updatedFoo1 := foo.DeepCopy()
	updatedFoo1.Status.ReadyComputeWorkers = make(map[string]v1alpha1.ComputeInstanceState)
	updatedFoo1.Status.ReadyComputeWorkers["default/groupp1-worker0"] = v1alpha1.ComputeInstanceReadyState
	w2 := utils.MakeDeployment(foo.Spec.ComputeLeaders, "groupp1-worker1", "default", "", fmt.Sprintf("default-groupp1"), false)
	w2.Spec.Replicas = int32Ptr(2)

	updatedFoo2 := foo.DeepCopy()
	updatedFoo2.Status.ReadyComputeWorkers = make(map[string]v1alpha1.ComputeInstanceState)
	updatedFoo2.Status.ReadyComputeWorkers["default/groupp1-worker1"] = v1alpha1.ComputeInstanceReadyState
	w2.Spec.Replicas = int32Ptr(2)
	for _, env := range w1.Spec.Template.Spec.Containers[0].Env {
		if env.Name == controller.FUSE_QUERY_PRIORITY {
			env.Value = "3"
		}
	}
	utils.AddDeployOwnership(w2, foo)
	f.expectCreateDeploymentAction(d)
	f.expectCreateDeploymentAction(w1)
	f.expectCreateDeploymentAction(w2)
	f.expectUpdateGroupAction(updatedFoo)
	f.expectUpdateGroupAction(updatedFoo1)
	f.expectUpdateGroupAction(updatedFoo2)
	f.runGroup(foo)
}

func int32Ptr(i int32) *int32   { return &i }
func strPtr(str string) *string { return &str }
