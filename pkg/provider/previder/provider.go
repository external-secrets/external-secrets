/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package previder

import (
	"context"
	"errors"
	"fmt"

	previderclient "github.com/previder/vault-cli/pkg"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errNotImplemented = "not implemented"
)

var _ esv1beta1.Provider = &SecretManager{}

type SecretManager struct {
	VaultClient previderclient.PreviderVaultClient
}

func init() {
	esv1beta1.Register(&SecretManager{}, &esv1beta1.SecretStoreProvider{
		Previder: &esv1beta1.PreviderProvider{},
	})
}

func (s *SecretManager) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	if store == nil {
		return nil, fmt.Errorf("secret store not found: %v", "nil store")
	}
	storeSpec := store.GetSpec().Provider.Previder

	storeKind := store.GetObjectKind().GroupVersionKind().Kind
	accessToken, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &storeSpec.Auth.SecretRef.AccessToken)
	if err != nil {
		return nil, fmt.Errorf(accessToken, err)
	}

	s.VaultClient, err = previderclient.NewVaultClient(storeSpec.BaseURI, accessToken)

	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SecretManager) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	previderSpec := storeSpec.Provider.Previder
	if previderSpec == nil {
		return nil, errors.New("missing Previder spec")
	}
	if previderSpec.Auth.SecretRef == nil {
		return nil, errors.New("missing Previder Auth SecretRef")
	}
	accessToken := previderSpec.Auth.SecretRef.AccessToken

	if accessToken.Name == "" {
		return nil, errors.New("missing Previder accessToken name")
	}
	if accessToken.Key == "" {
		return nil, errors.New("missing Previder accessToken key")
	}

	return nil, nil
}

func (s *SecretManager) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (s *SecretManager) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := s.VaultClient.DecryptSecret(ref.Key)
	if err != nil {
		return nil, err
	}
	return []byte(secret.Secret), nil
}

func (s *SecretManager) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
	return errors.New(errNotImplemented)
}

func (s *SecretManager) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

func (s *SecretManager) SecretExists(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

func (s *SecretManager) Validate() (esv1beta1.ValidationResult, error) {
	_, err := s.VaultClient.GetSecrets()
	if err != nil {
		return esv1beta1.ValidationResultError, err
	}

	return esv1beta1.ValidationResultReady, nil
}

func (s *SecretManager) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secrets, err := s.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	secretData := make(map[string][]byte)
	secretData[ref.Key] = secrets
	return secretData, nil
}

func (s *SecretManager) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

func (s *SecretManager) Close(ctx context.Context) error {
	return nil
}
