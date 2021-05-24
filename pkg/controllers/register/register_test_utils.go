package register

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"datafuselabs.io/datafuse-operator/pkg/apis/datafuse/v1alpha1"
	"datafuselabs.io/datafuse-operator/pkg/controllers/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type mockFuseQueryServer struct {
	Nodes  []ClusterNodeRequest
	Config string
	Server *httptest.Server
}

func int32Ptr(i int32) *int32   { return &i }
func strPtr(str string) *string { return &str }

var (
	DefaultComputeSpec = v1alpha1.DatafuseComputeSetSpec{
		Replicas: int32Ptr(1),
		DatafuseComputeInstanceSpec: v1alpha1.DatafuseComputeInstanceSpec{
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
		},
	}
)

func newHTTPTestServer(t *testing.T, config string, nodes *[]ClusterNodeRequest) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.Method == "GET" && r.URL.EscapedPath() == string(CLUSTER_CONFIG) {
			w.Write([]byte(config))
		}
		if r.Method == "GET" && r.URL.EscapedPath() == string(CLUSTER_LIST) {
			b, err := json.Marshal(*nodes)
			if err != nil {
				t.Fatal(err)
			}
			w.Write(b)
		}
		if r.Method == "POST" && r.URL.EscapedPath() == string(CLUSTER_ADD) {
			jsonbyte, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Body.Close()
			var node ClusterNodeRequest
			json.Unmarshal(jsonbyte, &node)
			for _, item := range *nodes {
				if item.Name == node.Name {
					return
				}
			}
			*nodes = append(*nodes, node)
		}
		if r.Method == "POST" && r.URL.EscapedPath() == string(CLUSTER_REMOVE) {
			jsonbyte, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Body.Close()
			var node ClusterNodeRequest
			json.Unmarshal(jsonbyte, &node)
			updatedList := []ClusterNodeRequest{}
			for _, item := range *nodes {
				if item.Name != node.Name {
					updatedList = append(updatedList, item)
				}
			}
			*nodes = updatedList
		}
	}))
	return server
}

func makeMockPod(groupKey, name, namespace, ip string, isLeader bool, isReady bool) *v1.Pod {
	pod := utils.MakeFuseQueryPod(&DefaultComputeSpec, name, namespace, groupKey, isLeader)
	if isReady {
		createMockStatus(pod, v1.PodReady)
	}
	pod.Status.PodIP = ip
	return pod
}

func makeIreleventMockPod(groupKey, name, namespace string, isLeader bool, isReady bool) *v1.Pod {
	pod := utils.MakeFuseQueryPod(&DefaultComputeSpec, name, namespace, groupKey, isLeader)
	pod.Labels = make(map[string]string)
	if isReady {
		createMockStatus(pod, v1.PodReady)
	}
	return pod
}

func makeMockPodFromServer(server *httptest.Server, groupKey, name, namespace string, isLeader, isReady bool) *v1.Pod {
	url := strings.TrimPrefix(server.URL, "http://")
	str := strings.SplitN(url, ":", 2)
	s, err := strconv.Atoi(str[1])
	if err != nil {
		return nil
	}

	// we change compute spec to accomply with mock server settings
	instance := DefaultComputeSpec.DeepCopy()

	instance.HTTPPort = int32Ptr(int32(s))
	pod := utils.MakeFuseQueryPod(instance, name, namespace, groupKey, isLeader)
	if isReady {
		createMockStatus(pod, v1.PodReady)
	}
	pod.Status.PodIP = str[0]
	return pod
}

func fakeK8sClienset(pods []*v1.Pod) (client kubernetes.Interface) {
	var csObjs []runtime.Object
	for _, op := range pods {
		csObjs = append(csObjs, op.DeepCopy())
	}

	client = fake.NewSimpleClientset(csObjs...)
	return client
}

func createMockStatus(pod *v1.Pod, conditionType v1.PodConditionType) {
	if pod.Status.Conditions == nil {
		pod.Status.Conditions = make([]v1.PodCondition, 0)
	}
	pod.Status.Conditions = append(pod.Status.Conditions, v1.PodCondition{Type: conditionType, Status: v1.ConditionTrue})
}
