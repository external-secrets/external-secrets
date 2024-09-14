//Copyright External Secrets Inc. All Rights Reserved

package conjur

import (
	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"
)

// SecretsClient is an interface for the Conjur client.
type SecretsClient interface {
	RetrieveSecret(secret string) (result []byte, err error)
	RetrieveBatchSecrets(variableIDs []string) (map[string][]byte, error)
	Resources(filter *conjurapi.ResourceFilter) (resources []map[string]interface{}, err error)
}

// SecretsClientFactory is an interface for creating a Conjur client.
type SecretsClientFactory interface {
	NewClientFromKey(config conjurapi.Config, loginPair authn.LoginPair) (SecretsClient, error)
	NewClientFromJWT(config conjurapi.Config) (SecretsClient, error)
}

// ClientAPIImpl is an implementation of the ClientAPI interface.
type ClientAPIImpl struct{}

func (c *ClientAPIImpl) NewClientFromKey(config conjurapi.Config, loginPair authn.LoginPair) (SecretsClient, error) {
	return conjurapi.NewClientFromKey(config, loginPair)
}

// NewClientFromJWT creates a new Conjur client from a JWT token.
func (c *ClientAPIImpl) NewClientFromJWT(config conjurapi.Config) (SecretsClient, error) {
	return conjurapi.NewClientFromJwt(config)
}
