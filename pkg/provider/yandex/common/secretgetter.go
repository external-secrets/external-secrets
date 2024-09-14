//Copyright External Secrets Inc. All Rights Reserved

package common

import (
	"context"
)

// Adapts the secrets received from a remote Yandex.Cloud service for the format expected by v1beta1.SecretsClient.
type SecretGetter interface {
	GetSecret(ctx context.Context, iamToken, resourceID, versionID, property string) ([]byte, error)
	GetSecretMap(ctx context.Context, iamToken, resourceID, versionID string) (map[string][]byte, error)
}
