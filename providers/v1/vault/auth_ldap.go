/*
Copyright © 2025 ESO Maintainer Team

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

package vault

import (
	"context"
	"strings"

	authldap "github.com/hashicorp/vault/api/auth/ldap"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

func setLdapAuthToken(ctx context.Context, v *client) (bool, error) {
	ldapAuth := v.store.Auth.Ldap
	if ldapAuth != nil {
		err := v.requestTokenWithLdapAuth(ctx, ldapAuth)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func (c *client) requestTokenWithLdapAuth(ctx context.Context, ldapAuth *esv1.VaultLdapAuth) error {
	username := strings.TrimSpace(ldapAuth.Username)
	password, err := resolvers.SecretKeyRef(ctx, c.kube, c.storeKind, c.namespace, &ldapAuth.SecretRef)
	if err != nil {
		return err
	}
	pass := authldap.Password{FromString: password}
	l, err := authldap.NewLDAPAuth(username, &pass, authldap.WithMountPath(ldapAuth.Path))
	if err != nil {
		return err
	}
	_, err = c.auth.Login(ctx, l)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return err
	}
	return nil
}
