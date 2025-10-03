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

// Package conjur provides a Conjur provider for External Secrets.
package conjur

import (
	"context"

	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// Provider implements the External Secrets provider interface for Conjur.
// It facilitates creation of Conjur clients and manages their lifecycle.
type Provider struct {
	NewConjurProvider func(context context.Context, store esv1.GenericStore, kube client.Client, namespace string, corev1 typedcorev1.CoreV1Interface, clientApi SecretsClientFactory) (esv1.SecretsClient, error)
}

// NewClient creates a new Conjur client using the provided store configuration.
// It sets up necessary Kubernetes clients and creates a new Conjur provider instance.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
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

// Capabilities returns the provider's supported capabilities.
// Conjur provider supports read-only access to secrets.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// newConjurProvider creates and returns a new Conjur client with the specified configuration.
func newConjurProvider(_ context.Context, store esv1.GenericStore, kube client.Client, namespace string, corev1 typedcorev1.CoreV1Interface, clientAPI SecretsClientFactory) (esv1.SecretsClient, error) {
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
	esv1.Register(&Provider{
		NewConjurProvider: newConjurProvider,
	}, &esv1.SecretStoreProvider{
		Conjur: &esv1.ConjurProvider{},
	}, esv1.MaintenanceStatusMaintained)
}
