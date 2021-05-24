package utils

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var KubeConfig string

func GetKubeConfigLocation() string {
	kubeconfig := KubeConfig
	if kubeconfig != "" {
		return kubeconfig
	}
	kubeconfig = os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = clientcmd.RecommendedHomeFile
	}
	return kubeconfig
}
func GetK8sConfig() (*rest.Config, error) {
	kubeconfig := GetKubeConfigLocation()
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error building config from kubeconfig located in %s: %w", kubeconfig, err)
	}
	return config, nil
}

func GetK8sClient() (*kubernetes.Clientset, error) {
	config, err := GetK8sConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	kubeCli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate k8s client: %w", err)
	}

	return kubeCli, nil
}
