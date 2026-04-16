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
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	awsv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/aws/v2alpha1"
)

type fakeV2STSAssumeRoleClient struct {
	input *sts.AssumeRoleInput
	err   error
}

func (f *fakeV2STSAssumeRoleClient) AssumeRole(_ context.Context, input *sts.AssumeRoleInput, _ ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	f.input = input
	if f.err != nil {
		return nil, f.err
	}
	return &sts.AssumeRoleOutput{}, nil
}

func TestNewParameterStoreV2ConfigUsesStaticSessionTokenSelector(t *testing.T) {
	t.Parallel()

	cfg := newParameterStoreV2Config("ns", "ps-static", awsV2AccessConfig{
		Region: "eu-central-1",
	})
	if cfg.TypeMeta.Kind != awsv2alpha1.ParameterStoreKind {
		t.Fatalf("expected kind %q, got %q", awsv2alpha1.ParameterStoreKind, cfg.TypeMeta.Kind)
	}
	if cfg.Spec.Auth.SecretRef == nil || cfg.Spec.Auth.SecretRef.SessionToken == nil {
		t.Fatal("expected session token selector to be configured for static auth")
	}
	if cfg.Spec.Auth.SecretRef.SessionToken.Name != "ps-static-credentials" || cfg.Spec.Auth.SecretRef.SessionToken.Key != "st" {
		t.Fatalf("unexpected session token selector: %+v", cfg.Spec.Auth.SecretRef.SessionToken)
	}
}

func TestParameterStoreConfigForExternalID(t *testing.T) {
	t.Parallel()

	cfg := newParameterStoreV2Config("ns", "ps-extid", awsV2AccessConfig{
		Region: "eu-west-1",
		Role:   awscommon.IAMRoleExternalID,
	}, awsV2AuthProfileExternalID)

	if cfg.Spec.ExternalID != awscommon.IAMTrustedExternalID {
		t.Fatalf("expected external ID %q, got %q", awscommon.IAMTrustedExternalID, cfg.Spec.ExternalID)
	}
	if cfg.Spec.Role != awscommon.IAMRoleExternalID {
		t.Fatalf("expected role %q, got %q", awscommon.IAMRoleExternalID, cfg.Spec.Role)
	}
}

func TestParameterStoreConfigForExternalIDDefaultsRoleWhenUnset(t *testing.T) {
	t.Parallel()

	cfg := newParameterStoreV2Config("ns", "ps-extid-default-role", awsV2AccessConfig{
		Region: "eu-west-1",
	}, awsV2AuthProfileExternalID)

	if cfg.Spec.Role != awscommon.IAMRoleExternalID {
		t.Fatalf("expected default role %q, got %q", awscommon.IAMRoleExternalID, cfg.Spec.Role)
	}
}

func TestParameterStoreConfigForSessionTags(t *testing.T) {
	t.Parallel()

	cfg := newParameterStoreV2Config("ns", "ps-tags", awsV2AccessConfig{
		Region: "eu-west-1",
		Role:   awscommon.IAMRoleSessionTags,
	}, awsV2AuthProfileSessionTags)

	if len(cfg.Spec.SessionTags) != 1 {
		t.Fatalf("expected one session tag, got %d", len(cfg.Spec.SessionTags))
	}
	if cfg.Spec.SessionTags[0].Key != "namespace" || cfg.Spec.SessionTags[0].Value != "e2e-test" {
		t.Fatalf("unexpected session tags: %+v", cfg.Spec.SessionTags)
	}
}

func TestParameterStoreConfigForSessionTagsDefaultsRoleWhenUnset(t *testing.T) {
	t.Parallel()

	cfg := newParameterStoreV2Config("ns", "ps-sess-default-role", awsV2AccessConfig{
		Region: "eu-west-1",
	}, awsV2AuthProfileSessionTags)

	if cfg.Spec.Role != awscommon.IAMRoleSessionTags {
		t.Fatalf("expected default role %q, got %q", awscommon.IAMRoleSessionTags, cfg.Spec.Role)
	}
}

func TestParameterStoreConfigForReferencedIRSA(t *testing.T) {
	t.Parallel()

	cfg := newParameterStoreV2Config("ns", "ps-irsa", awsV2AccessConfig{
		Region:      "eu-west-1",
		SAName:      "irsa-sa",
		SANamespace: "irsa-ns",
	}, awsV2AuthProfileReferencedIRSA)

	if cfg.Spec.Auth.JWTAuth == nil || cfg.Spec.Auth.JWTAuth.ServiceAccountRef == nil {
		t.Fatal("expected JWT service account reference")
	}
	ref := cfg.Spec.Auth.JWTAuth.ServiceAccountRef
	if ref.Name != "irsa-sa" {
		t.Fatalf("expected service account name %q, got %q", "irsa-sa", ref.Name)
	}
	if ref.Namespace == nil || *ref.Namespace != "irsa-ns" {
		t.Fatalf("expected service account namespace %q, got %v", "irsa-ns", ref.Namespace)
	}
	if cfg.Spec.Auth.SecretRef != nil {
		t.Fatalf("expected referenced IRSA auth to avoid secretRef, got %+v", cfg.Spec.Auth.SecretRef)
	}
}

