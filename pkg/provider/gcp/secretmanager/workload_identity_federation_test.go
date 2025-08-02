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
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2/google/externalaccount"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type workloadIdentityFederationTest struct {
	name              string
	wifConfig         *esv1.GCPWorkloadIdentityFederation
	configMapData     map[string]string
	kubeObjects       []client.Object
	genSAToken        func(context.Context, []string, string, string) (*authv1.TokenRequest, error)
	expectError       string
	expectTokenSource bool
}

const (
	testConfigMapName                  = "external-account-config"
	testConfigMapKey                   = "config.json"
	testServiceAccount                 = "test-sa"
	testAudience                       = "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/test-pool/providers/test-provider"
	testServiceAccountImpersonationURL = "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/test@test.iam.gserviceaccount.com:generateAccessToken"
	testSAToken                        = "test-sa-token"
	testAwsRegion                      = "us-west-2"
	// below values taken from https://docs.aws.amazon.com/sdkref/latest/guide/feature-static-credentials.html
	testAwsAccessKey = "AKIAIOSFODNN7EXAMPLE"
	testAwsSecretKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	// below value taken from https://docs.aws.amazon.com/STS/latest/APIReference/API_GetSessionToken.html
	testAwsSessionToken        = "AQoEXAMPLEH4aoAH0gNCAPyJxz4BlCFFxWNE1OPTgk5TthT+FvwqnKwRcOIfrRh3c/LTo6UDdyJwOOvEVPvLXCrrrUtdnniCEXAMPLE/IvU1dYUg2RVAJBanLiHb4IgRmpRV3zrkuWJOgQs8IZZaIv2BXIa2R4OlgkBN9bkUDNCJiBeb/AXlzBBko7b15fjrBs2+cTQtpZ3CYWFXG8C5zqx37wnOE49mRl/+OtkIKGO7fAE"
	testAwsTokenIPV4URL        = "http://169.254.169.254/latest/meta-data/iam/security-credentials"
	testAwsRegionIPv4URL       = "http://169.254.169.254/latest/meta-data/placement/availability-zone"
	testAwsSessionTokenIPv4URL = "http://169.254.169.254/latest/api/token"
	testAwsTokenIPV6URL        = "http://[fd00:ec2::254]/latest/meta-data/iam/security-credentials"
	testAwsRegionIPv6URL       = "http://[fd00:ec2::254]/latest/meta-data/placement/availability-zone"
	testAwsSessionTokenIPv6URL = "http://[fd00:ec2::254]/latest/api/token"
)

var (
	testNamespace = "external-secrets-tests"
)

func createValidK8sExternalAccountConfig(audience string) string {
	config := map[string]interface{}{
		"type":               externalAccountCredentialType,
		"audience":           audience,
		"subject_token_type": workloadIdentitySubjectTokenType,
		"token_url":          workloadIdentityTokenURL,
		"credential_source": map[string]interface{}{
			"file": autoMountedServiceAccountTokenPath,
		},
		"token_info_url": workloadIdentityTokenInfoURL,
	}
	data, _ := json.Marshal(config)
	return string(data)
}

func createValidAWSExternalAccountConfig(audience string) string {
	config := map[string]interface{}{
		"type":                              externalAccountCredentialType,
		"audience":                          audience,
		"subject_token_type":                workloadIdentitySubjectTokenType,
		"token_url":                         workloadIdentityTokenURL,
		"service_account_impersonation_url": testServiceAccountImpersonationURL,
		"credential_source": map[string]interface{}{
			"environment_id":           "aws1",
			"url":                      testAwsTokenIPV4URL,
			"region_url":               testAwsRegionIPv4URL,
			"imdsv2_session_token_url": testAwsSessionTokenIPv4URL,
		},
	}
	data, _ := json.Marshal(config)
	return string(data)
}

func createInvalidTypeExternalAccountConfig() string {
	config := map[string]interface{}{
		"type":     "service_account",
		"audience": testAudience,
	}
	data, _ := json.Marshal(config)
	return string(data)
}

func defaultSATokenGenerator(ctx context.Context, idPool []string, namespace, name string) (*authv1.TokenRequest, error) {
	return &authv1.TokenRequest{
		Status: authv1.TokenRequestStatus{
			Token: testSAToken,
		},
	}, nil
}

