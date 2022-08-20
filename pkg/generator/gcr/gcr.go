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

package gcr

import (
	"context"
	"fmt"
	"strconv"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider/gcp/secretmanager"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Generator struct{}

const (
	defaultLoginUsername = `oauth2accesstoken`

	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"
	errGetToken  = "unable to get authorization token: %w"
)

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, error) {
	if jsonSpec == nil {
		return nil, fmt.Errorf(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, fmt.Errorf(errParseSpec, err)
	}
	ts, err := secretmanager.NewTokenSource(ctx, res.Spec.Auth, res.Spec.ProjectID, false, kube, namespace)
	if err != nil {
		return nil, err
	}
	token, err := ts.Token()
	if err != nil {
		return nil, err
	}
	exp := strconv.FormatInt(token.Expiry.UTC().Unix(), 10)
	return map[string][]byte{
		"username": []byte(defaultLoginUsername),
		"password": []byte(token.AccessToken),
		"expiry":   []byte(exp),
	}, nil
}

func parseSpec(data []byte) (*genv1alpha1.GCRAccessToken, error) {
	var spec genv1alpha1.GCRAccessToken
	err := json.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.GCRAccessTokenKind, &Generator{})
}
