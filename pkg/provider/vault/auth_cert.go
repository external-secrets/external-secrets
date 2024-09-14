//Copyright External Secrets Inc. All Rights Reserved

package vault

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	vault "github.com/hashicorp/vault/api"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errVaultRequest = "error from Vault request: %w"
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

func (c *client) requestTokenWithCertAuth(ctx context.Context, certAuth *esv1beta1.VaultCertAuth, cfg *vault.Config) error {
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

	if transport, ok := cfg.HttpClient.Transport.(*http.Transport); ok {
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}

	url := strings.Join([]string{"auth", "cert", "login"}, "/")
	vaultResult, err := c.logical.WriteWithContext(ctx, url, nil)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultWriteSecretData, err)
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
