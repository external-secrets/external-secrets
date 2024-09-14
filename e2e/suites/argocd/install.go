//Copyright External Secrets Inc. All Rights Reserved

package argocd

import (
	"os"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework/addon"
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
		HelmParameters: []string{
			"image.tag=" + tag,
			"webhook.image.tag=" + tag,
			"certController.image.tag=" + tag,
		},
	}, cfg)
}
