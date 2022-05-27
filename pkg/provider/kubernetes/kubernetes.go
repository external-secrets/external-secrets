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

package kubernetes

import (
	"context"
	"fmt"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &ProviderKubernetes{}
var _ esv1beta1.Provider = &ProviderKubernetes{}

type KClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Secret, error)
}

type RClient interface {
	Create(ctx context.Context, SelfSubjectAccessReview *authv1.SelfSubjectAccessReview, opts metav1.CreateOptions) (*authv1.SelfSubjectAccessReview, error)
}

// ProviderKubernetes is a provider for Kubernetes.
type ProviderKubernetes struct {
	Client       KClient
	ReviewClient RClient
	Namespace    string
}

var _ esv1beta1.SecretsClient = &ProviderKubernetes{}

type BaseClient struct {
	kube        kclient.Client
	store       *esv1beta1.KubernetesProvider
	namespace   string
	storeKind   string
	Certificate []byte
	Key         []byte
	CA          []byte
	BearerToken []byte
}

func init() {
	esv1beta1.Register(&ProviderKubernetes{}, &esv1beta1.SecretStoreProvider{
		Kubernetes: &esv1beta1.KubernetesProvider{},
	})
}

// NewClient constructs a Kubernetes Provider.
func (p *ProviderKubernetes) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Kubernetes == nil {
		return nil, fmt.Errorf("no store type or wrong store type")
	}
	storeSpecKubernetes := storeSpec.Provider.Kubernetes

	bStore := BaseClient{
		kube:      kube,
		store:     storeSpecKubernetes,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	// allow SecretStore controller validation to pass
	// when using referent namespace.
	if bStore.storeKind == esv1beta1.ClusterSecretStoreKind && bStore.namespace == "" {
		return p, nil
	}

	if err := bStore.setAuth(ctx); err != nil {
		return nil, err
	}

	config := &rest.Config{
		Host:        bStore.store.Server.URL,
		BearerToken: string(bStore.BearerToken),
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: false,
			CertData: bStore.Certificate,
			KeyData:  bStore.Key,
			CAData:   bStore.CA,
		},
	}

	kubeClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error configuring clientset: %w", err)
	}

	p.Client = kubeClientSet.CoreV1().Secrets(bStore.store.RemoteNamespace)
	p.Namespace = bStore.store.RemoteNamespace
	p.ReviewClient = kubeClientSet.AuthorizationV1().SelfSubjectAccessReviews()
	return p, nil
}

func (p *ProviderKubernetes) Close(ctx context.Context) error {
	return nil
}

func (p *ProviderKubernetes) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if ref.Property == "" {
		return nil, fmt.Errorf(errPropertyNotFound)
	}
	payload, err := p.GetSecretMap(ctx, ref)
	if err != nil {
		return nil, err
	}
	val, ok := payload[ref.Property]
	if !ok {
		return nil, fmt.Errorf("property %s does not exist in key %s", ref.Property, ref.Key)
	}
	return val, nil
}

func (p *ProviderKubernetes) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := p.Client.Get(ctx, ref.Key, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (p *ProviderKubernetes) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
