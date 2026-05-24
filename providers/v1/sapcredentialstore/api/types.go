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

// Package api contains the HTTP types and client interface for the SAP Credential Store provider.
package api

// Credential is the full credential payload returned by GET requests to
// /api/v1/namespaces/{namespace}/credentials/{type}/{name}.
type Credential struct {
	Name     string            `json:"name"`
	Username string            `json:"username,omitempty"` // password type only
	Value    string            `json:"value"`
	Key      string            `json:"key,omitempty"`      // certificate type: private key PEM
	Metadata map[string]string `json:"metadata,omitempty"`
}

// CredentialMeta is a list item returned by list endpoints.
type CredentialMeta struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// CredentialBody is the request payload for PUT (create/update) operations.
type CredentialBody struct {
	Value    string            `json:"value"`
	Username string            `json:"username,omitempty"`
	Key      string            `json:"key,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
