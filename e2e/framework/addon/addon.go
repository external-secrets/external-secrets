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

package addon

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework/log"
)

var globalAddons []Addon

func init() {
	globalAddons = make([]Addon, 0)
}

type Config struct {
	// KubeConfig which was used to create the connection.
	KubeConfig *rest.Config

	// Kubernetes API clientsets
	KubeClientSet kubernetes.Interface

	// controller-runtime client for newer controllers
	CRClient crclient.Client
}

type Addon interface {
	Setup(*Config) error
	Install() error
	Logs() error
	Uninstall() error
}

func InstallGlobalAddon(addon Addon, cfg *Config) {
	globalAddons = append(globalAddons, addon)

	ginkgo.By("installing global addon")
	err := addon.Setup(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	err = addon.Install()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func UninstallGlobalAddons() {
	for _, addon := range globalAddons {
		ginkgo.By("uninstalling addon")
		err := addon.Uninstall()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}
}

func PrintLogs() {
	for _, addon := range globalAddons {
		err := addon.Logs()
		if err != nil {
			log.Logf("error fetching logs: %s", err.Error())
		}
	}
}
