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

package main

import (
	"strings"
	"testing"
)

func TestMainTemplateUsesGeneratorMappingType(t *testing.T) {
	tmpl, err := loadTemplate("templates/main.go.tmpl")
	if err != nil {
		t.Fatalf("load template: %v", err)
	}

	data := prepareTemplateData(&ProviderConfig{
		Provider: providerMetadata{
			Name:        "fake",
			DisplayName: "Fake",
			V2Package:   "github.com/external-secrets/external-secrets/apis/provider/fake/v2alpha1",
		},
		Stores: []storeConfig{{
			GVK: gvkConfig{
				Group:   "provider.external-secrets.io",
				Version: "v2alpha1",
				Kind:    "Fake",
			},
			V1Provider:     "github.com/external-secrets/external-secrets/providers/v2/fake/store",
			V1ProviderFunc: "NewProvider",
		}},
		Generators: []generatorConfig{{
			GVK: gvkConfig{
				Group:   "generators.external-secrets.io",
				Version: "v1alpha1",
				Kind:    "Fake",
			},
			V1Generator:     "github.com/external-secrets/external-secrets/providers/v2/fake/generator",
			V1GeneratorFunc: "NewGenerator",
		}},
	})

	rendered, err := executeTemplate(tmpl, data)
	if err != nil {
		t.Fatalf("execute template: %v", err)
	}

	renderedText := string(rendered)
	if strings.Contains(renderedText, "adaptergenerator.GeneratorMapping") {
		t.Fatalf("main template rendered stale generator mapping type:\n%s", renderedText)
	}
	if !strings.Contains(renderedText, "adaptergenerator.Mapping") {
		t.Fatalf("main template did not render adaptergenerator.Mapping:\n%s", renderedText)
	}
}

func TestMainTemplateStartsProviderMetricsServer(t *testing.T) {
	tmpl, err := loadTemplate("templates/main.go.tmpl")
	if err != nil {
		t.Fatalf("load template: %v", err)
	}

	data := prepareTemplateData(&ProviderConfig{
		Provider: providerMetadata{
			Name:        "kubernetes",
			DisplayName: "Kubernetes",
			V2Package:   "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1",
		},
		Stores: []storeConfig{{
			GVK: gvkConfig{
				Group:   "provider.external-secrets.io",
				Version: "v2alpha1",
				Kind:    "Kubernetes",
			},
			V1Provider:     "github.com/external-secrets/external-secrets/providers/v1/kubernetes",
			V1ProviderFunc: "NewProvider",
		}},
	})

	rendered, err := executeTemplate(tmpl, data)
	if err != nil {
		t.Fatalf("execute template: %v", err)
	}

	renderedText := string(rendered)
	if !strings.Contains(renderedText, "grpcserver.RunProviderServer(grpcserver.RuntimeOptions{") {
		t.Fatalf("main template did not use the shared provider runtime:\n%s", renderedText)
	}
	if !strings.Contains(renderedText, "Register: func(registrar grpc.ServiceRegistrar)") {
		t.Fatalf("main template did not register services through the shared provider runtime:\n%s", renderedText)
	}
}
