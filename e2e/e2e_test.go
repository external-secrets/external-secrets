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
package e2e

import (
	"testing"

	// nolint
	. "github.com/onsi/ginkgo"
	// nolint
	. "github.com/onsi/gomega"

	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/framework/addon"
	"github.com/external-secrets/external-secrets/e2e/framework/util"
	_ "github.com/external-secrets/external-secrets/e2e/suite"
)

var _ = SynchronizedBeforeSuite(func() []byte {
	cfg := &addon.Config{}
	cfg.KubeConfig, cfg.KubeClientSet, cfg.CRClient = framework.NewConfig()

	By("installing localstack")
	addon.InstallGlobalAddon(addon.NewLocalstack(), cfg)

	By("waiting for localstack")
	err := util.WaitForURL("http://localstack.default/health")
	Expect(err).ToNot(HaveOccurred())

	By("installing vault")
	addon.InstallGlobalAddon(addon.NewVault(), cfg)

	By("installing eso")
	addon.InstallGlobalAddon(addon.NewESO(), cfg)

	By("installing scoped eso")
	addon.InstallGlobalAddon(addon.NewScopedESO(), cfg)
	return nil
}, func([]byte) {})

var _ = SynchronizedAfterSuite(func() {}, func() {
	By("Cleaning up global addons")
	addon.UninstallGlobalAddons()
	if CurrentGinkgoTestDescription().Failed {
		addon.PrintLogs()
	}
})

func TestE2E(t *testing.T) {
	NewWithT(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "external-secrets e2e suite")
}
