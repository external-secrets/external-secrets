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
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

var errBadGCPToken = "could not get Auth.Gcp.SecretRef.JWT: %w"

// conjurClientFromGCP creates a Conjur client using the authn-gcp authenticator.
// If SecretRef is set, a GCP identity token is resolved from a Kubernetes Secret and
// used directly. Otherwise config.JWTContent is left empty and conjur-api-go fetches
// a token from the GCP Metadata Service automatically.
func (c *Client) conjurClientFromGCP(ctx context.Context, config conjurapi.Config, prov *esv1.ConjurProvider) (SecretsClient, error) {
	config.AuthnType = "gcp"
	config.Account = prov.Auth.Gcp.Account
	config.ServiceID = prov.Auth.Gcp.ServiceID
	config.JWTHostID = prov.Auth.Gcp.HostID

	if prov.Auth.Gcp.SecretRef != nil {
		token, err := resolvers.SecretKeyRef(ctx, c.kube, c.StoreKind, c.namespace, &prov.Auth.Gcp.SecretRef.JWT)
		if err != nil {
			return nil, fmt.Errorf(errBadGCPToken, err)
		}
		config.JWTContent = token
	}

	conjurClient, err := c.clientAPI.NewClientFromGCP(config)
	if err != nil {
		return nil, fmt.Errorf(errConjurClient, err)
	}

	c.client = conjurClient
	return conjurClient, nil
}
