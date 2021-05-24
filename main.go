/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/namsral/flag"
	"github.com/rs/zerolog/log"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	//+kubebuilder:scaffold:imports
	datafusev1alpha1 "datafuselabs.io/datafuse-operator/pkg/apis/datafuse/v1alpha1"
	crdclientset "datafuselabs.io/datafuse-operator/pkg/client/clientset/versioned"
	crdinformers "datafuselabs.io/datafuse-operator/pkg/client/informers/externalversions"
	"datafuselabs.io/datafuse-operator/pkg/controllers/operator"
	register "datafuselabs.io/datafuse-operator/pkg/controllers/register"
	"datafuselabs.io/datafuse-operator/pkg/controllers/utils"
	datafuseutils "datafuselabs.io/datafuse-operator/utils"
)

var (
	scheme              = runtime.NewScheme()
	master              = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	controllerThreads   = flag.Int("controller-threads", 10, "Number of worker threads used by the SparkApplication controller.")
	resyncInterval      = flag.Int("resync-interval", 30, "Informer resync interval in seconds.")
	namespace           = flag.String("namespace", apiv1.NamespaceAll, "The Kubernetes namespace to manage. Will manage custom resource objects of the managed CRD types for the whole cluster if unset.")
	metricsAddr         = flag.String("metrics-addr", ":8080", "The address the metric endpoint binds to.")
	enableLeaderElectio = flag.Bool("enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(datafusev1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	flag.StringVar(&datafuseutils.KubeConfig, "kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	flag.Parse()
	stopCh := datafuseutils.SetupSignalHandler()
	log.Info().Msg(datafuseutils.KubeConfig)
	kubeConfig, err := buildConfig(*master, datafuseutils.KubeConfig)
	if err != nil {
		log.Fatal().Msgf("Error building kubernetes config: %s", err)
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatal().Msgf("Error building kubernetes client: %s", err)
	}
	crdClient, err := crdclientset.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatal().Msgf("Error building crd client: %s", err)
	}
	// TODO prometheus etc controllers

	var (
		kubeInformerFactory kubeinformers.SharedInformerFactory
		crdInformerFactory  crdinformers.SharedInformerFactory
	)
	if *namespace != apiv1.NamespaceAll {
		kubeInformerFactory = kubeinformers.NewSharedInformerFactoryWithOptions(kubeClient, time.Second*30, kubeinformers.WithNamespace(*namespace))
		crdInformerFactory = crdinformers.NewSharedInformerFactoryWithOptions(crdClient, time.Second*30, crdinformers.WithNamespace(*namespace))
	} else {
		kubeInformerFactory = kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
		crdInformerFactory = crdinformers.NewSharedInformerFactory(crdClient, time.Second*30)
	}
	// controller := cluster.NewController(kubeClient, crdClient,
	// 	kubeInformerFactory.Apps().V1().Deployments(),
	// 	kubeInformerFactory.Core().V1().Services(),
	// 	crdInformerFactory.Datafuse().V1alpha1().DatafuseComputeGroups(),
	// 	crdInformerFactory.Datafuse().V1alpha1().DatafuseOperators(),
	// )
	controller := operator.NewController(&utils.OperatorSetter{K8sClient: kubeClient, Client: crdClient, AllNS: true}, kubeInformerFactory.Apps().V1().Deployments(), crdInformerFactory.Datafuse().V1alpha1().DatafuseComputeGroups(), crdInformerFactory.Datafuse().V1alpha1().DatafuseOperators())
	rc, err := register.NewRegisterController(&register.RegistSetter{AllNS: true, Namspaces: []string{}, K8sClient: kubeClient})
	go kubeInformerFactory.Start(stopCh)
	go crdInformerFactory.Start(stopCh)
	go controller.Start(12, stopCh)
	rc.Run(stopCh)
}

func buildConfig(masterURL string, kubeConfig string) (*rest.Config, error) {
	if kubeConfig != "" {
		return clientcmd.BuildConfigFromFlags(masterURL, kubeConfig)
	}
	return rest.InClusterConfig()
}
