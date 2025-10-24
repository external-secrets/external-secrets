/*
Copyright © 2025 ESO Maintainer Team

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

package vault

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	vault "github.com/hashicorp/vault/api"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	errVaultRequest          = "error from Vault request: %w"
	errUnusupportedTransport = "unsupported http client transport: %T"
)

func setCertAuthToken(ctx context.Context, v *client, cfg *vault.Config) (bool, error) {
	certAuth := v.store.Auth.Cert
	if certAuth != nil {
		err := v.requestTokenWithCertAuth(ctx, certAuth, cfg)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func (c *client) requestTokenWithCertAuth(ctx context.Context, certAuth *esv1.VaultCertAuth, cfg *vault.Config) error {
	clientKey, err := resolvers.SecretKeyRef(ctx, c.kube, c.storeKind, c.namespace, &certAuth.SecretRef)
	if err != nil {
		return err
	}

	clientCert, err := resolvers.SecretKeyRef(ctx, c.kube, c.storeKind, c.namespace, &certAuth.ClientCert)
	if err != nil {
		return err
	}

	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return fmt.Errorf(errClientTLSAuth, err)
	}

	switch transport := cfg.HttpClient.Transport.(type) {
	case *http.Transport:
		transport.TLSClientConfig.GetClientCertificate = func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return &cert, nil
		}
	default:
		return fmt.Errorf(errUnusupportedTransport, transport)
	}

	path := certAuth.Path
	if path == "" {
		path = "cert"
	}
	url := strings.Join([]string{"auth", path, "login"}, "/")
	vaultResult, err := c.logical.WriteWithContext(ctx, url, nil)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return fmt.Errorf(errVaultRequest, err)
	}
	token, err := vaultResult.TokenID()
	if err != nil {
		return fmt.Errorf(errVaultToken, err)
	}
	c.client.SetToken(token)
	return nil
}
