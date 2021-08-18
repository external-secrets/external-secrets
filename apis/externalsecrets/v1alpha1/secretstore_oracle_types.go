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

package v1alpha1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Configures an store to sync secrets using a Oracle Cloud Secrets Manager
// backend.
type OracleProvider struct {
	// Auth configures how secret-manager authenticates with the Oracle secrets manager.
	Auth OracleAuth `json:"auth"`

	// projectID is an access token specific to the secret.
	ProjectID *string `json:"projectID,omitempty"`
}

type OracleAuth struct {
	SecretRef OracleSecretRef `json:"secretRef"`
}

type OracleSecretRef struct {
	// The Access Token is used for authentication
	KeyId esmeta.SecretKeySelector `json:"token,omitempty"`
}
