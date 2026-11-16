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

package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestValidateGithubProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider *esv1.GithubProvider
		wantType esv1.GithubSecretType
		wantErr  string
	}{
		{
			name:     "omitted secret type defaults to Actions",
			provider: &esv1.GithubProvider{},
			wantType: esv1.GithubSecretTypeActions,
		},
		{
			name:     "explicit Actions",
			provider: &esv1.GithubProvider{SecretType: esv1.GithubSecretTypeActions},
			wantType: esv1.GithubSecretTypeActions,
		},
		{
			name: "Dependabot with empty environment",
			provider: &esv1.GithubProvider{
				SecretType:  esv1.GithubSecretTypeDependabot,
				Environment: "",
			},
			wantType: esv1.GithubSecretTypeDependabot,
		},
		{
			name: "Dependabot environment is unsupported",
			provider: &esv1.GithubProvider{
				SecretType:  esv1.GithubSecretTypeDependabot,
				Repository:  "repository",
				Environment: "production",
			},
			wantErr: "Dependabot secrets do not support environments",
		},
		{
			name:     "unsupported secret type",
			provider: &esv1.GithubProvider{SecretType: esv1.GithubSecretType("Unsupported")},
			wantErr:  "unsupported GitHub secret type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, err := validateGithubProvider(tt.provider)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, gotType)
		})
	}
}

func TestProviderValidateStoreRejectsInvalidSecretTypeConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		provider *esv1.GithubProvider
		wantErr  string
	}{
		{
			name:     "unsupported secret type",
			provider: &esv1.GithubProvider{SecretType: esv1.GithubSecretType("Unsupported")},
			wantErr:  "unsupported GitHub secret type",
		},
		{
			name: "Dependabot environment",
			provider: &esv1.GithubProvider{
				SecretType:  esv1.GithubSecretTypeDependabot,
				Environment: "production",
			},
			wantErr: "Dependabot secrets do not support environments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{Github: tt.provider},
				},
			}
			_, err := (&Provider{}).ValidateStore(store)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestNewClientRejectsInvalidSecretTypeBeforeAuthentication(t *testing.T) {
	tests := []struct {
		name     string
		provider *esv1.GithubProvider
		wantErr  string
	}{
		{
			name:     "unsupported secret type",
			provider: &esv1.GithubProvider{SecretType: esv1.GithubSecretType("Unsupported")},
			wantErr:  "unsupported GitHub secret type",
		},
		{
			name: "Dependabot environment",
			provider: &esv1.GithubProvider{
				SecretType:  esv1.GithubSecretTypeDependabot,
				Environment: "production",
			},
			wantErr: "Dependabot secrets do not support environments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{Github: tt.provider},
				},
			}
			_, err := newClient(context.Background(), store, nil, "default")
			require.ErrorContains(t, err, tt.wantErr)
			assert.NotContains(t, err.Error(), "private key")
		})
	}
}
