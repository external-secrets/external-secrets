//Copyright External Secrets Inc. All Rights Reserved

package vault

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/vault/api/auth/approle"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errInvalidAppRoleID = "invalid Auth.AppRole: neither `roleId` nor `roleRef` was supplied"
)

func setAppRoleToken(ctx context.Context, v *client) (bool, error) {
	appRole := v.store.Auth.AppRole
	if appRole != nil {
		err := v.requestTokenWithAppRoleRef(ctx, appRole)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func (c *client) requestTokenWithAppRoleRef(ctx context.Context, appRole *esv1beta1.VaultAppRole) error {
	var err error
	var roleID string // becomes the RoleID used to authenticate with HashiCorp Vault

	// prefer .auth.appRole.roleId, fallback to .auth.appRole.roleRef, give up after that.
	if appRole.RoleID != "" { // use roleId from CRD, if configured
		roleID = strings.TrimSpace(appRole.RoleID)
	} else if appRole.RoleRef != nil { // use RoleID from Secret, if configured
		roleID, err = resolvers.SecretKeyRef(ctx, c.kube, c.storeKind, c.namespace, appRole.RoleRef)
		if err != nil {
			return err
		}
	} else { // we ran out of ways to get RoleID. return an appropriate error
		return errors.New(errInvalidAppRoleID)
	}

	secretID, err := resolvers.SecretKeyRef(ctx, c.kube, c.storeKind, c.namespace, &appRole.SecretRef)
	if err != nil {
		return err
	}
	secret := approle.SecretID{FromString: secretID}
	appRoleClient, err := approle.NewAppRoleAuth(roleID, &secret, approle.WithMountPath(appRole.Path))
	if err != nil {
		return err
	}
	_, err = c.auth.Login(ctx, appRoleClient)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return err
	}
	return nil
}
