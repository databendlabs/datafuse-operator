package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd"
)

func TestGetKubeConfigLocation(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		original := os.Getenv("KUBECONFIG")
		os.Setenv("KUBECONFIG", "")
		defer os.Setenv("KUBECONFIG", original)

		config := GetKubeConfigLocation()
		assert.Equal(t, config, clientcmd.RecommendedHomeFile)
	})

	t.Run("KUBECONFIG", func(t *testing.T) {
		actual := "./testconfig"
		original := os.Getenv("KUBECONFIG")
		os.Setenv("KUBECONFIG", actual)
		defer os.Setenv("KUBECONFIG", original)

		config := GetKubeConfigLocation()
		assert.Equal(t, config, actual)
	})
	t.Run("local", func(t *testing.T) {
		actual := "./testconfig"
		original := os.Getenv("KUBECONFIG")
		os.Setenv("KUBECONFIG", "")
		defer os.Setenv("KUBECONFIG", original)

		KubeConfig = actual
		config := GetKubeConfigLocation()
		assert.Equal(t, config, actual)
		KubeConfig = "" //cleanup
	})
}
