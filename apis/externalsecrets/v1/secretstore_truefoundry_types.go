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

	// SecretRef points to a Kubernetes Secret entry holding the cluster
	// service token used to authenticate against the TrueFoundry control
	// plane. The value is sent verbatim as the
	// `Authorization: Bearer <token>` header on every request.
	SecretRef esmeta.SecretKeySelector `json:"secretRef"`
}
