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

type GithubProvider struct {
	// URL configures the Github instance URL. Defaults to https://github.com/.
	URL       string `json:"url,omitempty"`
	AppID     string `json:"appID"`
	InstallID string `json:"installID"`
	// Auth configures how secret-manager authenticates with a Github instance.
	Auth GithubAuth `json:"auth"`
}

type GithubAuth struct {
	SecretRef GithubSecretRef `json:"SecretRef"`
}

type GithubSecretRef struct {
	PrivatKey esmeta.SecretKeySelector `json:"privatKey"`
}
