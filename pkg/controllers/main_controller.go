// /*
// Copyright 2021.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package controller

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"github.com/rs/zerolog/log"
// 	"golang.org/x/time/rate"
// 	appsv1 "k8s.io/api/apps/v1"
// 	corev1 "k8s.io/api/core/v1"
// 	"k8s.io/apimachinery/pkg/api/errors"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
// 	"k8s.io/apimachinery/pkg/util/wait"
// 	appsinformers "k8s.io/client-go/informers/apps/v1"
// 	coreinformers "k8s.io/client-go/informers/core/v1"
// 	"k8s.io/client-go/kubernetes"
// 	"k8s.io/client-go/kubernetes/scheme"
// 	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
// 	appslisters "k8s.io/client-go/listers/apps/v1"
// 	corelisters "k8s.io/client-go/listers/core/v1"
// 	_ "k8s.io/client-go/plugin/pkg/client/auth"
// 	"k8s.io/client-go/tools/cache"
// 	"k8s.io/client-go/tools/record"
// 	"k8s.io/client-go/util/workqueue"
// 	"k8s.io/klog"

// 	"datafuselabs.io/datafuse-operator/pkg/apis/datafuse/v1alpha1"
// 	crdclientset "datafuselabs.io/datafuse-operator/pkg/client/clientset/versioned"
// 	crdscheme "datafuselabs.io/datafuse-operator/pkg/client/clientset/versioned/scheme"
// 	crdinformers "datafuselabs.io/datafuse-operator/pkg/client/informers/externalversions/datafuse/v1alpha1"
// 	crdlisters "datafuselabs.io/datafuse-operator/pkg/client/listers/datafuse/v1alpha1"
// )

const (
	ControllerAgentName = "datafuse-operator"
	// SuccessSynced is used as part of the Event 'reason' when a Tenant is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Tenant fails
	// to sync due to a StatefulSet of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceSynced is the message used for an Event fired when a Operator
	// is synced successfully
	MessageResourceSynced              = "datafuse operator synced successfully"
	queueTokenRefillRate               = 50
	queueTokenBucketSize               = 500
	OperatorLabel                      = "datafuse-operator"
	ComputeGroupLabel                  = "datafuse-computegroup"
	ComputeGroupRoleLabel              = "datafuse-computegrouprole"
	ComputeGroupRoleLeader             = "leader"
	ComputeGroupRoleFollower           = "follower"
	InstanceContainerName              = "fusequery"
	FUSE_QUERY_MYSQL_HANDLER_HOST      = "FUSE_QUERY_MYSQL_HANDLER_HOST"
	FUSE_QUERY_MYSQL_HANDLER_PORT      = "FUSE_QUERY_MYSQL_HANDLER_PORT"
	FUSE_QUERY_CLICKHOUSE_HANDLER_HOST = "FUSE_QUERY_CLICKHOUSE_HANDLER_HOST"
	FUSE_QUERY_CLICKHOUSE_HANDLER_PORT = "FUSE_QUERY_CLICKHOUSE_HANDLER_PORT"
	FUSE_QUERY_RPC_API_ADDRESS         = "FUSE_QUERY_FLIGHT_API_ADDRESS"
	FUSE_QUERY_HTTP_API_ADDRESS        = "FUSE_QUERY_HTTP_API_ADDRESS"
	FUSE_QUERY_METRIC_API_ADDRESS      = "FUSE_QUERY_METRIC_API_ADDRESS"
	FUSE_QUERY_PRIORITY                = "FUSE_QUERY_PRIORITY"
	ContainerHTTPPort                  = "http"
	ContainerMetricsPort               = "metrics"
	ContainerRPCPort                   = "rpc"
	ContainerMysqlPort                 = "mysql"
	ContainerClickhousePort            = "clickhouse"
)

