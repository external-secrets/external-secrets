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

package quay

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type Generator struct {
	httpClient *http.Client
}

const (
	defaultQuayURL = "quay.io"

	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"
	errGetToken  = "unable to get authorization token: %w"

	httpClientTimeout = 5 * time.Second
)

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(
		ctx,
		jsonSpec,
		kube,
		namespace,
	)
}

func (g *Generator) Cleanup(_ context.Context, jsonSpec *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *Generator) generate(
	ctx context.Context,
	jsonSpec *apiextensions.JSON,
	_ client.Client,
	namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	// Fetch the service account token
	token, err := fetchServiceAccountToken(ctx, res.Spec.ServiceAccountRef, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch service account token: %w", err)
	}
	url := res.Spec.URL
	if url == "" {
		url = defaultQuayURL
	}
	url = strings.TrimPrefix(url, "https://")

	accessToken, err := getQuayRobotToken(ctx, token, res.Spec.RobotAccount, url, g.httpClient)
	if err != nil {
		return nil, nil, err
	}
	exp, err := tokenExpiration(accessToken)
	if err != nil {
		return nil, nil, err
	}
	return map[string][]byte{
		"registry": []byte(url),
		"auth":     []byte(b64.StdEncoding.EncodeToString([]byte(res.Spec.RobotAccount + ":" + accessToken))),
		"expiry":   []byte(exp),
	}, nil, nil
}

func getClaims(tokenString string) (map[string]interface{}, error) {
	// Split the token into its three parts
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Decode the payload (the second part of the token)
	payload, err := b64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("error decoding payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("error un-marshaling claims: %w", err)
	}
	return claims, nil
}

func tokenExpiration(tokenString string) (string, error) {
	claims, err := getClaims(tokenString)
	if err != nil {
		return "", fmt.Errorf("error getting claims: %w", err)
	}
	exp, ok := claims["exp"].(float64)
	if ok {
		return strconv.FormatFloat(exp, 'f', -1, 64), nil
	}

	return "", fmt.Errorf("exp claim not found or wrong type")
}

// https://docs.projectquay.io/manage_quay.html#exchanging-oauth2-robot-account-token
func getQuayRobotToken(ctx context.Context, fedToken, robotAccount, url string, hc *http.Client) (string, error) {
	if hc == nil {
		hc = &http.Client{
			Timeout: httpClientTimeout,
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://"+url+"/oauth2/federation/robot/token", http.NoBody)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(robotAccount, fedToken)
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("request failed do to unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}
	token, ok := result["token"]
	if !ok {
		return "", fmt.Errorf("token not found in response")
	}
	tokenString, ok := token.(string)
	if !ok {
		return "", fmt.Errorf("error when typecasting token to string")
	}
	return tokenString, nil
}

func fetchServiceAccountToken(ctx context.Context, saRef esmeta.ServiceAccountSelector, namespace string) (string, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return "", err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences: saRef.Audiences,
		},
	}
	tokenResponse, err := kubeClient.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, saRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}
	return tokenResponse.Status.Token, nil
}

func parseSpec(data []byte) (*genv1alpha1.QuayAccessToken, error) {
	var spec genv1alpha1.QuayAccessToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.QuayAccessTokenKind, &Generator{})
}
