// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package register

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	controller "datafuselabs.io/datafuse-operator/pkg/controllers"
)

// Should sync with datafuse http api upstream
type ClusterNodeRequest struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Priority int    `json:"priority"`
}

type FuseQueryConfigs struct {
	MysqlPort      string `json:"mysqlPort"`
	HTTPPort       string `json:"httpPort"`
	RPCPort        string `json:"rpcPort"`
	ClickHousePort string `json:"clickhousePort"`
	MetricsPort    string `json:"metricsPort"`
	Priority       string `json:"priority"`
}

type CLUSTER_OPS string

const (
	CLUSTER_ADD    CLUSTER_OPS = "/v1/cluster/add"
	CLUSTER_REMOVE CLUSTER_OPS = "/v1/cluster/remove"
	CLUSTER_LIST   CLUSTER_OPS = "/v1/cluster/list"
	CLUSTER_CONFIG CLUSTER_OPS = "/v1/configs"
)

type RegistSetter struct {
	AllNS     bool
	Namspaces []string
	K8sClient kubernetes.Interface
}

func (r RegistSetter) GetLabelSelectors() []string {
	return []string{controller.ComputeGroupRoleLabel, controller.ComputeGroupLabel}
}

// TODO select only running instances with namespace specified
func (r RegistSetter) GetFieldSelectors() []string {
	return []string{}
}

func (r RegistSetter) IsGroupLeader(pod v1.Pod) bool {
	role, found := pod.Labels[controller.ComputeGroupRoleLabel]
	if !found {
		return false
	}
	return role == controller.ComputeGroupRoleLeader
}

func (r RegistSetter) IsGroupWorker(pod v1.Pod) bool {
	role, found := pod.Labels[controller.ComputeGroupRoleLabel]
	if !found {
		return false
	}
	return role == controller.ComputeGroupRoleFollower
}

func httpParse(url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	return "http://" + url
}

func postHTTP(url string, jsonStr []byte) error {
	validURL := httpParse(url)
	req, err := http.NewRequest("POST", validURL, bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Debug().Msgf("url %s with json %s has response Body: %v", url, string(jsonStr), string(body))
	return nil
}

// regist follower node to leader
// register name, <follower-namespace>/<follower-name>
// register ip, <follower pod ip>:<follower rpc port>
// register priority, defined in environment config
func (r RegistSetter) AddNode(leader, follower *v1.Pod) error {
	leaderConfig, err := BuildPodConfig(leader)
	if err != nil {
		return err
	}
	followerconfig, err := BuildPodConfig(follower)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s/%s", follower.Namespace, follower.Name)
	ip := fmt.Sprintf("%s:%s", follower.Status.PodIP, followerconfig.RPCPort)
	priority, err := strconv.Atoi(followerconfig.Priority)
	if err != nil {
		return fmt.Errorf("cannot convert priority to integer, %s", err.Error())
	}
	configReq := ClusterNodeRequest{Name: key, Address: ip, Priority: priority}
	receiver := fmt.Sprintf("%s:%s%s", leader.Status.PodIP, leaderConfig.HTTPPort, CLUSTER_ADD)
	fmt.Println(receiver)
	jsonbyte, err := json.Marshal(configReq)
	if err != nil {
		return err
	}
	fmt.Println(string(jsonbyte))
	err = postHTTP(receiver, jsonbyte)
	if err != nil {
		return err
	}
	return nil
}

// remove follower node from leader
// register name, <follower-namespace>/<follower-name>
// register ip, <follower pod ip>:<follower rpc port>
// register priority, defined in environment config
func (r RegistSetter) RemoveNode(leader, follower *v1.Pod) error {
	followerconfig, err := BuildPodConfig(follower)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s/%s", follower.Namespace, follower.Name)
	ip := fmt.Sprintf("%s:%s", follower.Status.PodIP, followerconfig.RPCPort)
	priority, err := strconv.Atoi(followerconfig.Priority)
	if err != nil {
		return fmt.Errorf("cannot convert priority to integer, %s", err.Error())
	}
	configReq := ClusterNodeRequest{Name: key, Address: ip, Priority: priority}
	err = r.RemoveConfig(leader, configReq)
	if err != nil {
		return err
	}
	return nil
}

func (r RegistSetter) RemoveConfig(pod *v1.Pod, config ClusterNodeRequest) error {
	leaderConfig, err := BuildPodConfig(pod)
	if err != nil {
		return err
	}
	receiver := fmt.Sprintf("%s:%s%s", pod.Status.PodIP, leaderConfig.HTTPPort, CLUSTER_REMOVE)
	jsonbyte, err := json.Marshal(config)
	if err != nil {
		return err
	}
	err = postHTTP(receiver, jsonbyte)
	if err != nil {
		return err
	}
	return nil
}

func getHTTP(url string) (body []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reponse status is not ok, status code: %d", resp.StatusCode)
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// build pod cluster info based on environment variables
func BuildPodConfig(pod *v1.Pod) (FuseQueryConfigs, error) {
	var config FuseQueryConfigs
	for _, env := range pod.Spec.Containers[0].Env {
		switch env.Name {
		case controller.FUSE_QUERY_HTTP_API_ADDRESS:
			str := strings.SplitN(env.Value, ":", 2)
			if len(str) != 2 {
				return FuseQueryConfigs{}, fmt.Errorf("cannot parse pod %s/%s http port with value %s", pod.Namespace, pod.Name, env.Value)
			}
			config.HTTPPort = str[1]
		case controller.FUSE_QUERY_MYSQL_HANDLER_PORT:
			config.MysqlPort = env.Value
		case controller.FUSE_QUERY_CLICKHOUSE_HANDLER_PORT:
			config.ClickHousePort = env.Value
		case controller.FUSE_QUERY_RPC_API_ADDRESS:
			str := strings.SplitN(env.Value, ":", 2)
			if len(str) != 2 {
				return FuseQueryConfigs{}, fmt.Errorf("cannot parse pod %s/%s rpc port with value %s", pod.Namespace, pod.Name, env.Value)
			}
			config.RPCPort = str[1]
		case controller.FUSE_QUERY_METRIC_API_ADDRESS:
			str := strings.SplitN(env.Value, ":", 2)
			if len(str) != 2 {
				return FuseQueryConfigs{}, fmt.Errorf("cannot parse pod %s/%s metrics port with value %s", pod.Namespace, pod.Name, env.Value)
			}
			config.MetricsPort = str[1]
		case controller.FUSE_QUERY_PRIORITY:
			config.Priority = env.Value
		default:
			continue
		}
	}
	return config, nil
}

// retrieve given pod's cluster information
func (r RegistSetter) GetClusterList(pod *v1.Pod) ([]ClusterNodeRequest, error) {
	var allNodes []ClusterNodeRequest
	config, err := BuildPodConfig(pod)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("http://%s:%s%s", pod.Status.PodIP, config.HTTPPort, CLUSTER_LIST)
	body, err := getHTTP(url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &allNodes)
	if err != nil {
		return nil, err
	}
	return allNodes, nil
}

// Could get throuh http://v1/config or http://v1/hello
// But it introduce some unneeded query loads
func (r RegistSetter) GetConfig(pod *v1.Pod) (string, error) {
	config, err := BuildPodConfig(pod)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("http://%s:%s%s", pod.Status.PodIP, config.HTTPPort, CLUSTER_CONFIG)
	body, err := getHTTP(url)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
