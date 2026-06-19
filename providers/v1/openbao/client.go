/*
Copyright © The ESO Authors

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

package openbao

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/openbao/openbao/api/auth/userpass/v2"
	"github.com/openbao/openbao/api/v2"
	v1 "k8s.io/api/core/v1"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/find"
)

var (
	_ esv1.SecretsClient = &client{}
)

const (
	errInvalidRevVersion      = "invalid Ref.Version: %w"
	errSecretKeyNotFound      = "cannot find secret data for key: %q"
	errFetchMount             = "error while validating %q: %w"
	errInvalidMountType       = `expected mount type "kv" found %q`
	errInvalidMountVersion    = "expected kv engine version %s found version %s"
	errKVv1VersionUnsupported = "OpenBao KVv1 secrets do not support versioning (use KVv2)"
	errCustomCA               = "cannot set OpenBao CA certificate: %w"
)

type client struct {
	client     *api.Client
	httpClient *http.Client
	store      *esv1.OpenBaoProvider
	storeKind  string
}

func (c *client) setup(ctx context.Context, kube k8sClient.Client, namespace string, httpClient httpClientFactory) error {
	c.httpClient = httpClient()

	config := api.DefaultConfig()
	config.HttpClient = c.httpClient
	config.Address = c.store.Server

	if len(c.store.CABundle) != 0 || c.store.CAProvider != nil {
		caCertPool := x509.NewCertPool()
		ca, err := esutils.FetchCACertFromSource(ctx, esutils.CreateCertOpts{
			CABundle:   c.store.CABundle,
			CAProvider: c.store.CAProvider,
			StoreKind:  c.storeKind,
			Namespace:  namespace,
			Client:     kube,
		})
		if err != nil {
			return fmt.Errorf(errCustomCA, err)
		}
		ok := caCertPool.AppendCertsFromPEM(ca)
		if !ok {
			return fmt.Errorf(errCustomCA, errors.New("failed add certificate to CertPool"))
		}

		if transport, ok := config.HttpClient.Transport.(*http.Transport); ok {
			transport = transport.Clone()
			if transport.TLSClientConfig == nil {
				transport.TLSClientConfig = &tls.Config{}
			}
			transport.TLSClientConfig.RootCAs = caCertPool
			config.HttpClient.Transport = transport
		}
	}

	client, err := api.NewClient(config)
	if err != nil {
		return err
	}
	c.client = client

	return c.setupAuth(ctx, kube, namespace)
}

func (c *client) setupAuth(ctx context.Context, kube k8sClient.Client, namespace string) error {
	if c.store.Auth == nil {
		return nil
	}

	switch {
	case c.store.Auth.TokenSecretRef != nil:
		token, err := resolvers.SecretKeyRef(ctx, kube, c.storeKind, namespace, c.store.Auth.TokenSecretRef)
		if err != nil {
			return err
		}

		c.client.SetToken(token)

	case c.store.Auth.UserPass != nil:
		userPass := c.store.Auth.UserPass
		password, err := resolvers.SecretKeyRef(ctx, kube, c.storeKind, namespace, &userPass.SecretRef)
		if err != nil {
			return err
		}

		auth, err := userpass.NewUserpassAuth(userPass.Username, &userpass.Password{
			FromString: password,
		}, userpass.WithMountPath(userPass.Path))
		if err != nil {
			return err
		}

		_, err = c.client.Auth().Login(ctx, auth)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *client) Close(_ context.Context) error {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
		c.httpClient = nil
	}
	c.client = nil
	c.store = nil
	return nil
}

func (c *client) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New("delete secret is not supported (the OpenBao provider is currently read only)")
}

func (c *client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Tags != nil {
		return nil, errors.New("tag based search is not implemented")
	}

	listPath := ""
	if ref.Path != nil {
		listPath = *ref.Path
	}

	var list func(ctx context.Context, secretPath string) (*api.KVList, error)
	if c.useV1() {
		list = c.client.KVv1(c.path()).List
	} else {
		list = c.client.KVv2(c.path()).List
	}

	meta, err := list(ctx, listPath)
	if err != nil {
		return nil, err
	}

	if meta == nil {
		return nil, nil
	}

	return c.findSecretsFromName(ctx, meta.Keys, *ref.Name)
}

func (c *client) findSecretsFromName(ctx context.Context, candidates []string, ref esv1.FindName) (map[string][]byte, error) {
	secrets := make(map[string][]byte)
	matcher, err := find.New(ref)
	if err != nil {
		return nil, err
	}
	for _, name := range candidates {
		ok := matcher.MatchName(name)
		if ok {
			secret, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: name})
			if errors.Is(err, esv1.NoSecretError{}) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if secret != nil {
				secrets[name] = secret
			}
		}
	}
	return secrets, nil
}

func (c *client) useV1() bool {
	return c.store.Version == esv1.OpenBaoKVStoreV1
}

func (c *client) path() string {
	if c.store.Path != nil {
		return *c.store.Path
	}
	return "kv"
}

func (c *client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	var data *api.KVSecret
	var err error

	if c.useV1() {
		if ref.Version != "" {
			return nil, errors.New(errKVv1VersionUnsupported)
		}

		kv := c.client.KVv1(c.path())
		data, err = kv.Get(ctx, ref.Key)
		if err != nil {
			return nil, err
		}
	} else {
		kv := c.client.KVv2(c.path())
		if ref.Version != "" {
			version, err := strconv.Atoi(ref.Version)
			if err != nil {
				return nil, fmt.Errorf(errInvalidRevVersion, err)
			}

			data, err = kv.GetVersion(ctx, ref.Key, version)
			if err != nil {
				return nil, err
			}
		} else {
			data, err = kv.Get(ctx, ref.Key)
			if err != nil {
				return nil, err
			}
		}
	}

	if ref.Property == "" {
		return json.Marshal(data.Data)
	}

	property, ok := data.Data[ref.Property]
	if !ok {
		return nil, fmt.Errorf(errSecretKeyNotFound, ref.Property)
	}

	return esutils.GetByteValue(property)
}

func (c *client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := c.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	var secretData map[string]any
	err = json.Unmarshal(data, &secretData)
	if err != nil {
		return nil, err
	}
	byteMap := make(map[string][]byte, len(secretData))
	for k, v := range secretData {
		byteMap[k], err = esutils.GetByteValue(v)
		if err != nil {
			return nil, err
		}
	}

	return byteMap, nil
}

func (c *client) PushSecret(_ context.Context, _ *v1.Secret, _ esv1.PushSecretData) error {
	return errors.New("push secret is not supported (the OpenBao provider is currently read only)")
}

func (c *client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

func (c *client) Validate() (esv1.ValidationResult, error) {
	// when using referent namespace we can not validate the token
	// because the namespace is not known yet when Validate() is called
	// from the SecretStore controller.
	if c.storeKind == esv1.ClusterSecretStoreKind && isReferentSpec(c.store) {
		return esv1.ValidationResultUnknown, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mount, err := c.client.Sys().MountInfoWithContext(ctx, c.path())
	if err != nil {
		return esv1.ValidationResultError, fmt.Errorf(errFetchMount, c.store.Server, err)
	}

	if mount.Type != "kv" {
		return esv1.ValidationResultError, fmt.Errorf(errInvalidMountType, mount.Type)
	}

	actualVersion := mount.Options["version"]
	expectedVersion := string(c.store.Version[1:]) // drop the "v" prefix
	if expectedVersion != actualVersion {
		return esv1.ValidationResultError, fmt.Errorf(errInvalidMountVersion, expectedVersion, actualVersion)
	}

	return esv1.ValidationResultReady, nil
}
