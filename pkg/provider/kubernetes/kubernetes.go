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
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
	"github.com/external-secrets/external-secrets/pkg/utils"
	corev1 "k8s.io/api/core/v1"
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
	errMissingCredentials                  = "missing Credentials: %v"
	errUninitalizedKubernetesProvider      = "provider kubernetes is not initialized"
	errJSONSecretUnmarshal                 = "unable to unmarshal secret: %w"
)

type KubernetesClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Secret, error)
}

// ProviderKubernetes is a provider for Kubernetes.
type ProviderKubernetes struct {
	projectID string
	Client    KubernetesClient
}

var _ provider.SecretsClient = &ProviderKubernetes{}

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

func init() {
	schema.Register(&ProviderKubernetes{}, &esv1alpha1.SecretStoreProvider{
		Kubernetes: &esv1alpha1.KubernetesProvider{},
	})
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

	if ref.Property == "" {
		return nil, fmt.Errorf("property field not found on extrenal secrets")
	}

	payload, err := k.GetSecretMap(ctx, ref)

	if err != nil {
		return nil, err
	}

	val, ok := payload[ref.Property]
	if !ok {
		return nil, fmt.Errorf("property %s does not exist in key %s", ref.Property, ref.Key)
	}
	return val, nil
}

func (k *ProviderKubernetes) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if utils.IsNil(k.Client) {
		return nil, fmt.Errorf(errUninitalizedKubernetesProvider)
	}
	opts := metav1.GetOptions{}
	secretOut, err := k.Client.Get(ctx, ref.Key, opts)

	if err != nil {
		return nil, err
	}

	var payload map[string][]byte
	if len(secretOut.Data) != 0 {
		payload = secretOut.Data
	}

	return payload, nil
}

func (k *BaseClient) setAuth(ctx context.Context) error {
	var err error
	k.Certificate, err = k.helper(ctx, k.store.Auth.SecretRef.Certificate, "cert")
	if err != nil {
		return err
	}
	k.Key, err = k.helper(ctx, k.store.Auth.SecretRef.Key, "key")
	if err != nil {
		return err
	}
	k.CA, err = k.helper(ctx, k.store.Auth.SecretRef.CA, "ca")
	if err != nil {
		return err
	}
	k.BearerToken, err = k.helper(ctx, k.store.Auth.SecretRef.BearerToken, "bearerToken")
	if err != nil {
		return err
	}
	return nil
}

func (k *BaseClient) helper(ctx context.Context, key esmeta.SecretKeySelector, component string) ([]byte, error) {
	keySecret := &corev1.Secret{}
	keySecretName := key.Name
	if keySecretName == "" {
		return nil, fmt.Errorf(errKubernetesCredSecretName)
	}
	objectKey := types.NamespacedName{
		Name:      keySecretName,
		Namespace: k.namespace,
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if k.storeKind == esv1alpha1.ClusterSecretStoreKind {
		if key.Namespace == nil {
			return nil, fmt.Errorf(errInvalidClusterStoreMissingNamespace)
		}
		objectKey.Namespace = *key.Namespace
	}

	err := k.kube.Get(ctx, objectKey, keySecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchCredentialsSecret, err)
	}

	check := keySecret.Data[key.Key]
	if (check == nil) || (len(check) == 0) {
		return nil, fmt.Errorf(errMissingCredentials, component)
	}
	return check, nil
}
