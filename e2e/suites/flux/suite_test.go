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

package flux

import (
	"testing"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"

	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	"github.com/external-secrets/external-secrets-e2e/framework/util"
)

var _ = SynchronizedBeforeSuite(func() []byte {
	cfg := &addon.Config{}
	cfg.KubeConfig, cfg.KubeClientSet, cfg.CRClient = util.NewConfig()
	installFlux()
	installESO(cfg)
	return nil
}, func([]byte) {
	// noop
})

var _ = SynchronizedAfterSuite(func() {
	// noop
}, func() {
	By("Cleaning up global addons")
	addon.UninstallGlobalAddons()
	if CurrentSpecReport().Failed() {
		addon.PrintLogs()
	}
})

func TestE2E(t *testing.T) {
	NewWithT(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "external-secrets flux e2e suite", Label("flux"))
}
