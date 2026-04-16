/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

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
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	"github.com/external-secrets/external-secrets-e2e/framework/util"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases"
	v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

var _ = SynchronizedBeforeSuite(func() []byte {
	if framework.IsV2ProviderMode() {
		By("installing eso in provider v2 mode")
		addon.InstallGlobalAddon(addon.NewESO(
			addon.WithCRDs(),
			addon.WithV2Namespace(),
			addon.WithV2KubernetesProvider(),
			addon.WithV2FakeProvider(),
			addon.WithV2AWSProvider(),
			addon.WithV2GCPProvider(),
		))
		return nil
	}

	By("installing eso")
	addon.InstallGlobalAddon(addon.NewESO(addon.WithCRDs()))

	return nil
}, func([]byte) {
	// noop
})

var _ = SynchronizedAfterSuite(func() {
	// noop
}, func() {
	cfg := &addon.Config{}
	cfg.KubeConfig, cfg.KubeClientSet, cfg.CRClient = util.NewConfig()

	By("Deleting any pending generator states")
	generatorStates := &genv1alpha1.GeneratorStateList{}
	err := cfg.CRClient.List(GinkgoT().Context(), generatorStates)
	if err == nil {
		for _, generatorState := range generatorStates.Items {
			err = cfg.CRClient.Delete(GinkgoT().Context(), &generatorState)
			Expect(err).ToNot(HaveOccurred())
		}
	} else if !util.IsMissingAPIResourceError(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	By("Deleting all ClusterExternalSecrets")
	externalSecretsList := &v1.ClusterExternalSecretList{}
	err = cfg.CRClient.List(GinkgoT().Context(), externalSecretsList)
	if err == nil {
		for _, externalSecret := range externalSecretsList.Items {
			err = cfg.CRClient.Delete(GinkgoT().Context(), &externalSecret)
			Expect(err).ToNot(HaveOccurred())
		}
	} else if !util.IsMissingAPIResourceError(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	By("Cleaning up global addons")
	addon.UninstallGlobalAddons()
	if CurrentSpecReport().Failed() {
		addon.PrintLogs()
	}
})

func TestE2E(t *testing.T) {
	NewWithT(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "external-secrets e2e suite", Label("e2e"))
}
