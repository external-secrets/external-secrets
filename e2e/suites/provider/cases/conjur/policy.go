/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package conjur

import (
	"bytes"
	"text/template"
)

const createVariablePolicyTemplate = `- !variable
  id: {{ .Key }}
  {{ if .Tags }}
  annotations:
    {{- range $key, $value := .Tags }}
    {{ $key }}: "{{ $value }}"
    {{- end }}
  {{ end }}

- !permit
  role: !host system:serviceaccount:{{ .Namespace }}:test-app-sa
  privilege: [ read, execute ]
  resource: !variable {{ .Key }}

- !permit
  role: !host system:serviceaccount:{{ .Namespace }}:test-app-hostid-sa
  privilege: [ read, execute ]
  resource: !variable {{ .Key }}`

const deleteVariablePolicyTemplate = `- !delete
  record: !variable {{ .Key }}`

const jwtHostPolicyTemplate = `- !host
  id: {{ .HostID }}
  annotations:
    authn-jwt/{{ .ServiceID }}/sub: "{{ .HostID }}"

- !permit
  role: !host {{ .HostID }}
  privilege: [ read, authenticate ]
  resource: !webservice conjur/authn-jwt/{{ .ServiceID }}`

func createVariablePolicy(key, namespace string, tags map[string]string) string {
	return renderTemplate(createVariablePolicyTemplate, map[string]interface{}{
		"Key":       key,
		"Namespace": namespace,
		"Tags":      tags,
	})
}

func deleteVariablePolicy(key string) string {
	return renderTemplate(deleteVariablePolicyTemplate, map[string]interface{}{
		"Key": key,
	})
}

func createJwtHostPolicy(hostID, serviceID string) string {
	return renderTemplate(jwtHostPolicyTemplate, map[string]interface{}{
		"HostID":    hostID,
		"ServiceID": serviceID,
	})
}

func renderTemplate(templateText string, data map[string]interface{}) string {
	// Use golang templates to render the policy
	tmpl, err := template.New("policy").Parse(templateText)
	if err != nil {
		// The templates are hardcoded, so this should never happen
		panic(err)
	}
	output := new(bytes.Buffer)
	err = tmpl.Execute(output, data)
	if err != nil {
		// The templates are hardcoded, so this should never happen
		panic(err)
	}
	return output.String()
}
