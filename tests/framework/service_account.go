// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package framework

import (
	"context"
	"encoding/json"
	"io"
	"os"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
)

func CreateServiceAccount(kubeClient kubernetes.Interface, namespace string, relativePath string) (FinalizerFn, error) {
	finalizerFn := func() error {
		return DeleteServiceAccount(kubeClient, namespace, relativePath)
	}

	serviceAccount, err := parseServiceAccountYaml(relativePath)
	if err != nil {
		return finalizerFn, err
	}
	serviceAccount.Namespace = namespace
	_, err = kubeClient.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), serviceAccount, metav1.CreateOptions{})
	if err != nil {
		return finalizerFn, err
	}

	return finalizerFn, nil
}

func parseServiceAccountYaml(relativePath string) (*v1.ServiceAccount, error) {
	var manifest *os.File
	var err error

	var serviceAccount v1.ServiceAccount
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

		if out.GetKind() == "ServiceAccount" {
			var marshaled []byte
			marshaled, err = out.MarshalJSON()
			json.Unmarshal(marshaled, &serviceAccount)
			break
		}
	}

	if err != io.EOF && err != nil {
		return nil, err
	}
	return &serviceAccount, nil
}

func DeleteServiceAccount(kubeClient kubernetes.Interface, namespace string, relativePath string) error {
	serviceAccount, err := parseServiceAccountYaml(relativePath)
	if err != nil {
		return err
	}

	return kubeClient.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), serviceAccount.Name, metav1.DeleteOptions{})
}
