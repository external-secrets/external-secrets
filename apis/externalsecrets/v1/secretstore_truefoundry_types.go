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
// by a cluster service token. See docs/provider/truefoundry.md.
type TrueFoundryProvider struct {
	// BaseURL is the TrueFoundry control-plane URL, e.g.
	// https://your-cluster.tfy-usea1-ctl.example.com. The provider appends
	// /api/svc/v1/control-plane/secret to this when fetching a secret.
	// +kubebuilder:validation:MinLength=1
	BaseURL string `json:"baseURL"`

	// Auth configures how the Operator authenticates with the TrueFoundry
	// control-plane API. Only cluster-token Bearer authentication is
	// supported today.
	Auth TrueFoundryAuth `json:"auth"`
}

// TrueFoundryAuth configures authentication against the TrueFoundry
// control-plane API.
type TrueFoundryAuth struct {
	// SecretRef authenticates using a TrueFoundry cluster service token
	// stored in a Kubernetes Secret. The token is sent verbatim as the
	// `Authorization: Bearer <token>` header on every request.
	SecretRef TrueFoundryAuthSecretRef `json:"secretRef"`
}

// TrueFoundryAuthSecretRef contains the SecretKeySelector that points to the
// Kubernetes Secret holding the TrueFoundry cluster service token.
type TrueFoundryAuthSecretRef struct {
	// ClusterToken references a key inside a Kubernetes Secret that holds
	// the cluster service token (e.g. the `CLUSTER_TOKEN` key inside the
	// `tfy-agent-internal-*-token` Secret provisioned by the TFY agent).
	ClusterToken esmeta.SecretKeySelector `json:"clusterToken"`
}
