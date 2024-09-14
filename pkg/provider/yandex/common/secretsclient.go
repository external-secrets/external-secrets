//Copyright External Secrets Inc. All Rights Reserved

package common

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errNotImplemented = "not implemented"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &yandexCloudSecretsClient{}

// Implementation of v1beta1.SecretsClient.
type yandexCloudSecretsClient struct {
	secretGetter SecretGetter
	secretSetter SecretSetter
	iamToken     string
}

func (c *yandexCloudSecretsClient) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return c.secretGetter.GetSecret(ctx, c.iamToken, ref.Key, ref.Version, ref.Property)
}

func (c *yandexCloudSecretsClient) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

func (c *yandexCloudSecretsClient) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

func (c *yandexCloudSecretsClient) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return errors.New(errNotImplemented)
}

func (c *yandexCloudSecretsClient) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

func (c *yandexCloudSecretsClient) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return c.secretGetter.GetSecretMap(ctx, c.iamToken, ref.Key, ref.Version)
}

func (c *yandexCloudSecretsClient) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, errors.New(errNotImplemented)
}

func (c *yandexCloudSecretsClient) Close(_ context.Context) error {
	return nil
}
