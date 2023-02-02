package scaleway

import (
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type secretApi interface {
	AccessSecretVersion(request *smapi.AccessSecretVersionRequest, option ...scw.RequestOption) (*smapi.AccessSecretVersionResponse, error)
	ListSecrets(request *smapi.ListSecretsRequest, option ...scw.RequestOption) (*smapi.ListSecretsResponse, error)
	CreateSecretVersion(request *smapi.CreateSecretVersionRequest, option ...scw.RequestOption) (*smapi.SecretVersion, error)
}
