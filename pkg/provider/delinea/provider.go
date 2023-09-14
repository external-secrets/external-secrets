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

package delinea

import (
	"context"
	"errors"
	"fmt"

	"github.com/DelineaXPM/dsv-sdk-go/v2/vault"
	corev1 "k8s.io/api/core/v1"
	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

var (
	errEmptyTenant                   = errors.New("tenant must not be empty")
	errEmptyClientID                 = errors.New("clientID must be set")
	errEmptyClientSecret             = errors.New("clientSecret must be set")
	errSecretRefAndValueConflict     = errors.New("cannot specify both secret reference and value")
	errSecretRefAndValueMissing      = errors.New("must specify either secret reference or direct value")
	errMissingStore                  = errors.New("missing store specification")
	errInvalidSpec                   = errors.New("invalid specification for delinea provider")
	errMissingSecretName             = errors.New("must specify a secret name")
	errMissingSecretKey              = errors.New("must specify a secret key")
	errClusterStoreRequiresNamespace = errors.New("when using a ClusterSecretStore, namespaces must be explicitly set")

	errNoSuchKeyFmt = "no such key in secret: %q"
)

type Provider struct{}

var _ esv1beta1.Provider = &Provider{}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kubeClient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	cfg, err := getConfig(store)
	if err != nil {
		return nil, err
	}

	if store.GetKind() == esv1beta1.ClusterSecretStoreKind && doesConfigDependOnNamespace(cfg) {
		// we are not attached to a specific namespace, but some config values are dependent on it
		return nil, errClusterStoreRequiresNamespace
	}

	clientID, err := loadConfigSecret(ctx, cfg.ClientID, kube, namespace)
	if err != nil {
		return nil, err
	}

	clientSecret, err := loadConfigSecret(ctx, cfg.ClientSecret, kube, namespace)
	if err != nil {
		return nil, err
	}

	dsvClient, err := vault.New(vault.Configuration{
		Credentials: vault.ClientCredential{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
		Tenant:      cfg.Tenant,
		TLD:         cfg.TLD,
		URLTemplate: cfg.URLTemplate,
	})
	if err != nil {
		return nil, err
	}

	return &client{
		api: dsvClient,
	}, nil
}

func loadConfigSecret(ctx context.Context, ref *esv1beta1.DelineaProviderSecretRef, kube kubeClient.Client, defaultNamespace string) (string, error) {
	if ref.SecretRef == nil {
		return ref.Value, nil
	}

	if err := validateSecretRef(ref); err != nil {
		return "", err
	}

	namespace := defaultNamespace
	if ref.SecretRef.Namespace != nil {
		namespace = *ref.SecretRef.Namespace
	}

	objKey := kubeClient.ObjectKey{Namespace: namespace, Name: ref.SecretRef.Name}
	secret := corev1.Secret{}
	err := kube.Get(ctx, objKey, &secret)
	if err != nil {
		return "", err
	}

	value, ok := secret.Data[ref.SecretRef.Key]
	if !ok {
		return "", fmt.Errorf(errNoSuchKeyFmt, ref.SecretRef.Key)
	}

	return string(value), nil
}

func validateStoreSecretRef(store esv1beta1.GenericStore, ref *esv1beta1.DelineaProviderSecretRef) error {
	if ref.SecretRef != nil {
		if err := utils.ValidateReferentSecretSelector(store, *ref.SecretRef); err != nil {
			return err
		}
	}

	return validateSecretRef(ref)
}

func validateSecretRef(ref *esv1beta1.DelineaProviderSecretRef) error {
	if ref.SecretRef != nil {
		if ref.Value != "" {
			return errSecretRefAndValueConflict
		}
		if ref.SecretRef.Name == "" {
			return errMissingSecretName
		}
		if ref.SecretRef.Key == "" {
			return errMissingSecretKey
		}
	} else if ref.Value == "" {
		return errSecretRefAndValueMissing
	}

	return nil
}

func doesConfigDependOnNamespace(cfg *esv1beta1.DelineaProvider) bool {
	if cfg.ClientID.SecretRef != nil && cfg.ClientID.SecretRef.Namespace == nil {
		return true
	}

	if cfg.ClientSecret.SecretRef != nil && cfg.ClientSecret.SecretRef.Namespace == nil {
		return true
	}

	return false
}

func getConfig(store esv1beta1.GenericStore) (*esv1beta1.DelineaProvider, error) {
	if store == nil {
		return nil, errMissingStore
	}
	storeSpec := store.GetSpec()

	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Delinea == nil {
		return nil, errInvalidSpec
	}
	cfg := storeSpec.Provider.Delinea

	if cfg.Tenant == "" {
		return nil, errEmptyTenant
	}

	if cfg.ClientID == nil {
		return nil, errEmptyClientID
	}

	if cfg.ClientSecret == nil {
		return nil, errEmptyClientSecret
	}

	err := validateStoreSecretRef(store, cfg.ClientID)
	if err != nil {
		return nil, err
	}

	err = validateStoreSecretRef(store, cfg.ClientSecret)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	_, err := getConfig(store)
	return err
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Delinea: &esv1beta1.DelineaProvider{},
	})
}
