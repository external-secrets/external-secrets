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

package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type fakeSTSAssumeRoleClient struct {
	input *sts.AssumeRoleInput
	err   error
}

func (f *fakeSTSAssumeRoleClient) AssumeRole(_ context.Context, input *sts.AssumeRoleInput, _ ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	f.input = input
	if f.err != nil {
		return nil, f.err
	}
	return &sts.AssumeRoleOutput{}, nil
}

func TestProviderAddressInNamespace(t *testing.T) {
	t.Parallel()

	got := frameworkv2.ProviderAddressInNamespace("aws", "aws-irsa-system")
	if got != "provider-aws.aws-irsa-system.svc:8080" {
		t.Fatalf("unexpected address: %s", got)
	}
}

func TestStaticAWSAuthUsesSessionTokenSelector(t *testing.T) {
	t.Parallel()

	auth := staticAWSAuth("aws-creds")
	if auth.SecretRef == nil || auth.SecretRef.SessionToken == nil {
		t.Fatal("expected session token selector to be preserved")
	}
	if auth.SecretRef.SessionToken.Name != "aws-creds" || auth.SecretRef.SessionToken.Key != "st" {
		t.Fatalf("unexpected session token selector: %+v", auth.SecretRef.SessionToken)
	}
}

func TestSecretsManagerConfigForExternalID(t *testing.T) {
	t.Parallel()

	provider := newSecretsManagerV2StoreProvider("sm-extid-credentials", awsAccessConfig{
		Region: "eu-west-1",
		Role:   awscommon.IAMRoleExternalID,
	}, awsAuthProfileExternalID, nil)
	if provider.AWS == nil {
		t.Fatal("expected AWS provider config")
	}
	if provider.AWS.ExternalID != awscommon.IAMTrustedExternalID {
		t.Fatalf("expected external ID %q, got %q", awscommon.IAMTrustedExternalID, provider.AWS.ExternalID)
	}
}

func TestSecretsManagerConfigForSessionTags(t *testing.T) {
	t.Parallel()

	provider := newSecretsManagerV2StoreProvider("sm-tags-credentials", awsAccessConfig{
		Region: "eu-west-1",
		Role:   awscommon.IAMRoleSessionTags,
	}, awsAuthProfileSessionTags, nil)
	if provider.AWS == nil {
		t.Fatal("expected AWS provider config")
	}
	if len(provider.AWS.SessionTags) != 1 {
		t.Fatalf("expected one session tag, got %d", len(provider.AWS.SessionTags))
	}
	if provider.AWS.SessionTags[0].Key != "namespace" || provider.AWS.SessionTags[0].Value != "e2e-test" {
		t.Fatalf("unexpected session tags: %+v", provider.AWS.SessionTags)
	}
}

func TestProviderConfigNamespaceForManifestScope(t *testing.T) {
	t.Parallel()

	if got := awscommon.ProviderConfigNamespace(esv1.AuthenticationScopeManifestNamespace, "provider-ns", "workload-ns"); got != "workload-ns" {
		t.Fatalf("expected workload namespace, got %q", got)
	}
}

func TestProviderConfigNamespaceForProviderScope(t *testing.T) {
	t.Parallel()

	if got := awscommon.ProviderConfigNamespace(esv1.AuthenticationScopeProviderNamespace, "provider-ns", "workload-ns"); got != "provider-ns" {
		t.Fatalf("expected provider namespace, got %q", got)
	}
}

func TestProviderReferenceNamespaceForManifestScope(t *testing.T) {
	t.Parallel()

	if got := awscommon.ProviderReferenceNamespace(esv1.AuthenticationScopeManifestNamespace, "provider-ns"); got != "" {
		t.Fatalf("expected empty provider reference namespace, got %q", got)
	}
}

