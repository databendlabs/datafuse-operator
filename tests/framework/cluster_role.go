// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package framework

import (
	"context"
	"encoding/json"
	"io"
	"os"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
)

func CreateClusterRole(kubeClient kubernetes.Interface, relativePath string) error {
	clusterRole, err := parseClusterRoleYaml(relativePath)
	if err != nil {
		return err
	}

	_, err = kubeClient.RbacV1().ClusterRoles().Get(context.TODO(), clusterRole.Name, metav1.GetOptions{})

	if err == nil {
		// ClusterRole already exists -> Update
		_, err = kubeClient.RbacV1().ClusterRoles().Update(context.TODO(), clusterRole, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

	} else {
		// ClusterRole doesn't exists -> Create
		_, err = kubeClient.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func DeleteClusterRole(kubeClient kubernetes.Interface, relativePath string) error {
	clusterRole, err := parseClusterRoleYaml(relativePath)
	if err != nil {
		return err
	}

	if err := kubeClient.RbacV1().ClusterRoles().Delete(context.TODO(), clusterRole.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

func parseClusterRoleYaml(relativePath string) (*rbacv1.ClusterRole, error) {
	var manifest *os.File
	var err error

	var clusterRole rbacv1.ClusterRole
	if manifest, err = PathToOSFile(relativePath); err != nil {
		return nil, err
	}

	decoder := yaml.NewYAMLOrJSONDecoder(manifest, 100)
	for {
		var out unstructured.Unstructured
		err = decoder.Decode(&out)
		if err != nil {
			// this would indicate it's malformed YAML.
			break
		}

		if out.GetKind() == "ClusterRole" {
			var marshaled []byte
			marshaled, err = out.MarshalJSON()
			json.Unmarshal(marshaled, &clusterRole)
			break
		}
	}

	if err != io.EOF && err != nil {
		return nil, err
	}
	return &clusterRole, nil
}
