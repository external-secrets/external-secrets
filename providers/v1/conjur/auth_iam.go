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
	"github.com/cyberark/conjur-api-go/conjurapi/authn"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

var (
	errBadIAMAccessKeyID     = "could not get Auth.Iam.SecretRef.AccessKeyIDSecretRef: %w"
	errBadIAMSecretAccessKey = "could not get Auth.Iam.SecretRef.SecretAccessKeySecretRef: %w"
	errBadIAMSessionToken    = "could not get Auth.Iam.SecretRef.SessionTokenSecretRef: %w"
)

// conjurClientFromIAM creates a Conjur client using the authn-iam authenticator.
func (c *Client) conjurClientFromIAM(ctx context.Context, config conjurapi.Config, prov *esv1.ConjurProvider) (SecretsClient, error) {
	config.AuthnType = "iam"
	config.Account = prov.Auth.Iam.Account
	config.ServiceID = prov.Auth.Iam.ServiceID
	config.JWTHostID = prov.Auth.Iam.HostID

	var creds *authn.IAMCredentials
	if prov.Auth.Iam.SecretRef != nil {
		var err error
		creds, err = c.resolveIAMCredentials(ctx, prov)
		if err != nil {
			return nil, err
		}
	}

	conjurClient, err := c.clientAPI.NewClientFromIAM(config, creds)
	if err != nil {
		return nil, fmt.Errorf(errConjurClient, err)
	}

	c.client = conjurClient
	return conjurClient, nil
}

// resolveIAMCredentials reads AWS credentials from Kubernetes Secrets.
func (c *Client) resolveIAMCredentials(ctx context.Context, prov *esv1.ConjurProvider) (*authn.IAMCredentials, error) {
	ref := prov.Auth.Iam.SecretRef

	accessKeyID, err := resolvers.SecretKeyRef(ctx, c.kube, c.StoreKind, c.namespace, &ref.AccessKeyIDSecretRef)
	if err != nil {
		return nil, fmt.Errorf(errBadIAMAccessKeyID, err)
	}
	secretAccessKey, err := resolvers.SecretKeyRef(ctx, c.kube, c.StoreKind, c.namespace, &ref.SecretAccessKeySecretRef)
	if err != nil {
		return nil, fmt.Errorf(errBadIAMSecretAccessKey, err)
	}
	sessionToken := ""
	if ref.SessionTokenSecretRef != nil {
		sessionToken, err = resolvers.SecretKeyRef(ctx, c.kube, c.StoreKind, c.namespace, ref.SessionTokenSecretRef)
		if err != nil {
			return nil, fmt.Errorf(errBadIAMSessionToken, err)
		}
	}

	return &authn.IAMCredentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
	}, nil
}
