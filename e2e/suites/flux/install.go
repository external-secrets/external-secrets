/*
Copyright 2020 The cert-manager Authors.
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
	"fmt"
	"os"
	"os/exec"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"

	"github.com/external-secrets/external-secrets-e2e/framework/addon"
)

const (
	helmChartRevision = "0.0.0-e2e"
)

func installFlux() {
	By("installing flux")
	fluxVersion := "v0.29.3"
	url := fmt.Sprintf("https://github.com/fluxcd/flux2/releases/download/%s/install.yaml", fluxVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	out, err := cmd.CombinedOutput()
	Expect(err).ToNot(HaveOccurred(), string(out))
}

func installESO(cfg *addon.Config) {
	By("installing helm http server")
	addon.InstallGlobalAddon(&addon.HelmServer{
		ChartDir:      "/k8s/deploy/charts/external-secrets",
		ChartRevision: helmChartRevision,
	}, cfg)

	By("installing eso through flux helmrelease app")
	tag := os.Getenv("VERSION")
	addon.InstallGlobalAddon(&addon.FluxHelmRelease{
		Name:            "external-secrets",
		Namespace:       "flux-system",
		TargetNamespace: "external-secrets",
		HelmChart:       "external-secrets",
		HelmRepo:        "http://e2e-helmserver.default.svc.cluster.local",
		HelmRevision:    helmChartRevision,
		HelmValues: fmt.Sprintf(`{
			"installCRDs": true,
			"image": {
			  "tag": "%s"
			},
			"webhook": {
			  "image": {
				"tag": "%s"
			  }
			},
			"certController": {
			  "image": {
				"tag": "%s"
			  }
			}
		  }`, tag, tag, tag),
	}, cfg)
}
