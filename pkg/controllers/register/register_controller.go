// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.

package register

import (
	"fmt"
	"strings"
	"time"

	controller "datafuselabs.io/datafuse-operator/pkg/controllers"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubectl/pkg/util/podutils"
)

const (
	RegisterAgentName = "fuse-query-registration-controller"
)

type RegisterController struct {
	k8sClient    kubernetes.Interface
	setter       *RegistSetter
	podInformers map[string]cache.SharedInformer
	queue        workqueue.RateLimitingInterface
	recorder     record.EventRecorder
	debug        bool
}

// registration controller will list all
func NewRegisterController(setter *RegistSetter) (*RegisterController, error) {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Info().Msgf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: setter.K8sClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: RegisterAgentName})
	c := &RegisterController{
		k8sClient:    setter.K8sClient,
		queue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "datafuse-operator"),
		podInformers: make(map[string]cache.SharedInformer),
		recorder:     recorder,
		setter:       setter,
		debug:        false,
	}
	if setter.AllNS {
		registerListWatch := cache.NewFilteredListWatchFromClient(
			c.k8sClient.CoreV1().RESTClient(),
			"pods",
			metav1.NamespaceAll,
			func(options *metav1.ListOptions) {
				options.LabelSelector = strings.Join(setter.GetLabelSelectors(), ",")
				options.FieldSelector = strings.Join(setter.GetFieldSelectors(), ",")
			},
		)
		buildRegisterInformer(c, metav1.NamespaceAll, registerListWatch)
	}
	return c, nil
}

func buildRegisterInformer(c *RegisterController, namespace string, source cache.ListerWatcher) {
	informer := cache.NewSharedInformer(source, &corev1.Pod{}, 0)
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(newObj interface{}) {
			c.queue.AddRateLimited(newObj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.queue.AddRateLimited(newObj)
		},
		DeleteFunc: func(newObj interface{}) {
			// TODO garbage collection
			c.queue.AddRateLimited(newObj)
		},
	})
	c.podInformers[namespace] = informer
}