func TestProbeAssumeRoleAccessBuildsExternalIDRequest(t *testing.T) {
	t.Parallel()

	client := &fakeV2STSAssumeRoleClient{}
	access := awsV2AccessConfig{
		Role: awscommon.IAMRoleExternalID,
	}
	if err := probeAssumeRoleAccess(context.Background(), client, access, awsV2AuthProfileExternalID); err != nil {
		t.Fatalf("probeAssumeRoleAccess() error = %v", err)
	}
	if client.input == nil {
		t.Fatal("expected AssumeRole input to be recorded")
	}
	if got := aws.ToString(client.input.RoleArn); got != awscommon.IAMRoleExternalID {
		t.Fatalf("expected role ARN %q, got %q", awscommon.IAMRoleExternalID, got)
	}
	if got := aws.ToString(client.input.RoleSessionName); got != assumeRoleSessionName {
		t.Fatalf("expected role session name %q, got %q", assumeRoleSessionName, got)
	}
	if got := aws.ToString(client.input.ExternalId); got != awscommon.IAMTrustedExternalID {
		t.Fatalf("expected external ID %q, got %q", awscommon.IAMTrustedExternalID, got)
	}
	if len(client.input.Tags) != 0 {
		t.Fatalf("expected no session tags for external-id profile, got %d", len(client.input.Tags))
	}
}

func TestParameterStoreRemoteRefKeyAvoidsReservedPrefixes(t *testing.T) {
	t.Parallel()

	got := parameterStoreRemoteRefKey("aws-v2-ps-refresh-remote", "e2e-tests-eso-aws-ps-v2-6s27x")
	if !strings.HasPrefix(got, "/e2e/") {
		t.Fatalf("expected /e2e/ prefix, got %q", got)
	}
	if strings.HasPrefix(strings.TrimPrefix(got, "/"), "aws") || strings.HasPrefix(strings.TrimPrefix(got, "/"), "ssm") {
		t.Fatalf("expected non-reserved parameter prefix, got %q", got)
	}
	if !strings.Contains(got, "aws-v2-ps-refresh-remote") {
		t.Fatalf("expected remote key to retain base name, got %q", got)
	}
}

func TestProbeAssumeRoleAccessBuildsSessionTagsRequest(t *testing.T) {
	t.Parallel()

	client := &fakeV2STSAssumeRoleClient{}
	access := awsV2AccessConfig{
		Role: awscommon.IAMRoleSessionTags,
	}
	if err := probeAssumeRoleAccess(context.Background(), client, access, awsV2AuthProfileSessionTags); err != nil {
		t.Fatalf("probeAssumeRoleAccess() error = %v", err)
	}
	if client.input == nil {
		t.Fatal("expected AssumeRole input to be recorded")
	}
	if got := aws.ToString(client.input.RoleArn); got != awscommon.IAMRoleSessionTags {
		t.Fatalf("expected role ARN %q, got %q", awscommon.IAMRoleSessionTags, got)
	}
	if got := aws.ToString(client.input.RoleSessionName); got != assumeRoleSessionName {
		t.Fatalf("expected role session name %q, got %q", assumeRoleSessionName, got)
	}
	if client.input.ExternalId != nil {
		t.Fatalf("expected no external ID for session-tags profile, got %q", aws.ToString(client.input.ExternalId))
	}
	if len(client.input.Tags) != 1 {
		t.Fatalf("expected one session tag, got %d", len(client.input.Tags))
	}
	tag := client.input.Tags[0]
	if aws.ToString(tag.Key) != "namespace" || aws.ToString(tag.Value) != "e2e-test" {
		t.Fatalf("unexpected session tag: %+v", tag)
	}
}

func TestIsAssumeRoleAccessDeniedRecognizesAssumeRoleErrors(t *testing.T) {
	t.Parallel()

	err := errors.New("api error AccessDenied: User is not authorized to perform: sts:AssumeRole")
	if !isAssumeRoleAccessDenied(err) {
		t.Fatal("expected sts:AssumeRole access denied error to be recognized")
	}
}

func TestIsAssumeRoleAccessDeniedRecognizesTagSessionErrors(t *testing.T) {
	t.Parallel()

	err := errors.New("api error AccessDenied: User is not authorized to perform: sts:TagSession")
	if !isAssumeRoleAccessDenied(err) {
		t.Fatal("expected sts:TagSession access denied error to be recognized")
	}
}

func TestParameterStoreConfigForMountedIRSAUsesEmptyAWSAuth(t *testing.T) {
	t.Parallel()

	cfg := newParameterStoreV2Config("ns", "ps-mounted-irsa", awsV2AccessConfig{
		Region: "eu-west-1",
	}, awsV2AuthProfileMountedIRSA)
	if cfg.Spec.Auth != (esv1.AWSAuth{}) {
		t.Fatalf("expected mounted IRSA auth to be empty, got %+v", cfg.Spec.Auth)
	}
}
