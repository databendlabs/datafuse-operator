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
package register

import (
	"fmt"
	"testing"
	"time"

	"datafuselabs.io/datafuse-operator/tests/utils/retry"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	fcache "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/util/workqueue"
)

// simplified register controller
func newMockRegisterController(rs *RegistSetter) (c *RegisterController,
	podSourcers map[string]*fcache.FakeControllerSource) {
	c = &RegisterController{
		k8sClient:    rs.K8sClient,
		queue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		setter:       rs,
		podInformers: make(map[string]cache.SharedInformer),
		debug:        true,
	}
	// construct a series of pod controller according to the configmaps' namespace and labelselector
	podSourcers = make(map[string]*fcache.FakeControllerSource)
	if rs.AllNS {
		podSourcer := fcache.NewFakeControllerSource()
		podSourcers[metav1.NamespaceAll] = podSourcer
		buildRegisterInformer(c, metav1.NamespaceAll, podSourcer)
	} else {
		//TODO
	}
	return c, podSourcers
}

func TestRegisterController_ListReadyLeaders(t *testing.T) {
	tests := []struct {
		name        string
		existedPods []*v1.Pod
		namespace   string
		groupKey    string
		want        []*v1.Pod
	}{
		{
			name:        "ok",
			existedPods: []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", true, true)},
			groupKey:    "foo",
			namespace:   metav1.NamespaceAll,
			want:        []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", true, true)},
		},
		{
			name: "with irrelevent pod",
			existedPods: []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", true, true),
				makeIreleventMockPod("foo", "foo-irrelevent", metav1.NamespaceAll, true, true)},
			groupKey:  "foo",
			namespace: metav1.NamespaceAll,
			want:      []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", true, true)},
		},
		{
			name: "with unready pod",
			existedPods: []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", true, true),
				makeIreleventMockPod("foo", "foo-irrelevent", metav1.NamespaceAll, true, true),
				makeMockPod("foo", "foo-unready1", metav1.NamespaceAll, "1.1.1.2", true, false)},
			groupKey:  "foo",
			namespace: metav1.NamespaceAll,
			want:      []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", true, true)},
		},
		{
			name: "with multiple ready pod",
			existedPods: []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", true, true),
				makeMockPod("foo", "foo-ready2", metav1.NamespaceAll, "1.1.1.3", true, true),
				makeIreleventMockPod("foo", "foo-irrelevent", metav1.NamespaceAll, true, true),
				makeMockPod("foo", "foo-unready1", metav1.NamespaceAll, "1.1.1.2", true, false)},
			groupKey:  "foo",
			namespace: metav1.NamespaceAll,
			want:      []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", true, true), makeMockPod("foo", "foo-ready2", metav1.NamespaceAll, "1.1.1.3", true, true)},
		},
		{
			name: "with workers",
			existedPods: []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", true, true),
				makeMockPod("foo", "foo-ready2", metav1.NamespaceAll, "1.1.1.3", true, true),
				makeMockPod("foo", "foo-ready3", metav1.NamespaceAll, "1.1.1.4", false, true),
				makeMockPod("foo", "foo-ready4", metav1.NamespaceAll, "1.1.1.5", false, true),
				makeIreleventMockPod("foo", "foo-irrelevent", metav1.NamespaceAll, true, true),
				makeMockPod("foo", "foo-unready1", metav1.NamespaceAll, "1.1.1.2", true, false)},
			groupKey:  "foo",
			namespace: metav1.NamespaceAll,
			want:      []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", true, true), makeMockPod("foo", "foo-ready2", metav1.NamespaceAll, "1.1.1.3", true, true)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fakeK8sClienset(tt.existedPods)
			rs := RegistSetter{true, []string{}, client}
			rc, src := newMockRegisterController(&rs)
			stop := make(chan struct{})
			for _, pod := range tt.existedPods {
				if rs.AllNS {
					src[metav1.NamespaceAll].Add(pod)
				}
			}
			go rc.Run(stop)
			err := retry.UntilSuccess(func() error {
				if len(rc.ListReadyLeaders(tt.groupKey)) != len(tt.want) {
					return fmt.Errorf("current leaders %d, expected leaders %d", len(rc.ListReadyLeaders(tt.groupKey)), len(tt.want))
				}
				return nil
			}, *retry.NewConfig().SetCoverage(1).SetInterval(10 * time.Millisecond).SetTimeOut(1 * time.Second))
			assert.NoError(t, err)

		})
	}
}

