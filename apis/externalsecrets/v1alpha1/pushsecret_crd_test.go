package v1alpha1

import (
	"os"
	"path/filepath"
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
		if versionMap["name"] != "v1alpha1" {
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