func (rc *RegisterController) Run(stopCh <-chan struct{}) {
	timeSetter := func() time.Duration {
		if rc.debug {
			return 10 * time.Millisecond
		} else {
			return 1 * time.Second
		}
	}
	for _, podController := range rc.podInformers {
		go podController.Run(stopCh)
		if !cache.WaitForCacheSync(stopCh, podController.HasSynced) {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
		// This will run the func every 1 second until stopCh is sent
		go wait.Until(
			func() {
				// Runs processNextItem in a loop, if it returns false it will
				// be restarted by wait.Until unless stopCh is sent.
				for rc.processNextItem() {
				}
			},
			timeSetter(),
			stopCh,
		)

		<-stopCh
		log.Info().Msgf("Stopping register controller.")
	}
}

// Process the next available item in the work queue.
// Return false if exiting permanently, else return true
// so the loop keeps processing.
func (rc *RegisterController) processNextItem() bool {
	errorHandler := func(obj interface{}, pod *v1.Pod, err error) {
		if pod == nil {
			return
		}
		key := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
		if err == nil {
			log.Debug().Msgf("Removing %s from work queue", key)
			rc.queue.Forget(obj)
		} else {
			log.Error().Msgf("Error while processing operator %s: %s", key, err.Error())
			if rc.queue.NumRequeues(obj) < 50 {
				log.Info().Msgf("Re-adding %s to work queue", key)
				rc.queue.AddRateLimited(obj)
			} else {
				log.Info().Msgf("Requeue limit reached, removing %s", key)
				rc.queue.Forget(obj)
				runtime.HandleError(err)
			}
		}
	}
	obj, quit := rc.queue.Get()
	if quit {
		// Exit permanently
		return false
	}
	defer rc.queue.Done(obj)
	pod, ok := obj.(*v1.Pod)
	if !ok {
		log.Error().Msgf("Error decoding object, invalid type. Dropping.")
		rc.queue.Forget(obj)
		// Short-circuit on this item, but return true to keep
		// processing.
		return true
	}

	if !podutils.IsPodReady(pod) {
		if rc.setter.IsGroupLeader(*pod) {
			errorHandler(obj, pod, fmt.Errorf("current leader pod %s in %s is not ready", pod.Name, pod.Namespace))
		} else if rc.setter.IsGroupWorker(*pod) {
			err := rc.ProcessUnreadyWorker(pod)
			errorHandler(obj, pod, err)
		} else {
			log.Error().Msgf("pod %s in %s is not managed by register controller, wrong labeling", pod.Name, pod.Namespace)
			rc.queue.Forget(obj)
			// Short-circuit on this item, but return true to keep
			// processing.
			return true
		}
	} else {
		if rc.setter.IsGroupLeader(*pod) {
			err := rc.ProcessLeader(pod)
			errorHandler(obj, pod, err)
		} else if rc.setter.IsGroupWorker(*pod) {
			err := rc.ProcessReadyWorker(pod)
			errorHandler(obj, pod, err)
		} else {
			log.Error().Msgf("pod %s in %s is not managed by register controller, wrong labeling", pod.Name, pod.Namespace)
			rc.queue.Forget(obj)
			// Short-circuit on this item, but return true to keep
			// processing.
			return true
		}
	}

	return true
}

// list all ready leaders from cache
func (rc *RegisterController) ListReadyLeaders(groupKey string) []*v1.Pod {
	ret := make([]*v1.Pod, 0)
	if rc.setter.AllNS {
		objs := rc.podInformers[metav1.NamespaceAll].GetStore().List()
		for _, obj := range objs {
			pod, ok := obj.(*v1.Pod)
			if !ok {
				continue
			}
			if !podutils.IsPodReady(pod) || !rc.setter.IsGroupLeader(*pod) {
				continue
			}
			ret = append(ret, pod)
		}
	}
	return ret
}

// list all ready followers from cache
func (rc *RegisterController) ListReadyWorkers(groupKey string) []*v1.Pod {
	ret := make([]*v1.Pod, 0)
	if rc.setter.AllNS {
		objs := rc.podInformers[metav1.NamespaceAll].GetStore().List()
		for _, obj := range objs {
			pod, ok := obj.(*v1.Pod)
			if !ok {
				continue
			}
			if !podutils.IsPodReady(pod) || !rc.setter.IsGroupWorker(*pod) {
				continue
			}
			ret = append(ret, pod)
		}
	}
	return ret
}

// retrieve all registered workers in the leader pod
func (rc *RegisterController) ListExistedWorkers(leader *v1.Pod) (map[string]*v1.Pod, error) {
	workers, err := rc.setter.GetClusterList(leader)
	if err != nil {
		return nil, err
	}
	existedWorkers := make(map[string]*v1.Pod, 0)
	needToRemove := []ClusterNodeRequest{}
	if rc.setter.AllNS {
		store := rc.podInformers[metav1.NamespaceAll].GetStore()
		for _, worker := range workers {
			key := worker.Name
			obj, existed, err := store.GetByKey(key)
			if !existed || err != nil {
				needToRemove = append(needToRemove, worker)
				continue
			}
			p, ok := obj.(*v1.Pod)
			if !ok {
				needToRemove = append(needToRemove, worker)
				continue
			}
			existedWorkers[key] = p
		}
		for _, item := range needToRemove {
			rc.setter.RemoveConfig(leader, item)
		}
	}
	// var podList []*v1.Pod
	return existedWorkers, nil
}

// regist all ready workers to this pod
func (rc *RegisterController) ProcessLeader(pod *v1.Pod) error {
	groupKey, found := pod.Labels[controller.ComputeGroupLabel]
	if !found {
		return fmt.Errorf("pod %s in %s does not have groupkey label", pod.Name, pod.Namespace)
	}
	existedWorkers, err := rc.ListExistedWorkers(pod)
	if err != nil {
		return err
	}
	workers := rc.ListReadyWorkers(groupKey)
	// should also consider leader itself for query processing
	workers = append(workers, pod)
	neededToAdd := make([]*v1.Pod, 0)    // minize Add and Remove cost per pod
	neededToRemove := make([]*v1.Pod, 0) // minize Add and Remove cost per pod
	removeHelper := make(map[string]*v1.Pod)
	for _, worker := range workers {
		key := fmt.Sprintf("%s/%s", worker.Namespace, worker.Name)
		if _, found := existedWorkers[key]; found {
			removeHelper[key] = worker
		} else {
			neededToAdd = append(neededToAdd, worker)
		}
	}
	for key, pod := range existedWorkers {
		if _, found := removeHelper[key]; found {
			delete(removeHelper, key)
		} else {
			neededToRemove = append(neededToRemove, pod)
		}
	}

	for _, item := range neededToAdd {
		err := rc.setter.AddNode(pod, item)
		if err != nil {
			return err
		}
	}
	for _, item := range neededToRemove {
		err := rc.setter.RemoveNode(pod, item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rc *RegisterController) ProcessUnreadyWorker(pod *v1.Pod) error {
	groupKey, found := pod.Labels[controller.ComputeGroupLabel]
	if !found {
		return fmt.Errorf("pod %s in %s does not have groupkey label", pod.Name, pod.Namespace)
	}
	if rc.setter.AllNS {
		leaders := rc.ListReadyLeaders(groupKey)
		// should also consider leader itself for query processing
		for _, leader := range leaders {
			err := rc.setter.RemoveNode(leader, pod)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (rc *RegisterController) ProcessReadyWorker(pod *v1.Pod) error {
	groupKey, found := pod.Labels[controller.ComputeGroupLabel]
	if !found {
		return fmt.Errorf("pod %s in %s does not have groupkey label", pod.Name, pod.Namespace)
	}
	if rc.setter.AllNS {
		leaders := rc.ListReadyLeaders(groupKey)
		// should also consider leader itself for query processing
		for _, leader := range leaders {
			err := rc.setter.AddNode(leader, pod)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
