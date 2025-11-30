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

package certificatemanager

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	acmtypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	fakecm "github.com/external-secrets/external-secrets/providers/v1/aws/certificatemanager/fake"
)

const (
	testCertificateARN = "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012"
)

var (
	testCertificate = "-----BEGIN CERTIFICATE-----\nMIICljCCAX4CCQDhsYy..."
	testPrivateKey  = "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG..."
	testChain       = "-----BEGIN CERTIFICATE-----\nMIICljCCAX4CCQDhsYy...\n-----END CERTIFICATE-----"
	testCertWithAll = testCertificate + "\n" + testChain + "\n" + testPrivateKey + "\n"
)

type certificateManagerTestCase struct {
	fakeClient      *fakecm.Client
	apiInput        *acm.ExportCertificateInput
	apiOutput       *acm.ExportCertificateOutput
	describeCertOut *acm.DescribeCertificateOutput
	describeCertErr error
	remoteRef       *esv1.ExternalSecretDataRemoteRef
	apiErr          error
	expectError     string
	expectedSecret  string
}

func makeValidCertificateManagerTestCase() *certificateManagerTestCase {
	return &certificateManagerTestCase{
		fakeClient:      &fakecm.Client{},
		apiInput:        makeValidACMInput(),
		apiOutput:       makeValidACMOutput(),
		describeCertOut: makeValidDescribeCertOutput(),
		describeCertErr: nil,
		remoteRef:       makeValidCMRemoteRef(),
		apiErr:          nil,
		expectError:     "",
		expectedSecret:  "",
	}
}

func makeValidACMInput() *acm.ExportCertificateInput {
	return &acm.ExportCertificateInput{
		CertificateArn: aws.String(testCertificateARN),
	}
}

func makeValidACMOutput() *acm.ExportCertificateOutput {
	return &acm.ExportCertificateOutput{
		Certificate:      aws.String(testCertificate),
		CertificateChain: aws.String(testChain),
		PrivateKey:       aws.String(testPrivateKey),
	}
}

func makeValidDescribeCertOutput() *acm.DescribeCertificateOutput {
	return &acm.DescribeCertificateOutput{
		Certificate: &acmtypes.CertificateDetail{
			CertificateArn: aws.String(testCertificateARN),
		},
	}
}

func makeValidCMRemoteRef() *esv1.ExternalSecretDataRemoteRef {
	return &esv1.ExternalSecretDataRemoteRef{
		Key: testCertificateARN,
	}
}

func makeValidCertificateManagerTestCaseCustom(tweaks ...func(tc *certificateManagerTestCase)) *certificateManagerTestCase {
	tc := makeValidCertificateManagerTestCase()
	for _, fn := range tweaks {
		fn(tc)
	}
	tc.fakeClient.WithValue(tc.apiInput, tc.apiOutput, tc.apiErr)
	return tc
}

func TestACMResolver(t *testing.T) {
	endpointEnvKey := ACMEndpointEnv
	endpointURL := "http://acm.foo"

	t.Setenv(endpointEnvKey, endpointURL)

	f, err := customEndpointResolver{}.ResolveEndpoint(context.Background(), acm.EndpointParameters{})

	assert.Nil(t, err)
	assert.Equal(t, endpointURL, f.URI.String())
}

func TestGetSecret(t *testing.T) {
	setValidCertificate := func(tc *certificateManagerTestCase) {
		tc.apiOutput.Certificate = aws.String(testCertificate)
		tc.apiOutput.CertificateChain = aws.String(testChain)
		tc.apiOutput.PrivateKey = aws.String(testPrivateKey)
		tc.expectedSecret = testCertWithAll
	}

	setCertificateOnly := func(tc *certificateManagerTestCase) {
		tc.apiOutput.Certificate = aws.String(testCertificate)
		tc.apiOutput.CertificateChain = nil
		tc.apiOutput.PrivateKey = nil
		tc.expectedSecret = testCertificate + "\n"
	}

	setCertificateAndChain := func(tc *certificateManagerTestCase) {
		tc.apiOutput.Certificate = aws.String(testCertificate)
		tc.apiOutput.CertificateChain = aws.String(testChain)
		tc.apiOutput.PrivateKey = nil
		tc.expectedSecret = testCertificate + "\n" + testChain + "\n"
	}

	setNoDataReturned := func(tc *certificateManagerTestCase) {
		tc.apiOutput.Certificate = nil
		tc.apiOutput.CertificateChain = nil
		tc.apiOutput.PrivateKey = nil
		tc.expectError = "no data returned"
	}

	setExportCertFail := func(tc *certificateManagerTestCase) {
		tc.apiErr = errors.New("certificate not exportable")
		tc.expectError = "certificate not exportable"
	}

	setDescribeCertFail := func(tc *certificateManagerTestCase) {
		tc.describeCertErr = errors.New("certificate not found")
		tc.expectError = "certificate not found"
	}
	setEmptyARN := func(tc *certificateManagerTestCase) {
		tc.remoteRef.Key = ""
		tc.expectError = "certificate ARN must be specified in remoteRef.key"
	}

	successCases := []*certificateManagerTestCase{
		makeValidCertificateManagerTestCaseCustom(setValidCertificate),
		makeValidCertificateManagerTestCaseCustom(setCertificateOnly),
		makeValidCertificateManagerTestCaseCustom(setCertificateAndChain),
		makeValidCertificateManagerTestCaseCustom(setNoDataReturned),
		makeValidCertificateManagerTestCaseCustom(setExportCertFail),
		makeValidCertificateManagerTestCaseCustom(setDescribeCertFail),
		makeValidCertificateManagerTestCaseCustom(setEmptyARN),
	}

	for k, tc := range successCases {
		cm := CertificateManager{
			client: tc.fakeClient,
		}
		tc.fakeClient.DescribeCertificateFn = fakecm.NewDescribeCertificateFn(tc.describeCertOut, tc.describeCertErr)
		out, err := cm.GetSecret(context.Background(), *tc.remoteRef)

		if !errorContains(err, tc.expectError) {
			t.Errorf("[%d] unexpected error: %v, expected: '%s'", k, err, tc.expectError)
		}
		if !cmp.Equal(string(out), tc.expectedSecret) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, tc.expectedSecret, string(out))
		}
	}
}

