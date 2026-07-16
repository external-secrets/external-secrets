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

package conjur

import (
	"io"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"
)

// SecretsClient is an interface for the Conjur client.
type SecretsClient interface {
	AddSecret(variable, secret string) error
	GetStaticSecretDetails(identifier string) (*conjurapi.StaticSecretResponse, error)
	LoadPolicy(policyMode conjurapi.PolicyMode, policyID string, policy io.Reader) (*conjurapi.PolicyResponse, error)
	RetrieveSecret(secret string) (result []byte, err error)
	RetrieveBatchSecrets(variableIDs []string) (map[string][]byte, error)
	Resources(filter *conjurapi.ResourceFilter) (resources []map[string]any, err error)
}

// SecretsClientFactory is an interface for creating a Conjur client.
type SecretsClientFactory interface {
	NewClientFromKey(config conjurapi.Config, loginPair authn.LoginPair) (SecretsClient, error)
	NewClientFromJWT(config conjurapi.Config) (SecretsClient, error)
}

// ClientAPIImpl is an implementation of the ClientAPI interface.
type ClientAPIImpl struct{}

// CompositeClient is the composite of the Client and ClientV2 mechanisms so that API methods from both are accessible.
type CompositeClient struct {
	*conjurapi.Client
	*conjurapi.ClientV2
}

// NewClientFromKey creates a new Conjur client using API key authentication.
func (c *ClientAPIImpl) NewClientFromKey(config conjurapi.Config, loginPair authn.LoginPair) (SecretsClient, error) {
	client, err := conjurapi.NewClientFromKey(config, loginPair)
	if err != nil {
		return nil, err
	}
	return CompositeClient{
		client,
		&conjurapi.ClientV2{Client: client},
	}, nil
}

// NewClientFromJWT creates a new Conjur client from a JWT token.
func (c *ClientAPIImpl) NewClientFromJWT(config conjurapi.Config) (SecretsClient, error) {
	client, err := conjurapi.NewClientFromJwt(config)
	if err != nil {
		return nil, err
	}
	return CompositeClient{
		client,
		&conjurapi.ClientV2{Client: client},
	}, nil
}
