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
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretsmanagertypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/log"
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmetav1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	awsv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/aws/v2alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	defaultV2WaitTimeout  = 60 * time.Second
	defaultV2PollInterval = 2 * time.Second
)

const (
	assumeRoleSessionName = "eso-e2e-probe"
)

type awsAuthProfile string

const (
	awsAuthProfileStatic         awsAuthProfile = "static"
	awsAuthProfileExternalID     awsAuthProfile = "external-id"
	awsAuthProfileSessionTags    awsAuthProfile = "session-tags"
	awsAuthProfileReferencedIRSA awsAuthProfile = "referenced-irsa"
	awsAuthProfileMountedIRSA    awsAuthProfile = "mounted-irsa"
)

type awsAccessConfig struct {
	KID         string
	SAK         string
	ST          string
	Region      string
	Role        string
	SAName      string
	SANamespace string
}

type secretsManagerBackend struct {
	access     awsAccessConfig
	client     *awssm.Client
	clientErr  error
	clientOnce sync.Once
	framework  *framework.Framework
}

type stsAssumeRoleClient interface {
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

type assumeRoleProbeKey struct {
	access  awsAccessConfig
	profile awsAuthProfile
}

type assumeRoleProbeResult struct {
	err error
}

var assumeRoleProbeCache sync.Map

func loadAWSAccessConfigFromEnv() awsAccessConfig {
	return awsAccessConfig{
		KID:         os.Getenv("AWS_ACCESS_KEY_ID"),
		SAK:         os.Getenv("AWS_SECRET_ACCESS_KEY"),
		ST:          os.Getenv("AWS_SESSION_TOKEN"),
		Region:      os.Getenv("AWS_REGION"),
		SAName:      os.Getenv("AWS_SA_NAME"),
		SANamespace: os.Getenv("AWS_SA_NAMESPACE"),
	}
}

func newBackendFromEnv(f *framework.Framework) *secretsManagerBackend {
	return newSecretsManagerBackend(f, loadAWSAccessConfigFromEnv())
}

func newSecretsManagerBackend(f *framework.Framework, access awsAccessConfig) *secretsManagerBackend {
	return &secretsManagerBackend{
		access:    access,
		framework: f,
	}
}

func (c awsAccessConfig) missingStaticCredentials() []string {
	var missing []string
	if c.KID == "" {
		missing = append(missing, "AWS_ACCESS_KEY_ID")
	}
	if c.SAK == "" {
		missing = append(missing, "AWS_SECRET_ACCESS_KEY")
	}
	if c.Region == "" {
		missing = append(missing, "AWS_REGION")
	}
	return missing
}

func skipIfAWSStaticCredentialsMissing(access awsAccessConfig) {
	if missing := access.missingStaticCredentials(); len(missing) > 0 {
		Skip("missing AWS e2e credentials: " + strings.Join(missing, ", "))
	}
}

func skipIfAWSManagedIRSAEnvMissing(access awsAccessConfig) {
	var missing []string
	if access.Region == "" {
		missing = append(missing, "AWS_REGION")
	}
	if access.SAName == "" {
		missing = append(missing, "AWS_SA_NAME")
	}
	if access.SANamespace == "" {
		missing = append(missing, "AWS_SA_NAMESPACE")
	}
	if len(missing) > 0 {
		Skip("missing AWS managed IRSA environment: " + strings.Join(missing, ", "))
	}
}

func skipIfAWSAssumeRoleProbeDenied(access awsAccessConfig, profile awsAuthProfile) {
	if profile != awsAuthProfileExternalID && profile != awsAuthProfileSessionTags {
		return
	}

	cacheKey := assumeRoleProbeKey{
		access:  access,
		profile: profile,
	}
	if cached, ok := assumeRoleProbeCache.Load(cacheKey); ok {
		handleAssumeRoleProbeResult(access, profile, cached.(assumeRoleProbeResult).err)
		return
	}

	cfg, err := loadAWSConfig(access)
	Expect(err).NotTo(HaveOccurred())

	err = probeAssumeRoleAccess(context.Background(), sts.NewFromConfig(cfg), access, profile)
	assumeRoleProbeCache.Store(cacheKey, assumeRoleProbeResult{err: err})
	handleAssumeRoleProbeResult(access, profile, err)
}

func handleAssumeRoleProbeResult(access awsAccessConfig, profile awsAuthProfile, err error) {
	if err == nil {
		return
	}
	if isAssumeRoleAccessDenied(err) {
		Skip(fmt.Sprintf("skipping AWS %s auth e2e: %s is not authorized to assume role %q with the current credentials", profile, assumeRoleAction(profile), roleARNForProfile(access, profile)))
	}
	Expect(err).NotTo(HaveOccurred())
}

func assumeRoleAction(profile awsAuthProfile) string {
	if profile == awsAuthProfileSessionTags {
		return "sts:TagSession"
	}
	return "sts:AssumeRole"
}

func staticAWSAuth(secretName string) esv1.AWSAuth {
	return esv1.AWSAuth{
		SecretRef: &esv1.AWSAuthSecretRef{
			AccessKeyID: esmetav1.SecretKeySelector{
				Name: secretName,
				Key:  awscommon.StaticAccessKeyIDKey,
			},
			SecretAccessKey: esmetav1.SecretKeySelector{
				Name: secretName,
				Key:  awscommon.StaticSecretAccessKeyKey,
			},
			SessionToken: &esmetav1.SecretKeySelector{
				Name: secretName,
				Key:  awscommon.StaticSessionTokenKey,
			},
		},
	}
}

func newStaticCredentialsSecret(namespace, name string, access awsAccessConfig) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: awscommon.StaticCredentialsSecretData(access.KID, access.SAK, access.ST),
	}
}

