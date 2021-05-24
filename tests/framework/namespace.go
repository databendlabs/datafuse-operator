package framework

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreateNamespace(kubeClient kubernetes.Interface, name string) (*v1.Namespace, error) {
	namespace, err := kubeClient.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	},
		metav1.CreateOptions{},
	)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to create namespace with name %v", name))
	}
	return namespace, nil
}

func (ctx *TestContext) CreateNamespace(t *testing.T, kubeClient kubernetes.Interface) string {
	name := ctx.GetObjID()
	if _, err := CreateNamespace(kubeClient, name); err != nil {
		t.Fatal(err)
	}

	namespaceFinalizerFn := func() error {
		return DeleteNamespace(kubeClient, name)
	}

	ctx.AddFinalizerFn(namespaceFinalizerFn)

	return name
}

func DeleteNamespace(kubeClient kubernetes.Interface, name string) error {
	return kubeClient.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
}
