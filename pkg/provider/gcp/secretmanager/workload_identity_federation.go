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

package secretmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google/externalaccount"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// workloadIdentityFederation holds the GCP external account cred config
// required to create a gcp oauth token.
type workloadIdentityFederation struct {
	Config *externalaccount.Config
}

// credentialsFile is the unmarshalled representation of a credentials file.
// sourced from github.com/golang/oauth2/google.credentialsFile as the
// type is not exported.
type credentialsFile struct {
	Type string `json:"type"`

	// External Account fields
	Audience                       string                           `json:"audience"`
	SubjectTokenType               string                           `json:"subject_token_type"`
	TokenURLExternal               string                           `json:"token_url"`
	TokenInfoURL                   string                           `json:"token_info_url"`
	ServiceAccountImpersonationURL string                           `json:"service_account_impersonation_url"`
	ServiceAccountImpersonation    serviceAccountImpersonationInfo  `json:"service_account_impersonation"`
	Delegates                      []string                         `json:"delegates"`
	ClientSecret                   string                           `json:"client_secret"`
	ClientID                       string                           `json:"client_id"`
	CredentialSource               externalaccount.CredentialSource `json:"credential_source"`
	QuotaProjectID                 string                           `json:"quota_project_id"`
	WorkforcePoolUserProject       string                           `json:"workforce_pool_user_project"`
	UniverseDomain                 string                           `json:"universe_domain"`
}

type serviceAccountImpersonationInfo struct {
	TokenLifetimeSeconds int `json:"token_lifetime_seconds"`
}

type k8sSATokenReader struct {
	Audience         string
	SubjectTokenType string
}

const (
	// autoMountedServiceAccountTokenPath is the kubernetes service account token filepath
	// made available by automountServiceAccountToken option in pod spec.
	autoMountedServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// externalAccountCredentialType is the external account type indicator in the credentials files.
	externalAccountCredentialType = "external_account"

	// defaultUniverseDomain is the domain which will be used in the STS token URL.
	defaultUniverseDomain = "googleapis.com"
)

// newWorkloadIdentityFederation returns an instance of workloadIdentityFederation which can be
// used for obtaining gcp oauth federated token.
func newWorkloadIdentityFederation(ctx context.Context, wifConfig *esv1beta1.GCPWorkloadIdentityFederation, kube kclient.Client) (*workloadIdentityFederation, error) {
	return readCredConfig(ctx, wifConfig, kube)
}

// readCredConfig is for loading the json cred config stored in the provided configmap.
func readCredConfig(ctx context.Context, wifConfig *esv1beta1.GCPWorkloadIdentityFederation, kube kclient.Client) (*workloadIdentityFederation, error) {
	key := types.NamespacedName{
		Name:      wifConfig.CredConfig.Name,
		Namespace: wifConfig.CredConfig.Namespace,
	}
	cm := &corev1.ConfigMap{}
	if err := kube.Get(ctx, key, cm); err != nil {
		return nil, fmt.Errorf("failed to fetch external acccount credentials configmap: %w", err)
	}

	configMapKey := wifConfig.CredConfig.Key
	if wifConfig.CredConfig.Key == "" {
		if len(cm.Data) == 0 {
			return nil, fmt.Errorf("no external acccount credentials found in %q configmap", wifConfig.CredConfig.Name)
		}
		for configMapKey = range cm.Data {
			break
		}
	}
	credJSON, ok := cm.Data[configMapKey]
	if !ok {
		return nil, fmt.Errorf("missing key %q in configmap %q", configMapKey, wifConfig.CredConfig.Name)
	}

	credFile := &credentialsFile{}
	if err := json.Unmarshal([]byte(credJSON), credFile); err != nil {
		return nil, err
	}

	config, err := generateExternalAccountConfig(credFile, wifConfig)
	if err != nil {
		return nil, err
	}

	return &workloadIdentityFederation{
		Config: config,
	}, nil
}

func generateExternalAccountConfig(credFile *credentialsFile, wifConfig *esv1beta1.GCPWorkloadIdentityFederation) (*externalaccount.Config, error) {
	if credFile.Type != externalAccountCredentialType {
		return nil, fmt.Errorf("invalid credentials: 'type' field is %q (expected %q)", credFile.Type, externalAccountCredentialType)
	}

	config := &externalaccount.Config{
		Audience:                       credFile.Audience,
		SubjectTokenType:               credFile.SubjectTokenType,
		TokenURL:                       credFile.TokenURLExternal,
		TokenInfoURL:                   credFile.TokenInfoURL,
		ServiceAccountImpersonationURL: credFile.ServiceAccountImpersonationURL,
		ServiceAccountImpersonationLifetimeSeconds: credFile.ServiceAccountImpersonation.TokenLifetimeSeconds,
		ClientSecret:             credFile.ClientSecret,
		ClientID:                 credFile.ClientID,
		CredentialSource:         &credFile.CredentialSource,
		QuotaProjectID:           credFile.QuotaProjectID,
		Scopes:                   secretmanager.DefaultAuthScopes(),
		WorkforcePoolUserProject: credFile.WorkforcePoolUserProject,
	}

	if wifConfig.Audience != "" {
		config.Audience = wifConfig.Audience
	}
	if config.CredentialSource == nil {
		config.SubjectTokenSupplier = &k8sSATokenReader{
			Audience:         config.Audience,
			SubjectTokenType: config.SubjectTokenType,
		}
	}
	if config.CredentialSource.File != "" && config.CredentialSource.File != autoMountedServiceAccountTokenPath {
		if _, err := os.Stat(config.CredentialSource.File); os.IsNotExist(err) {
			config.CredentialSource.File = autoMountedServiceAccountTokenPath
		}
	}
	if config.UniverseDomain == "" {
		config.UniverseDomain = defaultUniverseDomain
	}

	return config, nil
}

func (r *k8sSATokenReader) SubjectToken(ctx context.Context, options externalaccount.SupplierOptions) (string, error) {
	if options.Audience != r.Audience || options.SubjectTokenType != r.SubjectTokenType {
		return "", fmt.Errorf("invalid subject token request, audience is %s(expected %s) and subject_token_type is %s(expected %s)", options.Audience, r.Audience, options.SubjectTokenType, r.SubjectTokenType)
	}

	token, err := os.ReadFile(autoMountedServiceAccountTokenPath)
	if err != nil {
		return "", err
	}
	return string(token), nil
}

// TokenSource creates a new external account token source for generating federated tokens.
func (wif *workloadIdentityFederation) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	return externalaccount.NewTokenSource(ctx, *wif.Config)
}
