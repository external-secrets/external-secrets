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

package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	grafanaclient "github.com/grafana/grafana-openapi-client-go/client"
	grafanasa "github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/grafana-openapi-client-go/models"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type Grafana struct{}

func (w *Grafana) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kclient client.Client, ns string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	gen, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, err
	}

	cl, err := newClient(ctx, gen, kclient, ns)
	if err != nil {
		return nil, nil, err
	}

	state, err := createOrGetServiceAccount(cl, gen)
	if err != nil {
		return nil, nil, err
	}

	// create new token
	res, err := cl.ServiceAccounts.CreateToken(&grafanasa.CreateTokenParams{
		ServiceAccountID: *state.ServiceAccount.ServiceAccountID,
		Body: &models.AddServiceAccountTokenCommand{
			Name: uuid.New().String(),
		},
	}, nil)
	if err != nil {
		return nil, nil, err
	}
	state.ServiceAccount.ServiceAccountTokenID = ptr.To(res.Payload.ID)
	return tokenResponse(state, res.Payload.Key)
}

func (w *Grafana) Cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, previousStatus genv1alpha1.GeneratorProviderState, kclient client.Client, ns string) error {
	if previousStatus == nil {
		return fmt.Errorf("missing previous status")
	}
	status, err := parseStatus(previousStatus.Raw)
	if err != nil {
		return err
	}
	gen, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return err
	}
	cl, err := newClient(ctx, gen, kclient, ns)
	if err != nil {
		return err
	}
	_, err = cl.ServiceAccounts.DeleteToken(*status.ServiceAccount.ServiceAccountTokenID, *status.ServiceAccount.ServiceAccountID)
	if err != nil && !strings.Contains(err.Error(), "service account token not found") {
		return err
	}
	return nil
}

func newClient(ctx context.Context, gen *genv1alpha1.Grafana, kclient client.Client, ns string) (*grafanaclient.GrafanaHTTPAPI, error) {
	parsedURL, err := url.Parse(gen.Spec.URL)
	if err != nil {
		return nil, err
	}

	cfg := &grafanaclient.TransportConfig{
		Host:     parsedURL.Host,
		BasePath: parsedURL.JoinPath("/api").Path,
		Schemes:  []string{parsedURL.Scheme},
	}

	if err := setGrafanaClientCredentials(ctx, gen, kclient, ns, cfg); err != nil {
		return nil, err
	}

	return grafanaclient.NewHTTPClientWithConfig(nil, cfg), nil
}

func setGrafanaClientCredentials(ctx context.Context, gen *genv1alpha1.Grafana, kclient client.Client, ns string, cfg *grafanaclient.TransportConfig) error {
	// First try to use service account auth
	if gen.Spec.Auth.Token != nil {
		serviceAccountAPIKey, err := resolvers.SecretKeyRef(ctx, kclient, resolvers.EmptyStoreKind, ns, &esmeta.SecretKeySelector{
			Namespace: &ns,
			Name:      gen.Spec.Auth.Token.Name,
			Key:       gen.Spec.Auth.Token.Key,
		})
		if err != nil {
			return err
		}

		cfg.APIKey = serviceAccountAPIKey
		return nil
	}

	// Next try to use basic auth
	if gen.Spec.Auth.Basic != nil {
		basicAuthPassword, err := resolvers.SecretKeyRef(ctx, kclient, resolvers.EmptyStoreKind, ns, &esmeta.SecretKeySelector{
			Namespace: &ns,
			Name:      gen.Spec.Auth.Basic.Password.Name,
			Key:       gen.Spec.Auth.Basic.Password.Key,
		})
		if err != nil {
			return err
		}

		cfg.BasicAuth = url.UserPassword(gen.Spec.Auth.Basic.Username, basicAuthPassword)
		return nil
	}

	// No auth found, fail
	return fmt.Errorf("no auth configuration found")
}

func createOrGetServiceAccount(cl *grafanaclient.GrafanaHTTPAPI, gen *genv1alpha1.Grafana) (*genv1alpha1.GrafanaServiceAccountTokenState, error) {
	saList, err := cl.ServiceAccounts.SearchOrgServiceAccountsWithPaging(&grafanasa.SearchOrgServiceAccountsWithPagingParams{
		Query: ptr.To(gen.Spec.ServiceAccount.Name),
	})
	if err != nil {
		return nil, err
	}
	for _, sa := range saList.Payload.ServiceAccounts {
		if sa.Name == gen.Spec.ServiceAccount.Name {
			return &genv1alpha1.GrafanaServiceAccountTokenState{
				ServiceAccount: genv1alpha1.GrafanaStateServiceAccount{
					ServiceAccountID:    &sa.ID,
					ServiceAccountLogin: &sa.Login,
				},
			}, nil
		}
	}

	res, err := cl.ServiceAccounts.CreateServiceAccount(&grafanasa.CreateServiceAccountParams{
		Body: &models.CreateServiceAccountForm{
			Name: gen.Spec.ServiceAccount.Name,
			Role: gen.Spec.ServiceAccount.Role,
		},
	}, nil)
	if err != nil {
		return nil, err
	}

	return &genv1alpha1.GrafanaServiceAccountTokenState{
		ServiceAccount: genv1alpha1.GrafanaStateServiceAccount{
			ServiceAccountID:    ptr.To(res.Payload.ID),
			ServiceAccountLogin: &res.Payload.Login,
		},
	}, nil
}

func tokenResponse(state *genv1alpha1.GrafanaServiceAccountTokenState, token string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	newStateJSON, err := json.Marshal(state)
	if err != nil {
		return nil, nil, err
	}
	return map[string][]byte{
		"login": []byte(*state.ServiceAccount.ServiceAccountLogin),
		"token": []byte(token),
	}, &apiextensions.JSON{Raw: newStateJSON}, nil
}

func parseSpec(data []byte) (*genv1alpha1.Grafana, error) {
	var spec genv1alpha1.Grafana
	err := json.Unmarshal(data, &spec)
	return &spec, err
}

func parseStatus(data []byte) (*genv1alpha1.GrafanaServiceAccountTokenState, error) {
	var state genv1alpha1.GrafanaServiceAccountTokenState
	err := json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}
	return &state, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.GrafanaKind, &Grafana{})
}
