package scaleway

import (
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type secretApi interface {
	GetSecret(req *smapi.GetSecretRequest, opts ...scw.RequestOption) (*smapi.Secret, error)
	GetSecretByName(req *smapi.GetSecretByNameRequest, opts ...scw.RequestOption) (*smapi.Secret, error)
	GetSecretVersion(req *smapi.GetSecretVersionRequest, opts ...scw.RequestOption) (*smapi.SecretVersion, error)
	GetSecretVersionByName(req *smapi.GetSecretVersionByNameRequest, opts ...scw.RequestOption) (*smapi.SecretVersion, error)
	AccessSecretVersion(request *smapi.AccessSecretVersionRequest, option ...scw.RequestOption) (*smapi.AccessSecretVersionResponse, error)
	ListSecrets(request *smapi.ListSecretsRequest, option ...scw.RequestOption) (*smapi.ListSecretsResponse, error)
	CreateSecret(request *smapi.CreateSecretRequest, option ...scw.RequestOption) (*smapi.Secret, error)
	CreateSecretVersion(request *smapi.CreateSecretVersionRequest, option ...scw.RequestOption) (*smapi.SecretVersion, error)
	DeleteSecret(request *smapi.DeleteSecretRequest, option ...scw.RequestOption) error
}
