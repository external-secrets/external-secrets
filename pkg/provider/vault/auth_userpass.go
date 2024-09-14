//Copyright External Secrets Inc. All Rights Reserved

package vault

import (
	"context"
	"strings"

	authuserpass "github.com/hashicorp/vault/api/auth/userpass"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

func setUserPassAuthToken(ctx context.Context, v *client) (bool, error) {
	userPassAuth := v.store.Auth.UserPass
	if userPassAuth != nil {
		err := v.requestTokenWithUserPassAuth(ctx, userPassAuth)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func (c *client) requestTokenWithUserPassAuth(ctx context.Context, userPassAuth *esv1beta1.VaultUserPassAuth) error {
	username := strings.TrimSpace(userPassAuth.Username)
	password, err := resolvers.SecretKeyRef(ctx, c.kube, c.storeKind, c.namespace, &userPassAuth.SecretRef)
	if err != nil {
		return err
	}
	pass := authuserpass.Password{FromString: password}
	l, err := authuserpass.NewUserpassAuth(username, &pass, authuserpass.WithMountPath(userPassAuth.Path))
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
