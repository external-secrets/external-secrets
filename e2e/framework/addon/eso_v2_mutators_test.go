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

package addon

import "testing"

func TestWithV2FakeProvider(t *testing.T) {
	t.Setenv("VERSION", "test-version")

	eso := NewESO(WithV2FakeProvider())

	assertVarValue(t, eso.HelmChart, "providers.enabled", "true")
	assertVarValue(t, eso.HelmChart, "providers.list[1].name", "fake")
	assertVarValue(t, eso.HelmChart, "providers.list[1].type", "fake")
	assertVarValue(t, eso.HelmChart, "providers.list[1].enabled", "true")
	assertVarValue(t, eso.HelmChart, "providers.list[1].replicaCount", "1")
	assertVarValue(t, eso.HelmChart, "providers.list[1].image.repository", "ghcr.io/external-secrets/provider-fake")
	assertVarValue(t, eso.HelmChart, "providers.list[1].image.tag", "test-version")
	assertVarValue(t, eso.HelmChart, "providers.list[1].image.pullPolicy", "IfNotPresent")
}

func TestWithV2ProvidersCompose(t *testing.T) {
	t.Setenv("VERSION", "test-version")

	eso := NewESO(
		WithV2Namespace(),
		WithV2KubernetesProvider(),
		WithV2FakeProvider(),
	)

	if eso.HelmChart.Namespace != v2HelmNamespace {
		t.Fatalf("expected namespace %q, got %q", v2HelmNamespace, eso.HelmChart.Namespace)
	}
	if eso.HelmChart.ReleaseName != v2HelmReleaseName {
		t.Fatalf("expected release name %q, got %q", v2HelmReleaseName, eso.HelmChart.ReleaseName)
	}
	if !containsArg(eso.HelmChart.Args, "--create-namespace") {
		t.Fatalf("expected --create-namespace arg, got %v", eso.HelmChart.Args)
	}

	assertVarValue(t, eso.HelmChart, "v2.enabled", "true")
	assertVarValue(t, eso.HelmChart, "providers.enabled", "true")
	assertVarValue(t, eso.HelmChart, "providers.list[0].name", "kubernetes")
	assertVarValue(t, eso.HelmChart, "providers.list[0].image.repository", "ghcr.io/external-secrets/provider-kubernetes")
	assertVarValue(t, eso.HelmChart, "providers.list[0].image.tag", "test-version")
	assertVarValue(t, eso.HelmChart, "providers.list[1].name", "fake")
	assertVarValue(t, eso.HelmChart, "providers.list[1].type", "fake")
	assertVarValue(t, eso.HelmChart, "providers.list[1].image.repository", "ghcr.io/external-secrets/provider-fake")
	assertVarValue(t, eso.HelmChart, "providers.list[1].image.tag", "test-version")
}

func assertVarValue(t *testing.T, chart *HelmChart, key, wantValue string) {
	t.Helper()

	for _, variable := range chart.Vars {
		if variable.Key == key {
			if variable.Value != wantValue {
				t.Fatalf("expected %s=%s, got %s", key, wantValue, variable.Value)
			}
			return
		}
	}

	t.Fatalf("expected %s=%s to be set", key, wantValue)
}