func TestRegisterController_ListReadyWorkers(t *testing.T) {
	tests := []struct {
		name        string
		existedPods []*v1.Pod
		namespace   string
		groupKey    string
		want        []*v1.Pod
	}{
		{
			name:        "ok",
			existedPods: []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", false, true)},
			groupKey:    "foo",
			namespace:   metav1.NamespaceAll,
			want:        []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", false, true)},
		},
		{
			name: "with irrelevent pod",
			existedPods: []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", false, true),
				makeIreleventMockPod("foo", "foo-irrelevent", metav1.NamespaceAll, true, true)},
			groupKey:  "foo",
			namespace: metav1.NamespaceAll,
			want:      []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", false, true)},
		},
		{
			name: "with unready pod",
			existedPods: []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", false, true),
				makeIreleventMockPod("foo", "foo-irrelevent", metav1.NamespaceAll, true, true),
				makeMockPod("foo", "foo-unready1", metav1.NamespaceAll, "1.1.1.2", false, false)},
			groupKey:  "foo",
			namespace: metav1.NamespaceAll,
			want:      []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", false, true)},
		},
		{
			name: "with multiple ready pod",
			existedPods: []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", false, true),
				makeMockPod("foo", "foo-ready2", metav1.NamespaceAll, "1.1.1.3", false, true),
				makeIreleventMockPod("foo", "foo-irrelevent", metav1.NamespaceAll, false, true),
				makeMockPod("foo", "foo-unready1", metav1.NamespaceAll, "1.1.1.2", false, false)},
			groupKey:  "foo",
			namespace: metav1.NamespaceAll,
			want:      []*v1.Pod{makeMockPod("foo", "foo-ready1", metav1.NamespaceAll, "1.1.1.1", false, true), makeMockPod("foo", "foo-ready2", metav1.NamespaceAll, "1.1.1.3", false, true)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fakeK8sClienset(tt.existedPods)
			rs := RegistSetter{true, []string{}, client}
			rc, src := newMockRegisterController(&rs)
			stop := make(chan struct{})
			for _, pod := range tt.existedPods {
				if rs.AllNS {
					src[metav1.NamespaceAll].Add(pod)
				}
			}
			go rc.Run(stop)
			err := retry.UntilSuccess(func() error {
				if len(rc.ListReadyWorkers(tt.groupKey)) != len(tt.want) {
					return fmt.Errorf("current leaders %d, expected leaders %d", len(rc.ListReadyWorkers(tt.groupKey)), len(tt.want))
				}
				return nil
			}, *retry.NewConfig().SetCoverage(1).SetInterval(10 * time.Millisecond).SetTimeOut(1 * time.Second))
			assert.NoError(t, err)

		})
	}
}

func TestRegisterControllerProcessReadyWorker(t *testing.T) {
	nodes := []ClusterNodeRequest{{Name: "pod1/default", Address: "1.1.1.1", Priority: 1}}
	leaderConfig := "foo"
	followerConfig := "bar"
	leaderServer := newHTTPTestServer(t, leaderConfig, &nodes)
	defer leaderServer.Close()
	followerServer := newHTTPTestServer(t, followerConfig, &[]ClusterNodeRequest{})
	defer followerServer.Close()
	leader := makeMockPodFromServer(leaderServer, "default-default-group", "pod1", "default", true, true)
	follower := makeMockPodFromServer(followerServer, "default-default-group", "pod2", "default", false, true)
	existedPods := []*v1.Pod{leader, follower}
	client := fakeK8sClienset(existedPods)
	rs := RegistSetter{true, []string{}, client}
	rc, src := newMockRegisterController(&rs)
	for _, pod := range existedPods {
		if rs.AllNS {
			src[metav1.NamespaceAll].Add(pod)
		}
	}
	stop := make(chan struct{})
	go rc.Run(stop)
	expected := []ClusterNodeRequest{{Name: "pod1/default", Address: "1.1.1.1", Priority: 1}, {Name: "default/pod2", Address: follower.Status.PodIP, Priority: 1}}
	err := retry.UntilSuccess(func() error {
		leaders := rc.ListReadyLeaders("default-default-group")
		if len(leaders) != 1 {
			return fmt.Errorf("ready leaders should be 1")
		}
		err := rc.ProcessReadyWorker(follower)
		assert.NoError(t, err)
		actual, err := rc.setter.GetClusterList(leader)
		if err != nil {
			return err
		}
		assert.Equal(t, len(expected), len(actual))
		return nil
	}, *retry.NewConfig().SetCoverage(1).SetInterval(10 * time.Millisecond).SetTimeOut(1 * time.Second))
	assert.NoError(t, err)
}

func TestRegisterControllerProcessNextItem(t *testing.T) {
	nodes := []ClusterNodeRequest{{Name: "pod1/default", Address: "1.1.1.1", Priority: 1}}
	leaderConfig := "foo"
	followerConfig := "bar"
	leaderServer := newHTTPTestServer(t, leaderConfig, &nodes)
	defer leaderServer.Close()
	followerServer := newHTTPTestServer(t, followerConfig, &[]ClusterNodeRequest{})
	defer followerServer.Close()
	leader := makeMockPodFromServer(leaderServer, "default-default-group", "pod1", "default", true, true)
	follower := makeMockPodFromServer(followerServer, "default-default-group", "pod2", "default", false, true)
	existedPods := []*v1.Pod{leader, follower}
	client := fakeK8sClienset(existedPods)
	rs := RegistSetter{true, []string{}, client}
	rc, src := newMockRegisterController(&rs)
	for _, pod := range existedPods {
		if rs.AllNS {
			src[metav1.NamespaceAll].Add(pod)
		}
	}
	stop := make(chan struct{})
	go rc.Run(stop)
	expected := []ClusterNodeRequest{{Name: "default/pod2", Address: follower.Status.PodIP + ":9091", Priority: 1}, {Name: "default/pod1", Address: leader.Status.PodIP + ":9091", Priority: 1}}
	err := retry.UntilSuccess(func() error {
		leaders := rc.ListReadyLeaders("default-default-group")
		if len(leaders) != 1 {
			return fmt.Errorf("ready leaders should be 1")
		}
		actual, err := rc.setter.GetClusterList(leader)
		if err != nil {
			return err
		}
		assert.Equal(t, expected, actual)
		return nil
	}, *retry.NewConfig().SetCoverage(1).SetInterval(500 * time.Millisecond).SetTimeOut(10 * time.Second))
	assert.NoError(t, err)
}