// type Controller struct {
// 	crdClient              crdclientset.Interface
// 	kubeClient             kubernetes.Interface
// 	operatorQueue          workqueue.RateLimitingInterface
// 	groupLister            crdlisters.DatafuseComputeGroupLister
// 	groupListerSynced      cache.InformerSynced
// 	computeGroupQueue      workqueue.RateLimitingInterface
// 	operatorLister         crdlisters.DatafuseOperatorLister
// 	operatorListerSynced   cache.InformerSynced
// 	deploymentsLister      appslisters.DeploymentLister
// 	deploymentListerSynced cache.InformerSynced
// 	serviceLister          corelisters.ServiceLister
// 	serviceListerSynced    cache.InformerSynced

// 	groupController    cache.Controller
// 	operatorController cache.Controller
// 	cacheSynced        cache.InformerSynced
// 	recorder           record.EventRecorder
// }

// // NewController creates a new Controller.
// func NewController(
// 	kubeClientSet kubernetes.Interface,
// 	crdClientSet crdclientset.Interface,
// 	deploymentInformer appsinformers.DeploymentInformer,
// 	serviceInformer coreinformers.ServiceInformer,
// 	computeGroupInformer crdinformers.DatafuseComputeGroupInformer,
// 	operatorInformer crdinformers.DatafuseOperatorInformer) *Controller {
// 	crdscheme.AddToScheme(scheme.Scheme)
// 	klog.V(4).Info("Creating event broadcaster")
// 	eventBroadcaster := record.NewBroadcaster()
// 	eventBroadcaster.StartLogging(klog.Infof)
// 	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})
// 	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
// 	return newDatafuseOperatorController(kubeClientSet, crdClientSet, deploymentInformer, serviceInformer, computeGroupInformer, operatorInformer, recorder)
// }

// func newDatafuseOperatorController(kubeClientSet kubernetes.Interface,
// 	crdClientSet crdclientset.Interface,
// 	deploymentInformer appsinformers.DeploymentInformer,
// 	serviceInformer coreinformers.ServiceInformer,
// 	computeGroupInformer crdinformers.DatafuseComputeGroupInformer,
// 	operatorInformer crdinformers.DatafuseOperatorInformer,
// 	eventRecorder record.EventRecorder) *Controller {
// 	queue := workqueue.NewNamedRateLimitingQueue(&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(queueTokenRefillRate), queueTokenBucketSize)},
// 		"datafuse-operator-controller")
// 	groupQueue := workqueue.NewNamedRateLimitingQueue(&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(queueTokenRefillRate), queueTokenBucketSize)},
// 		"datafuse-computegroup-controller")
// 	controller := &Controller{
// 		crdClient:              crdClientSet,
// 		kubeClient:             kubeClientSet,
// 		groupLister:            computeGroupInformer.Lister(),
// 		groupListerSynced:      computeGroupInformer.Informer().HasSynced,
// 		operatorLister:         operatorInformer.Lister(),
// 		operatorListerSynced:   operatorInformer.Informer().HasSynced,
// 		deploymentsLister:      deploymentInformer.Lister(),
// 		deploymentListerSynced: deploymentInformer.Informer().HasSynced,
// 		serviceLister:          serviceInformer.Lister(),
// 		serviceListerSynced:    serviceInformer.Informer().HasSynced,
// 		recorder:               eventRecorder,
// 		operatorQueue:          queue,
// 		computeGroupQueue:      groupQueue,
// 	}
// 	klog.Info("Setting up event handlers")
// 	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
// 		AddFunc: controller.handleObject,
// 		UpdateFunc: func(old, new interface{}) {
// 			newDepl := new.(*appsv1.Deployment)
// 			oldDepl := old.(*appsv1.Deployment)
// 			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
// 				// Periodic resync will send update events for all known Deployments.
// 				// Two different versions of the same Deployments will always have different RVs.
// 				return
// 			}
// 			controller.handleObject(new)
// 		},
// 		DeleteFunc: controller.handleObject,
// 	})
// 	operatorInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
// 		AddFunc: controller.enqueueOperator,
// 		UpdateFunc: func(old, new interface{}) {
// 			controller.enqueueOperator(new)
// 		},
// 	})

