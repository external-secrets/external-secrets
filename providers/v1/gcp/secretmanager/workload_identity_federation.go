/*
Copyright © 2025 ESO Maintainer Team

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

package secretmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	gsmapiv1 "cloud.google.com/go/secretmanager/apiv1"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google/externalaccount"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

// workloadIdentityFederation holds the clients and generators needed
// to create a gcp oauth token.
type workloadIdentityFederation struct {
	kubeClient       kclient.Client
	saTokenGenerator saTokenGenerator
	config           *esv1.GCPWorkloadIdentityFederation
	isClusterKind    bool
	namespace        string
}

// k8sSATokenReader holds the data for generating the federated token.
type k8sSATokenReader struct {
	audience         string
	subjectTokenType string
	saTokenGenerator saTokenGenerator
	saAudience       []string
	serviceAccount   types.NamespacedName
}

type awsSecurityCredentialsReader struct {
	region                 string
	awsSecurityCredentials *externalaccount.AwsSecurityCredentials
}

// credentialsFile is the unmarshalled representation of a credentials file.
// sourced from https://github.com/golang/oauth2/blob/master/google/google.go#L108-L144
// as the type is not exported.
type credentialsFile struct {
	Type string `json:"type"`

	// Service Account fields
	ClientEmail    string `json:"client_email"`
	PrivateKeyID   string `json:"private_key_id"`
	PrivateKey     string `json:"private_key"`
	AuthURL        string `json:"auth_uri"`
	TokenURL       string `json:"token_uri"`
	ProjectID      string `json:"project_id"`
	UniverseDomain string `json:"universe_domain"`

	// User Credential fields
	// (These typically come from gcloud auth.)
	ClientSecret string `json:"client_secret"`
	ClientID     string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`

	// External Account fields
	Audience                       string                           `json:"audience"`
	SubjectTokenType               string                           `json:"subject_token_type"`
	TokenURLExternal               string                           `json:"token_url"`
	TokenInfoURL                   string                           `json:"token_info_url"`
	ServiceAccountImpersonationURL string                           `json:"service_account_impersonation_url"`
	ServiceAccountImpersonation    serviceAccountImpersonationInfo  `json:"service_account_impersonation"`
	Delegates                      []string                         `json:"delegates"`
	CredentialSource               externalaccount.CredentialSource `json:"credential_source"`
	QuotaProjectID                 string                           `json:"quota_project_id"`
	WorkforcePoolUserProject       string                           `json:"workforce_pool_user_project"`

	// External Account Authorized User fields
	RevokeURL string `json:"revoke_url"`

	// Service account impersonation
	SourceCredentials *credentialsFile `json:"source_credentials"`
}

type serviceAccountImpersonationInfo struct {
	TokenLifetimeSeconds int `json:"token_lifetime_seconds"`
}

var (
	awsSTSTokenURLRegex                 = regexp.MustCompile(`^http://(metadata\.google\.internal|169\.254\.169\.254|\[fd00:ec2::254\])/latest/meta-data/iam/security-credentials$`)
	awsRegionURLRegex                   = regexp.MustCompile(`^http://(metadata\.google\.internal|169\.254\.169\.254|\[fd00:ec2::254\])/latest/meta-data/placement/availability-zone$`)
	awsSessionTokenURLRegex             = regexp.MustCompile(`^http://(metadata\.google\.internal|169\.254\.169\.254|\[fd00:ec2::254\])/latest/api/token$`)
	serviceAccountImpersonationURLRegex = regexp.MustCompile(`^https://iamcredentials\.googleapis\.com/v1/projects/-/serviceAccounts/(\S+):generateAccessToken$`)
)

const (
	// autoMountedServiceAccountTokenPath is the kubernetes service account token filepath
	// made available by automountServiceAccountToken option in pod spec.
	autoMountedServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// externalAccountCredentialType is the external account type indicator in the credentials files.
	externalAccountCredentialType = "external_account"

	awsEnvironmentIDPrefix    = "aws"
	awsAccessKeyIDKeyName     = "aws_access_key_id"
	awsSecretAccessKeyKeyName = "aws_secret_access_key"
	awsSessionTokenKeyName    = "aws_session_token"
)

func newWorkloadIdentityFederation(kube kclient.Client, wif *esv1.GCPWorkloadIdentityFederation, isClusterKind bool, namespace string) (*workloadIdentityFederation, error) {
	satg, err := newSATokenGenerator()
	if err != nil {
		return nil, err
	}
	return &workloadIdentityFederation{
		kubeClient:       kube,
		saTokenGenerator: satg,
		config:           wif,
		isClusterKind:    isClusterKind,
		namespace:        namespace,
	}, nil
}

func (w *workloadIdentityFederation) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	if w.config == nil {
		return nil, nil
	}

	invalidConfigErrPrefix := "invalid workloadIdentityFederation config"
	count := 0
	if w.config.CredConfig != nil {
		count++
	}
	if w.config.ServiceAccountRef != nil {
		count++
	}
	if w.config.AwsSecurityCredentials != nil {
		count++
	}
	if count != 1 {
		return nil, fmt.Errorf("%s: exactly one of credConfig, awsSecurityCredentials or serviceAccountRef must be provided", invalidConfigErrPrefix)
	}
	if (w.config.ServiceAccountRef != nil || w.config.AwsSecurityCredentials != nil) && w.config.Audience == "" {
		return nil, fmt.Errorf("%s: audience must be provided, when serviceAccountRef or awsSecurityCredentials is provided", invalidConfigErrPrefix)
	}

	config, err := w.readCredConfig(ctx)
	if err != nil {
		return nil, err
	}
	return externalaccount.NewTokenSource(ctx, *config)
}

// readCredConfig is for loading the json cred config stored in the provided configmap.
func (w *workloadIdentityFederation) readCredConfig(ctx context.Context) (*externalaccount.Config, error) {
	if w.config.CredConfig == nil {
		return w.generateExternalAccountConfig(ctx, nil)
	}

	key := types.NamespacedName{
		Name:      w.config.CredConfig.Name,
		Namespace: w.namespace,
	}
	if w.isClusterKind && w.config.CredConfig.Namespace != "" {
		key.Namespace = w.config.CredConfig.Namespace
	}

	cm := &corev1.ConfigMap{}
	if err := w.kubeClient.Get(ctx, key, cm); err != nil {
		return nil, fmt.Errorf("failed to fetch external acccount credentials configmap %q: %w", key, err)
	}

	credKeyName := w.config.CredConfig.Key
	credJSON, ok := cm.Data[credKeyName]
	if !ok {
		return nil, fmt.Errorf("missing key %q in configmap %q", credKeyName, w.config.CredConfig.Name)
	}
	if credJSON == "" {
		return nil, fmt.Errorf("key %q in configmap %q has empty value", credKeyName, w.config.CredConfig.Name)
	}
	credFile := &credentialsFile{}
	if err := json.Unmarshal([]byte(credJSON), credFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal external acccount config in %q: %w", w.config.CredConfig.Name, err)
	}

	return w.generateExternalAccountConfig(ctx, credFile)
}

func (w *workloadIdentityFederation) generateExternalAccountConfig(ctx context.Context, credFile *credentialsFile) (*externalaccount.Config, error) {
	var config = new(externalaccount.Config)

	if err := w.updateExternalAccountConfigWithCredFileValues(config, credFile); err != nil {
		return nil, err
	}
	w.updateExternalAccountConfigWithSubjectTokenSupplier(config)
	if err := w.updateExternalAccountConfigWithAWSCredentialsSupplier(ctx, config); err != nil {
		return nil, err
	}
	w.updateExternalAccountConfigWithDefaultValues(config)
	if err := validateExternalAccountConfig(config, w.config); err != nil {
		return nil, err
	}

	return config, nil
}

func (w *workloadIdentityFederation) updateExternalAccountConfigWithCredFileValues(config *externalaccount.Config, credFile *credentialsFile) error {
	if credFile == nil {
		return nil
	}

	if credFile.Type != externalAccountCredentialType {
		return fmt.Errorf("invalid credentials: 'type' field is %q (expected %q)", credFile.Type, externalAccountCredentialType)
	}

	config.Audience = credFile.Audience
	config.SubjectTokenType = credFile.SubjectTokenType
	config.TokenURL = credFile.TokenURLExternal
	config.TokenInfoURL = credFile.TokenInfoURL
	config.ServiceAccountImpersonationURL = credFile.ServiceAccountImpersonationURL
	config.ServiceAccountImpersonationLifetimeSeconds = credFile.ServiceAccountImpersonation.TokenLifetimeSeconds
	config.ClientSecret = credFile.ClientSecret
	config.ClientID = credFile.ClientID
	config.QuotaProjectID = credFile.QuotaProjectID
	config.UniverseDomain = credFile.UniverseDomain

	// disallow using token of operator serviceaccount, not everyone gets
	// same access defined to the operator. To use operator serviceaccount
	// once has to provide the service account reference explicitly.
	if !reflect.ValueOf(credFile.CredentialSource).IsZero() &&
		credFile.CredentialSource.File != autoMountedServiceAccountTokenPath {
		config.CredentialSource = &credFile.CredentialSource
	}

	return nil
}

func (w *workloadIdentityFederation) updateExternalAccountConfigWithDefaultValues(config *externalaccount.Config) {
	config.Scopes = gsmapiv1.DefaultAuthScopes()
	if w.config.Audience != "" {
		config.Audience = w.config.Audience
	}
	if config.SubjectTokenType == "" {
		config.SubjectTokenType = workloadIdentitySubjectTokenType
	}
	if config.TokenURL == "" {
		config.TokenURL = workloadIdentityTokenURL
	}
	if config.TokenInfoURL == "" {
		config.TokenInfoURL = workloadIdentityTokenInfoURL
	}
	if config.UniverseDomain == "" {
		config.UniverseDomain = defaultUniverseDomain
	}
}

func (w *workloadIdentityFederation) updateExternalAccountConfigWithAWSCredentialsSupplier(ctx context.Context, config *externalaccount.Config) error {
	awsCredentialsSupplier, err := w.readAWSSecurityCredentials(ctx)
	if err != nil {
		return err
	}
	if awsCredentialsSupplier != nil {
		config.AwsSecurityCredentialsSupplier = awsCredentialsSupplier
		config.SubjectTokenType = workloadIdentitySubjectTokenTypeAWS
	}
	return nil
}

func (w *workloadIdentityFederation) updateExternalAccountConfigWithSubjectTokenSupplier(config *externalaccount.Config) {
	if w.config.ServiceAccountRef == nil {
		return
	}

	ns := w.namespace
	if w.isClusterKind && w.config.ServiceAccountRef.Namespace != nil {
		ns = *w.config.ServiceAccountRef.Namespace
	}
	config.SubjectTokenSupplier = &k8sSATokenReader{
		audience:         w.config.Audience,
		subjectTokenType: workloadIdentitySubjectTokenType,
		saTokenGenerator: w.saTokenGenerator,
		saAudience:       w.config.ServiceAccountRef.Audiences,
		serviceAccount: types.NamespacedName{
			Name:      w.config.ServiceAccountRef.Name,
			Namespace: ns,
		},
	}
}

func (w *workloadIdentityFederation) readAWSSecurityCredentials(ctx context.Context) (*awsSecurityCredentialsReader, error) {
	awsCreds := w.config.AwsSecurityCredentials
	if awsCreds == nil {
		return nil, nil
	}

	key := types.NamespacedName{
		Name:      awsCreds.AwsCredentialsSecretRef.Name,
		Namespace: w.namespace,
	}
	if w.isClusterKind && awsCreds.AwsCredentialsSecretRef.Namespace != "" {
		key.Namespace = awsCreds.AwsCredentialsSecretRef.Namespace
	}
	secret := &corev1.Secret{}
	if err := w.kubeClient.Get(ctx, key, secret); err != nil {
		return nil, fmt.Errorf("failed to fetch AwsSecurityCredentials secret %q: %w", key, err)
	}

	accessKeyID := string(secret.Data[awsAccessKeyIDKeyName])
	secretAccessKey := string(secret.Data[awsSecretAccessKeyKeyName])
	sessionToken := string(secret.Data[awsSessionTokenKeyName])
	if accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("%s and %s keys must be present in AwsSecurityCredentials secret", awsAccessKeyIDKeyName, awsSecretAccessKeyKeyName)
	}

	return &awsSecurityCredentialsReader{
		region: w.config.AwsSecurityCredentials.Region,
		awsSecurityCredentials: &externalaccount.AwsSecurityCredentials{
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
			SessionToken:    sessionToken,
		},
	}, nil
}

// validateExternalAccountConfig is for validating the external_account credentials configurations, based on
// suggestions made at https://cloud.google.com/docs/authentication/client-libraries#external-credentials.
func validateExternalAccountConfig(config *externalaccount.Config, wif *esv1.GCPWorkloadIdentityFederation) error {
	var errs []error
	errs = append(errs, fmt.Errorf("invalid %s config", externalAccountCredentialType))

	if config.Audience == "" {
		errs = append(errs, fmt.Errorf("audience is empty"))
	}
	if config.ServiceAccountImpersonationURL != "" &&
		!serviceAccountImpersonationURLRegex.MatchString(config.ServiceAccountImpersonationURL) {
		errs = append(errs, fmt.Errorf("service_account_impersonation_url \"%s\" does not have expected value", config.ServiceAccountImpersonationURL))
	}
	if config.TokenURL != workloadIdentityTokenURL {
		errs = append(errs, fmt.Errorf("token_url \"%s\" must match %s", config.TokenURL, workloadIdentityTokenURL))
	}
	if config.CredentialSource != nil {
		errs = append(errs, validateCredConfigCredentialSource(config.CredentialSource, wif)...)
	}
	if len(errs) > 1 {
		return errors.Join(errs...)
	}

	return nil
}

func validateCredConfigCredentialSource(credSource *externalaccount.CredentialSource, wif *esv1.GCPWorkloadIdentityFederation) []error {
	var errs []error
	// restricting the use of executables from security standpoint, since executables can't be validated.
	if credSource.Executable != nil {
		errs = append(errs, fmt.Errorf("credential_source.executable.command is not allowed"))
	}
	if credSource.File == "" && credSource.URL == "" && credSource.EnvironmentID == "" {
		errs = append(errs, fmt.Errorf("one of credential_source.file, credential_source.url, credential_source.aws.url or credential_source_environment_id should be provided"))
	}
	if credSource.EnvironmentID == "" && credSource.URL != wif.ExternalTokenEndpoint {
		errs = append(errs, fmt.Errorf("credential_source.url \"%s\" does not match with the configured %s externalTokenEndpoint", credSource.URL, wif.ExternalTokenEndpoint))
	}
	errs = append(errs, validateCredConfigAWSCredentialSource(credSource)...)

	return errs
}

func validateCredConfigAWSCredentialSource(credSource *externalaccount.CredentialSource) []error {
	var errs []error
	if credSource.EnvironmentID != "" {
		if !strings.HasPrefix(strings.ToLower(credSource.EnvironmentID), awsEnvironmentIDPrefix) {
			errs = append(errs, fmt.Errorf("credential_source.environment_id \"%s\" must start with %s", credSource.EnvironmentID, awsEnvironmentIDPrefix))
		}
		if !awsSTSTokenURLRegex.MatchString(credSource.URL) {
			errs = append(errs, fmt.Errorf("credential_source.aws.url \"%s\" does not have expected value", credSource.URL))
		}
		if !awsRegionURLRegex.MatchString(credSource.RegionURL) {
			errs = append(errs, fmt.Errorf("credential_source.aws.region_url \"%s\" does not have expected value", credSource.RegionURL))
		}
		if credSource.IMDSv2SessionTokenURL != "" && !awsSessionTokenURLRegex.MatchString(credSource.IMDSv2SessionTokenURL) {
			errs = append(errs, fmt.Errorf("credential_source.aws.imdsv2_session_token_url \"%s\" does not have expected value", credSource.IMDSv2SessionTokenURL))
		}
	}
	return errs
}

func (r *k8sSATokenReader) SubjectToken(ctx context.Context, options externalaccount.SupplierOptions) (string, error) {
	if options.Audience != r.audience || options.SubjectTokenType != r.subjectTokenType {
		return "", fmt.Errorf(
			"invalid subject token request, audience is %s(expected %s) and subject_token_type is %s(expected %s)",
			options.Audience,
			r.audience,
			options.SubjectTokenType,
			r.subjectTokenType,
		)
	}

	resp, err := r.saTokenGenerator.Generate(ctx, r.saAudience, r.serviceAccount.Name, r.serviceAccount.Namespace)
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMGenerateSAToken, err)
	if err != nil {
		return "", fmt.Errorf(errFetchPodToken, err)
	}

	return resp.Status.Token, nil
}

// AwsRegion returns the AWS region for workload identity federation.
func (a *awsSecurityCredentialsReader) AwsRegion(_ context.Context, _ externalaccount.SupplierOptions) (string, error) {
	return a.region, nil
}

// AwsSecurityCredentials returns AWS security credentials for workload identity federation.
func (a *awsSecurityCredentialsReader) AwsSecurityCredentials(_ context.Context, _ externalaccount.SupplierOptions) (*externalaccount.AwsSecurityCredentials, error) {
	return a.awsSecurityCredentials, nil
}
