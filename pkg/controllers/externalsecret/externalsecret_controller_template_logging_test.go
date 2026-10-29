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

package externalsecret

import (
	"context"
	"testing"

	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestTemplateRenderDebugLogs(t *testing.T) {
	var lines []string
	r := &Reconciler{Log: funcr.NewJSON(func(line string) { lines = append(lines, line) }, funcr.Options{Verbosity: 1})}
	es := &esv1.ExternalSecret{
		Name: "test-es", Namespace: "default",
		Spec: esv1.ExternalSecretSpec{Target: esv1.ExternalSecretTarget{Template: &esv1.ExternalSecretTemplate{
			EngineVersion: esv1.TemplateEngineV2, Data: map[string]string{"key": "{{ .password }}"},
		}}},
	}

	require.NoError(t, r.ApplyTemplate(
		context.Background(), es, &v1.Secret{}, map[string][]byte{"password": []byte("supersecret")},
	))
	es.Spec.Target.Template.Data["key"] = "{{ .missing }}"
	obj := &unstructured.Unstructured{Object: map[string]any{"data": map[string]any{}}}
	_, err := r.renderTemplatedManifest(context.Background(), es, obj, nil)
	require.Error(t, err)

	require.Len(t, lines, 4)
	for i, line := range lines {
		if i%2 == 0 {
			require.Contains(t, line, "template render started")
		} else {
			require.Contains(t, line, "template render finished")
		}
		require.Contains(t, line, `"namespace":"default"`)
		require.Contains(t, line, `"name":"test-es"`)
		require.NotContains(t, line, "supersecret")
		require.NotContains(t, line, "{{ .password }}")
	}
}
