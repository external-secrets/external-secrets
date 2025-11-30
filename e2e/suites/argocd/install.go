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
	"os"
	"path/filepath"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework/addon"
)

const (
	helmChartRevision = "0.0.0-e2e"
)

func installArgo() {
	By("installing argocd")
	addon.InstallGlobalAddon(&addon.HelmChart{
		Namespace:    "argocd",
		ReleaseName:  "argocd",
		Chart:        "argo-cd/argo-cd",
		ChartVersion: "8.5.0",
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
	})
}

func installESO() {
	By("installing helm http server")
	tag := os.Getenv("VERSION")
	addon.InstallGlobalAddon(&addon.HelmServer{
		ChartDir:      filepath.Join(addon.AssetDir(), "deploy/charts/external-secrets"),
		ChartRevision: helmChartRevision,
	})

	By("installing eso through argo app")
	addon.InstallGlobalAddon(&addon.ArgoCDApplication{
		Name:                 "external-secrets",
		Namespace:            "argocd",
		DestinationNamespace: "external-secrets",
		HelmChart:            "external-secrets",
		HelmRepo:             "http://e2e-helmserver.default.svc.cluster.local",
		HelmRevision:         helmChartRevision,
		HelmParameters: []string{
			"image.tag=" + tag,
			"webhook.image.tag=" + tag,
			"certController.image.tag=" + tag,
		},
	})
}
