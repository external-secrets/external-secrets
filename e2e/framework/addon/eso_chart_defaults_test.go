package addon

import (
	"os"
	"path/filepath"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestExternalSecretsChartDefaultsEnableV2StoreAPIs(t *testing.T) {
	path := filepath.Join("..", "..", "..", "deploy", "charts", "external-secrets", "values.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read values.yaml: %v", err)
	}

	var values map[string]any
	if err := yaml.Unmarshal(data, &values); err != nil {
		t.Fatalf("unmarshal values.yaml: %v", err)
	}

	crds := requireMap(t, values, "crds")
	v2 := requireMap(t, values, "v2")

	requireBool(t, crds, "createClusterProviderClass", true)
	requireBool(t, crds, "createProviderStore", true)
	requireBool(t, crds, "createClusterProviderStore", true)
	requireBool(t, v2, "enabled", true)
}

func requireMap(t *testing.T, values map[string]any, key string) map[string]any {
	t.Helper()

	raw, ok := values[key]
	if !ok {
		t.Fatalf("missing %q", key)
	}

	out, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("%q is %T, want map[string]any", key, raw)
	}

	return out
}

func requireBool(t *testing.T, values map[string]any, key string, want bool) {
	t.Helper()

	raw, ok := values[key]
	if !ok {
		t.Fatalf("missing %q", key)
	}

	got, ok := raw.(bool)
	if !ok {
		t.Fatalf("%q is %T, want bool", key, raw)
	}

	if got != want {
		t.Fatalf("%q = %t, want %t", key, got, want)
	}
}
