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

package template

import (
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/template/v2/sprig"
)

func TestTemplateLimits(t *testing.T) {
	for _, tc := range []struct {
		name, input, errorText string
	}{
		{"until", `{{ until 35000000 | join " " }}`, "safety limit"},
		{"negative until", `{{ until -35000000 }}`, "safety limit"},
		{"untilStep", `{{ untilStep 0 35000000 1 }}`, "safety limit"},
		{"seq", `{{ seq 35000000 }}`, "safety limit"},
		{"repeat", `{{ repeat 35000000 "x" }}`, "safety limit"},
		{"join", `{{ until 10000 | join (repeat 100000 "x") }}`, "safety limit"},
		{"random", `{{ randAlpha 35000000 }}`, "safety limit"},
		{"random bytes", `{{ randBytes 35000000 }}`, "safety limit"},
		{"indent", `{{ indent 35000000 "x" }}`, "safety limit"},
		{"replace", `{{ replace "x" "xxxxx" .value }}`, "safety limit"},
		{"range output", `{{ range until 10000 }}xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx{{ end }}`, "template output exceeds"},
		{"small functions", `{{ until 3 | join "," }} {{ seq 3 }} {{ repeat 2 "x" }}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			secret := &corev1.Secret{}
			err := Execute(
				map[string][]byte{"key": []byte(tc.input)},
				map[string][]byte{"value": []byte(strings.Repeat("x", 300000))},
				esapi.TemplateScopeValues, esapi.TemplateTargetData, secret, esapi.ExternalSecretDecodeNone,
			)
			if tc.errorText != "" {
				require.ErrorContains(t, err, tc.errorText)
				require.Empty(t, secret.Data)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []byte("0,1,2 1 2 3 xx"), secret.Data["key"])
		})
	}
}

func TestExecuteTemplateOutputBoundary(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse(`{{ .a }}{{ .b }}`))
	output, err := ExecuteTemplate(tmpl, map[string]string{"a": strings.Repeat("x", sprig.MaxGeneratedBytes-1), "b": "y"})
	require.NoError(t, err)
	require.Len(t, output, sprig.MaxGeneratedBytes)

	output, err = ExecuteTemplate(tmpl, map[string]string{"a": strings.Repeat("x", sprig.MaxGeneratedBytes), "b": "y"})
	require.ErrorContains(t, err, "template output exceeds")
	require.Nil(t, output)

	literal := template.Must(template.New("literal").Parse(strings.Repeat("x", sprig.MaxGeneratedBytes+1)))
	output, err = ExecuteTemplate(literal, nil)
	require.ErrorContains(t, err, "template output exceeds")
	require.Nil(t, output)
}
