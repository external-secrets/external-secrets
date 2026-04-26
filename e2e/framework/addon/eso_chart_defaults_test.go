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
