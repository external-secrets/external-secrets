/*
Copyright © 2026 SSH Communications

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

package privx

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type fakeSecretsClient struct {
	getSecretFn     func(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error)
	pushSecretFn    func(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error
	deleteSecretFn  func(ctx context.Context, ref esv1.PushSecretRemoteRef) error
	secretExistsFn  func(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error)
	validateFn      func() (esv1.ValidationResult, error)
	getSecretMapFn  func(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error)
	getAllSecretsFn func(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error)
	closeFn         func(ctx context.Context) error
}

func (f *fakeSecretsClient) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, ref)
	}
	return nil, nil
}

func (f *fakeSecretsClient) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	if f.pushSecretFn != nil {
		return f.pushSecretFn(ctx, secret, data)
	}
	return nil
}

func (f *fakeSecretsClient) DeleteSecret(ctx context.Context, ref esv1.PushSecretRemoteRef) error {
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, ref)
	}
	return nil
}

func (f *fakeSecretsClient) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	if f.secretExistsFn != nil {
		return f.secretExistsFn(ctx, ref)
	}
	return false, nil
}

func (f *fakeSecretsClient) Validate() (esv1.ValidationResult, error) {
	if f.validateFn != nil {
		return f.validateFn()
	}
	return esv1.ValidationResultUnknown, nil
}

func (f *fakeSecretsClient) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if f.getSecretMapFn != nil {
		return f.getSecretMapFn(ctx, ref)
	}
	return nil, nil
}

func (f *fakeSecretsClient) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if f.getAllSecretsFn != nil {
		return f.getAllSecretsFn(ctx, ref)
	}
	return nil, nil
}

func (f *fakeSecretsClient) Close(ctx context.Context) error {
	if f.closeFn != nil {
		return f.closeFn(ctx)
	}
	return nil
}
