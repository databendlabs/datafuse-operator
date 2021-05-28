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

func CreateRole(kubeClient kubernetes.Interface, ns string, relativePath string) error {
	role, err := parseRoleYaml(relativePath)
	if err != nil {
		return err
	}

	role.Namespace = ns

	_, err = kubeClient.RbacV1().Roles(ns).Get(context.TODO(), role.Name, metav1.GetOptions{})

	if err == nil {
		// Role already exists -> Update
		_, err = kubeClient.RbacV1().Roles(ns).Update(context.TODO(), role, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

	} else {
		// Role doesn't exists -> Create
		_, err = kubeClient.RbacV1().Roles(ns).Create(context.TODO(), role, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func DeleteRole(kubeClient kubernetes.Interface, ns string, relativePath string) error {
	role, err := parseRoleYaml(relativePath)
	if err != nil {
		return err
	}

	if err := kubeClient.RbacV1().Roles(ns).Delete(context.TODO(), role.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

func parseRoleYaml(relativePath string) (*rbacv1.Role, error) {
	var manifest *os.File
	var err error

	var role rbacv1.Role
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

		if out.GetKind() == "Role" {
			var marshaled []byte
			marshaled, err = out.MarshalJSON()
			json.Unmarshal(marshaled, &role)
			break
		}
	}

	if err != io.EOF && err != nil {
		return nil, err
	}
	return &role, nil
}