func createStaticCredentialsSecret(f *framework.Framework, namespace, name string, access awsAccessConfig) {
	Expect(f.CRClient.Create(GinkgoT().Context(), newStaticCredentialsSecret(namespace, name, access))).To(Succeed())
}

func roleARNForProfile(access awsAccessConfig, profile awsAuthProfile) string {
	if access.Role != "" {
		return access.Role
	}
	switch profile {
	case awsAuthProfileExternalID:
		return awscommon.IAMRoleExternalID
	case awsAuthProfileSessionTags:
		return awscommon.IAMRoleSessionTags
	default:
		return ""
	}
}

func sessionTagsForProfile(profile awsAuthProfile) []ststypes.Tag {
	if profile != awsAuthProfileSessionTags {
		return nil
	}

	return []ststypes.Tag{{
		Key:   aws.String("namespace"),
		Value: aws.String("e2e-test"),
	}}
}

func probeAssumeRoleAccess(ctx context.Context, client stsAssumeRoleClient, access awsAccessConfig, profile awsAuthProfile) error {
	if profile != awsAuthProfileExternalID && profile != awsAuthProfileSessionTags {
		return nil
	}

	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleARNForProfile(access, profile)),
		RoleSessionName: aws.String(assumeRoleSessionName),
		Tags:            sessionTagsForProfile(profile),
	}
	if profile == awsAuthProfileExternalID {
		input.ExternalId = aws.String(awscommon.IAMTrustedExternalID)
	}

	_, err := client.AssumeRole(ctx, input)
	return err
}

func isAssumeRoleAccessDenied(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "accessdenied") {
		return false
	}
	return strings.Contains(msg, "sts:assumerole") || strings.Contains(msg, "sts:tagsession")
}

func newSecretsManagerV2StoreProvider(secretName string, access awsAccessConfig, profile awsAuthProfile, authNamespace *string) *esv1.SecretStoreProvider {
	provider := &esv1.SecretStoreProvider{
		AWS: &esv1.AWSProvider{
			Service: esv1.AWSServiceSecretsManager,
			Region:  access.Region,
		},
	}
	switch profile {
	case awsAuthProfileStatic:
		provider.AWS.Auth = staticAWSAuth(secretName)
	case awsAuthProfileExternalID:
		provider.AWS.Auth = staticAWSAuth(secretName)
		provider.AWS.Role = access.Role
		if provider.AWS.Role == "" {
			provider.AWS.Role = awscommon.IAMRoleExternalID
		}
		provider.AWS.ExternalID = awscommon.IAMTrustedExternalID
	case awsAuthProfileSessionTags:
		provider.AWS.Auth = staticAWSAuth(secretName)
		provider.AWS.Role = access.Role
		if provider.AWS.Role == "" {
			provider.AWS.Role = awscommon.IAMRoleSessionTags
		}
		provider.AWS.SessionTags = []*esv1.Tag{{
			Key:   "namespace",
			Value: "e2e-test",
		}}
	case awsAuthProfileReferencedIRSA:
		provider.AWS.Auth = esv1.AWSAuth{
			JWTAuth: &esv1.AWSJWTAuth{
				ServiceAccountRef: &esmetav1.ServiceAccountSelector{
					Name:      access.SAName,
					Namespace: &access.SANamespace,
				},
			},
		}
	case awsAuthProfileMountedIRSA:
		provider.AWS.Auth = esv1.AWSAuth{}
	default:
		provider.AWS.Auth = staticAWSAuth(secretName)
	}
	if authNamespace != nil && provider.AWS.Auth.SecretRef != nil {
		provider.AWS.Auth.SecretRef.AccessKeyID.Namespace = authNamespace
		provider.AWS.Auth.SecretRef.SecretAccessKey.Namespace = authNamespace
		if provider.AWS.Auth.SecretRef.SessionToken != nil {
			provider.AWS.Auth.SecretRef.SessionToken.Namespace = authNamespace
		}
	}
	if authNamespace != nil && provider.AWS.Auth.JWTAuth != nil && provider.AWS.Auth.JWTAuth.ServiceAccountRef != nil {
		provider.AWS.Auth.JWTAuth.ServiceAccountRef.Namespace = authNamespace
	}
	return provider
}

