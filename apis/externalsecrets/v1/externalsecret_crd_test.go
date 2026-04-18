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

package v1

import (
	"os"
	"path/filepath"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestExternalSecretCRDSecretStoreRefKindOmitsProviderKinds(t *testing.T) {
	kindEnum := externalSecretCRDKindEnum(t, "v1")

	assertContains := func(value string) {
		t.Helper()
		for _, candidate := range kindEnum {
			if candidate == value {
				return
			}
		}
		t.Fatalf("kind enum does not contain %q: %v", value, kindEnum)
	}

	assertNotContains := func(value string) {
		t.Helper()
		for _, candidate := range kindEnum {
			if candidate == value {
				t.Fatalf("kind enum unexpectedly contains %q: %v", value, kindEnum)
			}
		}
	}

	assertContains("ProviderStore")
	assertContains("ClusterProviderStore")
	assertNotContains("Provider")
	assertNotContains("ClusterProvider")
}

func externalSecretCRDKindEnum(t *testing.T, versionName string) []string {
	t.Helper()

	crdPath := filepath.Join("..", "..", "..", "config", "crds", "bases", "external-secrets.io_externalsecrets.yaml")
	data, err := os.ReadFile(crdPath)
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}

	var crd map[string]any
	if err := yaml.Unmarshal(data, &crd); err != nil {
		t.Fatalf("unmarshal CRD: %v", err)
	}

	versions := asSlice(t, asMap(t, crd["spec"], "spec")["versions"], "spec.versions")
	for _, version := range versions {
		versionMap := asMap(t, version, "spec.versions[]")
		if versionMap["name"] != versionName {
			continue
		}

		schema := asMap(t, versionMap["schema"], "spec.versions[].schema")
		openAPIV3 := asMap(t, schema["openAPIV3Schema"], "spec.versions[].schema.openAPIV3Schema")
		properties := asMap(t, openAPIV3["properties"], "spec.versions[].schema.openAPIV3Schema.properties")
		specProperties := asMap(t, asMap(t, properties["spec"], "spec property")["properties"], "spec.properties")
		secretStoreRef := asMap(t, asMap(t, specProperties["secretStoreRef"], "spec.properties.secretStoreRef")["properties"], "spec.properties.secretStoreRef.properties")
		kindSchema := asMap(t, secretStoreRef["kind"], "spec.properties.secretStoreRef.properties.kind")
		return asStringSlice(t, kindSchema["enum"], "spec.properties.secretStoreRef.properties.kind.enum")
	}

	t.Fatalf("did not find %s secretStoreRef.kind enum", versionName)
	return nil
}

func asMap(t *testing.T, v any, path string) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("%s is %T, want map[string]any", path, v)
	}
	return m
}

func asSlice(t *testing.T, v any, path string) []any {
	t.Helper()
	s, ok := v.([]any)
	if !ok {
		t.Fatalf("%s is %T, want []any", path, v)
	}
	return s
}

func asStringSlice(t *testing.T, v any, path string) []string {
	t.Helper()
	s := asSlice(t, v, path)
	out := make([]string, 0, len(s))
	for i, entry := range s {
		str, ok := entry.(string)
		if !ok {
			t.Fatalf("%s[%d] is %T, want string", path, i, entry)
		}
		out = append(out, str)
	}
	return out
}
