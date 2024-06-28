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

package conjur

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/cyberark/conjur-api-go/conjurapi"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const JwtLifespan = 600 // 10 minutes

// getJWTToken retrieves a JWT token either using the TokenRequest API for a specified service account, or from a jwt stored in a k8s secret.
func (c *Client) getJWTToken(ctx context.Context, conjurJWTConfig *esv1beta1.ConjurJWT) (string, error) {
	if conjurJWTConfig.ServiceAccountRef != nil {
		// Should work for Kubernetes >=v1.22: fetch token via TokenRequest API
		jwtToken, err := c.getJwtFromServiceAccountTokenRequest(ctx, *conjurJWTConfig.ServiceAccountRef, nil, JwtLifespan)
		if err != nil {
			return "", err
		}
		return jwtToken, nil
	} else if conjurJWTConfig.SecretRef != nil {
		tokenRef := conjurJWTConfig.SecretRef
		if tokenRef.Key == "" {
			tokenRef = conjurJWTConfig.SecretRef.DeepCopy()
			tokenRef.Key = "token"
		}
		jwtToken, err := resolvers.SecretKeyRef(
			ctx,
			c.kube,
			c.StoreKind,
			c.namespace,
			tokenRef)
		if err != nil {
			return "", err
		}
		return jwtToken, nil
	}
	return "", fmt.Errorf("missing ServiceAccountRef or SecretRef")
}

// getJwtFromServiceAccountTokenRequest uses the TokenRequest API to get a JWT token for the given service account.
func (c *Client) getJwtFromServiceAccountTokenRequest(ctx context.Context, serviceAccountRef esmeta.ServiceAccountSelector, additionalAud []string, expirationSeconds int64) (string, error) {
	audiences := serviceAccountRef.Audiences
	if len(additionalAud) > 0 {
		audiences = append(audiences, additionalAud...)
	}
	tokenRequest := &authenticationv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.namespace,
		},
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}
	if (c.StoreKind == esv1beta1.ClusterSecretStoreKind) &&
		(serviceAccountRef.Namespace != nil) {
		tokenRequest.Namespace = *serviceAccountRef.Namespace
	}
	tokenResponse, err := c.corev1.ServiceAccounts(tokenRequest.Namespace).CreateToken(ctx, serviceAccountRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf(errGetKubeSATokenRequest, serviceAccountRef.Name, err)
	}
	return tokenResponse.Status.Token, nil
}

// newClientFromJwt creates a new Conjur client using the given JWT Auth Config.
func (c *Client) newClientFromJwt(ctx context.Context, config conjurapi.Config, jwtAuth *esv1beta1.ConjurJWT) (SecretsClient, error) {
	jwtToken, getJWTError := c.getJWTToken(ctx, jwtAuth)
	if getJWTError != nil {
		return nil, getJWTError
	}

	client, clientError := c.clientAPI.NewClientFromJWT(config, jwtToken, jwtAuth.ServiceID, jwtAuth.HostID)
	if clientError != nil {
		return nil, clientError
	}

	return client, nil
}

// newHTTPSClient creates a new HTTPS client with the given cert.
func newHTTPSClient(cert []byte) (*http.Client, error) {
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(cert)
	if !ok {
		return nil, fmt.Errorf("can't append Conjur SSL cert")
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}
	return &http.Client{Transport: tr, Timeout: time.Second * 10}, nil
}
