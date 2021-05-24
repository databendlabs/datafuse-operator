package register

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubectl/pkg/util/podutils"
)

func TestBuildPodConfig(t *testing.T) {
	nodes := []ClusterNodeRequest{{Name: "foo", Address: "1.1.1.1", Priority: 1}}
	config := "foo"
	server := newHTTPTestServer(t, config, &nodes)
	defer server.Close()
	pod := makeMockPodFromServer(server, "default-default-group", "pod1", "default", true, true)
	url := strings.TrimPrefix(server.URL, "http://")
	str := strings.SplitN(url, ":", 2)
	pConfig, err := BuildPodConfig(pod)
	assert.NoError(t, err)
	assert.Equal(t, pConfig.MysqlPort, strconv.Itoa(int(*DefaultComputeSpec.MysqlPort)))
	assert.Equal(t, pConfig.ClickHousePort, strconv.Itoa(int(*DefaultComputeSpec.ClickhousePort)))
	assert.Equal(t, pConfig.MetricsPort, strconv.Itoa(int(*DefaultComputeSpec.MetricsPort)))
	assert.Equal(t, pConfig.RPCPort, strconv.Itoa(int(*DefaultComputeSpec.RPCPort)))
	assert.Equal(t, pConfig.Priority, strconv.Itoa(int(*DefaultComputeSpec.Priority)))
	assert.Equal(t, pConfig.HTTPPort, str[1])
}

func TestGetConfig(t *testing.T) {
	nodes := []ClusterNodeRequest{}
	config := "foo"
	server := newHTTPTestServer(t, config, &nodes)
	defer server.Close()
	pod := makeMockPodFromServer(server, "default-default-group", "pod1", "default", true, true)
	client := fakeK8sClienset([]*v1.Pod{pod})
	rs := RegistSetter{true, []string{}, client}
	str, err := rs.GetConfig(pod)
	assert.NoError(t, err)
	assert.Equal(t, config, str)
}

func TestGetClusterList(t *testing.T) {
	nodes := []ClusterNodeRequest{{Name: "foo", Address: "1.1.1.1", Priority: 1}}
	config := "foo"
	server := newHTTPTestServer(t, config, &nodes)
	defer server.Close()
	pod := makeMockPodFromServer(server, "default-default-group", "pod1", "default", true, true)
	client := fakeK8sClienset([]*v1.Pod{pod})
	rs := RegistSetter{true, []string{}, client}
	str, err := rs.GetConfig(pod)
	assert.NoError(t, err)
	assert.Equal(t, config, str)
	list, err := rs.GetClusterList(pod)
	assert.NoError(t, err)
	assert.Equal(t, list, nodes)
	assert.Equal(t, true, podutils.IsPodReady(pod))
}

func TestAddNode(t *testing.T) {
	nodes := []ClusterNodeRequest{{Name: "foo", Address: "1.1.1.1", Priority: 1}}
	leaderConfig := "foo"
	followerConfig := "bar"
	leaderServer := newHTTPTestServer(t, leaderConfig, &nodes)
	defer leaderServer.Close()
	followerServer := newHTTPTestServer(t, followerConfig, &[]ClusterNodeRequest{})
	defer followerServer.Close()
	leader := makeMockPodFromServer(leaderServer, "default-default-group", "pod1", "default", true, true)
	follower := makeMockPodFromServer(followerServer, "default-default-group", "pod2", "default", false, true)
	client := fakeK8sClienset([]*v1.Pod{leader, follower})
	rs := RegistSetter{true, []string{}, client}
	err := rs.AddNode(leader, follower)
	assert.NoError(t, err)
	str, err := rs.GetConfig(leader)
	assert.NoError(t, err)
	assert.Equal(t, leaderConfig, str)
	list, err := rs.GetClusterList(leader)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, list, nodes)
	assert.Equal(t, true, rs.IsGroupLeader(*leader))
	assert.Equal(t, true, rs.IsGroupWorker(*follower))
}

func TestRemoveNode(t *testing.T) {
	nodes := []ClusterNodeRequest{{Name: "foo", Address: "1.1.1.1", Priority: 1}}
	leaderConfig := "foo"
	followerConfig := "bar"
	leaderServer := newHTTPTestServer(t, leaderConfig, &nodes)
	defer leaderServer.Close()
	followerServer := newHTTPTestServer(t, followerConfig, &[]ClusterNodeRequest{})
	defer followerServer.Close()
	leader := makeMockPodFromServer(leaderServer, "default-default-group", "pod1", "default", true, true)
	follower := makeMockPodFromServer(followerServer, "default-default-group", "pod2", "default", false, true)
	client := fakeK8sClienset([]*v1.Pod{leader, follower})
	rs := RegistSetter{true, []string{}, client}
	err := rs.AddNode(leader, follower)
	assert.NoError(t, err)
	str, err := rs.GetConfig(leader)
	assert.NoError(t, err)
	assert.Equal(t, leaderConfig, str)
	list, err := rs.GetClusterList(leader)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, list, nodes)
	err = rs.RemoveNode(leader, follower)
	assert.NoError(t, err)
	list, err = rs.GetClusterList(leader)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, list, nodes)
}
