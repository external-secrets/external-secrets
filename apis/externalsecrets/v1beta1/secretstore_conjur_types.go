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

import esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

type ConjurProvider struct {
	URL      string     `json:"url"`
	CABundle string     `json:"caBundle,omitempty"`
	Auth     ConjurAuth `json:"auth"`
}

type ConjurAuth struct {
	Apikey *ConjurApikey `json:"apikey"`
}

type ConjurApikey struct {
	Account   string                    `json:"account"`
	UserRef   *esmeta.SecretKeySelector `json:"userRef"`
	APIKeyRef *esmeta.SecretKeySelector `json:"apiKeyRef"`
}
