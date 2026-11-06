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
	"context"
	"fmt"

	"github.com/cyberark/conjur-api-go/conjurapi"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// conjurClientFromAzure creates a Conjur client using the authn-azure authenticator.
// If ServiceAccountRef is set, a Kubernetes ServiceAccount token is requested via the
// TokenRequest API and used as the Azure JWT. Otherwise config.JWTContent is left empty
// and conjur-api-go fetches a token from the Azure IMDS endpoint automatically.
func (c *Client) conjurClientFromAzure(ctx context.Context, config conjurapi.Config, prov *esv1.ConjurProvider) (SecretsClient, error) {
	config.AuthnType = "azure"
	config.Account = prov.Auth.Azure.Account
	config.ServiceID = prov.Auth.Azure.ServiceID
	config.JWTHostID = prov.Auth.Azure.HostID
	config.AzureClientID = prov.Auth.Azure.ClientID

	if prov.Auth.Azure.ServiceAccountRef != nil {
		token, err := c.getJwtFromServiceAccountTokenRequest(ctx, *prov.Auth.Azure.ServiceAccountRef, nil, JwtLifespan)
		if err != nil {
			return nil, fmt.Errorf("could not get Azure JWT from ServiceAccount: %w", err)
		}
		config.JWTContent = token
	}

	conjurClient, err := c.clientAPI.NewClientFromAzure(config)
	if err != nil {
		return nil, fmt.Errorf(errConjurClient, err)
	}

	c.client = conjurClient
	return conjurClient, nil
}