func TestWorkloadIdentityFederation(t *testing.T) {
	tests := []*workloadIdentityFederationTest{
		{
			name: "successful kubernetes service account token federation",
			wifConfig: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name:      testConfigMapName,
					Key:       testConfigMapKey,
					Namespace: testNamespace,
				},
				ServiceAccountRef: esmeta.ServiceAccountSelector{
					Name:      testServiceAccount,
					Namespace: &testNamespace,
					Audiences: []string{testAudience},
				},
			},
			configMapData: map[string]string{
				testConfigMapKey: createValidK8sExternalAccountConfig(testAudience),
			},
			kubeObjects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testConfigMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{
						testConfigMapKey: createValidK8sExternalAccountConfig(testAudience),
					},
				},
			},
			genSAToken: func(c context.Context, s1 []string, s2, s3 string) (*authv1.TokenRequest, error) {
				return &authv1.TokenRequest{
					Status: authv1.TokenRequestStatus{
						Token: testSAToken,
					},
				}, nil
			},
			expectTokenSource: true,
		},
		{
			name: "successful AWS federation with security credentials",
			wifConfig: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name:      testConfigMapName,
					Key:       testConfigMapKey,
					Namespace: testNamespace,
				},
			},
			kubeObjects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testConfigMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{
						testConfigMapKey: createValidAWSExternalAccountConfig(testAudience),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws-creds",
						Namespace: testNamespace,
					},
					Data: map[string][]byte{
						awsAccessKeyIdKeyName:     []byte(testAwsAccessKey),
						awsSecretAccessKeyKeyName: []byte(testAwsSecretKey),
						awsSessionTokenKeyName:    []byte(testAwsSessionToken),
					},
				},
			},
			expectTokenSource: true,
		},
		{
			name: "external account creds configmap not present",
			wifConfig: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name:      testConfigMapName,
					Key:       testConfigMapKey,
					Namespace: testNamespace,
				},
			},
			kubeObjects: []client.Object{},
			expectError: `failed to fetch external acccount credentials configmap "external-secrets-tests/external-account-config": configmaps "external-account-config" not found`,
		},
		{
			name: "creds configmap has invalid type",
			wifConfig: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name:      testConfigMapName,
					Key:       testConfigMapKey,
					Namespace: testNamespace,
				},
			},
			kubeObjects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testConfigMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{
						testConfigMapKey: createInvalidTypeExternalAccountConfig(),
					},
				},
			},
			expectError: `invalid credentials: 'type' field is "service_account" (expected "external_account")`,
		},
		{
			name: "creds configmap has non-json data",
			wifConfig: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name:      testConfigMapName,
					Key:       testConfigMapKey,
					Namespace: testNamespace,
				},
			},
			kubeObjects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testConfigMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{
						testConfigMapKey: "invalid-json",
					},
				},
			},
			expectError: "invalid character 'i' looking for beginning of value",
		},
		{
			name: "non-existent AWS credentials secret",
			wifConfig: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name:      testConfigMapName,
					Key:       testConfigMapKey,
					Namespace: testNamespace,
				},
				AwsSecurityCredentials: &esv1.AwsCredentialsConfig{
					Region: testAwsRegion,
					AwsCredentialsSecretRef: &esv1.SecretReference{
						Name:      "non-existent-aws-creds",
						Namespace: testNamespace,
					},
				},
			},
			kubeObjects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testConfigMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{
						testConfigMapKey: createValidAWSExternalAccountConfig(testAudience),
					},
				},
			},
			expectError: `failed to fetch AwsSecurityCredentials secret "external-secrets-tests/non-existent-aws-creds": secrets "non-existent-aws-creds" not found`,
		},
		{
			name: "invalid AWS credentials - aws_access_key_id not provided",
			wifConfig: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name:      testConfigMapName,
					Key:       testConfigMapKey,
					Namespace: testNamespace,
				},
				AwsSecurityCredentials: &esv1.AwsCredentialsConfig{
					Region: testAwsRegion,
					AwsCredentialsSecretRef: &esv1.SecretReference{
						Name:      "aws-creds",
						Namespace: testNamespace,
					},
				},
			},
			kubeObjects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testConfigMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{
						testConfigMapKey: createValidAWSExternalAccountConfig(testAudience),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws-creds",
						Namespace: testNamespace,
					},
					Data: map[string][]byte{
						awsSecretAccessKeyKeyName: []byte(testAwsSecretKey),
					},
				},
			},
			expectError: "aws_access_key_id and aws_secret_access_key keys must be present in AwsSecurityCredentials secret",
		},
		{
			name: "credConfig is empty",
			wifConfig: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: nil,
			},
			expectError: "workloadIdentityFederation.credConfig must not be empty",
		},
		{
			name: "both credential_source in credConfig and AwsCredentialsConfig are set",
			wifConfig: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name:      testConfigMapName,
					Key:       testConfigMapKey,
					Namespace: testNamespace,
				},
				AwsSecurityCredentials: &esv1.AwsCredentialsConfig{
					Region: testAwsRegion,
					AwsCredentialsSecretRef: &esv1.SecretReference{
						Name:      "aws-creds",
						Namespace: testNamespace,
					},
				},
			},
			kubeObjects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testConfigMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{
						testConfigMapKey: createValidAWSExternalAccountConfig(testAudience),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws-creds",
						Namespace: testNamespace,
					},
					Data: map[string][]byte{
						awsAccessKeyIdKeyName:     []byte(testAwsAccessKey),
						awsSecretAccessKeyKeyName: []byte(testAwsSecretKey),
						awsSessionTokenKeyName:    []byte(testAwsSessionToken),
					},
				},
			},
			expectError: "invalid external_account config external-account-config: when credConfig has `credential_source` set, AwsSecurityCredentialsSupplier must be not configured",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := clientfake.NewClientBuilder().WithObjects(tc.kubeObjects...).Build()

			fakeSATG := &fakeSATokenGen{
				GenerateFunc: tc.genSAToken,
			}
			if tc.genSAToken == nil {
				fakeSATG.GenerateFunc = defaultSATokenGenerator
			}

			wif := &workloadIdentityFederation{
				kubeClient:       fakeClient,
				saTokenGenerator: fakeSATG,
				config:           tc.wifConfig,
				isClusterKind:    true,
				namespace:        testNamespace,
			}

			ts, err := wif.TokenSource(context.Background())
			if tc.expectError != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectError, err.Error())
				assert.Nil(t, ts)
			} else {
				assert.NoError(t, err)
				if tc.expectTokenSource {
					assert.NotNil(t, ts)
				}
			}
		})
	}
}

func TestValidateCredConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *externalaccount.Config
		wif         *esv1.GCPWorkloadIdentityFederation
		expectError string
	}{
		{
			name: "valid kubernetes provider config",
			config: &externalaccount.Config{
				Audience:                       testAudience,
				SubjectTokenType:               workloadIdentitySubjectTokenType,
				TokenURL:                       workloadIdentityTokenURL,
				ServiceAccountImpersonationURL: testServiceAccountImpersonationURL,
				CredentialSource: &externalaccount.CredentialSource{
					File: autoMountedServiceAccountTokenPath,
				},
			},
			wif: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{Name: testConfigMapName},
			},
			expectError: "",
		},
		{
			name: "valid AWS provider config",
			config: &externalaccount.Config{
				Audience:                       testAudience,
				SubjectTokenType:               workloadIdentitySubjectTokenType,
				TokenURL:                       workloadIdentityTokenURL,
				ServiceAccountImpersonationURL: testServiceAccountImpersonationURL,
				CredentialSource: &externalaccount.CredentialSource{
					EnvironmentID:         "aws1",
					URL:                   testAwsTokenIPV6URL,
					RegionURL:             testAwsRegionIPv6URL,
					IMDSv2SessionTokenURL: testAwsSessionTokenIPv6URL,
				},
			},
			wif: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{Name: testConfigMapName},
			},
			expectError: "",
		},
		{
			name: "invalid service account impersonation URL",
			config: &externalaccount.Config{
				Audience:                       testAudience,
				TokenURL:                       workloadIdentityTokenURL,
				ServiceAccountImpersonationURL: "https://invalid-url.com",
			},
			wif: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{Name: testConfigMapName},
			},
			expectError: `invalid external_account config external-account-config: service_account_impersonation_url "https://invalid-url.com" does not have expected value`,
		},
		{
			name: "invalid token URL",
			config: &externalaccount.Config{
				Audience:                       testAudience,
				TokenURL:                       "https://invalid-token-url.com",
				ServiceAccountImpersonationURL: testServiceAccountImpersonationURL,
			},
			wif: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{Name: testConfigMapName},
			},
			expectError: `invalid external_account config external-account-config: token_url "https://invalid-token-url.com" must match https://sts.googleapis.com/v1/token`,
		},
		{
			name: "invalid credential source file",
			config: &externalaccount.Config{
				Audience:                       testAudience,
				TokenURL:                       workloadIdentityTokenURL,
				ServiceAccountImpersonationURL: testServiceAccountImpersonationURL,
				CredentialSource: &externalaccount.CredentialSource{
					File: "/invalid/path/to/token",
				},
			},
			wif: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{Name: testConfigMapName},
			},
			expectError: `invalid external_account config external-account-config: credential_source.file "/invalid/path/to/token" is not a recognized path`,
		},
		{
			name: "executable is configured",
			config: &externalaccount.Config{
				Audience:                       testAudience,
				TokenURL:                       workloadIdentityTokenURL,
				ServiceAccountImpersonationURL: testServiceAccountImpersonationURL,
				CredentialSource: &externalaccount.CredentialSource{
					Executable: &externalaccount.ExecutableConfig{
						Command: "/usr/local/bin/token-issuer",
					},
				},
			},
			wif: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{Name: testConfigMapName},
			},
			expectError: "invalid external_account config external-account-config: credential_source.executable.command is not allowed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCredConfig(tc.config, tc.wif)
			if tc.expectError != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectError, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestK8sSATokenReader(t *testing.T) {
	r := &k8sSATokenReader{
		audience:         testAudience,
		subjectTokenType: workloadIdentitySubjectTokenType,
		saTokenGenerator: &fakeSATokenGen{
			GenerateFunc: defaultSATokenGenerator,
		},
		saAudience: []string{testAudience},
		serviceAccount: types.NamespacedName{
			Name:      testServiceAccount,
			Namespace: testNamespace,
		},
	}

	ctx := context.Background()

	// Test successful token generation
	token, err := r.SubjectToken(ctx, externalaccount.SupplierOptions{
		Audience:         testAudience,
		SubjectTokenType: workloadIdentitySubjectTokenType,
	})
	assert.NoError(t, err)
	assert.Equal(t, testSAToken, token)

	// Test invalid audience
	_, err = r.SubjectToken(ctx, externalaccount.SupplierOptions{
		Audience:         "invalid-audience",
		SubjectTokenType: workloadIdentitySubjectTokenType,
	})
	assert.Error(t, err)
	assert.Equal(t,
		`invalid subject token request, audience is invalid-audience(expected //iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/test-pool/providers/test-provider) and subject_token_type is urn:ietf:params:oauth:token-type:jwt(expected urn:ietf:params:oauth:token-type:jwt)`,
		err.Error())

	// Test invalid subject token type
	_, err = r.SubjectToken(ctx, externalaccount.SupplierOptions{
		Audience:         testAudience,
		SubjectTokenType: "invalid-type",
	})
	assert.Error(t, err)
	assert.Equal(t,
		`invalid subject token request, audience is //iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/test-pool/providers/test-provider(expected //iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/test-pool/providers/test-provider) and subject_token_type is invalid-type(expected urn:ietf:params:oauth:token-type:jwt)`,
		err.Error())
}

