//Copyright External Secrets Inc. All Rights Reserved

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
