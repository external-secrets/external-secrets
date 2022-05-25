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

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	api "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets/e2e/framework/addon"
	"github.com/external-secrets/external-secrets/e2e/framework/log"
	"github.com/external-secrets/external-secrets/e2e/framework/util"
)

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
	f.KubeConfig, f.KubeClientSet, f.CRClient = util.NewConfig()

	BeforeEach(f.BeforeEach)
	AfterEach(f.AfterEach)

	return f
}

// BeforeEach creates a namespace.
func (f *Framework) BeforeEach() {
	var err error
	f.Namespace, err = util.CreateKubeNamespace(f.BaseName, f.KubeClientSet)
	log.Logf("created test namespace %s", f.Namespace.Name)
	Expect(err).ToNot(HaveOccurred())
}

// AfterEach deletes the namespace and cleans up the registered addons.
func (f *Framework) AfterEach() {
	for _, a := range f.Addons {
		if CurrentSpecReport().Failed() {
			err := a.Logs()
			Expect(err).ToNot(HaveOccurred())
		}
		err := a.Uninstall()
		Expect(err).ToNot(HaveOccurred())
	}
	// reset addons to default once the run is done
	f.Addons = []addon.Addon{}
	log.Logf("deleting test namespace %s", f.Namespace.Name)
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

// Compose helps define multiple testcases with same/different auth methods.
func Compose(descAppend string, f *Framework, fn func(f *Framework) (string, func(*TestCase)), tweaks ...func(*TestCase)) TableEntry {
	// prepend common fn to tweaks
	desc, cfn := fn(f)
	tweaks = append([]func(*TestCase){cfn}, tweaks...)

	// need to convert []func to []interface{}
	ifs := make([]interface{}, len(tweaks))
	for i := 0; i < len(tweaks); i++ {
		ifs[i] = tweaks[i]
	}
	te := Entry(desc+" "+descAppend, ifs...)

	return te
}
