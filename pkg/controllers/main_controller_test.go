/*
Copyright 2017 The Kubernetes Authors.
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

package controller

// import (
// 	"context"
// 	"reflect"
// 	"testing"
// 	"time"

// 	"github.com/rs/zerolog/log"
// 	"github.com/stretchr/testify/assert"
// 	apps "k8s.io/api/apps/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/runtime"
// 	"k8s.io/apimachinery/pkg/runtime/schema"
// 	"k8s.io/apimachinery/pkg/util/diff"
// 	kubeinformers "k8s.io/client-go/informers"
// 	k8sfake "k8s.io/client-go/kubernetes/fake"
// 	core "k8s.io/client-go/testing"
// 	"k8s.io/client-go/tools/cache"
// 	"k8s.io/client-go/tools/record"

// 	"datafuselabs.io/datafuse-operator/pkg/apis/datafuse/v1alpha1"
// 	"datafuselabs.io/datafuse-operator/pkg/client/clientset/versioned/fake"
// 	crdinformers "datafuselabs.io/datafuse-operator/pkg/client/informers/externalversions"
// 	"datafuselabs.io/datafuse-operator/tests/utils/retry"
// 	testutils "datafuselabs.io/datafuse-operator/tests/utils/retry"
// )

// var (
// 	alwaysReady        = func() bool { return true }
// 	noResyncPeriodFunc = func() time.Duration { return 0 }
// )

// type fixture struct {
// 	t *testing.T

// 	client     *fake.Clientset
// 	kubeclient *k8sfake.Clientset
// 	// Objects to put in the store.
// 	operatorLister     []*v1alpha1.DatafuseOperator
// 	computeGroupLister []*v1alpha1.DatafuseComputeGroup
// 	// Actions expected to happen on the client.
// 	kubeactions []core.Action
// 	actions     []core.Action
// 	// Objects from here preloaded into NewSimpleFake.
// 	kubeobjects []runtime.Object
// 	objects     []runtime.Object
// }

// func newFixture(t *testing.T) *fixture {
// 	f := &fixture{}
// 	f.t = t
// 	f.objects = []runtime.Object{}
// 	f.kubeobjects = []runtime.Object{}
// 	return f
// }

// func (f *fixture) newController() (*Controller, crdinformers.SharedInformerFactory, kubeinformers.SharedInformerFactory) {
// 	f.client = fake.NewSimpleClientset(f.objects...)
// 	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)

// 	i := crdinformers.NewSharedInformerFactory(f.client, noResyncPeriodFunc())
// 	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())
// 	c := NewController(f.kubeclient, f.client, k8sI.Apps().V1().Deployments(),
// 		k8sI.Core().V1().Services(),
// 		i.Datafuse().V1alpha1().DatafuseComputeGroups(),
// 		i.Datafuse().V1alpha1().DatafuseOperators())
// 	c.operatorListerSynced = alwaysReady
// 	c.groupListerSynced = alwaysReady
// 	c.deploymentListerSynced = alwaysReady
// 	c.serviceListerSynced = alwaysReady
// 	c.recorder = &record.FakeRecorder{}

// 	for _, o := range f.operatorLister {
// 		i.Datafuse().V1alpha1().DatafuseOperators().Informer().GetIndexer().Add(o)
// 	}

// 	for _, g := range f.computeGroupLister {
// 		i.Datafuse().V1alpha1().DatafuseComputeGroups().Informer().GetIndexer().Add(g)
// 	}

// 	return c, i, k8sI
// }

// func (f *fixture) run(key string) {
// 	f.runController(key, true, false)
// }

// func (f *fixture) runExpectError(key string) {
// 	f.runController(key, true, true)
// }

// func (f *fixture) runController(operatorKey string, startInformers bool, expectError bool) {
// 	c, i, k8sI := f.newController()
// 	if startInformers {
// 		stopCh := make(chan struct{})
// 		defer close(stopCh)
// 		i.Start(stopCh)
// 		k8sI.Start(stopCh)
// 	}

// 	err := c.syncOperatorHandler(operatorKey)
// 	if !expectError && err != nil {
// 		f.t.Errorf("error syncing operator: %v", err)
// 	} else if expectError && err == nil {
// 		f.t.Error("expected error syncing foo, got nil")
// 	}

// 	actions := filterInformerActions(f.client.Actions())
// 	for i, action := range actions {
// 		if len(f.actions) < i+1 {
// 			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.actions), actions[i:])
// 			break
// 		}
// 		log.Debug().Msgf("current action %+v", action)
// 		expectedAction := f.actions[i]
// 		checkAction(expectedAction, action, f.t)
// 	}

// 	if len(f.actions) > len(actions) {
// 		f.t.Errorf("%d additional expected actions:%+v", len(f.actions)-len(actions), f.actions[len(actions):])
// 	}

// 	k8sActions := filterInformerActions(f.kubeclient.Actions())
// 	for i, action := range k8sActions {
// 		if len(f.kubeactions) < i+1 {
// 			f.t.Errorf("%d unexpected actions: %+v", len(k8sActions)-len(f.kubeactions), k8sActions[i:])
// 			break
// 		}

// 		expectedAction := f.kubeactions[i]
// 		checkAction(expectedAction, action, f.t)
// 	}

// 	if len(f.kubeactions) > len(k8sActions) {
// 		f.t.Errorf("%d additional expected actions:%+v", len(f.kubeactions)-len(k8sActions), f.kubeactions[len(k8sActions):])
// 	}
// }

// // checkAction verifies that expected and actual actions are equal and both have
// // same attached resources
// func checkAction(expected, actual core.Action, t *testing.T) {
// 	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
// 		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
// 		return
// 	}

// 	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
// 		t.Errorf("Action has wrong type. Expected: %t. Got: %t", expected, actual)
// 		return
// 	}

// 	switch a := actual.(type) {
// 	case core.GetActionImpl:
// 		e, _ := expected.(core.GetActionImpl)
// 		assert.Equal(t, a.Name, e.Name)
// 		assert.Equal(t, a.Namespace, e.Namespace)
// 		assert.Equal(t, a.Verb, e.Verb)
// 	case core.CreateActionImpl:
// 		e, _ := expected.(core.CreateActionImpl)
// 		expObject := e.GetObject()
// 		object := a.GetObject()

// 		if !reflect.DeepEqual(expObject, object) {
// 			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
// 				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
// 		}
// 	case core.UpdateActionImpl:
// 		e, _ := expected.(core.UpdateActionImpl)
// 		expObject := e.GetObject()
// 		object := a.GetObject()

// 		if !reflect.DeepEqual(expObject, object) {
// 			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
// 				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
// 		}
// 	case core.PatchActionImpl:
// 		e, _ := expected.(core.PatchActionImpl)
// 		expPatch := e.GetPatch()
// 		patch := a.GetPatch()

// 		if !reflect.DeepEqual(expPatch, patch) {
// 			t.Errorf("Action %s %s has wrong patch\nDiff:\n %s",
// 				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expPatch, patch))
// 		}
// 	default:
// 		t.Errorf("Uncaptured Action %s %s, you should explicitly add a case to capture it",
// 			actual.GetVerb(), actual.GetResource().Resource)
// 	}
// }

// // filterInformerActions filters list and watch actions for testing resources.
// // Since list and watch don't change resource state we can filter it to lower
// // nose level in our tests.
// func filterInformerActions(actions []core.Action) []core.Action {
// 	ret := []core.Action{}
// 	for _, action := range actions {
// 		if len(action.GetNamespace()) == 0 &&
// 			(action.Matches("list", "datafuseoperators") ||
// 				action.Matches("watch", "datafuseoperators") ||
// 				action.Matches("list", "datafusecomputegroups") ||
// 				action.Matches("watch", "datafusecomputegroups") ||
// 				action.Matches("list", "deployments") ||
// 				action.Matches("watch", "deployments") ||
// 				action.Matches("list", "services") ||
// 				action.Matches("watch", "services")) {
// 			continue
// 		}
// 		log.Debug().Msgf("action verb: %s, namespace: %s, resource name: %s", action.GetVerb(), action.GetNamespace(), action.GetResource().Resource)
// 		ret = append(ret, action)
// 	}

// 	return ret
// }

// func (f *fixture) expectGetComputeGroupAction(g *v1alpha1.DatafuseComputeGroup) {
// 	action := core.NewGetAction(schema.GroupVersionResource{Resource: "datafusecomputegroups"}, g.Namespace, g.Name)
// 	f.actions = append(f.actions, action)
// }

// func (f *fixture) expectCreateDeploymentAction(d *apps.Deployment) {
// 	f.kubeactions = append(f.kubeactions, core.NewCreateAction(schema.GroupVersionResource{Resource: "deployments"}, d.Namespace, d))
// }

// func (f *fixture) expectUpdateDeploymentAction(d *apps.Deployment) {
// 	f.kubeactions = append(f.kubeactions, core.NewUpdateAction(schema.GroupVersionResource{Resource: "deployments"}, d.Namespace, d))
// }

// func (f *fixture) expectCreateGroupAction(g *v1alpha1.DatafuseComputeGroup) {
// 	action := core.NewCreateAction(schema.GroupVersionResource{Resource: "datafusecomputegroups"}, g.Namespace, g)
// 	f.actions = append(f.actions, action)
// }

// func (f *fixture) expectUpdateGroupAction(g *v1alpha1.DatafuseComputeGroup) {
// 	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "datafusecomputegroups"}, g.Namespace, g)
// 	f.actions = append(f.actions, action)
// }

// func (f *fixture) expectCreateOperatorAction(o *v1alpha1.DatafuseOperator) {
// 	action := core.NewCreateAction(schema.GroupVersionResource{Resource: "datafuseoperators"}, o.Namespace, o)
// 	f.actions = append(f.actions, action)
// }

// func (f *fixture) expectUpdateOperatorAction(o *v1alpha1.DatafuseOperator) {
// 	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "datafuseoperators"}, o.Namespace, o)
// 	f.actions = append(f.actions, action)
// }

// func getOperatorKey(foo *v1alpha1.DatafuseOperator, t *testing.T) string {
// 	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(foo)
// 	if err != nil {
// 		t.Errorf("Unexpected error getting key for foo %v: %v", foo.Name, err)
// 		return ""
// 	}
// 	return key
// }

// func updateOperatorStatus(t *testing.T, operator *v1alpha1.DatafuseOperator, group *v1alpha1.DatafuseComputeGroup) *v1alpha1.DatafuseOperator {
// 	ret := operator.DeepCopy()
// 	if ret.Status.ComputeGroupStates == nil {
// 		ret.Status.ComputeGroupStates = make(map[string]v1alpha1.ComputeGroupState)
// 	}
// 	key, err := cache.MetaNamespaceKeyFunc(group)
// 	if err != nil {
// 		t.Fatalf("Cannot retrieve key from group %v: %v", group.Name, err)
// 	}
// 	ret.Status.ComputeGroupStates[key] = group.Status.Status
// 	return ret
// }

// func TestOperatorGenerateGroups(t *testing.T) {
// 	f := newFixture(t)
// 	groupSetting := []*workerSettings{{priority: int32Ptr(1), replicas: int32Ptr(3)}}
// 	group := newComputeGroup("test-group1", "default", int32Ptr(1), groupSetting)
// 	operator := newOperator("test", []*v1alpha1.DatafuseComputeGroup{group})
// 	f.operatorLister = append(f.operatorLister, operator)
// 	f.objects = append(f.objects, operator)
// 	expectGroup := group.DeepCopy()
// 	expectGroup = addOwnership(operator, expectGroup)
// 	newOperator := updateOperatorStatus(t, operator, expectGroup)
// 	f.expectCreateGroupAction(expectGroup)
// 	f.expectUpdateOperatorAction(newOperator)
// 	f.run(getKey(operator, t))
// }

// func addOwnership(operator *v1alpha1.DatafuseOperator, group *v1alpha1.DatafuseComputeGroup) *v1alpha1.DatafuseComputeGroup {
// 	newGroup := group.DeepCopy()
// 	v1alpha1.SetDatafuseComputeGroupDefaults(operator, newGroup)
// 	return newGroup
// }

// func TestDoNothing(t *testing.T) {
// 	f := newFixture(t)
// 	groupSetting := []*workerSettings{{priority: int32Ptr(1), replicas: int32Ptr(3)}}
// 	group := newComputeGroup("test-group1", "default", int32Ptr(1), groupSetting)
// 	operator := newOperator("test", []*v1alpha1.DatafuseComputeGroup{group})
// 	newGroup := addOwnership(operator, group)
// 	f.operatorLister = append(f.operatorLister, operator)
// 	f.computeGroupLister = append(f.computeGroupLister, newGroup)
// 	f.objects = append(f.objects, operator)
// 	f.objects = append(f.objects, group)
// 	//log.Fatal().Msgf("%s %s %s", group.Name, group.Namespace, group.Kind)
// 	newOperator := updateOperatorStatus(t, operator, newGroup)
// 	//f.expectGetComputeGroupAction(newGroup)
// 	f.expectUpdateOperatorAction(newOperator)
// 	f.run(getKey(operator, t))
// }

// // func TestAddMoreLeaderReplicas(t *testing.T) {
// // 	f := newFixture(t)
// // 	groupSetting := []*workerSettings{{priority: int32Ptr(1), replicas: int32Ptr(3)}}
// // 	existedGroup := newComputeGroup("test-group1", "default", int32Ptr(1), groupSetting)

// // 	// Update leader replicas
// // 	updatedGroup := newComputeGroup("test-group1", "default", int32Ptr(5), groupSetting)
// // 	operator := newOperator("test", []*v1alpha1.DatafuseComputeGroup{updatedGroup})
// // 	existedGroup = addOwnership(operator, existedGroup)
// // 	updatedGroup = addOwnership(operator, updatedGroup)
// // 	f.operatorLister = append(f.operatorLister, operator)
// // 	f.computeGroupLister = append(f.computeGroupLister, existedGroup)
// // 	f.objects = append(f.objects, operator)
// // 	f.objects = append(f.objects, existedGroup)
// // 	f.expectUpdateGroupAction(updatedGroup)
// // 	f.run(getKey(operator, t))
// // }

// // func TestNotControlledByUs(t *testing.T) {
// // 	f := newFixture(t)
// // 	foo := newFoo("test", int32Ptr(1))
// // 	d := newDeployment(foo)

// // 	d.ObjectMeta.OwnerReferences = []metav1.OwnerReference{}

// // 	f.fooLister = append(f.fooLister, foo)
// // 	f.objects = append(f.objects, foo)
// // 	f.deploymentLister = append(f.deploymentLister, d)
// // 	f.kubeobjects = append(f.kubeobjects, d)

// // 	f.runExpectError(getKey(foo, t))
// // }

func int32Ptr(i int32) *int32 { return &i }

// func getKey(object interface{}, t *testing.T) string {
// 	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(object)
// 	if err != nil {
// 		return ""
// 	}
// 	return key
// }

// // // func TestUpdateReplica(t *testing.T) {
// // 	f := newFixture(t)
// // 	controller, i, k8sI := f.newController()
// // 	controller.syncOperatorHandler()
// // }

// func (f *fixture) updateWithRetries(name, namespace string, replicate int32, pollInterval, pollTimeout time.Duration) (*v1alpha1.DatafuseComputeGroup, error) {
// 	retry.UntilSuccess(func() error {
// 		g, err := f.client.DatafuseV1alpha1().DatafuseComputeGroups(namespace).Get(context.TODO(), name, metav1.GetOptions{})
// 		if err != nil {
// 			log.Info().Msgf("failted get")
// 			return err
// 		}
// 		g = newComputeGroup(name, namespace, &replicate, []*workerSettings{})
// 		g, err = f.client.DatafuseV1alpha1().DatafuseComputeGroups(namespace).Update(context.TODO(), g, metav1.UpdateOptions{})
// 		if err != nil {
// 			log.Info().Msgf("failted update")
// 			return err
// 		}
// 		return nil
// 	}, *testutils.NewConfig().SetCoverage(1).SetInterval(pollInterval).SetTimeOut(pollTimeout))
// 	return nil, nil
// }
