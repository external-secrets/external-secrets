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
	"errors"
	"fmt"

	authv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &Client{}
var _ esv1beta1.Provider = &Provider{}

type KClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Secret, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.SecretList, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Create(ctx context.Context, secret *v1.Secret, opts metav1.CreateOptions) (*v1.Secret, error)
	Update(ctx context.Context, secret *v1.Secret, opts metav1.UpdateOptions) (*v1.Secret, error)
}

type RClient interface {
	Create(ctx context.Context, selfSubjectRulesReview *authv1.SelfSubjectRulesReview, opts metav1.CreateOptions) (*authv1.SelfSubjectRulesReview, error)
}

// Provider implements Secret Provider interface
// for Kubernetes.
type Provider struct{}

// Client implements Secret Client interface
// for Kubernetes.
type Client struct {
	// ctrlClient is a controller-runtime client
	// with RBAC scope of the controller (privileged!)
	ctrlClient kclient.Client
	// ctrlClientset is a client-go CoreV1() client
	// with RBAC scope of the controller (privileged!)
	ctrlClientset typedcorev1.CoreV1Interface
	// userSecretClient is a client-go CoreV1().Secrets() client
	// with user-defined scope.
	userSecretClient KClient
	// userReviewClient is a SelfSubjectAccessReview client with
	// user-defined scope.
	userReviewClient RClient

	// store is the Kubernetes Provider spec
	// which contains the configuration for this provider.
	store     *esv1beta1.KubernetesProvider
	storeKind string

	// namespace is the namespace of the
	// ExternalSecret referencing this provider.
	namespace string
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Kubernetes: &esv1beta1.KubernetesProvider{},
	})
}

func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}
func (p *Provider) Convert(_ esv1beta1.GenericStore) (kclient.Object, error) {
	return nil, nil
}

func (p *Provider) ApplyReferent(spec kclient.Object, _ esmeta.ReferentCallOrigin, _ string) (kclient.Object, error) {
	return spec, nil
}

func (p *Provider) NewClientFromObj(_ context.Context, _ kclient.Object, _ kclient.Client, _ string) (esv1beta1.SecretsClient, error) {
	return nil, fmt.Errorf("not implemented")
}

// NewClient constructs a Kubernetes Provider.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	return p.newClient(ctx, store, kube, clientset, namespace)
}

func (p *Provider) newClient(ctx context.Context, store esv1beta1.GenericStore, ctrlClient kclient.Client, ctrlClientset kubernetes.Interface, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Kubernetes == nil {
		return nil, errors.New("no store type or wrong store type")
	}
	storeSpecKubernetes := storeSpec.Provider.Kubernetes
	client := &Client{
		ctrlClientset: ctrlClientset.CoreV1(),
		ctrlClient:    ctrlClient,
		store:         storeSpecKubernetes,
		namespace:     namespace,
		storeKind:     store.GetObjectKind().GroupVersionKind().Kind,
	}

	// allow SecretStore controller validation to pass
	// when using referent namespace.
	if client.storeKind == esv1beta1.ClusterSecretStoreKind && client.namespace == "" && isReferentSpec(storeSpecKubernetes) {
		return client, nil
	}

	cfg, err := client.getAuth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare auth: %w", err)
	}

	userClientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("error configuring clientset: %w", err)
	}
	client.userSecretClient = userClientset.CoreV1().Secrets(client.store.RemoteNamespace)
	client.userReviewClient = userClientset.AuthorizationV1().SelfSubjectRulesReviews()
	return client, nil
}

func isReferentSpec(prov *esv1beta1.KubernetesProvider) bool {
	if prov.Auth.Cert != nil {
		if prov.Auth.Cert.ClientCert.Namespace == nil {
			return true
		}
		if prov.Auth.Cert.ClientKey.Namespace == nil {
			return true
		}
	}
	if prov.Auth.ServiceAccount != nil {
		if prov.Auth.ServiceAccount.Namespace == nil {
			return true
		}
	}
	if prov.Auth.Token != nil {
		if prov.Auth.Token.BearerToken.Namespace == nil {
			return true
		}
	}
	return false
}

func (p *Provider) Close(_ context.Context) error {
	return nil
}
