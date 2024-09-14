//Copyright External Secrets Inc. All Rights Reserved

package vault

import (
	"context"
	"errors"
	"fmt"
	"strings"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errJwtNoTokenSource = "neither `secretRef` nor `kubernetesServiceAccountToken` was supplied as token source for jwt authentication"
)

func setJwtAuthToken(ctx context.Context, v *client) (bool, error) {
	jwtAuth := v.store.Auth.Jwt
	if jwtAuth != nil {
		err := v.requestTokenWithJwtAuth(ctx, jwtAuth)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func (c *client) requestTokenWithJwtAuth(ctx context.Context, jwtAuth *esv1beta1.VaultJwtAuth) error {
	role := strings.TrimSpace(jwtAuth.Role)
	var jwt string
	var err error
	if jwtAuth.SecretRef != nil {
		jwt, err = resolvers.SecretKeyRef(ctx, c.kube, c.storeKind, c.namespace, jwtAuth.SecretRef)
	} else if k8sServiceAccountToken := jwtAuth.KubernetesServiceAccountToken; k8sServiceAccountToken != nil {
		audiences := k8sServiceAccountToken.Audiences
		if audiences == nil {
			audiences = &[]string{"vault"}
		}
		expirationSeconds := k8sServiceAccountToken.ExpirationSeconds
		if expirationSeconds == nil {
			tmp := int64(600)
			expirationSeconds = &tmp
		}
		jwt, err = createServiceAccountToken(
			ctx,
			c.corev1,
			c.storeKind,
			c.namespace,
			k8sServiceAccountToken.ServiceAccountRef,
			*audiences,
			*expirationSeconds)
	} else {
		err = errors.New(errJwtNoTokenSource)
	}
	if err != nil {
		return err
	}

	parameters := map[string]any{
		"role": role,
		"jwt":  jwt,
	}
	url := strings.Join([]string{"auth", jwtAuth.Path, "login"}, "/")
	vaultResult, err := c.logical.WriteWithContext(ctx, url, parameters)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
	if err != nil {
		return err
	}

	token, err := vaultResult.TokenID()
	if err != nil {
		return fmt.Errorf(errVaultToken, err)
	}
	c.client.SetToken(token)
	return nil
}
