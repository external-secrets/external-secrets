/*
Copyright © 2025 ESO Maintainer Team

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

package argocd

import (
	"testing"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"

	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	"github.com/external-secrets/external-secrets-e2e/framework/util"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

var _ = SynchronizedBeforeSuite(func() []byte {
	installArgo()
	installESO()
	return nil
}, func([]byte) {
	// noop
})

var _ = SynchronizedAfterSuite(func() {
	// noop
}, func() {
	_, _, cl := util.NewConfig()
	By("Deleting any pending generator states")
	generatorStates := &genv1alpha1.GeneratorStateList{}
	err := cl.List(GinkgoT().Context(), generatorStates)
	Expect(err).ToNot(HaveOccurred())
	for _, generatorState := range generatorStates.Items {
		err = cl.Delete(GinkgoT().Context(), &generatorState)
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
	RunSpecs(t, "external-secrets argocd e2e suite", Label("argocd"))
}
