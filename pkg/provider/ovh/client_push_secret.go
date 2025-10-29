package ovh

import (
	"context"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	corev1 "k8s.io/api/core/v1"
)

func (c *client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	return nil
}

func (c *client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	return nil
}
