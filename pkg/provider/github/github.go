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

package github

import (
	"context"
	crypto_rand "crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	github "github.com/google/go-github/v56/github"
	"golang.org/x/crypto/nacl/box"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errUnexpectedStoreSpec = "unexpected store spec"

	errInvalidStore      = "invalid store"
	errInvalidStoreSpec  = "invalid store spec"
	errInvalidStoreProv  = "invalid store provider"
	errInvalidGithubProv = "invalid github provider"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &Github{}
var _ esv1beta1.Provider = &Github{}

type ActionsServiceClient interface {
	CreateOrUpdateOrgSecret(ctx context.Context, org string, eSecret *github.EncryptedSecret) (response *github.Response, err error)
	GetOrgSecret(ctx context.Context, org string, name string) (*github.Secret, *github.Response, error)
	ListOrgSecrets(ctx context.Context, org string, opts *github.ListOptions) (*github.Secrets, *github.Response, error)
}

type Github struct {
	crClient   client.Client
	kubeClient kcorev1.CoreV1Interface
	store      esv1beta1.GenericStore
	provider   *esv1beta1.GithubProvider
	baseClient github.ActionsService
	namespace  string
	storeKind  string
}

func init() {
	esv1beta1.Register(&Github{}, &esv1beta1.SecretStoreProvider{
		Github: &esv1beta1.GithubProvider{},
	})
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (g *Github) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreWriteOnly
}

// NewClient constructs a new secrets client based on the provided store.
func (g *Github) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace)
}

func newClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	g := &Github{
		crClient:   kube,
		kubeClient: kubeClient.CoreV1(),
		store:      store,
		namespace:  namespace,
		provider:   provider,
		storeKind:  store.GetObjectKind().GroupVersionKind().Kind,
	}

	privateKeySecret := &corev1.Secret{}
	privateKeySecretName := g.provider.Auth.PrivateKey.Name
	if privateKeySecretName == "" {
		return nil, fmt.Errorf("errGithubCredSecretName")
	}

	objectKey := types.NamespacedName{
		Name:      privateKeySecretName,
		Namespace: g.namespace,
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if g.storeKind == esv1beta1.ClusterSecretStoreKind {
		if g.provider.Auth.PrivateKey.Namespace == nil {
			return nil, fmt.Errorf("invalid clusterStore missing SAK namespace")
		}
		objectKey.Namespace = *g.provider.Auth.PrivateKey.Namespace
	}

	err = g.crClient.Get(ctx, objectKey, privateKeySecret)
	if err != nil {
		return nil, fmt.Errorf("couldn't find secret on cluster: %w", err)
	}

	privateKey := privateKeySecret.Data[g.provider.Auth.PrivateKey.Key]
	if len(privateKey) == 0 {
		return nil, fmt.Errorf("no credentials found in secret %s/%s", privateKeySecret.Namespace, privateKeySecret.Name)
	}

	itr, err := ghinstallation.New(http.DefaultTransport, g.provider.AppID, g.provider.InstallationID, privateKey)

	var client *github.Client
	if (g.provider.URL != "") && (g.provider.URL != "https://github.com/") {
		client, err = github.NewClient(&http.Client{Transport: itr}).WithEnterpriseURLs(g.provider.URL, g.provider.URL)
		if err != nil {
			return nil, fmt.Errorf("github.NewClient couldn't create a client with the enterprise URL '%s': %w", g.provider.URL, err)
		}
	} else {
		client = github.NewClient(&http.Client{Transport: itr})
	}

	g.baseClient = *client.Actions

	return g, err
}

func getProvider(store esv1beta1.GenericStore) (*esv1beta1.GithubProvider, error) {
	spc := store.GetSpec()
	if spc == nil || spc.Provider.Github == nil {
		return nil, errors.New(errUnexpectedStoreSpec)
	}

	return spc.Provider.Github, nil
}

