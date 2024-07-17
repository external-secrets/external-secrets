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

package vault

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	vault "github.com/hashicorp/vault/api"
	corev1 "k8s.io/api/core/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/util"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

var _ esv1beta1.SecretsClient = &client{}

type client struct {
	kube      kclient.Client
	store     *esv1beta1.VaultProvider
	log       logr.Logger
	corev1    typedcorev1.CoreV1Interface
	client    util.Client
	auth      util.Auth
	logical   util.Logical
	token     util.Token
	namespace string
	storeKind string
}

func (c *client) newConfig(ctx context.Context) (*vault.Config, error) {
	cfg := vault.DefaultConfig()
	cfg.Address = c.store.Server

	if len(c.store.CABundle) != 0 || c.store.CAProvider != nil {
		caCertPool := x509.NewCertPool()
		ca, err := utils.FetchCACertFromSource(ctx, utils.CreateCertOpts{
			CABundle:   c.store.CABundle,
			CAProvider: c.store.CAProvider,
			StoreKind:  c.storeKind,
			Namespace:  c.namespace,
			Client:     c.kube,
		})
		if err != nil {
			return nil, err
		}
		ok := caCertPool.AppendCertsFromPEM(ca)
		if !ok {
			return nil, fmt.Errorf(errVaultCert, errors.New("failed to parse certificates from CertPool"))
		}

		if transport, ok := cfg.HttpClient.Transport.(*http.Transport); ok {
			transport.TLSClientConfig.RootCAs = caCertPool
		}
	}

	err := c.configureClientTLS(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// If either read-after-write consistency feature is enabled, enable ReadYourWrites
	cfg.ReadYourWrites = c.store.ReadYourWrites || c.store.ForwardInconsistent

	return cfg, nil
}

func (c *client) configureClientTLS(ctx context.Context, cfg *vault.Config) error {
	clientTLS := c.store.ClientTLS
	if clientTLS.CertSecretRef != nil && clientTLS.KeySecretRef != nil {
		if clientTLS.KeySecretRef.Key == "" {
			clientTLS.KeySecretRef.Key = corev1.TLSPrivateKeyKey
		}
		clientKey, err := resolvers.SecretKeyRef(ctx, c.kube, c.storeKind, c.namespace, clientTLS.KeySecretRef)
		if err != nil {
			return err
		}

		if clientTLS.CertSecretRef.Key == "" {
			clientTLS.CertSecretRef.Key = corev1.TLSCertKey
		}
		clientCert, err := resolvers.SecretKeyRef(ctx, c.kube, c.storeKind, c.namespace, clientTLS.CertSecretRef)
		if err != nil {
			return err
		}

		cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
		if err != nil {
			return fmt.Errorf(errClientTLSAuth, err)
		}

		if transport, ok := cfg.HttpClient.Transport.(*http.Transport); ok {
			transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
		}
	}
	return nil
}

func (c *client) Close(ctx context.Context) error {
	// Revoke the token if we have one set, it wasn't sourced from a TokenSecretRef,
	// and token caching isn't enabled
	if !enableCache && c.client.Token() != "" && c.store.Auth.TokenSecretRef == nil {
		err := revokeTokenIfValid(ctx, c.client)
		if err != nil {
			return err
		}
	}
	return nil
}
