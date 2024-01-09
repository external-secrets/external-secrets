/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://wwg.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package github

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

type Github struct {
	http      *http.Client
	kube      client.Client
	namespace string
	store     esv1beta1.GenericStore
	storeKind string
	url       string
}

func (g *Github) getPrivateKey(ctx context.Context) (*rsa.PrivateKey, error) {
	provider, err := getProvider(g.store)
	if err != nil {
		return nil, fmt.Errorf("can't get provider: %w", err)
	}

	key, err := g.getStoreSecret(ctx, provider.Auth.SecretRef.PrivatKey)
	if err != nil {
		return nil, fmt.Errorf("can't get provider auth secret: %w", err)
	}

	pk, err := jwt.ParseRSAPrivateKeyFromPEM(key.Data[provider.Auth.SecretRef.PrivatKey.Key])
	if err != nil {
		return nil, fmt.Errorf("error parsing RSA private key: %w", err)
	}
	return pk, nil
}

// Get github installation token.
func (g *Github) getInstallationToken(key *rsa.PrivateKey) (string, error) {
	provider, err := getProvider(g.store)
	if err != nil {
		return "", fmt.Errorf("can't get provider: %w", err)
	}

	claims := jwt.RegisteredClaims{
		Issuer:    provider.AppID,
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Second * 10)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * 300)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("error signing token: %w", err)
	}

	return signedToken, nil
}

func (g *Github) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	key, err := g.getPrivateKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("error parsing RSA private key: %w", err)
	}

	itoken, err := g.getInstallationToken(key)
	if err != nil {
		return nil, fmt.Errorf("can't get InstallationToken: %w", err)
	}

	// Github api expects POST request
	req, err := http.NewRequestWithContext(ctx, "POST", g.url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+itoken)
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	resp, err := g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	}
	defer resp.Body.Close()

	// git access token
	var gat map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&gat); err != nil && resp.StatusCode < 300 {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	accessToken, ok := gat[ref.Key].(string)
	if !ok {
		return nil, fmt.Errorf("token is not a string")
	}

	return []byte(accessToken), nil
}
