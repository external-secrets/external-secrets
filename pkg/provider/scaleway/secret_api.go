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

package scaleway

import (
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type secretAPI interface {
	GetSecret(req *smapi.GetSecretRequest, opts ...scw.RequestOption) (*smapi.Secret, error)
	GetSecretVersion(req *smapi.GetSecretVersionRequest, opts ...scw.RequestOption) (*smapi.SecretVersion, error)
	AccessSecretVersion(request *smapi.AccessSecretVersionRequest, option ...scw.RequestOption) (*smapi.AccessSecretVersionResponse, error)
	DisableSecretVersion(request *smapi.DisableSecretVersionRequest, option ...scw.RequestOption) (*smapi.SecretVersion, error)
	ListSecrets(request *smapi.ListSecretsRequest, option ...scw.RequestOption) (*smapi.ListSecretsResponse, error)
	CreateSecret(request *smapi.CreateSecretRequest, option ...scw.RequestOption) (*smapi.Secret, error)
	CreateSecretVersion(request *smapi.CreateSecretVersionRequest, option ...scw.RequestOption) (*smapi.SecretVersion, error)
	DeleteSecret(request *smapi.DeleteSecretRequest, option ...scw.RequestOption) error
}
