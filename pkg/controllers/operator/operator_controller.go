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

package operator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"datafuselabs.io/datafuse-operator/pkg/apis/datafuse/v1alpha1"
	crdclientset "datafuselabs.io/datafuse-operator/pkg/client/clientset/versioned"
	crdscheme "datafuselabs.io/datafuse-operator/pkg/client/clientset/versioned/scheme"
	crdinformers "datafuselabs.io/datafuse-operator/pkg/client/informers/externalversions/datafuse/v1alpha1"
	crdlisters "datafuselabs.io/datafuse-operator/pkg/client/listers/datafuse/v1alpha1"
	controller "datafuselabs.io/datafuse-operator/pkg/controllers"
	utils "datafuselabs.io/datafuse-operator/pkg/controllers/utils"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

type OperatorController struct {
	k8sclient         client.Interface
	client            crdclientset.Interface
	deploymentLister  appslisters.DeploymentLister
	serviceLister     corelisters.ServiceLister
	deploymentsSynced cache.InformerSynced
	serviceSynced     cache.InformerSynced
	operatorLister    crdlisters.DatafuseOperatorLister
	operatorSynced    cache.InformerSynced
	groupLister       crdlisters.DatafuseComputeGroupLister
	groupSynced       cache.InformerSynced

	operatorQueue workqueue.RateLimitingInterface
	groupQueue    workqueue.RateLimitingInterface

	// // map namespace to their operators, depreciated
	// cachedOperatorInformer map[string]cache.SharedInformer

	// // map namespace to their groups, depreciated
	// cachedGroupInformer map[string]cache.SharedInformer

	recorder record.EventRecorder
	setter   *utils.OperatorSetter
}

func NewController(setter *utils.OperatorSetter, deployInformer appsinformers.DeploymentInformer,
	serviceInformer coreinformers.ServiceInformer,
	groupInformer crdinformers.DatafuseComputeGroupInformer,
	operatorInformer crdinformers.DatafuseOperatorInformer) *OperatorController {
	utilruntime.Must(crdscheme.AddToScheme(scheme.Scheme))
	log.Info().Msgf("Creating operator controller event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Info().Msgf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: setter.K8sClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controller.ControllerAgentName})
	c := &OperatorController{
		k8sclient:         setter.K8sClient,
		client:            setter.Client,
		operatorQueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "datafuse-operator"),
		groupQueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "register"),
		deploymentLister:  deployInformer.Lister(),
		serviceLister:     serviceInformer.Lister(),
		groupLister:       groupInformer.Lister(),
		operatorLister:    operatorInformer.Lister(),
		groupSynced:       groupInformer.Informer().HasSynced,
		operatorSynced:    operatorInformer.Informer().HasSynced,
		deploymentsSynced: deployInformer.Informer().HasSynced,
		serviceSynced:     serviceInformer.Informer().HasSynced,
		recorder:          recorder,
		setter:            setter,
	}
	log.Info().Msgf("Setting up event handlers")
	groupInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(newObj interface{}) {
			c.groupQueue.AddRateLimited(newObj)
		},
		UpdateFunc: func(old, new interface{}) {
			oldD, ok := old.(v1alpha1.DatafuseComputeGroup)
			if !ok {
				return
			}
			newD, ok := new.(v1alpha1.DatafuseComputeGroup)
			if !ok {
				return
			}
			if oldD.ResourceVersion == newD.ResourceVersion {
				return
			}
			c.handleUnneededDeployments(oldD, newD)
			c.groupQueue.AddRateLimited(new)
		},
	})
	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a Foo resource will enqueue that Foo resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	deployInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.Deployment)
			oldDepl := old.(*appsv1.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			c.handleObject(new)
		},
		DeleteFunc: c.handleObject,
	})
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*corev1.Service)
			oldDepl := old.(*corev1.Service)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			c.handleObject(new)
		},
		DeleteFunc: c.handleObject,
	})
	// Operator informer
	operatorInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(newObj interface{}) {
			c.operatorQueue.AddRateLimited(newObj)
		},
		UpdateFunc: func(old, new interface{}) {
			oldD, ok := old.(v1alpha1.DatafuseOperator)
			if !ok {
				return
			}
			newD, ok := new.(v1alpha1.DatafuseOperator)
			if !ok {
				return
			}
			if oldD.ResourceVersion == newD.ResourceVersion {
				return
			}
			// c.handleUnneededDeployments(oldD, newD)
			c.operatorQueue.AddRateLimited(new)
		},
	})
	return c
}

