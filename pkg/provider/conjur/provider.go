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

// Package conjur provides a Conjur provider for External Secrets.
package conjur

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/conjur/util"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

type Provider struct {
	NewConjurProvider func(context context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string, corev1 typedcorev1.CoreV1Interface, clientApi SecretsClientFactory) (esv1beta1.SecretsClient, error)
}

// NewClient creates a new Conjur client.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	// controller-runtime/client does not support TokenRequest or other subresource APIs
	// so we need to construct our own client and use it to create a TokenRequest
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	return p.NewConjurProvider(ctx, store, kube, namespace, clientset.CoreV1(), &ClientAPIImpl{})
}

// ValidateStore validates the store.
func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	prov, err := util.GetConjurProvider(store)
	if err != nil {
		return nil, err
	}

	if prov.URL == "" {
		return nil, fmt.Errorf("conjur URL cannot be empty")
	}
	if prov.Auth.APIKey != nil {
		if prov.Auth.APIKey.Account == "" {
			return nil, fmt.Errorf("missing Auth.ApiKey.Account")
		}
		if prov.Auth.APIKey.UserRef == nil {
			return nil, fmt.Errorf("missing Auth.Apikey.UserRef")
		}
		if prov.Auth.APIKey.APIKeyRef == nil {
			return nil, fmt.Errorf("missing Auth.Apikey.ApiKeyRef")
		}
		if err := utils.ValidateReferentSecretSelector(store, *prov.Auth.APIKey.UserRef); err != nil {
			return nil, fmt.Errorf("invalid Auth.Apikey.UserRef: %w", err)
		}
		if err := utils.ValidateReferentSecretSelector(store, *prov.Auth.APIKey.APIKeyRef); err != nil {
			return nil, fmt.Errorf("invalid Auth.Apikey.ApiKeyRef: %w", err)
		}
	}

	if prov.Auth.Jwt != nil {
		if prov.Auth.Jwt.Account == "" {
			return nil, fmt.Errorf("missing Auth.Jwt.Account")
		}
		if prov.Auth.Jwt.ServiceID == "" {
			return nil, fmt.Errorf("missing Auth.Jwt.ServiceID")
		}
		if prov.Auth.Jwt.ServiceAccountRef == nil && prov.Auth.Jwt.SecretRef == nil {
			return nil, fmt.Errorf("must specify Auth.Jwt.SecretRef or Auth.Jwt.ServiceAccountRef")
		}
		if prov.Auth.Jwt.SecretRef != nil {
			if err := utils.ValidateReferentSecretSelector(store, *prov.Auth.Jwt.SecretRef); err != nil {
				return nil, fmt.Errorf("invalid Auth.Jwt.SecretRef: %w", err)
			}
		}
		if prov.Auth.Jwt.ServiceAccountRef != nil {
			if err := utils.ValidateReferentServiceAccountSelector(store, *prov.Auth.Jwt.ServiceAccountRef); err != nil {
				return nil, fmt.Errorf("invalid Auth.Jwt.ServiceAccountRef: %w", err)
			}
		}
	}

	// At least one auth must be configured
	if prov.Auth.APIKey == nil && prov.Auth.Jwt == nil {
		return nil, fmt.Errorf("missing Auth.* configuration")
	}

	return nil, nil
}

// Capabilities returns the provider Capabilities (Read, Write, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func newConjurProvider(_ context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string, corev1 typedcorev1.CoreV1Interface, clientAPI SecretsClientFactory) (esv1beta1.SecretsClient, error) {
	return &Client{
		StoreKind: store.GetObjectKind().GroupVersionKind().Kind,
		store:     store,
		kube:      kube,
		namespace: namespace,
		corev1:    corev1,
		clientAPI: clientAPI,
	}, nil
}

func init() {
	esv1beta1.Register(&Provider{
		NewConjurProvider: newConjurProvider,
	}, &esv1beta1.SecretStoreProvider{
		Conjur: &esv1beta1.ConjurProvider{},
	})
}
