package main

import (
	"strings"
	"testing"
)

func TestMainTemplateStartsMetricsServer(t *testing.T) {
	tmpl, err := loadTemplate("templates/main.go.tmpl")
	if err != nil {
		t.Fatalf("loadTemplate() error = %v", err)
	}

	data := prepareTemplateData(&ProviderConfig{
		Provider: providerMetadata{
			Name:        "kubernetes",
			DisplayName: "Kubernetes",
			V2Package:   "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1",
		},
		Stores: []storeConfig{
			{
				GVK: gvkConfig{
					Group:   "provider.external-secrets.io",
					Version: "v2alpha1",
					Kind:    "Kubernetes",
				},
				V1Provider:     "github.com/external-secrets/external-secrets/providers/v1/kubernetes",
				V1ProviderFunc: "NewProvider",
			},
		},
		ConfigPackage: "./config.go",
	})

	rendered, err := executeTemplate(tmpl, data)
	if err != nil {
		t.Fatalf("executeTemplate() error = %v", err)
	}

	output := string(rendered)
	expectedSnippets := []string{
		"ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)",
		"metricsServer := grpcserver.NewMetricsServer(grpcserver.DefaultMetricsPort, nil)",
		"if err := grpcserver.RegisterMetrics(metricsServer.GetRegistry()); err != nil {",
		"go func() {",
		"if err := metricsServer.Start(ctx); err != nil {",
		"log.Fatalf(\"Failed to start metrics server: %v\", err)",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("generated main.go missing snippet %q\noutput:\n%s", snippet, output)
		}
	}
}
