/*
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
package framework

import (
	"os"

	// nolint
	. "github.com/onsi/ginkgo"

	// nolint
	. "github.com/onsi/gomega"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/e2e/framework/addon"
	"github.com/external-secrets/external-secrets/e2e/framework/util"
)

var Scheme = runtime.NewScheme()

func init() {
	_ = kscheme.AddToScheme(Scheme)
	_ = esv1alpha1.AddToScheme(Scheme)
}

type Framework struct {
	BaseName string

	// KubeConfig which was used to create the connection.
	KubeConfig *rest.Config

	// Kubernetes API clientsets
	KubeClientSet kubernetes.Interface

	// controller-runtime client for newer controllers
	CRClient crclient.Client

	// Namespace in which all test resources should reside
	Namespace *api.Namespace

	Addons []addon.Addon
}

// New returns a new framework instance with defaults.
func New(baseName string) *Framework {
	f := &Framework{
		BaseName: baseName,
	}
	f.KubeConfig, f.KubeClientSet, f.CRClient = NewConfig()

	BeforeEach(f.BeforeEach)
	AfterEach(f.AfterEach)

	return f
}

// BeforeEach creates a namespace.
func (f *Framework) BeforeEach() {
	var err error
	By("Building a namespace api object")
	f.Namespace, err = util.CreateKubeNamespace(f.BaseName, f.KubeClientSet)
	Expect(err).NotTo(HaveOccurred())

	By("Using the namespace " + f.Namespace.Name)
}

// AfterEach deletes the namespace and cleans up the registered addons.
func (f *Framework) AfterEach() {
	for _, a := range f.Addons {
		err := a.Uninstall()
		Expect(err).ToNot(HaveOccurred())
	}
	// reset addons to default once the run is done
	f.Addons = []addon.Addon{}
	By("deleting test namespace")
	err := util.DeleteKubeNamespace(f.Namespace.Name, f.KubeClientSet)
	Expect(err).NotTo(HaveOccurred())
}

func (f *Framework) Install(a addon.Addon) {
	f.Addons = append(f.Addons, a)

	By("installing addon")
	err := a.Setup(&addon.Config{
		KubeConfig:    f.KubeConfig,
		KubeClientSet: f.KubeClientSet,
		CRClient:      f.CRClient,
	})
	Expect(err).NotTo(HaveOccurred())

	err = a.Install()
	Expect(err).NotTo(HaveOccurred())
}

// NewConfig loads and returns the kubernetes credentials from the environment.
// KUBECONFIG env var takes precedence and falls back to in-cluster config.
func NewConfig() (*rest.Config, *kubernetes.Clientset, crclient.Client) {
	var kubeConfig *rest.Config
	var err error
	kcPath := os.Getenv("KUBECONFIG")
	if kcPath != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kcPath)
		Expect(err).NotTo(HaveOccurred())
	} else {
		kubeConfig, err = rest.InClusterConfig()
		Expect(err).NotTo(HaveOccurred())
	}

	By("creating a kubernetes client")
	kubeClientSet, err := kubernetes.NewForConfig(kubeConfig)
	Expect(err).NotTo(HaveOccurred())

	By("creating a controller-runtime client")
	CRClient, err := crclient.New(kubeConfig, crclient.Options{Scheme: Scheme})
	Expect(err).NotTo(HaveOccurred())

	return kubeConfig, kubeClientSet, CRClient
}
