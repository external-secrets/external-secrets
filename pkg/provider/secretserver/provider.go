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

package secretserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

var (
	errEmptyUserName                 = errors.New("username must not be empty")
	errEmptyPassword                 = errors.New("password must be set")
	errEmptyServerURL                = errors.New("serverURL must be set")
	errSecretRefAndValueConflict     = errors.New("cannot specify both secret reference and value")
	errSecretRefAndValueMissing      = errors.New("must specify either secret reference or direct value")
	errMissingStore                  = errors.New("missing store specification")
	errInvalidSpec                   = errors.New("invalid specification for secret server provider")
	errClusterStoreRequiresNamespace = errors.New("when using a ClusterSecretStore, namespaces must be explicitly set")
	errMissingSecretName             = errors.New("must specify a secret name")

	errMissingSecretKey = errors.New("must specify a secret key")
)

type Provider struct{}

var _ esv1beta1.Provider = &Provider{}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}
func (p *Provider) ApplyReferent(spec kubeClient.Object, _ esmeta.ReferentCallOrigin, _ string) (kubeClient.Object, error) {
	return spec, nil
}

func (p *Provider) Convert(_ esv1beta1.GenericStore) (kubeClient.Object, error) {
	return nil, nil
}

func (p *Provider) NewClientFromObj(_ context.Context, _ kubeClient.Object, _ kubeClient.Client, _ string) (esv1beta1.SecretsClient, error) {
	return nil, fmt.Errorf("not implemented")
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
	username, err := loadConfigSecret(ctx, store.GetKind(), cfg.Username, kube, namespace)
	if err != nil {
		return nil, err
	}
	password, err := loadConfigSecret(ctx, store.GetKind(), cfg.Password, kube, namespace)
	if err != nil {
		return nil, err
	}

	secretServer, err := server.New(server.Configuration{
		Credentials: server.UserCredential{
			Username: username,
			Password: password,
		},
		ServerURL: cfg.ServerURL,
	})
	if err != nil {
		return nil, err
	}

	return &client{
		api: secretServer,
	}, nil
}

func loadConfigSecret(
	ctx context.Context,
	storeKind string,
	ref *esv1beta1.SecretServerProviderRef,
	kube kubeClient.Client,
	namespace string) (string, error) {
	if ref.SecretRef == nil {
		return ref.Value, nil
	}
	if err := validateSecretRef(ref); err != nil {
		return "", err
	}
	return resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, ref.SecretRef)
}

func validateStoreSecretRef(store esv1beta1.GenericStore, ref *esv1beta1.SecretServerProviderRef) error {
	if ref.SecretRef != nil {
		if err := utils.ValidateReferentSecretSelector(store, *ref.SecretRef); err != nil {
			return err
		}
	}
	return validateSecretRef(ref)
}

func validateSecretRef(ref *esv1beta1.SecretServerProviderRef) error {
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

func doesConfigDependOnNamespace(cfg *esv1beta1.SecretServerProvider) bool {
	if cfg.Username.SecretRef != nil && cfg.Username.SecretRef.Namespace == nil {
		return true
	}
	if cfg.Password.SecretRef != nil && cfg.Password.SecretRef.Namespace == nil {
		return true
	}
	return false
}

func getConfig(store esv1beta1.GenericStore) (*esv1beta1.SecretServerProvider, error) {
	if store == nil {
		return nil, errMissingStore
	}
	storeSpec := store.GetSpec()

	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.SecretServer == nil {
		return nil, errInvalidSpec
	}
	cfg := storeSpec.Provider.SecretServer

	if cfg.Username == nil {
		return nil, errEmptyUserName
	}
	if cfg.Password == nil {
		return nil, errEmptyPassword
	}
	if cfg.ServerURL == "" {
		return nil, errEmptyServerURL
	}

	err := validateStoreSecretRef(store, cfg.Username)
	if err != nil {
		return nil, err
	}
	err = validateStoreSecretRef(store, cfg.Password)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	_, err := getConfig(store)
	return nil, err
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		SecretServer: &esv1beta1.SecretServerProvider{},
	})
}