// 	computeGroupInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
// 		AddFunc: controller.enqueueComputeGroup,
// 		UpdateFunc: func(old, new interface{}) {
// 			controller.enqueueComputeGroup(new)
// 		},
// 	})

// 	return controller
// }

// func (c *Controller) Start(workers int, stopCh <-chan struct{}) error {
// 	defer utilruntime.HandleCrash()
// 	defer c.operatorQueue.ShutDown()
// 	log.Info().Msg("start main controller")
// 	log.Info().Msg("wait for cache sync")
// 	if ok := cache.WaitForCacheSync(stopCh, c.deploymentListerSynced, c.operatorListerSynced, c.groupListerSynced); !ok {
// 		return fmt.Errorf("failed to wait for cache sync")
// 	}
// 	log.Info().Msgf("start %d workers", workers)
// 	for i := 0; i < workers; i++ {
// 		go wait.Until(c.runOperatorWorker, time.Second, stopCh)
// 		go wait.Until(c.runGroupWorker, time.Second, stopCh)
// 	}
// 	log.Info().Msgf("Starting main workers")
// 	<-stopCh
// 	log.Info().Msgf("Shutting down main workers")
// 	return nil
// }

// // runOperatorWorker is a long-running function that will continually call the
// // processNextOperatorItem function in order to read and process a message on the
// // operatorQueue
// func (c *Controller) runOperatorWorker() {
// 	for c.processNextOperatorItem() {
// 	}
// }

// // runGroupWorker is a long-running function that will continually call the
// // processNextGroupItem function in order to read and process a message on the
// // groupQueue
// func (c *Controller) runGroupWorker() {
// 	for c.processNextGroupItem() {
// 	}
// }

// func (c *Controller) processNextGroupItem() bool {
// 	obj, shutdown := c.computeGroupQueue.Get()

// 	if shutdown {
// 		return false
// 	}
// 	// We wrap this block in a func so we can defer c.workqueue.Done.
// 	err := func(obj interface{}) error {
// 		// We call Done here so the workqueue knows we have finished
// 		// processing this item. We also must remember to call Forget if we
// 		// do not want this work item being re-queued. For example, we do
// 		// not call Forget if a transient error occurs, instead the item is
// 		// put back on the workqueue and attempted again after a back-off
// 		// period.
// 		defer c.computeGroupQueue.Done(obj)
// 		var key string
// 		var ok bool
// 		// We expect strings to come off the workqueue. These are of the
// 		// form namespace/name. We do this as the delayed nature of the
// 		// workqueue means the items in the informer cache may actually be
// 		// more up to date that when the item was initially put onto the
// 		// workqueue.
// 		if key, ok = obj.(string); !ok {
// 			// As the item in the workqueue is actually invalid, we call
// 			// Forget here else we'd go into a loop of attempting to
// 			// process a work item that is invalid.
// 			c.computeGroupQueue.Forget(obj)
// 			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
// 			return nil
// 		}
// 		log.Info().Msgf("start to process compute group key: %q", key)
// 		defer log.Info().Msgf("processed  compute group key: %q", key)
// 		// Run the syncHandler, passing it the namespace/name string of the
// 		// Foo resource to be synced.
// 		if err := c.syncGroupHandler(key); err != nil {
// 			// Put the item back on the workqueue to handle any transient errors.
// 			c.computeGroupQueue.AddRateLimited(key)
// 			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
// 		}
// 		// Finally, if no error occurs we Forget this item so it does not
// 		// get queued again until another change happens.
// 		c.computeGroupQueue.Forget(obj)
// 		klog.Infof("Successfully synced '%s'", key)
// 		return nil
// 	}(obj)

// 	if err != nil {
// 		utilruntime.HandleError(err)
// 		return true
// 	}

// 	return true
// }

// // processNextWorkItem will read a single work item off the workqueue and
// // attempt to process it, by calling the syncHandler.
// func (c *Controller) processNextOperatorItem() bool {
// 	obj, shutdown := c.operatorQueue.Get()

