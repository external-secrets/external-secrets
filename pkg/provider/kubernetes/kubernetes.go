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

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	errKubernetesCredSecretName            = "kubernetes credentials are empty"
	errInvalidClusterStoreMissingNamespace = "invalid clusterStore missing Cert namespace"
	errFetchCredentialsSecret              = "could not fetch Credentials secret: %w"
	errMissingCredentials                  = "missing Credentials"
	errUninitalizedKubernetesProvider      = "provider kubernetes is not initialized"
)

type KubernetesClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Secret, error)
	// List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
	// Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
	// Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
	// Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error
	// DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error
	// Status() client.StatusWriter
	// Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
}

// ProviderKubernetes is a provider for Kubernetes.
type ProviderKubernetes struct {
	projectID string
	Client    KubernetesClient
}

type BaseClient struct {
	kube        kclient.Client
	store       *esv1alpha1.KubernetesProvider
	namespace   string
	storeKind   string
	Server      string
	User        string
	Certificate []byte
	Key         []byte
	CA          []byte
	BearerToken []byte
}

// NewClient constructs a Kubernetes Provider.
func (k *ProviderKubernetes) NewClient(ctx context.Context, store esv1alpha1.GenericStore, kube kclient.Client, namespace string) (provider.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Kubernetes == nil {
		return nil, fmt.Errorf("no store type or wrong store type")
	}
	storeSpecKubernetes := storeSpec.Provider.Kubernetes

	kStore := BaseClient{
		kube:      kube,
		store:     storeSpecKubernetes,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
		Server:    storeSpecKubernetes.Server,
		User:      storeSpecKubernetes.User,
		// Certificate: xsxx,
		// Key:,
		// CA:,
	}

	if err := kStore.setAuth(ctx); err != nil {
		return nil, err
	}

	config := &rest.Config{
		Host:        kStore.store.Server,
		BearerToken: string(kStore.BearerToken),
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: false,
			CertData: kStore.Certificate,
			KeyData:  kStore.Key,
			CAData:   kStore.CA,
		},
	}

	kubeClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error configuring clientset: %w", err)
	}

	k.Client = kubeClientSet.CoreV1().Secrets(kStore.store.RemoteNamespace)

	return k, nil

}

func (k *ProviderKubernetes) Close(ctx context.Context) error {
	return nil
}

func (k *ProviderKubernetes) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {

	return []byte{}, nil
}

func (k *ProviderKubernetes) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	result := make(map[string][]byte)
	return result, nil
}

func (k *BaseClient) setAuth(ctx context.Context) error {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := k.store.Auth.SecretRef.Certificate.Name
	if credentialsSecretName == "" {
		return fmt.Errorf(errKubernetesCredSecretName)
	}
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: k.namespace,
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if k.storeKind == esv1alpha1.ClusterSecretStoreKind {
		if k.store.Auth.SecretRef.Certificate.Namespace == nil {
			return fmt.Errorf(errInvalidClusterStoreMissingNamespace)
		}
		objectKey.Namespace = *k.store.Auth.SecretRef.Certificate.Namespace
	}

	err := k.kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return fmt.Errorf(errFetchCredentialsSecret, err)
	}

	k.Certificate = credentialsSecret.Data[k.store.Auth.SecretRef.Certificate.Key]
	if (k.Certificate == nil) || (len(k.Certificate) == 0) {
		return fmt.Errorf(errMissingCredentials)
	}

	k.Key = credentialsSecret.Data[k.store.Auth.SecretRef.Key.Key]
	if (k.Key == nil) || (len(k.Key) == 0) {
		return fmt.Errorf(errMissingCredentials)
	}

	k.CA = credentialsSecret.Data[k.store.Auth.SecretRef.CA.Key]
	if (k.CA == nil) || (len(k.CA) == 0) {
		return fmt.Errorf(errMissingCredentials)
	}

	k.BearerToken = credentialsSecret.Data[k.store.Auth.SecretRef.BearerToken.Key]
	if (k.BearerToken == nil) || (len(k.BearerToken) == 0) {
		return fmt.Errorf(errMissingCredentials)
	}

	return nil
}
