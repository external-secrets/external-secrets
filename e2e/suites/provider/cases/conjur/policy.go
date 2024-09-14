// Copyright External Secrets Inc. All Rights Reserved
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