func (g *Github) ValidateStore(store esv1beta1.GenericStore) error {
	if store == nil {
		return fmt.Errorf(errInvalidStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return fmt.Errorf(errInvalidStoreSpec)
	}
	if spc.Provider == nil {
		return fmt.Errorf(errInvalidStoreProv)
	}
	p := spc.Provider.Github
	if p == nil {
		return fmt.Errorf(errInvalidGithubProv)
	}

	return nil
}

func (g *Github) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushRemoteRef) error {
	return nil
}

// PushSecret stores secrets into a Github Secret.
func (g *Github) PushSecret(ctx context.Context, value []byte, secretType corev1.SecretType, _ *apiextensionsv1.JSON, remoteRef esv1beta1.PushRemoteRef) error {
	secret, _, err := g.baseClient.GetOrgSecret(ctx, g.provider.OrganisationName, remoteRef.GetRemoteKey())
	if err != nil {
		return fmt.Errorf("error fetching secret: %w", err)
	}

	// If the secret already exists, we need to update it.
	// First at all, we need the organization public key to encrypt the secret.
	publicKey, _, err := g.baseClient.GetOrgPublicKey(ctx, g.provider.OrganisationName)
	if err != nil {
		return fmt.Errorf("error fetching %s Github Organisation's public key: %w", g.provider.OrganisationName, err)
	}

	decodedPublicKey, err := base64.StdEncoding.DecodeString(publicKey.GetKey())
	if err != nil {
		return fmt.Errorf("base64.StdEncoding.DecodeString was unable to decode public key: %w", err)
	}

	var boxKey [32]byte
	copy(boxKey[:], decodedPublicKey)

	encryptedBytes, err := box.SealAnonymous([]byte{}, value, &boxKey, crypto_rand.Reader)
	if err != nil {
		return fmt.Errorf("box.SealAnonymous failed with error %w", err)
	}

	encryptedString := base64.StdEncoding.EncodeToString(encryptedBytes)
	keyID := publicKey.GetKeyID()
	encryptedSecret := &github.EncryptedSecret{
		Name:           secret.Name,
		KeyID:          keyID,
		EncryptedValue: encryptedString,
		Visibility:     secret.Visibility,
	}

	if _, err := g.baseClient.CreateOrUpdateOrgSecret(ctx, g.provider.OrganisationName, encryptedSecret); err != nil {
		return fmt.Errorf("Actions.CreateOrUpdateRepoSecret returned error: %w", err)
	}

	return nil
}

// Implements store.Client.GetAllSecrets Interface.
// Retrieves a map[string][]byte with the secret names as key and the secret itself as the calue.
func (g *Github) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	secretsMap := make(map[string][]byte)

	var err error
	var secrets *github.Secrets
	secrets, _, err = g.baseClient.ListOrgSecrets(ctx, g.provider.OrganisationName, &github.ListOptions{})

	if err != nil {
		return nil, fmt.Errorf("error fetching secrets: %w", err)
	}

	for _, secret := range secrets.Secrets {
		secretsMap[secret.Name] = []byte("")
	}

	return secretsMap, err
}

// Implements store.Client.GetSecret Interface.
// Retrieves a secret/Key/Certificate/Tag with the secret name defined in ref.Name
// The Object Type is defined as a prefix in the ref.Name , if no prefix is defined , we assume a secret is required.
func (g *Github) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, fmt.Errorf("not implemented. ESO Github provider is a push only provider, the Github API does not allow to retrieve secrets.")
}

// Implements store.Client.GetSecretMap Interface.
// New version of GetSecretMap.
func (g *Github) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *Github) Close(ctx context.Context) error {
	ctx.Done()
	return nil
}

func (g *Github) Validate() (esv1beta1.ValidationResult, error) {
	if g.store.GetKind() == esv1beta1.ClusterSecretStoreKind {
		return esv1beta1.ValidationResultUnknown, nil
	}
	return esv1beta1.ValidationResultReady, nil
}
