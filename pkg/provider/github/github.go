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
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/golang-jwt/jwt/v5"
)

const (
	ghAPIUrl = "https://api.github.com/app/installations/%s/access_tokens"
)

// Get github installation token
func (g *Github) getInstallationToken(ctx context.Context) (string, error) {
	provider, err := getProvider(g.store)
	if err != nil {
		return "", fmt.Errorf("Can't get provider: %w", err)
	}

	key, err := g.getStoreSecret(ctx, provider.Auth.SecretRef.PrivatKey)
	if err != nil {
		return "", fmt.Errorf("Can't get provider auth secret: %w", err)
	}

	pk, err := jwt.ParseRSAPrivateKeyFromPEM(key.Data[provider.Auth.SecretRef.PrivatKey.Key])
	if err != nil {
		return "", fmt.Errorf("error parsing RSA private key: %w", err)
	}

	claims := jwt.RegisteredClaims{
		Issuer:    provider.AppID,
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Second * 10)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * 300)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(pk)
	if err != nil {
		return "", fmt.Errorf("error signing token: %w", err)
	}

	return signedToken, nil
}

func (g *Github) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	provider, err := getProvider(g.store)
	if err != nil {
		return nil, fmt.Errorf("Can't get provider: %w", err)
	}

	itoken, err := g.getInstallationToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("Can't get InstallationToken: %w", err)
	}

	ghAPI := fmt.Sprintf(ghAPIUrl, provider.InstallID)

	// Github api expects POST request
	req, err := http.NewRequestWithContext(ctx, "POST", ghAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+itoken)
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	}
	defer resp.Body.Close()

	var gat map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&gat); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	accessToken, ok := gat["token"].(string)
	if !ok {
		return nil, fmt.Errorf("token is not a string")
	}

	return []byte(accessToken), nil
}
