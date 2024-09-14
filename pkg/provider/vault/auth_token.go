//Copyright External Secrets Inc. All Rights Reserved

package vault

import (
	"context"

	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

func setSecretKeyToken(ctx context.Context, v *client) (bool, error) {
	tokenRef := v.store.Auth.TokenSecretRef
	if tokenRef != nil {
		token, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, tokenRef)
		if err != nil {
			return true, err
		}
		v.client.SetToken(token)
		return true, nil
	}
	return false, nil
}
