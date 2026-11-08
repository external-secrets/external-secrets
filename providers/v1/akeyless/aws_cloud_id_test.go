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

package akeyless

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	testRoleARN = "arn:aws:iam::123456789012:role/team-a"
	testSAToken = "sa-jwt"
)

type fakeSTS struct {
	input *sts.AssumeRoleWithWebIdentityInput
}

func (f *fakeSTS) AssumeRoleWithWebIdentity(_ context.Context, in *sts.AssumeRoleWithWebIdentityInput, _ ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error) {
	f.input = in
	return &sts.AssumeRoleWithWebIdentityOutput{
		Credentials: &ststypes.Credentials{
			AccessKeyId:     aws.String("ASSUMEDAKID"),
			SecretAccessKey: aws.String("ASSUMEDSECRET"),
			SessionToken:    aws.String("ASSUMEDSESSION"),
			Expiration:      aws.Time(time.Now().Add(time.Hour)),
		},
	}, nil
}

type tokenRequestCapture struct {
	namespace string
	name      string
	audiences []string
}

type awsTestEnv struct {
	base    *akeylessBase
	sts     *fakeSTS
	tokenRq *tokenRequestCapture
}

func newAWSTestEnv(t *testing.T, storeKind, namespace string, objs ...*corev1.ServiceAccount) *awsTestEnv {
	t.Helper()

	builder := clientfake.NewClientBuilder()
	for _, o := range objs {
		builder = builder.WithObjects(o)
	}

	capture := &tokenRequestCapture{}
	clientset := k8sfake.NewClientset()
	clientset.PrependReactor("create", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
		create, ok := action.(k8stesting.CreateActionImpl)
		if !ok || create.GetSubresource() != "token" {
			return false, nil, nil
		}
		tr := create.GetObject().(*authenticationv1.TokenRequest)
		capture.namespace = create.GetNamespace()
		capture.name = create.Name
		capture.audiences = tr.Spec.Audiences
		return true, &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: testSAToken}}, nil
	})

	fsts := &fakeSTS{}
	origLoad, origSTS := loadAWSConfig, newAWSSTSClient
	loadAWSConfig = func(context.Context) (aws.Config, error) { return aws.Config{Region: "us-west-2"}, nil }
	newAWSSTSClient = func(aws.Config) stscreds.AssumeRoleWithWebIdentityAPIClient { return fsts }
	t.Cleanup(func() { loadAWSConfig, newAWSSTSClient = origLoad, origSTS })

	return &awsTestEnv{
		base: &akeylessBase{
			kube:      builder.Build(),
			corev1:    clientset.CoreV1(),
			namespace: namespace,
			storeKind: storeKind,
		},
		sts:     fsts,
		tokenRq: capture,
	}
}

func irsaServiceAccount(name, namespace string, annotations map[string]string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{Name: name, Namespace: namespace, Annotations: annotations}
}

func decodeAWSCloudIDHeaders(t *testing.T, cloudID string) (string, http.Header) {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(cloudID)
	require.NoError(t, err)
	var payload map[string]string
	require.NoError(t, json.Unmarshal(raw, &payload))
	u, err := base64.StdEncoding.DecodeString(payload["sts_request_url"])
	require.NoError(t, err)
	h, err := base64.StdEncoding.DecodeString(payload["sts_request_headers"])
	require.NoError(t, err)
	var headers http.Header
	require.NoError(t, json.Unmarshal(h, &headers))
	return string(u), headers
}

func TestGetAWSCloudIDWithoutServiceAccountRefUsesControllerIdentity(t *testing.T) {
	orig := defaultAWSCloudID
	defaultAWSCloudID = func() (string, error) { return "controller-identity", nil }
	t.Cleanup(func() { defaultAWSCloudID = orig })

	env := newAWSTestEnv(t, esv1.SecretStoreKind, "team-a")

	for _, auth := range []*esv1.AkeylessAuth{nil, {}} {
		got, err := env.base.getCloudID(context.Background(), "aws_iam", "", auth)
		require.NoError(t, err)
		require.Equal(t, "controller-identity", got)
	}
	require.Nil(t, env.sts.input, "STS must not be called without serviceAccountRef")
	require.Empty(t, env.tokenRq.name, "TokenRequest must not be issued without serviceAccountRef")
}