// 	if shutdown {
// 		return false
// 	}

// 	// We wrap this block in a func so we can defer c.workqueue.Done.
// 	err := func(obj interface{}) error {
// 		// We call Done here so the workqueue knows we have finished
// 		// processing this item. We also must remember to call Forget if we
// 		// do not want this work item being re-queued. For example, we do
// 		// not call Forget if a transient error occurs, instead the item is
// 		// put back on the workqueue and attempted again after a back-off
// 		// period.
// 		defer c.operatorQueue.Done(obj)
// 		var key string
// 		var ok bool
// 		// We expect strings to come off the workqueue. These are of the
// 		// form namespace/name. We do this as the delayed nature of the
// 		// workqueue means the items in the informer cache may actually be
// 		// more up to date that when the item was initially put onto the
// 		// workqueue.
// 		if key, ok = obj.(string); !ok {
// 			// As the item in the workqueue is actually invalid, we call
// 			// Forget here else we'd go into a loop of attempting to
// 			// process a work item that is invalid.
// 			c.operatorQueue.Forget(obj)
// 			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
// 			return nil
// 		}
// 		log.Info().Msgf("start to process key: %q", key)
// 		defer log.Info().Msgf("processed key: %q", key)
// 		// Run the syncHandler, passing it the namespace/name string of the
// 		// Foo resource to be synced.
// 		if err := c.syncOperatorHandler(key); err != nil {
// 			// Put the item back on the workqueue to handle any transient errors.
// 			c.operatorQueue.AddRateLimited(key)
// 			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
// 		}
// 		// Finally, if no error occurs we Forget this item so it does not
// 		// get queued again until another change happens.
// 		c.operatorQueue.Forget(obj)
// 		klog.Infof("Successfully synced '%s'", key)
// 		return nil
// 	}(obj)

// 	if err != nil {
// 		utilruntime.HandleError(err)
// 		return true
// 	}

// 	return true
// }

// func (c *Controller) getDatafuseComputeGroup(namespace, name string) (*v1alpha1.DatafuseComputeGroup, error) {
// 	op, err := c.groupLister.DatafuseComputeGroups(namespace).Get(name)
// 	if err != nil {
// 		if errors.IsNotFound(err) {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}
// 	return op, nil
// }
// func (c *Controller) getDatafuseOperator(namespace, name string) (*v1alpha1.DatafuseOperator, error) {
// 	op, err := c.operatorLister.DatafuseOperators(namespace).Get(name)
// 	if err != nil {
// 		if errors.IsNotFound(err) {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}
// 	return op, nil
// }

// // syncHandler compares the actual state with the desired, and attempts to
// // converge the two. It then updates the Status block of the Foo resource
// // with the current status of the resource.
// func (c *Controller) syncGroupHandler(key string) error {
// 	// Convert the namespace/name string into a distinct namespace and name
// 	namespace, name, err := cache.SplitMetaNamespaceKey(key)
// 	if err != nil {
// 		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s, with error %s", key, err.Error()))
// 		return nil
// 	}
// 	operator, err := c.getDatafuseOperator(namespace, name)
// 	if err != nil {
// 		return err
// 	}

// 	if operator == nil {
// 		return nil
// 	}

// 	//deploy fuse query leaders
// 	leaders, err := c.syncQueryLeaders(operator)
// 	if err != nil {
// 		return err
// 	}
// 	//deploy fuse query workers
// 	workers, err := c.syncQueryWorkers(operator)
// 	if err != nil {
// 		return err
// 	}
// 	err = c.updateOperatorStatus(operator, leaders, workers)
// 	if err != nil {
// 		return err
// 	}
// 	c.recorder.Event(operator, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
// 	return nil
// }