func createSecretsManagerV2ProviderConfig(f *framework.Framework, namespace, name, providerRefNamespace string, provider *esv1.SecretStoreProvider) *esv1.StoreProviderRef {
	Expect(provider).NotTo(BeNil())
	Expect(provider.AWS).NotTo(BeNil())
	Expect(f.CreateObjectWithRetry(&awsv2alpha1.SecretsManager{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "provider.external-secrets.io/v2alpha1",
			Kind:       "SecretsManager",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: awsv2alpha1.SecretsManagerSpec{
			Auth:              provider.AWS.Auth,
			Role:              provider.AWS.Role,
			Region:            provider.AWS.Region,
			AdditionalRoles:   provider.AWS.AdditionalRoles,
			ExternalID:        provider.AWS.ExternalID,
			SessionTags:       provider.AWS.SessionTags,
			SecretsManager:    provider.AWS.SecretsManager,
			TransitiveTagKeys: provider.AWS.TransitiveTagKeys,
			Prefix:            provider.AWS.Prefix,
		},
	})).To(Succeed())

	return &esv1.StoreProviderRef{
		APIVersion: "provider.external-secrets.io/v2alpha1",
		Kind:       "SecretsManager",
		Name:       name,
		Namespace:  providerRefNamespace,
	}
}

func loadAWSConfig(access awsAccessConfig) (aws.Config, error) {
	loadOptions := []func(*config.LoadOptions) error{
		config.WithRegion(access.Region),
	}
	if access.KID != "" || access.SAK != "" || access.ST != "" {
		loadOptions = append(loadOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(access.KID, access.SAK, access.ST),
		))
	}

	return config.LoadDefaultConfig(context.Background(), loadOptions...)
}

func (b *secretsManagerBackend) ensureClient() {
	b.clientOnce.Do(func() {
		cfg, err := loadAWSConfig(b.access)
		if err != nil {
			b.clientErr = err
			return
		}
		b.client = awssm.NewFromConfig(cfg)
	})

	Expect(b.clientErr).ToNot(HaveOccurred())
	Expect(b.client).NotTo(BeNil())
}

func (b *secretsManagerBackend) CreateSecret(key string, val framework.SecretEntry) {
	b.ensureClient()

	smTags := make([]secretsmanagertypes.Tag, 0, len(val.Tags))
	for tagKey, tagValue := range val.Tags {
		smTags = append(smTags, secretsmanagertypes.Tag{
			Key:   aws.String(tagKey),
			Value: aws.String(tagValue),
		})
	}

	attempts := 20
	for {
		log.Logf("creating secret %s / attempts left: %d", key, attempts)
		_, err := b.client.CreateSecret(GinkgoT().Context(), &awssm.CreateSecretInput{
			Name:         aws.String(key),
			SecretString: aws.String(val.Value),
			Tags:         smTags,
		})
		if err == nil {
			return
		}
		attempts--
		if attempts < 0 {
			Fail("unable to create secret: " + err.Error())
		}
		<-time.After(5 * time.Second)
	}
}

func (b *secretsManagerBackend) DeleteSecret(key string) {
	b.ensureClient()

	log.Logf("deleting secret %s", key)
	_, err := b.client.DeleteSecret(GinkgoT().Context(), &awssm.DeleteSecretInput{
		SecretId:                   aws.String(key),
		ForceDeleteWithoutRecovery: aws.Bool(true),
	})
	var notFound *secretsmanagertypes.ResourceNotFoundException
	if errors.As(err, &notFound) {
		return
	}
	Expect(err).ToNot(HaveOccurred())
}

func (b *secretsManagerBackend) WaitForSecretValue(name, expectedValue string) {
	b.ensureClient()

	Eventually(func(g Gomega) {
		out, err := b.client.GetSecretValue(GinkgoT().Context(), &awssm.GetSecretValueInput{
			SecretId: aws.String(name),
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(secretValueString(out)).To(Equal(expectedValue))
	}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())
}

func (b *secretsManagerBackend) ExpectSecretAbsent(name string) {
	b.ensureClient()

	Eventually(func() bool {
		_, err := b.client.GetSecretValue(GinkgoT().Context(), &awssm.GetSecretValueInput{
			SecretId: aws.String(name),
		})
		return secretReadErrorIndicatesAbsence(err)
	}, defaultV2WaitTimeout, defaultV2PollInterval).Should(BeTrue(), fmt.Sprintf("expected AWS secret %q to be absent", name))
}

func secretValueString(out *awssm.GetSecretValueOutput) string {
	if out == nil {
		return ""
	}
	if out.SecretString != nil {
		return aws.ToString(out.SecretString)
	}
	if len(out.SecretBinary) > 0 {
		return string(out.SecretBinary)
	}
	return ""
}

func secretReadErrorIndicatesAbsence(err error) bool {
	if err == nil {
		return false
	}

	var notFound *secretsmanagertypes.ResourceNotFoundException
	if errors.As(err, &notFound) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "marked for deletion") || strings.Contains(msg, "scheduled for deletion")
}