func TestGetSecretMap(t *testing.T) {
	cm := CertificateManager{
		client: &fakecm.Client{},
	}

	_, err := cm.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "GetSecretMap is not supported")
}

func TestGetAllSecrets(t *testing.T) {
	cm := CertificateManager{
		client: &fakecm.Client{},
	}

	_, err := cm.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "GetAllSecrets is not supported")
}

func TestSecretExists(t *testing.T) {
	cm := CertificateManager{
		client: &fakecm.Client{},
	}

	_, err := cm.SecretExists(context.Background(), &fakePushSecretRemoteRef{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "SecretExists is not supported")
}

func TestPushSecret(t *testing.T) {
	cm := CertificateManager{
		client: &fakecm.Client{},
	}

	err := cm.PushSecret(context.Background(), nil, &fakePushSecretData{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "PushSecret is not supported")
}

func TestDeleteSecret(t *testing.T) {
	cm := CertificateManager{
		client: &fakecm.Client{},
	}

	err := cm.DeleteSecret(context.Background(), &fakePushSecretRemoteRef{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "DeleteSecret is not supported")
}

func TestValidate(t *testing.T) {
	cm := CertificateManager{
		referentAuth: true,
	}
	result, err := cm.Validate()
	assert.Nil(t, err)
	assert.Equal(t, esv1.ValidationResultUnknown, result)

	cfg := &aws.Config{
		Credentials: mockCredentialsProvider{},
	}
	cm = CertificateManager{
		cfg:          cfg,
		referentAuth: false,
	}
	result, err = cm.Validate()
	assert.Nil(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)

	cfg = &aws.Config{
		Credentials: mockCredentialsProviderError{},
	}
	cm = CertificateManager{
		cfg:          cfg,
		referentAuth: false,
	}
	result, err = cm.Validate()
	assert.NotNil(t, err)
	assert.Equal(t, esv1.ValidationResultError, result)
}

func TestClose(t *testing.T) {
	cm := CertificateManager{
		client: &fakecm.Client{},
	}

	err := cm.Close(context.Background())
	assert.Nil(t, err)
}

func TestNew(t *testing.T) {
	cfg := &aws.Config{}
	cm, err := New(context.Background(), cfg, "test-prefix", false)

	require.NoError(t, err)
	assert.NotNil(t, cm)
	assert.Equal(t, "test-prefix", cm.prefix)
	assert.False(t, cm.referentAuth)
	assert.NotNil(t, cm.client)
}

func errorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

// mockCredentialsProvider is a mock implementation of aws.CredentialsProvider.
type mockCredentialsProvider struct{}

func (m mockCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     "mock-access-key",
		SecretAccessKey: "mock-secret-key",
	}, nil
}

type mockCredentialsProviderError struct{}

func (m mockCredentialsProviderError) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{}, errors.New("credentials not available")
}

type fakePushSecretRemoteRef struct{}

func (f *fakePushSecretRemoteRef) GetRemoteKey() string {
	return "fake-remote-key"
}

func (f *fakePushSecretRemoteRef) GetProperty() string {
	return "fake-property"
}

type fakePushSecretData struct{}

func (f *fakePushSecretData) GetRemoteKey() string {
	return "fake-remote-key"
}

func (f *fakePushSecretData) GetSecretKey() string {
	return "fake-secret-key"
}

func (f *fakePushSecretData) GetProperty() string {
	return "fake-property"
}

func (f *fakePushSecretData) GetMetadata() *apiextensionsv1.JSON {
	return nil
}
