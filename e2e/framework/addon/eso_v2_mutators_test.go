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

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestWithV2FakeProvider(t *testing.T) {
	t.Setenv("VERSION", "test-version")

	eso := NewESO(WithV2FakeProvider())

	assertV2ProviderBaseVars(t, eso.HelmChart)
	assertVarValue(t, eso.HelmChart, "providers.enabled", "true")
	assertProvider(
		t,
		eso.HelmChart,
		"fake",
		"fake",
		"ghcr.io/external-secrets/provider-fake",
		"test-version",
	)
	assertSequentialProviderIndexes(t, eso.HelmChart)

	providers := providerEntries(t, eso.HelmChart)
	if len(providers) != 1 {
		t.Fatalf("expected exactly one provider entry, got %d", len(providers))
	}
	if providers[0].Name != "fake" {
		t.Fatalf("expected fake to be at index 0 when standalone, got index 0 name %q", providers[0].Name)
	}
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

	assertV2ProviderBaseVars(t, eso.HelmChart)
	assertVarValue(t, eso.HelmChart, "providers.enabled", "true")
	assertProvider(
		t,
		eso.HelmChart,
		"kubernetes",
		"kubernetes",
		"ghcr.io/external-secrets/provider-kubernetes",
		"test-version",
	)
	assertProvider(
		t,
		eso.HelmChart,
		"fake",
		"fake",
		"ghcr.io/external-secrets/provider-fake",
		"test-version",
	)
	assertSequentialProviderIndexes(t, eso.HelmChart)
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

func assertV2ProviderBaseVars(t *testing.T, chart *HelmChart) {
	t.Helper()

	assertVarValue(t, chart, "replicaCount", "1")
	assertVarValue(t, chart, "v2.enabled", "true")
	assertVarValue(t, chart, "crds.createProvider", "true")
	assertVarValue(t, chart, "crds.createClusterProvider", "true")
	assertVarValue(t, chart, "providerDefaults.replicaCount", "1")
}

func assertProvider(t *testing.T, chart *HelmChart, name, providerType, imageRepository, imageTag string) {
	t.Helper()

	for _, provider := range providerEntries(t, chart) {
		if provider.Name != name {
			continue
		}
		if provider.Type != providerType {
			t.Fatalf("expected provider %q to have type %q, got %q", name, providerType, provider.Type)
		}
		if provider.Enabled != "true" {
			t.Fatalf("expected provider %q to be enabled, got %q", name, provider.Enabled)
		}
		if provider.ReplicaCount != "1" {
			t.Fatalf("expected provider %q replicaCount 1, got %q", name, provider.ReplicaCount)
		}
		if provider.ImageRepository != imageRepository {
			t.Fatalf("expected provider %q image repository %q, got %q", name, imageRepository, provider.ImageRepository)
		}
		if provider.ImageTag != imageTag {
			t.Fatalf("expected provider %q image tag %q, got %q", name, imageTag, provider.ImageTag)
		}
		if provider.ImagePullPolicy != "IfNotPresent" {
			t.Fatalf("expected provider %q image pull policy IfNotPresent, got %q", name, provider.ImagePullPolicy)
		}
		return
	}

	t.Fatalf("expected provider %q to exist", name)
}

func assertSequentialProviderIndexes(t *testing.T, chart *HelmChart) {
	t.Helper()

	providers := providerEntries(t, chart)
	for i := 0; i < len(providers); i++ {
		if _, ok := providers[i]; !ok {
			t.Fatalf("expected provider index %d to exist, got indexes %v", i, sortedProviderIndexes(providers))
		}
	}
}

type providerEntry struct {
	Name            string
	Type            string
	Enabled         string
	ReplicaCount    string
	ImageRepository string
	ImageTag        string
	ImagePullPolicy string
}

var providerVarPattern = regexp.MustCompile(`^providers\.list\[(\d+)\]\.(.+)$`)

func providerEntries(t *testing.T, chart *HelmChart) map[int]providerEntry {
	t.Helper()

	providers := make(map[int]providerEntry)
	for _, variable := range chart.Vars {
		matches := providerVarPattern.FindStringSubmatch(variable.Key)
		if matches == nil {
			continue
		}
		index, err := strconv.Atoi(matches[1])
		if err != nil {
			t.Fatalf("unable to parse provider index from key %q: %v", variable.Key, err)
		}
		field := matches[2]

		entry := providers[index]
		switch field {
		case "name":
			entry.Name = variable.Value
		case "type":
			entry.Type = variable.Value
		case "enabled":
			entry.Enabled = variable.Value
		case "replicaCount":
			entry.ReplicaCount = variable.Value
		case "image.repository":
			entry.ImageRepository = variable.Value
		case "image.tag":
			entry.ImageTag = variable.Value
		case "image.pullPolicy":
			entry.ImagePullPolicy = variable.Value
		default:
			if strings.HasPrefix(field, "image.") {
				t.Fatalf("unexpected provider image field %q in key %q", field, variable.Key)
			}
		}
		providers[index] = entry
	}
	return providers
}

func sortedProviderIndexes(providers map[int]providerEntry) []int {
	indexes := make([]int, 0, len(providers))
	for index := range providers {
		indexes = append(indexes, index)
	}

	for i := 0; i < len(indexes); i++ {
		for j := i + 1; j < len(indexes); j++ {
			if indexes[j] < indexes[i] {
				indexes[i], indexes[j] = indexes[j], indexes[i]
			}
		}
	}

	return indexes
}
