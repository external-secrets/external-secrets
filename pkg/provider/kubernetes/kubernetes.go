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
	"encoding/json"
	"fmt"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &ProviderKubernetes{}
var _ esv1beta1.Provider = &ProviderKubernetes{}

type KClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Secret, error)
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.SecretList, error)
}

type RClient interface {
	Create(ctx context.Context, selfSubjectRulesReview *authv1.SelfSubjectRulesReview, opts metav1.CreateOptions) (*authv1.SelfSubjectRulesReview, error)
}

// ProviderKubernetes is a provider for Kubernetes.
type ProviderKubernetes struct {
	Client       KClient
	ReviewClient RClient
	Namespace    string
	store        *esv1beta1.KubernetesProvider
	storeKind    string
}

var _ esv1beta1.SecretsClient = &ProviderKubernetes{}

type BaseClient struct {
	kube          kclient.Client
	kubeClientset typedcorev1.CoreV1Interface
	store         *esv1beta1.KubernetesProvider
	storeKind     string
	namespace     string
	Certificate   []byte
	Key           []byte
	CA            []byte
	BearerToken   []byte
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
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	client := BaseClient{
		kubeClientset: clientset.CoreV1(),
		kube:          kube,
		store:         storeSpecKubernetes,
		namespace:     namespace,
		storeKind:     store.GetObjectKind().GroupVersionKind().Kind,
	}
	p.Namespace = client.store.RemoteNamespace
	p.store = storeSpecKubernetes
	p.storeKind = store.GetObjectKind().GroupVersionKind().Kind

	// allow SecretStore controller validation to pass
	// when using referent namespace.
	if client.storeKind == esv1beta1.ClusterSecretStoreKind && client.namespace == "" && isReferentSpec(storeSpecKubernetes) {
		return p, nil
	}

	if err := client.setAuth(ctx); err != nil {
		return nil, err
	}

	config := &rest.Config{
		Host:        client.store.Server.URL,
		BearerToken: string(client.BearerToken),
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: false,
			CertData: client.Certificate,
			KeyData:  client.Key,
			CAData:   client.CA,
		},
	}

	kubeClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error configuring clientset: %w", err)
	}
	p.Client = kubeClientSet.CoreV1().Secrets(client.store.RemoteNamespace)
	p.ReviewClient = kubeClientSet.AuthorizationV1().SelfSubjectRulesReviews()
	return p, nil
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

func (p *ProviderKubernetes) Close(ctx context.Context) error {
	return nil
}

func (p *ProviderKubernetes) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secretMap, err := p.GetSecretMap(ctx, ref)
	if err != nil {
		return nil, err
	}
	if ref.Property != "" {
		val, ok := secretMap[ref.Property]
		if !ok {
			return nil, fmt.Errorf("property %s does not exist in key %s", ref.Property, ref.Key)
		}
		return val, nil
	}
	strMap := make(map[string]string)
	for k, v := range secretMap {
		strMap[k] = string(v)
	}
	jsonStr, err := json.Marshal(strMap)
	if err != nil {
		return nil, fmt.Errorf("unabled to marshal json: %w", err)
	}
	return jsonStr, nil
}

func (p *ProviderKubernetes) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := p.Client.Get(ctx, ref.Key, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (p *ProviderKubernetes) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Tags != nil {
		return p.findByTags(ctx, ref)
	}
	if ref.Name != nil {
		return p.findByName(ctx, ref)
	}
	return nil, fmt.Errorf("unexpected find operator: %#v", ref)
}

func (p *ProviderKubernetes) findByTags(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// empty/nil tags = everything
	sel, err := labels.ValidatedSelectorFromSet(ref.Tags)
	if err != nil {
		return nil, fmt.Errorf("unable to validate selector tags: %w", err)
	}
	secrets, err := p.Client.List(ctx, metav1.ListOptions{LabelSelector: sel.String()})
	if err != nil {
		return nil, fmt.Errorf("unable to list secrets: %w", err)
	}
	data := make(map[string][]byte)
	for _, secret := range secrets.Items {
		jsonStr, err := json.Marshal(convertMap(secret.Data))
		if err != nil {
			return nil, err
		}
		data[secret.Name] = jsonStr
	}
	return utils.ConvertKeys(ref.ConversionStrategy, data)
}

func (p *ProviderKubernetes) findByName(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	secrets, err := p.Client.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to list secrets: %w", err)
	}
	matcher, err := find.New(*ref.Name)
	if err != nil {
		return nil, err
	}
	data := make(map[string][]byte)
	for _, secret := range secrets.Items {
		if !matcher.MatchName(secret.Name) {
			continue
		}
		jsonStr, err := json.Marshal(convertMap(secret.Data))
		if err != nil {
			return nil, err
		}
		data[secret.Name] = jsonStr
	}
	return utils.ConvertKeys(ref.ConversionStrategy, data)
}

func convertMap(in map[string][]byte) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		out[k] = string(v)
	}
	return out
}
