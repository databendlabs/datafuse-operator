// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package framework

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"datafuselabs.io/datafuse-operator/pkg/apis/datafuse/v1alpha1"
	crdclientset "datafuselabs.io/datafuse-operator/pkg/client/clientset/versioned"
	testutils "datafuselabs.io/datafuse-operator/tests/utils"
)

func MakeDatafuseOperatorFromYaml(pathToYaml string) (*v1alpha1.DatafuseOperator, error) {
	manifest, err := testutils.PathToOSFile(pathToYaml)
	if err != nil {
		return nil, err
	}
	operator := v1alpha1.DatafuseOperator{}
	if err := yaml.NewYAMLOrJSONDecoder(manifest, 100).Decode(&operator); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to decode file %s", pathToYaml))
	}

	return &operator, nil
}

func CreateDatafuseOperator(crdclientset crdclientset.Interface, namespace string, op *v1alpha1.DatafuseOperator) error {
	_, err := crdclientset.DatafuseV1alpha1().DatafuseOperators(namespace).Create(context.TODO(), op, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to create DatafuseOperator %s in %s", op.Name, op.Namespace))
	}
	return nil
}

func UpdateDatafuseOperator(crdclientset crdclientset.Interface, namespace string, op *v1alpha1.DatafuseOperator) error {
	_, err := crdclientset.DatafuseV1alpha1().DatafuseOperators(namespace).Update(context.TODO(), op, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to create DatafuseOperator %s in %s", op.Name, op.Namespace))
	}
	return nil
}

func GetDatafuseOperator(crdclientset crdclientset.Interface, namespace string, name string) error {
	_, err := crdclientset.DatafuseV1alpha1().DatafuseOperators(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to create DatafuseOperator %s in %s", name, namespace))
	}
	return nil
}

func DeleteDatafuseOperator(crdclientset crdclientset.Interface, namespace string, name string) error {
	err := crdclientset.DatafuseV1alpha1().DatafuseOperators(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to create DatafuseOperator %s in %s", name, namespace))
	}
	return nil
}
