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
package argocd

import (
	"fmt"
	"os"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets/e2e/framework/addon"
)

const (
	helmChartRevision = "0.0.0-e2e"
)

func installArgo(cfg *addon.Config) {
	By("installing argocd")
	addon.InstallGlobalAddon(&addon.HelmChart{
		Namespace:    "argocd",
		ReleaseName:  "argocd",
		Chart:        "argo-cd/argo-cd",
		ChartVersion: "3.35.4",
		Repo: addon.ChartRepo{
			Name: "argo-cd",
			URL:  "https://argoproj.github.io/argo-helm",
		},
		Vars: []addon.StringTuple{
			{
				Key:   "controller.args.appResyncPeriod",
				Value: "15",
			},
		},
		Args: []string{"--create-namespace"},
	}, cfg)
}

func installESO(cfg *addon.Config) {
	By("installing helm http server")
	repo := os.Getenv("IMAGE_REGISTRY")
	tag := os.Getenv("VERSION")
	addon.InstallGlobalAddon(&addon.HelmServer{
		ChartDir:      "/k8s/deploy/charts/external-secrets",
		ChartRevision: helmChartRevision,
	}, cfg)

	By("installing eso through argo app")
	addon.InstallGlobalAddon(&addon.ArgoCDApplication{
		Name:                 "external-secrets",
		Namespace:            "argocd",
		DestinationNamespace: "external-secrets",
		HelmChart:            "external-secrets",
		HelmRepo:             "http://e2e-helmserver.default.svc.cluster.local",
		HelmRevision:         helmChartRevision,
		HelmValues: fmt.Sprintf(`
installCRDs: true
image:
  repository: %s
  tag: %s
webhook:
  image:
    repository: %s
    tag: %s
certController:
  image:
    repository: %s
    tag: %s`, repo, tag, repo, tag, repo, tag),
	}, cfg)
}