func (oc *OperatorController) handleUnneededDeployments(old, new v1alpha1.DatafuseComputeGroup) {
	buildNamedSpec := func(group v1alpha1.DatafuseComputeGroup) (map[string]v1alpha1.DatafuseComputeSetSpec, error) {
		ans := make(map[string]v1alpha1.DatafuseComputeSetSpec)
		if group.Spec.ComputeLeaders.Name == nil {
			ans[fmt.Sprintf("%s-leader", group.GetName())] = *group.Spec.ComputeLeaders
		} else {
			ans[*group.Spec.ComputeLeaders.Name] = *group.Spec.ComputeLeaders
		}
		for i, worker := range group.Spec.ComputeWorkers {
			if worker.Name == nil {
				ans[fmt.Sprintf("%s-worker%d", group.GetName(), i)] = *worker
			} else {
				ans[*worker.Name] = *worker
			}
		}
		return ans, nil
	}
	oldNames, err := buildNamedSpec(old)
	if err != nil {
		return
	}
	newNames, err := buildNamedSpec(new)
	if err != nil {
		return
	}
	for name := range oldNames {
		_, found := newNames[name]
		if !found {
			// old names are not found, so we delete related instances and reset group status
			oc.k8sclient.AppsV1().Deployments(old.Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})

			// TODO update status
		}
	}
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the Foo resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Foo resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *OperatorController) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		log.Info().Msgf("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	log.Info().Msgf("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Foo, we should not do anything more
		// with it.
		if ownerRef.Kind != "Foo" {
			return
		}

		foo, err := c.groupLister.DatafuseComputeGroups(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			log.Info().Msgf("ignoring orphaned object '%s' of foo '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.groupQueue.Add(foo)
		return
	}
}

func (c *OperatorController) Start(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.operatorQueue.ShutDown()
	defer c.groupQueue.ShutDown()
	log.Info().Msgf("Starting group controller")
	log.Info().Msgf("Wait for group and deployment cache sync")
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.groupSynced, c.operatorSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	log.Info().Msgf("Starting group workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runGroupWorker, time.Second, stopCh)
	}
	log.Info().Msgf("Started group workers")
	log.Info().Msgf("Starting operator workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runOperatorWorker, time.Second, stopCh)
	}
	log.Info().Msgf("Started operator workers")
	<-stopCh
	log.Info().Msgf("Shutting down workers")
	return nil
}

// runWorker is a long-running function that will continually call the
// processNextComputeGroup function in order to read and process a message on the
// workqueue.
func (c *OperatorController) runGroupWorker() {
	for c.processNextComputeGroup() {
	}
}

// runWorker is a long-running function that will continually call the
// processNextComputeGroup function in order to read and process a message on the
// workqueue.
func (c *OperatorController) runOperatorWorker() {
	for c.processNextOperator() {
	}
}

// func NewOperatorController(setter *utils.OperatorSetter) (*OperatorController, error) {
// 	utilruntime.Must(crdscheme.AddToScheme(scheme.Scheme))
// 	log.Info().Msgf("Creating operator controller event broadcaster")
// 	eventBroadcaster := record.NewBroadcaster()
// 	eventBroadcaster.StartLogging(log.Info().Msgf)
// 	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: setter.K8sClient.CoreV1().Events("")})
// 	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controller.ControllerAgentName})
// 	c := &OperatorController{
// 		k8sclient:              setter.K8sClient,
// 		client:                 setter.Client,
// 		operatorQueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "datafuse-operator"),
// 		groupQueue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "compute-group"),
// 		cachedOperatorInformer: make(map[string]cache.SharedInformer),
// 		cachedGroupInformer:    make(map[string]cache.SharedInformer),
// 		recorder:               recorder,
// 		setter:                 setter,
// 	}
// 	if setter.AllNS {
// 		operatorListWatch := cache.NewFilteredListWatchFromClient(
// 			c.client.DatafuseV1alpha1().RESTClient(),
// 			"datafuseoperators",
// 			metav1.NamespaceAll,
// 			func(options *metav1.ListOptions) {})
// 		buildOperatorInformer(c, metav1.NamespaceAll, operatorListWatch)
// 		computeGroupListWatch := cache.NewFilteredListWatchFromClient(
// 			c.client.DatafuseV1alpha1().RESTClient(),
// 			"datafusecomputegroups",
// 			metav1.NamespaceAll,
// 			func(options *metav1.ListOptions) {})
// 		buildComputeGroupInformer(c, metav1.NamespaceAll, computeGroupListWatch)
// 	}
// 	return c, nil
// }

// func buildOperatorInformer(c *OperatorController, namespace string, source cache.ListerWatcher) {
// 	informer := cache.NewSharedInformer(source, &v1alpha1.DatafuseOperator{}, 0)
// 	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
// 		AddFunc: func(newObj interface{}) {
// 			log.Info().Msgf("added an obj")
// 			c.operatorQueue.AddRateLimited(newObj)
// 		},
// 		UpdateFunc: func(oldObj, newObj interface{}) {
// 			oldOp, ok := oldObj.(*v1alpha1.DatafuseOperator)
// 			if !ok {
// 				return
// 			}

// 			newOp, ok := newObj.(*v1alpha1.DatafuseOperator)
// 			if !ok {
// 				return
// 			}
// 			// The informer will call this function on non-updated resources during resync, avoid
// 			// enqueuing unchanged operator
// 			if oldOp.ResourceVersion == oldOp.ResourceVersion {
// 				return
// 			}
// 			// The spec has changed. This is currently best effort as we can potentially miss updates
// 			// and end up in an inconsistent state.
// 			if !equality.Semantic.DeepEqual(oldOp.Spec, newOp.Spec) {
// 				// Force-set the application status to Invalidating which handles clean-up and application re-run.

// 			}
// 			c.operatorQueue.AddRateLimited(newObj)
// 		},
// 		DeleteFunc: func(newObj interface{}) {
// 			// TODO garbage collection
// 		},
// 	})
// 	c.cachedOperatorInformer[namespace] = informer
// }

// func buildComputeGroupInformer(c *OperatorController, namespace string, source cache.ListerWatcher) {
// 	informer := cache.NewSharedInformer(source, &v1alpha1.DatafuseComputeGroup{}, 0)
// 	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
// 		AddFunc: func(newObj interface{}) {
// 			log.Info().Msgf("added an obj to group")
// 			c.groupQueue.AddRateLimited(newObj)
// 		},
// 		UpdateFunc: func(oldObj, newObj interface{}) {
// 			oldOp, ok := oldObj.(*v1alpha1.DatafuseOperator)
// 			if !ok {
// 				return
// 			}

// 			newOp, ok := newObj.(*v1alpha1.DatafuseOperator)
// 			if !ok {
// 				return
// 			}
// 			// The informer will call this function on non-updated resources during resync, avoid
// 			// enqueuing unchanged operator
// 			if oldOp.ResourceVersion == oldOp.ResourceVersion {
// 				return
// 			}
// 			// The spec has changed. This is currently best effort as we can potentially miss updates
// 			// and end up in an inconsistent state.
// 			if !equality.Semantic.DeepEqual(oldOp.Spec, newOp.Spec) {
// 				// Force-set the application status to Invalidating which handles clean-up and application re-run.

// 			}
// 			// TODO delete unneeeded groups in updated operator
// 			c.operatorQueue.AddRateLimited(newObj)
// 		},
// 		DeleteFunc: func(newObj interface{}) {
// 			// TODO garbage collection
// 		},
// 	})
// 	c.cachedGroupInformer[namespace] = informer
// }

// func (oc *OperatorController) Run(stopCh <-chan struct{}) {
// 	for _, operatorController := range oc.cachedOperatorInformer {
// 		go operatorController.Run(stopCh)
// 		// wait for cache sync up
// 		if !cache.WaitForCacheSync(stopCh, operatorController.HasSynced) {
// 			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
// 			return
// 		}
// 	}
// 	log.Info().Msgf("operators are synced")
// 	for _, groupController := range oc.cachedGroupInformer {
// 		go groupController.Run(stopCh)
// 		// wait for cache sync up
// 		if !cache.WaitForCacheSync(stopCh, groupController.HasSynced) {
// 			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
// 			return
// 		}
// 	}

// 	log.Info().Msgf("compute groups are synced")
// 	// This will run the func every 1 second until stopCh is sent
// 	go wait.Until(
// 		func() {
// 			// Runs processNextItem in a loop, if it returns false it will
// 			// be restarted by wait.Until unless stopCh is sent.
// 			for oc.processNextOperator() {
// 			}
// 		},
// 		1*time.Second,
// 		stopCh)
// 	go wait.Until(
// 		func() {
// 			// Runs processNextItem in a loop, if it returns false it will
// 			// be restarted by wait.Until unless stopCh is sent.
// 			for oc.processNextComputeGroup() {
// 			}
// 		},
// 		1*time.Second,
// 		stopCh)
// 	<-stopCh
// 	log.Info().Msgf("stop operator controller")
// }

func (oc *OperatorController) processNextOperator() bool {
	errorHandler := func(obj interface{}, op *v1alpha1.DatafuseOperator, err error) {
		opName := fmt.Sprintf("%s/%s", op.Namespace, op.Name)
		if err == nil {
			log.Debug().Msgf("Removing %s from work queue", opName)
			oc.operatorQueue.Forget(obj)
		} else {
			log.Error().Msgf("Error while processing operator %s: %s", opName, err.Error())
			if oc.operatorQueue.NumRequeues(obj) < 50 {
				log.Info().Msgf("Re-adding %s to work queue", opName)
				oc.operatorQueue.AddRateLimited(obj)
			} else {
				log.Info().Msgf("Requeue limit reached, removing %s", opName)
				oc.operatorQueue.Forget(obj)
				runtime.HandleError(err)
			}
		}
	}
	obj, quit := oc.operatorQueue.Get()
	if quit {
		// Exit permanently
		return false
	}
	defer oc.operatorQueue.Done(obj)
	op, ok := obj.(*v1alpha1.DatafuseOperator)
	if !ok {
		log.Error().Msgf("Error decoding object, invalid type. Dropping.")
		oc.operatorQueue.Forget(obj)
		// Short-circuit on this item, but return true to keep
		// processing.
		return true
	}
	// check for datafuse operator status
	utils.IsOperatorCreated(op)
	err := oc.processNewOperator(op)
	errorHandler(obj, op, err)

	// Return true to let the loop process the next item
	return true
}

func (oc *OperatorController) processNewOperator(op *v1alpha1.DatafuseOperator) error {
	opCopy := op.DeepCopy()
	var err error
	v1alpha1.SetDatafuseOperatorDefault(opCopy)
	for _, group := range opCopy.Spec.ComputeGroups {
		// err := oc.syncGroup(opCopy, group)
		e := oc.syncGroup2(opCopy, group)
		multierr.Append(err, e)
	}
	return err
}

func (oc *OperatorController) syncGroup2(op *v1alpha1.DatafuseOperator, groupSpec *v1alpha1.DatafuseComputeGroupSpec) error {
	group := utils.MakeComputeGroup(op, *groupSpec)
	v1alpha1.SetDatafuseComputeGroupDefaults(op, group)
	g, err := oc.groupLister.DatafuseComputeGroups(groupSpec.Namespace).Get(group.Name)
	if err != nil && errors.IsNotFound(err) {
		log.Debug().Msgf("deployment not found, create %s in namespace %s", group.GetName(), group.GetNamespace())
		g, err = oc.client.DatafuseV1alpha1().DatafuseComputeGroups(group.Namespace).Create(context.TODO(), group, metav1.CreateOptions{})
	}
	if err != nil {
		log.Debug().Msgf("group %s in namespace %s cannot be created, %s", group.Name, group.Namespace, err.Error())
	}

	if !metav1.IsControlledBy(g, op) {
		msg := fmt.Sprintf("group %s in %s existed but not controlled by operator %s", g.Name, g.Namespace, op.Name)
		oc.recorder.Event(op, corev1.EventTypeWarning, controller.ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	if err != nil {
		return err
	}

	err = oc.updateOperatorStatus(g, op)
	if err != nil {
		return err
	}
	oc.recorder.Event(op, corev1.EventTypeNormal, controller.SuccessSynced, controller.MessageResourceSynced)
	log.Info().Msgf("Successfully Created compute group %s/%s", g.GetNamespace(), g.GetName())
	return nil
}

func (oc *OperatorController) updateOperatorStatus(cg *v1alpha1.DatafuseComputeGroup, op *v1alpha1.DatafuseOperator) error {
	opCopy := op.DeepCopy()
	if opCopy.Status.ComputeGroupStates == nil {
		opCopy.Status.ComputeGroupStates = make(map[string]v1alpha1.ComputeGroupState)
	}
	cgKey := fmt.Sprintf("%s/%s", cg.Namespace, cg.Name)
	opCopy.Status.ComputeGroupStates[cgKey] = v1alpha1.ComputeGroupDeployed
	opCopy.Status.Status = v1alpha1.OperatorReady
	_, err := oc.client.DatafuseV1alpha1().DatafuseOperators(op.Namespace).Update(context.TODO(), opCopy, metav1.UpdateOptions{})
	return err
}

// func (oc *OperatorController) syncGroup(op *v1alpha1.DatafuseOperator, groupSpec *v1alpha1.DatafuseComputeGroupSpec) error {
// 	group := utils.MakeComputeGroup(op, *groupSpec)
// 	v1alpha1.SetDatafuseComputeGroupDefaults(op, group)
// 	var groupInformer cache.SharedInformer
// 	var err error
// 	var g *v1alpha1.DatafuseComputeGroup
// 	if oc.setter.AllNS {
// 		groupInformer = oc.cachedGroupInformer[metav1.NamespaceAll]
// 	} else {
// 		groupInformer = oc.cachedGroupInformer[group.Namespace]
// 	}
// 	gObj, exists, err := groupInformer.GetStore().Get(group)
// 	if !exists || (err != nil && strings.Contains(err.Error(), "not found")) {
// 		log.Debug().Msgf("compute group not found, create %s in namespace %s", group.GetName(), group.GetNamespace())
// 		g, err = oc.client.DatafuseV1alpha1().DatafuseComputeGroups(group.GetNamespace()).Create(context.TODO(), group, metav1.CreateOptions{})
// 	}
// 	if err != nil {
// 		log.Debug().Msgf("compute group %s in namespace %s cannot be created, %s", group.GetName(), group.GetNamespace(), err.Error())
// 		return err
// 	}

// 	g, ok := gObj.(*v1alpha1.DatafuseComputeGroup)
// 	if !ok {
// 		return fmt.Errorf("cannot retreive compute group given namespace %s and name %s", group.GetNamespace(), group.GetName())
// 	}
// 	log.Info().Msgf("Successfully Created compute group %s/%s", g.GetNamespace(), g.GetName())
// 	return nil
// }

func (oc *OperatorController) ProcessUpdatedOperator(op *v1alpha1.DatafuseOperator) error {
	return nil
}

func (oc *OperatorController) processNextComputeGroup() bool {
	errorHandler := func(obj interface{}, cg *v1alpha1.DatafuseComputeGroup, err error) {
		opName := fmt.Sprintf("%s/%s", cg.Namespace, cg.Name)
		if err == nil {
			log.Debug().Msgf("Removing %s from work queue", opName)
			oc.groupQueue.Forget(obj)
		} else {
			log.Error().Msgf("Error while processing pod %s: %s", opName, err.Error())
			if oc.groupQueue.NumRequeues(obj) < 50 {
				log.Info().Msgf("Re-adding %s to work queue", opName)
				oc.groupQueue.AddRateLimited(obj)
			} else {
				log.Info().Msgf("Requeue limit reached, removing %s", opName)
				oc.groupQueue.Forget(obj)
				runtime.HandleError(err)
			}
		}
	}
	obj, quit := oc.groupQueue.Get()
	if quit {
		// Exit permanently
		return false
	}
	defer oc.groupQueue.Done(obj)
	group, ok := obj.(*v1alpha1.DatafuseComputeGroup)
	if !ok {
		log.Error().Msgf("Error decoding object, invalid type. Dropping.")
		oc.groupQueue.Forget(obj)
		// Short-circuit on this item, but return true to keep
		// processing.
		return true
	}

	if utils.IsComputeGroupCreated(group) {
		err := oc.processNewComputeGroup(group)
		errorHandler(obj, group, err)
	} else {
		err := oc.processUpdatedComputeGroup(group)
		errorHandler(obj, group, err)
	}

	return true
}

func (oc *OperatorController) processNewComputeGroup(cg *v1alpha1.DatafuseComputeGroup) error {
	var err error
	fillName := func(instanceSpec v1alpha1.DatafuseComputeInstanceSpec, basic string) string {
		if instanceSpec.Name == nil {
			return basic
		}
		return *instanceSpec.Name
	}
	cgCopy := cg.DeepCopy()

	err1 := oc.syncComputeGroup(cgCopy, cgCopy.Spec.ComputeLeaders, fillName(cgCopy.Spec.ComputeLeaders.DatafuseComputeInstanceSpec, fmt.Sprintf("%s-leader", cgCopy.GetName())), true)
	if err != nil {
		multierr.Append(err, err1)
	}
	log.Info().Msgf("successfully synced group leader %s/%s", cgCopy.Namespace, fmt.Sprintf("%s-leader", cgCopy.GetName()))
	for i, worker := range cgCopy.Spec.ComputeWorkers {
		err1 = oc.syncComputeGroup(cgCopy, worker, fillName(worker.DatafuseComputeInstanceSpec, fmt.Sprintf("%s-worker%d", cgCopy.GetName(), i)), false)
		if err != nil {
			multierr.Append(err, err1)
		}
		log.Info().Msgf("successfully synced group worker %s/%s", cgCopy.Namespace, fmt.Sprintf("%s-worker%d", cgCopy.GetName(), i))
	}
	return err
}

func (oc *OperatorController) syncComputeGroup(cg *v1alpha1.DatafuseComputeGroup,
	set *v1alpha1.DatafuseComputeSetSpec, name string, isLeader bool) error {
	opKey, groupKey, err := oc.GetComputeGroupKeys(cg)
	if err != nil {
		return err
	}
	deploy := utils.MakeDeployment(set, name, cg.Namespace, opKey, groupKey, isLeader)
	utils.AddDeployOwnership(deploy, cg)
	// var deployInformer cache.SharedInformer
	var d *v1.Deployment

	d, err = oc.deploymentLister.Deployments(deploy.Namespace).Get(deploy.Name)
	if err != nil && errors.IsNotFound(err) {
		log.Debug().Msgf("deployment not found, create %s in namespace %s", deploy.GetName(), deploy.GetNamespace())
		d, err = oc.k8sclient.AppsV1().Deployments(deploy.Namespace).Create(context.TODO(), deploy, metav1.CreateOptions{})
	}
	if err != nil {
		log.Debug().Msgf("deployment %s in namespace %s cannot be created, %s", deploy.GetName(), deploy.GetNamespace(), err.Error())
		return err
	}
	// check whether group service is bootstraped
	if isLeader {
		service := utils.MakeService(cg.Name, deploy)
		_, err = oc.serviceLister.Services(service.Namespace).Get(service.Name)
		if err != nil && errors.IsNotFound(err) {
			log.Debug().Msgf("service not found, create %s in namespace %s", service.GetName(), service.GetNamespace())
			_, err = oc.k8sclient.CoreV1().Services(service.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
		}
		if err != nil {
			log.Debug().Msgf("service %s in namespace %s cannot be created, %s", service.GetName(), service.GetNamespace(), err.Error())
			return err
		}
	}

	// If the Deployment is not controlled by this Foo resource, we should log
	// a warning to the event recorder and return error msg.
	if !metav1.IsControlledBy(d, cg) {
		msg := fmt.Sprintf("deployment %s in %s existed but not controlled by group %s", d.Name, d.Namespace, cg.Name)
		oc.recorder.Event(cg, corev1.EventTypeWarning, controller.ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}
	if set.Replicas != nil && *set.Replicas != *d.Spec.Replicas {
		log.Info().Msgf("deployment %s in namespace %s required replicas: %d, actually: %d", d.Name, d.Namespace, *set.Replicas, *d.Spec.Replicas)
		d, err = oc.k8sclient.AppsV1().Deployments(cg.Namespace).Update(context.TODO(), deploy, metav1.UpdateOptions{})
	}
	if err != nil {
		return err
	}

	err = oc.updateComputeGroupStatus(cg, d, isLeader)
	if err != nil {
		return err
	}
	oc.recorder.Event(cg, corev1.EventTypeNormal, controller.SuccessSynced, controller.MessageResourceSynced)
	log.Info().Msgf("Successfully Created deployment %s/%s", d.GetNamespace(), d.GetName())
	return nil
}

func (oc *OperatorController) updateComputeGroupStatus(cg *v1alpha1.DatafuseComputeGroup, d *appsv1.Deployment, isLeader bool) error {
	cgCopy := cg.DeepCopy()
	deployKey := fmt.Sprintf("%s/%s", d.Namespace, d.Name)
	if isLeader {
		if cgCopy.Status.Status != v1alpha1.ComputeGroupDeployed {
			cgCopy.Status.Status = v1alpha1.ComputeGroupDeployed
		}
		if cgCopy.Status.ReadyComputeLeaders == nil {
			cgCopy.Status.ReadyComputeLeaders = make(map[string]v1alpha1.ComputeInstanceState)
		}
		cgCopy.Status.ReadyComputeLeaders[deployKey] = v1alpha1.ComputeInstanceReadyState
	} else {
		if cgCopy.Status.ReadyComputeWorkers == nil {
			cgCopy.Status.ReadyComputeWorkers = make(map[string]v1alpha1.ComputeInstanceState)
		}
		cgCopy.Status.ReadyComputeWorkers[deployKey] = v1alpha1.ComputeInstanceReadyState
	}

	_, err := oc.client.DatafuseV1alpha1().DatafuseComputeGroups(cgCopy.Namespace).Update(context.TODO(), cgCopy, metav1.UpdateOptions{})
	return err
}

func (oc *OperatorController) GetComputeGroupKeys(cg *v1alpha1.DatafuseComputeGroup) (opKey, groupKey string, err error) {
	if cg.GetName() == "" || cg.GetNamespace() == "" {
		return "", "", fmt.Errorf("compute group name or namespace is empty")
	}
	opKey, found := cg.Labels[controller.OperatorLabel]
	if !found {
		opKey = ""
		// return "", "", fmt.Errorf("cannot retrieve datafuse operator label")
	}
	return opKey, fmt.Sprintf("%s-%s", cg.GetNamespace(), cg.GetName()), nil
}

func (oc *OperatorController) processUpdatedComputeGroup(op *v1alpha1.DatafuseComputeGroup) error {
	if op.Status.ReadyComputeLeaders == nil && op.Status.ReadyComputeWorkers == nil {
		err := oc.processNewComputeGroup(op)
		return err
	}
	opCopy := op.DeepCopy()
	for key := range op.Status.ReadyComputeLeaders {
		ns, name := strings.SplitN(key, "/", 2)[0], strings.SplitN(key, "/", 2)[1]
		_, err := oc.deploymentLister.Deployments(ns).Get(name)
		if err != nil && errors.IsNotFound(err) {
			opCopy.Status.Status = v1alpha1.ComputeGroupPending
			delete(opCopy.Status.ReadyComputeLeaders, key)
		}
	}

	for key := range op.Status.ReadyComputeWorkers {

		ns, name := strings.SplitN(key, "/", 2)[0], strings.SplitN(key, "/", 2)[1]
		_, err := oc.deploymentLister.Deployments(ns).Get(name)
		if err != nil && errors.IsNotFound(err) {
			opCopy.Status.Status = v1alpha1.ComputeGroupPending
			delete(opCopy.Status.ReadyComputeWorkers, key)
		}
	}
	oc.client.DatafuseV1alpha1().DatafuseComputeGroups(op.Namespace).Update(context.TODO(), opCopy, metav1.UpdateOptions{})

	return oc.processNewComputeGroup(op)
}
