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
	"fmt"
	"net/http"
	"strings"

	bao "github.com/hashicorp/vault/api"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	errOpenBaoRequest        = "error from OpenBao request: %w"
	errUnusupportedTransport = "unsupported http client transport: %T"
)

func setCertAuthToken(ctx context.Context, v *client, cfg *bao.Config) (bool, error) {
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

func (c *client) requestTokenWithCertAuth(ctx context.Context, certAuth *esv1.OpenBaoCertAuth, cfg *bao.Config) error {
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

	var loginData map[string]any
	if certAuth.OpenBaoRole != "" {
		loginData = map[string]any{
			"name": certAuth.OpenBaoRole,
		}
	}

	url := strings.Join([]string{"auth", path, "login"}, "/")
	baoResult, err := c.logical.WriteWithContext(ctx, url, loginData)
	metrics.ObserveAPICall(constants.ProviderOpenBao, constants.CallOpenBaoLogin, err)
	if err != nil {
		return fmt.Errorf(errOpenBaoRequest, err)
	}
	token, err := baoResult.TokenID()
	if err != nil {
		return fmt.Errorf(errOpenBaoToken, err)
	}
	c.client.SetToken(token)
	return nil
}