// // syncHandler compares the actual state with the desired, and attempts to
// // converge the two. It then updates the Status block of the Foo resource
// // with the current status of the resource.
// func (c *Controller) syncOperatorHandler(key string) error {
// 	// Convert the namespace/name string into a distinct namespace and name
// 	namespace, name, err := cache.SplitMetaNamespaceKey(key)
// 	if err != nil {
// 		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s, with error %s", key, err.Error()))
// 		return nil
// 	}
// 	operator, err := c.getDatafuseOperator(namespace, name)
// 	if err != nil {
// 		return err
// 	}

// 	if operator == nil {
// 		return nil
// 	}
// 	operatorCopy := operator.DeepCopy()
// 	_, err = c.syncOperatorComputeGroups(operatorCopy)
// 	if err != nil {
// 		return err
// 	}

// 	c.recorder.Event(operator, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
// 	return nil
// }
// func (c *Controller) syncOperatorComputeGroups(operator *v1alpha1.DatafuseOperator) ([]*v1alpha1.DatafuseComputeGroup, error) {

// 	groups := []*v1alpha1.DatafuseComputeGroup{}
// 	for _, group := range operator.Spec.ComputeGroups {
// 		g, err := c.groupLister.DatafuseComputeGroups(group.Namespace).Get(group.Name)
// 		// If the resource doesn't exist, we'll create it
// 		if errors.IsNotFound(err) {
// 			groupCopy := group.DeepCopy()
// 			v1alpha1.SetDatafuseComputeGroupDefaults(operator, groupCopy)
// 			g, err = c.crdClient.DatafuseV1alpha1().DatafuseComputeGroups(groupCopy.Namespace).Create(context.TODO(), groupCopy, metav1.CreateOptions{})
// 		}
// 		if err != nil {
// 			return nil, err
// 		}
// 		// If the compute group is not controlled by this operator resource, we should log
// 		// a warning to the event recorder and return error msg.
// 		if !metav1.IsControlledBy(g, operator) {
// 			msg := fmt.Sprintf("Resource %q already exists and is not managed by Operator %s", group.Name, operator.Name)
// 			c.recorder.Event(operator, corev1.EventTypeWarning, ErrResourceExists, msg)
// 			return nil, fmt.Errorf(msg)
// 		}
// 		// Update groups
// 		g, err = c.updateGroups(group, g)
// 		if err != nil {
// 			return nil, err
// 		}
// 		err = c.updateOperatorComputeGroupStatus(operator, g)
// 		if err != nil {
// 			return nil, err
// 		}
// 		groups = append(groups, g)
// 	}
// 	return groups, nil
// }

// func (c *Controller) updateGroups(operatorGroup, clusterGroup *v1alpha1.DatafuseComputeGroup) (finalGroup *v1alpha1.DatafuseComputeGroup, err error) {
// 	finalGroup = clusterGroup.DeepCopy()
// 	needUpdate := false
// 	if *operatorGroup.Spec.ComputeLeaders.Replicas != *clusterGroup.Spec.ComputeLeaders.Replicas {
// 		log.Info().Msgf("group %s in namspace %s should has %d leaders but has %d leaders",
// 			operatorGroup.Name, operatorGroup.Namespace,
// 			*operatorGroup.Spec.ComputeLeaders.Replicas,
// 			*clusterGroup.Spec.ComputeLeaders.Replicas)
// 		finalGroup.Spec.ComputeLeaders.Replicas = operatorGroup.Spec.ComputeLeaders.Replicas
// 		needUpdate = true
// 	}
// 	if len(operatorGroup.Spec.ComputeWorkers) != len(clusterGroup.Spec.ComputeWorkers) {
// 		log.Info().Msgf("group %s in namspace %s should has %d but has %d worker group",
// 			operatorGroup.Name, operatorGroup.Namespace,
// 			*operatorGroup.Spec.ComputeLeaders.Replicas,
// 			*clusterGroup.Spec.ComputeLeaders.Replicas)
// 		needUpdate = true
// 	}
// 	if needUpdate {
// 		finalGroup, err = c.crdClient.DatafuseV1alpha1().DatafuseComputeGroups(finalGroup.Namespace).Update(context.TODO(), finalGroup, metav1.UpdateOptions{})
// 	}
// 	return finalGroup, err
// }

