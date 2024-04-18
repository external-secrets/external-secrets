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

package passbolt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/passbolt/go-passbolt/api"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errPassboltStoreMissingProvider                = "missing: spec.provider.passbolt"
	errPassboltStoreMissingAuth                    = "missing: spec.provider.passbolt.auth"
	errPassboltStoreMissingAuthPassword            = "missing: spec.provider.passbolt.auth.passwordSecretRef"
	errPassboltStoreMissingAuthPrivateKey          = "missing: spec.provider.passbolt.auth.privateKeySecretRef"
	errPassboltStoreMissingHost                    = "missing: spec.provider.passbolt.host"
	errPassboltExternalSecretMissingFindNameRegExp = "missing: find.name.regexp"
	errPassboltStoreHostSchemeNotHTTPS             = "host Url has to be https scheme"
	errPassboltSecretPropertyInvalid               = "property must be one of name, username, uri, password or description"
	errNotImplemented                              = "not implemented"
)

type ProviderPassbolt struct {
	client Client
}

func (provider *ProviderPassbolt) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

type Client interface {
	CheckSession(ctx context.Context) bool
	Login(ctx context.Context) error
	Logout(ctx context.Context) error
	GetResource(ctx context.Context, resourceID string) (*api.Resource, error)
	GetResources(ctx context.Context, opts *api.GetResourcesOptions) ([]api.Resource, error)
	GetResourceType(ctx context.Context, typeID string) (*api.ResourceType, error)
	DecryptMessage(message string) (string, error)
	GetSecret(ctx context.Context, resourceID string) (*api.Secret, error)
}

func (provider *ProviderPassbolt) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	config := store.GetSpec().Provider.Passbolt

	password, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		config.Auth.PasswordSecretRef,
	)
	if err != nil {
		return nil, err
	}

	privateKey, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		config.Auth.PrivateKeySecretRef,
	)
	if err != nil {
		return nil, err
	}

	client, err := api.NewClient(nil, "", config.Host, privateKey, password)
	if err != nil {
		return nil, err
	}

	provider.client = client
	return provider, nil
}

func (provider *ProviderPassbolt) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf(errNotImplemented)
}

func (provider *ProviderPassbolt) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if err := assureLoggedIn(ctx, provider.client); err != nil {
		return nil, err
	}

	secret, err := provider.getPassboltSecret(ctx, ref.Key)
	if err != nil {
		return nil, err
	}

	if ref.Property == "" {
		return utils.JSONMarshal(secret)
	}

	return secret.GetProp(ref.Property)
}

func (provider *ProviderPassbolt) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return fmt.Errorf(errNotImplemented)
}

func (provider *ProviderPassbolt) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return fmt.Errorf(errNotImplemented)
}

func (provider *ProviderPassbolt) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultUnknown, nil
}

func (provider *ProviderPassbolt) GetSecretMap(_ context.Context, _ esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf(errNotImplemented)
}

func (provider *ProviderPassbolt) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	res := make(map[string][]byte)

	if ref.Name == nil || ref.Name.RegExp == "" {
		return res, errors.New(errPassboltExternalSecretMissingFindNameRegExp)
	}

	if err := assureLoggedIn(ctx, provider.client); err != nil {
		return nil, err
	}

	resources, err := provider.client.GetResources(ctx, &api.GetResourcesOptions{})
	if err != nil {
		return nil, err
	}

	nameRegexp, err := regexp.Compile(ref.Name.RegExp)
	if err != nil {
		return nil, err
	}

	for _, resource := range resources {
		if !nameRegexp.MatchString(resource.Name) {
			continue
		}

		secret, err := provider.getPassboltSecret(ctx, resource.ID)
		if err != nil {
			return nil, err
		}
		marshaled, err := utils.JSONMarshal(secret)
		if err != nil {
			return nil, err
		}
		res[resource.ID] = marshaled
	}

	return res, nil
}

func (provider *ProviderPassbolt) Close(ctx context.Context) error {
	return provider.client.Logout(ctx)
}

func (provider *ProviderPassbolt) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	config := store.GetSpec().Provider.Passbolt
	if config == nil {
		return nil, errors.New(errPassboltStoreMissingProvider)
	}

	if config.Auth == nil {
		return nil, errors.New(errPassboltStoreMissingAuth)
	}

	if config.Auth.PasswordSecretRef == nil || config.Auth.PasswordSecretRef.Name == "" || config.Auth.PasswordSecretRef.Key == "" {
		return nil, errors.New(errPassboltStoreMissingAuthPassword)
	}

	if config.Auth.PrivateKeySecretRef == nil || config.Auth.PrivateKeySecretRef.Name == "" || config.Auth.PrivateKeySecretRef.Key == "" {
		return nil, errors.New(errPassboltStoreMissingAuthPrivateKey)
	}
	if config.Host == "" {
		return nil, errors.New(errPassboltStoreMissingHost)
	}

	host, err := url.Parse(config.Host)
	if err != nil {
		return nil, err
	}

	if host.Scheme != "https" {
		return nil, errors.New(errPassboltStoreHostSchemeNotHTTPS)
	}

	return nil, nil
}

func init() {
	esv1beta1.Register(&ProviderPassbolt{}, &esv1beta1.SecretStoreProvider{
		Passbolt: &esv1beta1.PassboltProvider{},
	})
}

type Secret struct {
	Name        string `json:"name"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	URI         string `json:"uri"`
	Description string `json:"description"`
}

func (ps Secret) GetProp(key string) ([]byte, error) {
	switch key {
	case "name":
		return []byte(ps.Name), nil
	case "username":
		return []byte(ps.Username), nil
	case "uri":
		return []byte(ps.URI), nil
	case "password":
		return []byte(ps.Password), nil
	case "description":
		return []byte(ps.Description), nil
	default:
		return nil, errors.New(errPassboltSecretPropertyInvalid)
	}
}

func (provider *ProviderPassbolt) getPassboltSecret(ctx context.Context, id string) (*Secret, error) {
	resource, err := provider.client.GetResource(ctx, id)
	if err != nil {
		return nil, err
	}

	secret, err := provider.client.GetSecret(ctx, resource.ID)
	if err != nil {
		return nil, err
	}
	res := Secret{
		Name:        resource.Name,
		Username:    resource.Username,
		URI:         resource.URI,
		Description: resource.Description,
	}

	raw, err := provider.client.DecryptMessage(secret.Data)
	if err != nil {
		return nil, err
	}

	resourceType, err := provider.client.GetResourceType(ctx, resource.ResourceTypeID)
	if err != nil {
		return nil, err
	}

	switch resourceType.Slug {
	case "password-string":
		res.Password = raw
	case "password-and-description", "password-description-totp":
		var pwAndDesc api.SecretDataTypePasswordAndDescription
		if err := json.Unmarshal([]byte(raw), &pwAndDesc); err != nil {
			return nil, err
		}
		res.Password = pwAndDesc.Password
		res.Description = pwAndDesc.Description
	case "totp":
	default:
		return nil, fmt.Errorf("UnknownPassboltResourceType: %q", resourceType)
	}

	return &res, nil
}

func assureLoggedIn(ctx context.Context, client Client) error {
	if client.CheckSession(ctx) {
		return nil
	}

	return client.Login(ctx)
}