func TestAWSSecurityCredentialsReader(t *testing.T) {
	r := &awsSecurityCredentialsReader{
		region: testAwsRegion,
		awsSecurityCredentials: &externalaccount.AwsSecurityCredentials{
			AccessKeyID:     testAwsAccessKey,
			SecretAccessKey: testAwsSecretKey,
			SessionToken:    testAwsSessionToken,
		},
	}

	ctx := context.Background()
	options := externalaccount.SupplierOptions{}

	// Test region retrieval
	region, err := r.AwsRegion(ctx, options)
	assert.NoError(t, err)
	assert.Equal(t, testAwsRegion, region)

	// Test credentials retrieval
	creds, err := r.AwsSecurityCredentials(ctx, options)
	assert.NoError(t, err)
	assert.Equal(t, testAwsAccessKey, creds.AccessKeyID)
	assert.Equal(t, testAwsSecretKey, creds.SecretAccessKey)
	assert.Equal(t, testAwsSessionToken, creds.SessionToken)
}

func TestNewWorkloadIdentityFederation(t *testing.T) {
	tests := []struct {
		name        string
		config      *esv1.GCPWorkloadIdentityFederation
		expectError bool
	}{
		{
			name: "successful creation",
			config: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name:      testConfigMapName,
					Namespace: testNamespace,
				},
			},
			expectError: false,
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := clientfake.NewClientBuilder().Build()

			wif, err := newWorkloadIdentityFederation(fakeClient, tc.config, false, testNamespace)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, wif)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, wif)
				assert.Equal(t, fakeClient, wif.kubeClient)
				assert.Equal(t, tc.config, wif.config)
				assert.NotNil(t, wif.saTokenGenerator)
			}
		})
	}
}

func TestReadCredConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *esv1.GCPWorkloadIdentityFederation
		kubeObjects []client.Object
		expectError string
	}{
		{
			name: "cred configmap key is not configured",
			config: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name:      testConfigMapName,
					Namespace: testNamespace,
				},
			},
			kubeObjects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testConfigMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{
						"auto-detected-key": createValidK8sExternalAccountConfig(testAudience),
					},
				},
			},
			expectError: "",
		},
		{
			name: "cred configmap has empty data",
			config: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name:      testConfigMapName,
					Namespace: testNamespace,
				},
			},
			kubeObjects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testConfigMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{},
				},
			},
			expectError: `no external acccount credentials found in "external-account-config" configmap`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := clientfake.NewClientBuilder().WithObjects(tc.kubeObjects...).Build()

			wif := &workloadIdentityFederation{
				kubeClient:       fakeClient,
				saTokenGenerator: &fakeSATokenGen{GenerateFunc: defaultSATokenGenerator},
				config:           tc.config,
				isClusterKind:    false,
				namespace:        testNamespace,
			}

			ctx := context.Background()
			_, err := wif.readCredConfig(ctx)

			if tc.expectError != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectError, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateExternalAccountConfig(t *testing.T) {
	wif := &esv1.GCPWorkloadIdentityFederation{
		CredConfig: &esv1.ConfigMapReference{
			Name:      testConfigMapName,
			Namespace: testNamespace,
		},
		AwsSecurityCredentials: &esv1.AwsCredentialsConfig{
			Region: testAwsRegion,
			AwsCredentialsSecretRef: &esv1.SecretReference{
				Name:      "aws-creds",
				Namespace: testNamespace,
			},
		},
		Audience: testAudience,
	}

	kubeObjects := []client.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aws-creds",
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				awsAccessKeyIdKeyName:     []byte(testAwsAccessKey),
				awsSecretAccessKeyKeyName: []byte(testAwsSecretKey),
				awsSessionTokenKeyName:    []byte(testAwsSessionToken),
			},
		},
	}

	fakeClient := clientfake.NewClientBuilder().WithObjects(kubeObjects...).Build()

	wifInstance := &workloadIdentityFederation{
		kubeClient:       fakeClient,
		saTokenGenerator: &fakeSATokenGen{GenerateFunc: defaultSATokenGenerator},
		config:           wif,
		isClusterKind:    false,
		namespace:        testNamespace,
	}

	ctx := context.Background()
	credFile := &credentialsFile{
		Type:                           externalAccountCredentialType,
		Audience:                       testAudience,
		SubjectTokenType:               workloadIdentitySubjectTokenType,
		TokenURLExternal:               workloadIdentityTokenURL,
		ServiceAccountImpersonationURL: testServiceAccountImpersonationURL,
	}

	config, err := wifInstance.generateExternalAccountConfig(ctx, credFile)
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.NotNil(t, config.AwsSecurityCredentialsSupplier)
	assert.Equal(t, testAudience, config.Audience)
}
