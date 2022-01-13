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

// AkeylessProvider Configures an store to sync secrets using Akeyless KV.
type AkeylessProvider struct {

	// Akeyless GW API Url from which the secrets to be fetched from.
	AkeylessGWApiURL *string `json:"akeylessGWApiURL"`

	// Auth configures how the operator authenticates with Akeyless.
	Auth *AkeylessAuth `json:"authSecretRef"`
}

type AkeylessAuth struct {
	SecretRef AkeylessAuthSecretRef `json:"secretRef"`
}

// AkeylessAuthSecretRef
//AKEYLESS_ACCESS_TYPE_PARAM: AZURE_OBJ_ID OR GCP_AUDIENCE OR ACCESS_KEY OR KUB_CONFIG_NAME.
type AkeylessAuthSecretRef struct {
	// The SecretAccessID is used for authentication
	AccessID        esmeta.SecretKeySelector `json:"accessID,omitempty"`
	AccessType      esmeta.SecretKeySelector `json:"accessType,omitempty"`
	AccessTypeParam esmeta.SecretKeySelector `json:"accessTypeParam,omitempty"`
}