func TestProviderReferenceNamespaceForProviderScope(t *testing.T) {
	t.Parallel()

	if got := awscommon.ProviderReferenceNamespace(esv1.AuthenticationScopeProviderNamespace, "provider-ns"); got != "provider-ns" {
		t.Fatalf("expected provider namespace, got %q", got)
	}
}

func TestSecretValueStringUsesSecretString(t *testing.T) {
	t.Parallel()

	got := secretValueString(&awssm.GetSecretValueOutput{
		SecretString: aws.String(`{"value":"from-string"}`),
		SecretBinary: []byte(`{"value":"from-binary"}`),
	})
	if got != `{"value":"from-string"}` {
		t.Fatalf("expected SecretString payload, got %q", got)
	}
}

func TestSecretValueStringFallsBackToSecretBinary(t *testing.T) {
	t.Parallel()

	got := secretValueString(&awssm.GetSecretValueOutput{
		SecretBinary: []byte(`{"value":"from-binary"}`),
	})
	if got != `{"value":"from-binary"}` {
		t.Fatalf("expected SecretBinary payload, got %q", got)
	}
}

func TestSecretReadErrorIndicatesAbsenceRecognizesMarkedForDeletion(t *testing.T) {
	t.Parallel()

	err := errors.New("InvalidRequestException: You can't perform this operation on the secret because it was marked for deletion.")
	if !secretReadErrorIndicatesAbsence(err) {
		t.Fatal("expected marked-for-deletion error to be treated as absence")
	}
}

func TestProbeAssumeRoleAccessBuildsExternalIDRequest(t *testing.T) {
	t.Parallel()

	client := &fakeSTSAssumeRoleClient{}
	access := awsAccessConfig{
		Role: awscommon.IAMRoleExternalID,
	}
	if err := probeAssumeRoleAccess(context.Background(), client, access, awsAuthProfileExternalID); err != nil {
		t.Fatalf("probeAssumeRoleAccess() error = %v", err)
	}
	if client.input == nil {
		t.Fatal("expected AssumeRole input to be recorded")
	}
	if got := aws.ToString(client.input.RoleArn); got != awscommon.IAMRoleExternalID {
		t.Fatalf("expected role ARN %q, got %q", awscommon.IAMRoleExternalID, got)
	}
	if got := aws.ToString(client.input.ExternalId); got != awscommon.IAMTrustedExternalID {
		t.Fatalf("expected external ID %q, got %q", awscommon.IAMTrustedExternalID, got)
	}
}

func TestProbeAssumeRoleAccessBuildsSessionTagsRequest(t *testing.T) {
	t.Parallel()

	client := &fakeSTSAssumeRoleClient{}
	access := awsAccessConfig{
		Role: awscommon.IAMRoleSessionTags,
	}
	if err := probeAssumeRoleAccess(context.Background(), client, access, awsAuthProfileSessionTags); err != nil {
		t.Fatalf("probeAssumeRoleAccess() error = %v", err)
	}
	if client.input == nil {
		t.Fatal("expected AssumeRole input to be recorded")
	}
	if got := aws.ToString(client.input.RoleArn); got != awscommon.IAMRoleSessionTags {
		t.Fatalf("expected role ARN %q, got %q", awscommon.IAMRoleSessionTags, got)
	}
	if len(client.input.Tags) != 1 {
		t.Fatalf("expected one session tag, got %d", len(client.input.Tags))
	}
	tag := client.input.Tags[0]
	if aws.ToString(tag.Key) != "namespace" || aws.ToString(tag.Value) != "e2e-test" {
		t.Fatalf("unexpected session tag: %+v", tag)
	}
}

func TestIsAssumeRoleAccessDeniedRecognizesSTSAccessDeniedErrors(t *testing.T) {
	t.Parallel()

	err := errors.New("api error AccessDenied: User is not authorized to perform: sts:TagSession")
	if !isAssumeRoleAccessDenied(err) {
		t.Fatal("expected sts access denied error to be recognized")
	}
}
