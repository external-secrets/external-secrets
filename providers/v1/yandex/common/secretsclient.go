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

package ydxcommon

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errNotImplemented = "not implemented"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &yandexCloudSecretsClient{}

// Implementation of v1beta1.SecretsClient.
type yandexCloudSecretsClient struct {
	secretGetter    SecretGetter
	secretSetter    SecretSetter
	iamToken        string
	resourceKeyType ResourceKeyType
	folderID        string
}

func (c *yandexCloudSecretsClient) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return c.secretGetter.GetSecret(ctx, c.iamToken, ref.Key, c.resourceKeyType, c.folderID, ref.Version, ref.Property)
}

func (c *yandexCloudSecretsClient) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

func (c *yandexCloudSecretsClient) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

func (c *yandexCloudSecretsClient) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errors.New(errNotImplemented)
}

func (c *yandexCloudSecretsClient) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

func (c *yandexCloudSecretsClient) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return c.secretGetter.GetSecretMap(ctx, c.iamToken, ref.Key, c.resourceKeyType, c.folderID, ref.Version)
}

func (c *yandexCloudSecretsClient) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, errors.New(errNotImplemented)
}

func (c *yandexCloudSecretsClient) Close(_ context.Context) error {
	return nil
}
