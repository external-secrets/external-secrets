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
	"fmt"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	notImplemented = "not implemented"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &Github{}
var _ esv1beta1.Provider = &Provider{}

type Provider struct{}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Github: &esv1beta1.GithubProvider{},
	})
}

func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	ghClient := &Github{
		http:      &http.Client{},
		kube:      kube,
		store:     store,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}
	key, appID, err := ghClient.GetPrivateKeyAppID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting private key: %w", err)
	}
	if ghClient.installTkn, err = GetInstallationToken(key, appID); err != nil {
		return nil, fmt.Errorf("can't get InstallationToken: %w", err)
	}
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}

	ghAPI := fmt.Sprintf("/app/installations/%s/access_tokens", provider.InstallID)

	ghClient.url = "https://api.github.com" + ghAPI
	if provider.URL != "" {
		ghClient.url = provider.URL + ghAPI
	}

	return ghClient, nil
}

func getProvider(store esv1beta1.GenericStore) (*esv1beta1.GithubProvider, error) {
	spc := store.GetSpec()
	if spc == nil || spc.Provider == nil || spc.Provider.Github == nil {
		return nil, fmt.Errorf("missing store provider github")
	}
	return spc.Provider.Github, nil
}

func (g *Github) getStoreSecret(ctx context.Context, ref esmeta.SecretKeySelector) (*corev1.Secret, error) {
	k := client.ObjectKey{
		Name:      ref.Name,
		Namespace: g.namespace,
	}
	if g.storeKind == esv1beta1.ClusterSecretStoreKind {
		if ref.Namespace == nil {
			return nil, fmt.Errorf("no namespace on ClusterSecretStore GH secret %s", ref.Name)
		}
		k.Namespace = *ref.Namespace
	}
	secret := &corev1.Secret{}
	if err := g.kube.Get(ctx, k, secret); err != nil {
		return nil, fmt.Errorf("failed to get clustersecretstore GH secret %s: %w", ref.Name, err)
	}
	return secret, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	pSpec := store.GetSpec().Provider.Github
	privatKey := pSpec.Auth.PrivatKey.SecretRef
	err := utils.ValidateSecretSelector(store, privatKey)
	if err != nil {
		return nil, err
	}

	if pSpec.AppID == "" || pSpec.InstallID == "" {
		return nil, fmt.Errorf("appID and instllIDs must not be empty")
	}

	if privatKey.Key == "" {
		return nil, fmt.Errorf("privatKey.key cannot be empty")
	}

	if privatKey.Name == "" {
		return nil, fmt.Errorf("privatKey.name cannot be empty")
	}

	return nil, nil
}

func (g *Github) Validate() (esv1beta1.ValidationResult, error) {
	timeout := 15 * time.Second
	url := g.url

	if err := utils.NetworkValidate(url, timeout); err != nil {
		return esv1beta1.ValidationResultError, err
	}
	return esv1beta1.ValidationResultReady, nil
}

func (g *Github) Close(_ context.Context) error {
	return nil
}

func (g *Github) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return fmt.Errorf(notImplemented)
}

// Not Implemented PushSecret.
func (g *Github) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return fmt.Errorf(notImplemented)
}

// Empty GetAllSecrets.
func (g *Github) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf(notImplemented)
}

func (g *Github) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	tkn, err := g.GetSecret(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("error getting secret %s: %w", ref.Key, err)
	}
	return map[string][]byte{"token": tkn}, nil
}
