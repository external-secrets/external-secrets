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

package v1beta1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Device42Provider configures a store to sync secrets with a Device42 instance.
type Device42Provider struct {
	// URL configures the Device42 instance URL.
	Host string `json:"host"`

	// Auth configures how secret-manager authenticates with a Device42 instance.
	Auth Device42Auth `json:"auth"`
}

type Device42Auth struct {
	SecretRef Device42SecretRef `json:"secretRef"`
}

type Device42SecretRef struct {
	// Username / Password is used for authentication.
	// +optional
	Credentials esmeta.SecretKeySelector `json:"credentials,omitempty"`
}