// func (c *Controller) updateOperatorComputeGroupStatus(operator *v1alpha1.DatafuseOperator, reconciledGroup *v1alpha1.DatafuseComputeGroup) error {
// 	operatorCopy := operator.DeepCopy()
// 	key, err := cache.MetaNamespaceKeyFunc(reconciledGroup)
// 	if err != nil {
// 		return err
// 	}
// 	if operatorCopy.Status.ComputeGroupStates == nil {
// 		operatorCopy.Status.ComputeGroupStates = make(map[string]v1alpha1.ComputeGroupState)
// 	}
// 	operatorCopy.Status.ComputeGroupStates[key] = reconciledGroup.Status.Status
// 	_, err = c.crdClient.DatafuseV1alpha1().DatafuseOperators(operator.Namespace).Update(context.TODO(), operatorCopy, metav1.UpdateOptions{})
// 	return nil
// }

// func (c *Controller) syncQueryLeaders(operator *v1alpha1.DatafuseOperator) (*appsv1.Deployment, error) {
// 	return nil, nil
// }

// func (c *Controller) syncQueryWorkers(operator *v1alpha1.DatafuseOperator) ([]*appsv1.Deployment, error) {
// 	return nil, nil
// }

// func (c *Controller) updateOperatorStatus(operator *v1alpha1.DatafuseOperator, leaders *appsv1.Deployment, workers []*appsv1.Deployment) error {
// 	return nil
// }

// // handleObject will take any resource implementing metav1.Object and attempt
// // to find the Foo resource that 'owns' it. It does this by looking at the
// // objects metadata.ownerReferences field for an appropriate OwnerReference.
// // It then enqueues that Foo resource to be processed. If the object does not
// // have an appropriate OwnerReference, it will simply be skipped.
// func (c *Controller) handleObject(obj interface{}) {
// 	var object metav1.Object
// 	var ok bool
// 	if object, ok = obj.(metav1.Object); !ok {
// 		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
// 		if !ok {
// 			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
// 			return
// 		}
// 		object, ok = tombstone.Obj.(metav1.Object)
// 		if !ok {
// 			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
// 			return
// 		}
// 		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
// 	}
// 	klog.V(4).Infof("Processing object: %s", object.GetName())
// 	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
// 		// If this object is not owned by a Foo, we should not do anything more
// 		// with it.
// 		if ownerRef.Kind != "DatafuseOperator" {
// 			return
// 		}

// 		df, err := c.operatorLister.DatafuseOperators(object.GetNamespace()).Get(ownerRef.Name)
// 		if err != nil {
// 			klog.V(4).Infof("ignoring orphaned object '%s' of foo '%s'", object.GetSelfLink(), ownerRef.Name)
// 			return
// 		}

// 		c.enqueueOperator(df)
// 		return
// 	}
// }

// // enqueueOperator takes a operator resource and convert it into a namespace/name
// // string which is then put onto the work queue, this method should not be passed to
// // resources of any type other than Operator
// func (c *Controller) enqueueOperator(obj interface{}) {
// 	var key string
// 	var err error
// 	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
// 		utilruntime.HandleError(err)
// 		return
// 	}
// 	c.operatorQueue.Add(key)
// }

// // enqueueGroup takes a operator resource and convert it into a namespace/name
// // string which is then put onto the work queue, this method should not be passed to
// // resources of any type other than Operator
// func (c *Controller) enqueueComputeGroup(obj interface{}) {
// 	var key string
// 	var err error
// 	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
// 		utilruntime.HandleError(err)
// 		return
// 	}
// 	c.computeGroupQueue.Add(key)
// }

// func (c *Controller) Stop() {
// 	log.Info().Msg("Stopping the DatafuseOperator controller")
// 	c.operatorQueue.ShutDown()
// 	c.computeGroupQueue.ShutDown()
// }
