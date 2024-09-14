//Copyright External Secrets Inc. All Rights Reserved

package generator

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

	By("installing eso")
	addon.InstallGlobalAddon(addon.NewESO(addon.WithCRDs()), cfg)

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
	RunSpecs(t, "external-secrets generator e2e suite", Label("generator"))
}
