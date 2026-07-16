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
package conjur

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/external-secrets/external-secrets-e2e/framework/addon"
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
  resource: !variable {{ .Key }}

- !permit
  role: !host vm-01
  privilege: [ read, execute ]
  resource: !variable {{ .Key }}

- !permit
  role: !host {{ .SpiffeHostID }}
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

// certHostIDPolicyTemplate creates the host authenticated by CreateCertHostIDStore, which
// supplies an explicit HostID in the authentication request (request mode). Request mode
// requires at least one certificate-matching restriction annotation; 'cn' is used here to
// match the client certificate's Common Name.
const certHostIDPolicyTemplate = `- !host
  id: vm-01
  annotations:
    authn-cert/{{ .ServiceID }}/cn: "vm-01"

- !permit
  role: !host vm-01
  privilege: [ read, authenticate ]
  resource: !webservice conjur/authn-cert/{{ .ServiceID }}`

// certSpiffePolicyTemplate creates the host authenticated by CreateCertStore, which omits
// HostID so the authn-cert authenticator derives the role from the SPIFFE URI SAN in the
// client certificate (spiffe mode). The host must exist at IdentityPath/WorkloadID to match
// the identity the authenticator derives from the certificate; no restriction annotation is
// required or checked in spiffe mode.
const certSpiffePolicyTemplate = `- !policy
  id: {{ .IdentityPath }}
  body:
    - !host {{ .WorkloadID }}

- !permit
  role: !host {{ .IdentityPath }}/{{ .WorkloadID }}
  privilege: [ read, authenticate ]
  resource: !webservice conjur/authn-cert/{{ .ServiceID }}`

func createVariablePolicy(key, namespace string, tags map[string]string) string {
	return renderTemplate(createVariablePolicyTemplate, map[string]interface{}{
		"Key":          key,
		"Namespace":    namespace,
		"Tags":         tags,
		"SpiffeHostID": fmt.Sprintf("%s/%s", addon.SpiffeIdentityPath, addon.SpiffeWorkloadID),
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

func createCertHostIDPolicy(serviceID string) string {
	return renderTemplate(certHostIDPolicyTemplate, map[string]interface{}{
		"ServiceID": serviceID,
	})
}

func createCertSpiffePolicy(serviceID, identityPath, workloadID string) string {
	return renderTemplate(certSpiffePolicyTemplate, map[string]interface{}{
		"ServiceID":    serviceID,
		"IdentityPath": identityPath,
		"WorkloadID":   workloadID,
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
