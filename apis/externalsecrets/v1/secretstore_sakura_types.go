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

package v1

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

// Configures a store to sync secrets with a Sakura Cloud Secret Manager.
type SakuraProvider struct {
	// Zone is the zone where the target vault is located.
	// +kubebuilder:default=is1a
	Zone SakuraZone `json:"zone,omitempty"`

	// VaultResourceID is the resource ID of the target vault.
	// +required
	VaultResourceID string `json:"vaultResourceID,omitempty"`

	// Auth defines the information necessary to authenticate against Sakura Cloud.
	// +required
	Auth SakuraAuth `json:"auth,omitempty"`
}

// SakuraZone is a enum that defines the zone where the target vault is located.
// +kubebuilder:validation:Enum=is1a;is1b;tk1a;tk1b
type SakuraZone string

const (
	// SakuraZoneIs1a is the zone is1a.
	SakuraZoneIs1a SakuraZone = "is1a"
	// SakuraZoneIs1b is the zone is1b.
	SakuraZoneIs1b SakuraZone = "is1b"
	// SakuraZoneTk1a is the zone tk1a.
	SakuraZoneTk1a SakuraZone = "tk1a"
	// SakuraZoneTk1b is the zone tk1b.
	SakuraZoneTk1b SakuraZone = "tk1b"
)

// SakuraAuth defines the information necessary to authenticate against Sakura Cloud.
type SakuraAuth struct {
	// +required
	SecretRef SakuraSecretRef `json:"secretRef,omitempty"`
}

// SakuraSecretRef holds secret references for Sakura Cloud credentials
// both AccessToken and AccessTokenSecret must be defined in order to properly authenticate.
type SakuraSecretRef struct {
	// The AccessToken is used for authentication
	// +required
	AccessToken esmeta.SecretKeySelector `json:"accessToken,omitempty"`

	// The AccessTokenSecret is used for authentication
	// +required
	AccessTokenSecret esmeta.SecretKeySelector `json:"accessTokenSecret,omitempty"`
}
