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

package v1alpha1

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestPushSecretCRDDoesNotDefaultSecretStoreRefKind(t *testing.T) {
	crdPath := filepath.Join("..", "..", "..", "config", "crds", "bases", "external-secrets.io_pushsecrets.yaml")
	data, err := os.ReadFile(crdPath)
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}

	var crd map[string]any
	if err := yaml.Unmarshal(data, &crd); err != nil {
		t.Fatalf("unmarshal CRD: %v", err)
	}

	versions := asSlice(t, asMap(t, crd["spec"], "spec")["versions"], "spec.versions")
	var kindSchema map[string]any
	for _, version := range versions {
		versionMap := asMap(t, version, "spec.versions[]")
		if versionMap["name"] != Version {
			continue
		}

		schema := asMap(t, versionMap["schema"], "spec.versions[].schema")
		openAPIV3 := asMap(t, schema["openAPIV3Schema"], "spec.versions[].schema.openAPIV3Schema")
		properties := asMap(t, openAPIV3["properties"], "spec.versions[].schema.openAPIV3Schema.properties")
		specProperties := asMap(t, asMap(t, properties["spec"], "spec property")["properties"], "spec.properties")
		secretStoreRefs := asMap(t, specProperties["secretStoreRefs"], "spec.properties.secretStoreRefs")
		items := asMap(t, secretStoreRefs["items"], "spec.properties.secretStoreRefs.items")
		itemProperties := asMap(t, items["properties"], "spec.properties.secretStoreRefs.items.properties")
		kindSchema = asMap(t, itemProperties["kind"], "spec.properties.secretStoreRefs.items.properties.kind")
		break
	}

	if kindSchema == nil {
		t.Fatal("did not find v1alpha1 secretStoreRefs.kind schema")
	}
	if def, ok := kindSchema["default"]; ok {
		t.Fatalf("secretStoreRefs.kind must not define a CRD default, got %v", def)
	}
}

func TestPushSecretCRDSecretStoreRefKindIncludesProviderStoreKinds(t *testing.T) {
	crdPath := filepath.Join("..", "..", "..", "config", "crds", "bases", "external-secrets.io_pushsecrets.yaml")
	data, err := os.ReadFile(crdPath)
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}

	var crd map[string]any
	if err := yaml.Unmarshal(data, &crd); err != nil {
		t.Fatalf("unmarshal CRD: %v", err)
	}

	versions := asSlice(t, asMap(t, crd["spec"], "spec")["versions"], "spec.versions")
	var kindEnum []string
	for _, version := range versions {
		versionMap := asMap(t, version, "spec.versions[]")
		if versionMap["name"] != Version {
			continue
		}

		schema := asMap(t, versionMap["schema"], "spec.versions[].schema")
		openAPIV3 := asMap(t, schema["openAPIV3Schema"], "spec.versions[].schema.openAPIV3Schema")
		properties := asMap(t, openAPIV3["properties"], "spec.versions[].schema.openAPIV3Schema.properties")
		specProperties := asMap(t, asMap(t, properties["spec"], "spec property")["properties"], "spec.properties")
		secretStoreRefs := asMap(t, specProperties["secretStoreRefs"], "spec.properties.secretStoreRefs")
		items := asMap(t, secretStoreRefs["items"], "spec.properties.secretStoreRefs.items")
		itemProperties := asMap(t, items["properties"], "spec.properties.secretStoreRefs.items.properties")
		kindSchema := asMap(t, itemProperties["kind"], "spec.properties.secretStoreRefs.items.properties.kind")
		kindEnum = asStringSlice(t, kindSchema["enum"], "spec.properties.secretStoreRefs.items.properties.kind.enum")
		break
	}
	if kindEnum == nil {
		t.Fatal("did not find v1alpha1 secretStoreRefs.kind enum")
	}

	assertContains := func(value string) {
		t.Helper()
		if slices.Contains(kindEnum, value) {
			return
		}
		t.Fatalf("kind enum does not contain %q: %v", value, kindEnum)
	}

	assertContains("ProviderStore")
	assertContains("ClusterProviderStore")
	assertNotContains := func(value string) {
		t.Helper()
		if slices.Contains(kindEnum, value) {
			t.Fatalf("kind enum unexpectedly contains %q: %v", value, kindEnum)
		}
	}

	assertNotContains("Provider")
	assertNotContains("ClusterProvider")
}

func TestPushSecretCRDDoesNotDefaultSecretStoreRefAPIVersion(t *testing.T) {
	crdPath := filepath.Join("..", "..", "..", "config", "crds", "bases", "external-secrets.io_pushsecrets.yaml")
	data, err := os.ReadFile(crdPath)
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}

	var crd map[string]any
	if err := yaml.Unmarshal(data, &crd); err != nil {
		t.Fatalf("unmarshal CRD: %v", err)
	}

	versions := asSlice(t, asMap(t, crd["spec"], "spec")["versions"], "spec.versions")
	var apiVersionSchema map[string]any
	for _, version := range versions {
		versionMap := asMap(t, version, "spec.versions[]")
		if versionMap["name"] != Version {
			continue
		}

		schema := asMap(t, versionMap["schema"], "spec.versions[].schema")
		openAPIV3 := asMap(t, schema["openAPIV3Schema"], "spec.versions[].schema.openAPIV3Schema")
		properties := asMap(t, openAPIV3["properties"], "spec.versions[].schema.openAPIV3Schema.properties")
		specProperties := asMap(t, asMap(t, properties["spec"], "spec property")["properties"], "spec.properties")
		secretStoreRefs := asMap(t, specProperties["secretStoreRefs"], "spec.properties.secretStoreRefs")
		items := asMap(t, secretStoreRefs["items"], "spec.properties.secretStoreRefs.items")
		itemProperties := asMap(t, items["properties"], "spec.properties.secretStoreRefs.items.properties")
		apiVersionSchema = asMap(t, itemProperties["apiVersion"], "spec.properties.secretStoreRefs.items.properties.apiVersion")
		break
	}

	if apiVersionSchema == nil {
		t.Fatal("did not find v1alpha1 secretStoreRefs.apiVersion schema")
	}
	if def, ok := apiVersionSchema["default"]; ok {
		t.Fatalf("secretStoreRefs.apiVersion must not define a CRD default, got %v", def)
	}
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
