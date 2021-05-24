package framework

import (
	"context"
	"fmt"
	"time"

	testutils "datafuselabs.io/datafuse-operator/tests/utils"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
)

// TODO add tests
func MakeDeployment(pathToYaml string) (*appsv1.Deployment, error) {
	manifest, err := testutils.PathToOSFile(pathToYaml)
	if err != nil {
		return nil, err
	}
	deployment := appsv1.Deployment{}
	if err := yaml.NewYAMLOrJSONDecoder(manifest, 100).Decode(&deployment); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to decode file %s", pathToYaml))
	}

	return &deployment, nil
}

func CreateDeployment(kubeClient kubernetes.Interface, namespace string, d *appsv1.Deployment) error {
	d.Namespace = namespace
	_, err := kubeClient.AppsV1().Deployments(namespace).Create(context.TODO(), d, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to create deployment %s", d.Name))
	}
	return nil
}

func DeleteDeployment(kubeClient kubernetes.Interface, namespace, name string) error {
	d, err := kubeClient.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	zero := int32(0)
	d.Spec.Replicas = &zero

	d, err = kubeClient.AppsV1().Deployments(namespace).Update(context.TODO(), d, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return kubeClient.AppsV1beta2().Deployments(namespace).Delete(context.TODO(), d.Name, metav1.DeleteOptions{})
}

func WaitUntilDeploymentGone(kubeClient kubernetes.Interface, namespace, name string, timeout time.Duration) error {
	return wait.Poll(time.Second, timeout, func() (bool, error) {
		_, err := kubeClient.
			AppsV1beta2().Deployments(namespace).
			Get(context.TODO(), name, metav1.GetOptions{})

		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		return false, nil
	})
}
