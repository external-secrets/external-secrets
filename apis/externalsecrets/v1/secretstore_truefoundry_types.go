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

package v1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// TrueFoundryProvider configures a store to sync secrets from TrueFoundry's
// control-plane secrets endpoint. Each secret is fetched by its fully-qualified
// name (FQN, "<tenant>:<group>:<secret>") through a single HTTP GET protected
// by a cluster service token. The provider wraps the FQN in a tfy-secret://
// URI before sending it as the secret_ref query parameter — users supply only
// the bare FQN. See docs/provider/truefoundry.md.
type TrueFoundryProvider struct {
	// BaseURL is the TrueFoundry control-plane URL, e.g.
	// https://your-cluster.tfy-usea1-ctl.example.com. The provider appends
	// /v1/control-plane/secret to this when fetching a secret.
	// +kubebuilder:validation:MinLength=1
	BaseURL string `json:"baseURL"`

	// Auth configures how the operator authenticates with the TrueFoundry
	// control-plane API. Today only cluster-token Bearer authentication is
	// supported; the wrapping struct leaves room to add additional methods
	// (e.g. ServiceAccount/OIDC token exchange) without a breaking change.
	Auth TrueFoundryAuth `json:"auth"`
}

// TrueFoundryAuth configures authentication against the TrueFoundry
// control-plane API. Exactly one of the contained methods must be set.
type TrueFoundryAuth struct {
	// SecretRef authenticates using a TrueFoundry cluster service token
	// stored in a Kubernetes Secret.
	// +optional
	SecretRef *TrueFoundryAuthSecretRef `json:"secretRef,omitempty"`
}

// TrueFoundryAuthSecretRef holds the per-credential SecretKeySelectors used
// for SecretRef-based authentication.
type TrueFoundryAuthSecretRef struct {
	// ClusterToken references a key inside a Kubernetes Secret that contains
	// the TrueFoundry cluster service token (e.g. the `CLUSTER_TOKEN` key
	// inside the `tfy-agent-internal-*-token` Secret provisioned by the TFY
	// agent). The value is sent verbatim as the `Authorization: Bearer <token>`
	// header on every request.
	ClusterToken esmeta.SecretKeySelector `json:"clusterToken"`
}