func TestGetAWSCloudIDFromServiceAccount(t *testing.T) {
	sa := irsaServiceAccount("akeyless-irsa", "team-a", map[string]string{annotationAWSRoleARN: testRoleARN})
	env := newAWSTestEnv(t, esv1.SecretStoreKind, "team-a", sa)

	auth := &esv1.AkeylessAuth{ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-irsa"}}
	cloudID, err := env.base.getCloudID(context.Background(), "aws_iam", "", auth)
	require.NoError(t, err)

	require.Equal(t, "team-a", env.tokenRq.namespace)
	require.Equal(t, "akeyless-irsa", env.tokenRq.name)
	require.Equal(t, []string{awsDefaultAudience}, env.tokenRq.audiences)

	require.NotNil(t, env.sts.input)
	require.Equal(t, testRoleARN, aws.ToString(env.sts.input.RoleArn))
	require.Equal(t, testSAToken, aws.ToString(env.sts.input.WebIdentityToken))
	require.Equal(t, awsRoleSessionName, aws.ToString(env.sts.input.RoleSessionName))

	url, headers := decodeAWSCloudIDHeaders(t, cloudID)
	require.Equal(t, "https://sts.us-west-2.amazonaws.com/", url)
	require.Equal(t, "ASSUMEDSESSION", headers.Get("X-Amz-Security-Token"))
	require.Contains(t, headers.Get("Authorization"), "Credential=ASSUMEDAKID/")
}

func TestGetAWSCloudIDDefaultsRegion(t *testing.T) {
	sa := irsaServiceAccount("akeyless-irsa", "team-a", map[string]string{annotationAWSRoleARN: testRoleARN})
	env := newAWSTestEnv(t, esv1.SecretStoreKind, "team-a", sa)
	var stsRegion string
	loadAWSConfig = func(context.Context) (aws.Config, error) { return aws.Config{}, nil }
	newAWSSTSClient = func(cfg aws.Config) stscreds.AssumeRoleWithWebIdentityAPIClient {
		stsRegion = cfg.Region
		return env.sts
	}

	cloudID, err := env.base.getAWSCloudID(context.Background(), &esv1.AkeylessAuth{
		ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-irsa"},
	})
	require.NoError(t, err)
	require.Equal(t, awsDefaultRegion, stsRegion)
	url, _ := decodeAWSCloudIDHeaders(t, cloudID)
	require.Equal(t, "https://sts.us-east-1.amazonaws.com/", url)
}

func TestGetAWSCloudIDAudienceAnnotation(t *testing.T) {
	sa := irsaServiceAccount("akeyless-irsa", "team-a", map[string]string{
		annotationAWSRoleARN:  testRoleARN,
		annotationAWSAudience: "custom-aud",
	})
	env := newAWSTestEnv(t, esv1.SecretStoreKind, "team-a", sa)

	ref := &esmeta.ServiceAccountSelector{Name: "akeyless-irsa", Audiences: []string{"extra"}}
	_, err := env.base.getAWSCloudID(context.Background(), &esv1.AkeylessAuth{ServiceAccountRef: ref})
	require.NoError(t, err)
	require.Equal(t, []string{"extra", "custom-aud"}, env.tokenRq.audiences)
}

func TestGetAWSCloudIDNamespaceScoping(t *testing.T) {
	other := "team-b"
	saA := irsaServiceAccount("akeyless-irsa", "team-a", map[string]string{annotationAWSRoleARN: testRoleARN})
	saB := irsaServiceAccount("akeyless-irsa", other, map[string]string{annotationAWSRoleARN: "arn:aws:iam::123456789012:role/team-b"})
	ref := &esmeta.ServiceAccountSelector{Name: "akeyless-irsa", Namespace: &other}

	t.Run("SecretStore ignores ref namespace", func(t *testing.T) {
		env := newAWSTestEnv(t, esv1.SecretStoreKind, "team-a", saA, saB)
		_, err := env.base.getAWSCloudID(context.Background(), &esv1.AkeylessAuth{ServiceAccountRef: ref})
		require.NoError(t, err)
		require.Equal(t, "team-a", env.tokenRq.namespace)
		require.Equal(t, testRoleARN, aws.ToString(env.sts.input.RoleArn))
	})

	t.Run("ClusterSecretStore honors ref namespace", func(t *testing.T) {
		env := newAWSTestEnv(t, esv1.ClusterSecretStoreKind, "team-a", saA, saB)
		_, err := env.base.getAWSCloudID(context.Background(), &esv1.AkeylessAuth{ServiceAccountRef: ref})
		require.NoError(t, err)
		require.Equal(t, other, env.tokenRq.namespace)
		require.Equal(t, "arn:aws:iam::123456789012:role/team-b", aws.ToString(env.sts.input.RoleArn))
	})
}

func TestGetAWSCloudIDErrors(t *testing.T) {
	t.Run("missing role-arn annotation", func(t *testing.T) {
		sa := irsaServiceAccount("akeyless-irsa", "team-a", nil)
		env := newAWSTestEnv(t, esv1.SecretStoreKind, "team-a", sa)
		_, err := env.base.getAWSCloudID(context.Background(), &esv1.AkeylessAuth{
			ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-irsa"},
		})
		require.ErrorContains(t, err, annotationAWSRoleARN)
		require.Nil(t, env.sts.input)
	})

	t.Run("missing service account", func(t *testing.T) {
		env := newAWSTestEnv(t, esv1.SecretStoreKind, "team-a")
		_, err := env.base.getAWSCloudID(context.Background(), &esv1.AkeylessAuth{
			ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "missing"},
		})
		require.ErrorContains(t, err, `failed to get service account "missing"`)
	})
}
