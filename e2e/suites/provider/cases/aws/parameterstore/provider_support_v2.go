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
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/aws/smithy-go"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/log"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmetav1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	awsv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/aws/v2alpha1"
)

const (
	awsProviderAPIVersion  = "provider.external-secrets.io/v2alpha1"
	defaultV2WaitTimeout   = 60 * time.Second
	defaultV2PollInterval  = 2 * time.Second
	assumeRoleSessionName  = "eso-e2e-probe"
	assumeRoleProbeTimeout = 15 * time.Second
)

type awsV2AuthProfile string

const (
	awsV2AuthProfileStatic         awsV2AuthProfile = "static"
	awsV2AuthProfileExternalID     awsV2AuthProfile = "external-id"
	awsV2AuthProfileSessionTags    awsV2AuthProfile = "session-tags"
	awsV2AuthProfileReferencedIRSA awsV2AuthProfile = "referenced-irsa"
	awsV2AuthProfileMountedIRSA    awsV2AuthProfile = "mounted-irsa"
)

type awsV2AccessConfig struct {
	KID         string
	SAK         string
	ST          string
	Region      string
	Role        string
	SAName      string
	SANamespace string
}

type parameterStoreBackend struct {
	access     awsV2AccessConfig
	client     *ssm.Client
	clientErr  error
	clientOnce sync.Once
}

type stsAssumeRoleV2Client interface {
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

type assumeRoleV2ProbeKey struct {
	access  awsV2AccessConfig
	profile awsV2AuthProfile
}

type assumeRoleV2ProbeResult struct {
	err error
}

var assumeRoleV2ProbeCache sync.Map

func loadAWSV2AccessConfigFromEnv() awsV2AccessConfig {
	role := os.Getenv("AWS_ROLE_ARN")
	if role == "" {
		role = os.Getenv("AWS_ROLE")
	}
	return awsV2AccessConfig{
		KID:         os.Getenv("AWS_ACCESS_KEY_ID"),
		SAK:         os.Getenv("AWS_SECRET_ACCESS_KEY"),
		ST:          os.Getenv("AWS_SESSION_TOKEN"),
		Region:      os.Getenv("AWS_REGION"),
		Role:        role,
		SAName:      os.Getenv("AWS_SA_NAME"),
		SANamespace: os.Getenv("AWS_SA_NAMESPACE"),
	}
}

func (c awsV2AccessConfig) missingStaticCredentials() []string {
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

func skipIfAWSV2StaticCredentialsMissing(access awsV2AccessConfig) {
	if missing := access.missingStaticCredentials(); len(missing) > 0 {
		Skip("missing AWS e2e credentials: " + strings.Join(missing, ", "))
	}
}

func skipIfAWSManagedIRSAEnvMissing(access awsV2AccessConfig) {
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

func skipIfAWSAssumeRoleProbeDenied(access awsV2AccessConfig, profile awsV2AuthProfile) {
	if profile != awsV2AuthProfileExternalID && profile != awsV2AuthProfileSessionTags {
		return
	}

	cacheKey := assumeRoleV2ProbeKey{
		access:  access,
		profile: profile,
	}
	if cached, ok := assumeRoleV2ProbeCache.Load(cacheKey); ok {
		handleAssumeRoleV2ProbeResult(access, profile, cached.(assumeRoleV2ProbeResult).err)
		return
	}

	cfg, err := loadParameterStoreAWSConfig(access)
	Expect(err).NotTo(HaveOccurred())

	probeCtx, cancel := context.WithTimeout(context.Background(), assumeRoleProbeTimeout)
	defer cancel()
	err = probeAssumeRoleAccess(probeCtx, sts.NewFromConfig(cfg), access, profile)
	assumeRoleV2ProbeCache.Store(cacheKey, assumeRoleV2ProbeResult{err: err})
	handleAssumeRoleV2ProbeResult(access, profile, err)
}

func handleAssumeRoleV2ProbeResult(access awsV2AccessConfig, profile awsV2AuthProfile, err error) {
	if err == nil {
		return
	}
	if isAssumeRoleAccessDenied(err) {
		Skip(fmt.Sprintf("skipping AWS %s auth e2e: %s is not authorized to assume role %q with the current credentials", profile, assumeRoleAction(profile), roleARNForProfile(access, profile)))
	}
	Expect(err).NotTo(HaveOccurred())
}

func assumeRoleAction(profile awsV2AuthProfile) string {
	if profile == awsV2AuthProfileSessionTags {
		return "sts:TagSession"
	}
	return "sts:AssumeRole"
}

func roleARNForProfile(access awsV2AccessConfig, profile awsV2AuthProfile) string {
	if access.Role != "" {
		return access.Role
	}
	switch profile {
	case awsV2AuthProfileExternalID:
		return awscommon.IAMRoleExternalID
	case awsV2AuthProfileSessionTags:
		return awscommon.IAMRoleSessionTags
	default:
		return ""
	}
}

func sessionTagsForProfile(profile awsV2AuthProfile) []ststypes.Tag {
	if profile != awsV2AuthProfileSessionTags {
		return nil
	}

	return []ststypes.Tag{{
		Key:   aws.String("namespace"),
		Value: aws.String("e2e-test"),
	}}
}

func probeAssumeRoleAccess(ctx context.Context, client stsAssumeRoleV2Client, access awsV2AccessConfig, profile awsV2AuthProfile) error {
	if profile != awsV2AuthProfileExternalID && profile != awsV2AuthProfileSessionTags {
		return nil
	}

	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleARNForProfile(access, profile)),
		RoleSessionName: aws.String(assumeRoleSessionName),
		Tags:            sessionTagsForProfile(profile),
	}
	if profile == awsV2AuthProfileExternalID {
		input.ExternalId = aws.String(awscommon.IAMTrustedExternalID)
	}

	_, err := client.AssumeRole(ctx, input)
	return err
}

func isAssumeRoleAccessDenied(err error) bool {
	if err == nil {
		return false
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if !strings.Contains(strings.ToLower(apiErr.ErrorCode()), "accessdenied") {
			return false
		}
		if hasAssumeRoleActionContext(strings.ToLower(apiErr.ErrorMessage())) {
			return true
		}

		var opErr *smithy.OperationError
		if errors.As(err, &opErr) && strings.EqualFold(opErr.Service(), "STS") && strings.EqualFold(opErr.Operation(), "AssumeRole") {
			return true
		}

		msg := strings.ToLower(err.Error())
		return hasAssumeRoleActionContext(msg)
	}

	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "accessdenied") {
		return false
	}
	return hasAssumeRoleActionContext(msg)
}

func hasAssumeRoleActionContext(msg string) bool {
	return strings.Contains(msg, "sts:assumerole") || strings.Contains(msg, "sts:tagsession")
}

func staticAWSV2Auth(secretName string) esv1.AWSAuth {
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

func newStaticCredentialsSecret(namespace, name string, access awsV2AccessConfig) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: awscommon.StaticCredentialsSecretData(access.KID, access.SAK, access.ST),
	}
}

func createStaticCredentialsSecret(f *framework.Framework, namespace, name string, access awsV2AccessConfig) {
	Expect(f.CRClient.Create(GinkgoT().Context(), newStaticCredentialsSecret(namespace, name, access))).To(Succeed())
}

func newParameterStoreV2Config(namespace, name string, access awsV2AccessConfig, profile ...awsV2AuthProfile) *awsv2alpha1.ParameterStore {
	authProfile := awsV2AuthProfileStatic
	if len(profile) > 0 {
		authProfile = profile[0]
	}

	cfg := &awsv2alpha1.ParameterStore{
		TypeMeta: metav1.TypeMeta{
			APIVersion: awsv2alpha1.GroupVersion.String(),
			Kind:       awsv2alpha1.ParameterStoreKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: awsv2alpha1.ParameterStoreSpec{
			Region: access.Region,
		},
	}

	switch authProfile {
	case awsV2AuthProfileStatic:
		cfg.Spec.Auth = staticAWSV2Auth(awscommon.CredentialsSecretName(name))
	case awsV2AuthProfileExternalID:
		cfg.Spec.Auth = staticAWSV2Auth(awscommon.CredentialsSecretName(name))
		cfg.Spec.Role = access.Role
		if cfg.Spec.Role == "" {
			cfg.Spec.Role = awscommon.IAMRoleExternalID
		}
		cfg.Spec.ExternalID = awscommon.IAMTrustedExternalID
	case awsV2AuthProfileSessionTags:
		cfg.Spec.Auth = staticAWSV2Auth(awscommon.CredentialsSecretName(name))
		cfg.Spec.Role = access.Role
		if cfg.Spec.Role == "" {
			cfg.Spec.Role = awscommon.IAMRoleSessionTags
		}
		cfg.Spec.SessionTags = []*esv1.Tag{{
			Key:   "namespace",
			Value: "e2e-test",
		}}
	case awsV2AuthProfileReferencedIRSA:
		cfg.Spec.Auth = esv1.AWSAuth{
			JWTAuth: &esv1.AWSJWTAuth{
				ServiceAccountRef: &esmetav1.ServiceAccountSelector{
					Name:      access.SAName,
					Namespace: &access.SANamespace,
				},
			},
		}
	case awsV2AuthProfileMountedIRSA:
		cfg.Spec.Auth = esv1.AWSAuth{}
	default:
		cfg.Spec.Auth = staticAWSV2Auth(awscommon.CredentialsSecretName(name))
	}

	return cfg
}

func createParameterStoreV2Config(f *framework.Framework, namespace, name string, access awsV2AccessConfig, profile ...awsV2AuthProfile) *awsv2alpha1.ParameterStore {
	authProfile := awsV2AuthProfileStatic
	if len(profile) > 0 {
		authProfile = profile[0]
	}

	if authProfile == awsV2AuthProfileStatic || authProfile == awsV2AuthProfileExternalID || authProfile == awsV2AuthProfileSessionTags {
		createStaticCredentialsSecret(f, namespace, awscommon.CredentialsSecretName(name), access)
	}

	cfg := newParameterStoreV2Config(namespace, name, access, authProfile)
	Expect(f.CRClient.Create(GinkgoT().Context(), cfg)).To(Succeed())
	return cfg
}

func loadParameterStoreAWSConfig(access awsV2AccessConfig) (aws.Config, error) {
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

func newParameterStoreBackend(access awsV2AccessConfig) *parameterStoreBackend {
	return &parameterStoreBackend{access: access}
}

func (b *parameterStoreBackend) ensureClient() {
	b.clientOnce.Do(func() {
		cfg, err := loadParameterStoreAWSConfig(b.access)
		if err != nil {
			b.clientErr = err
			return
		}
		b.client = ssm.NewFromConfig(cfg)
	})

	Expect(b.clientErr).ToNot(HaveOccurred())
	Expect(b.client).NotTo(BeNil())
}

func (b *parameterStoreBackend) CreateSecret(key string, val framework.SecretEntry) {
	b.ensureClient()

	psTags := make([]ssmtypes.Tag, 0, len(val.Tags))
	for tagKey, tagValue := range val.Tags {
		psTags = append(psTags, ssmtypes.Tag{
			Key:   aws.String(tagKey),
			Value: aws.String(tagValue),
		})
	}

	overwrite := len(psTags) == 0
	_, err := b.client.PutParameter(GinkgoT().Context(), &ssm.PutParameterInput{
		Name:      aws.String(key),
		Value:     aws.String(val.Value),
		DataType:  aws.String("text"),
		Type:      ssmtypes.ParameterTypeString,
		Overwrite: aws.Bool(overwrite),
		Tags:      psTags,
	})
	Expect(err).ToNot(HaveOccurred())
}

func (b *parameterStoreBackend) DeleteSecret(key string) {
	b.ensureClient()

	_, err := b.client.DeleteParameter(GinkgoT().Context(), &ssm.DeleteParameterInput{
		Name: aws.String(key),
	})
	var parameterNotFound *ssmtypes.ParameterNotFound
	var resourceNotFound *ssmtypes.ResourceNotFoundException
	if errors.As(err, &parameterNotFound) || errors.As(err, &resourceNotFound) {
		return
	}
	Expect(err).ToNot(HaveOccurred())
}

func (b *parameterStoreBackend) WaitForSecretValue(name, expectedValue string) {
	b.ensureClient()

	Eventually(func(g Gomega) {
		out, err := b.client.GetParameter(GinkgoT().Context(), &ssm.GetParameterInput{
			Name:           aws.String(name),
			WithDecryption: aws.Bool(true),
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out.Parameter).NotTo(BeNil())
		g.Expect(aws.ToString(out.Parameter.Value)).To(Equal(expectedValue))
	}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())
}

func (b *parameterStoreBackend) ExpectSecretAbsent(name string) {
	b.ensureClient()

	Eventually(func() bool {
		_, err := b.client.GetParameter(GinkgoT().Context(), &ssm.GetParameterInput{
			Name:           aws.String(name),
			WithDecryption: aws.Bool(true),
		})
		return parameterStoreReadErrorIndicatesAbsence(err)
	}, defaultV2WaitTimeout, defaultV2PollInterval).Should(BeTrue(), fmt.Sprintf("expected AWS parameter %q to be absent", name))
}

func parameterStoreReadErrorIndicatesAbsence(err error) bool {
	if err == nil {
		return false
	}
	var parameterNotFound *ssmtypes.ParameterNotFound
	var resourceNotFound *ssmtypes.ResourceNotFoundException
	return errors.As(err, &parameterNotFound) || errors.As(err, &resourceNotFound)
}

type ProviderV2 struct {
	access    awsV2AccessConfig
	backend   *parameterStoreBackend
	framework *framework.Framework
}

func NewProviderV2(f *framework.Framework) *ProviderV2 {
	access := loadAWSV2AccessConfigFromEnv()
	f.MakeRemoteRefKey = func(base string) string {
		if f.Namespace == nil {
			return parameterStoreRemoteRefKey(base, "")
		}
		return parameterStoreRemoteRefKey(base, f.Namespace.Name)
	}

	prov := &ProviderV2{
		access:    access,
		backend:   newParameterStoreBackend(access),
		framework: f,
	}

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			return
		}
		skipIfAWSV2StaticCredentialsMissing(access)
	})

	return prov
}

func parameterStoreRemoteRefKey(base, namespace string) string {
	base = strings.Trim(base, "/")
	if namespace == "" {
		return "/e2e/" + base
	}
	return fmt.Sprintf("/e2e/%s/%s", namespace, base)
}

func (p *ProviderV2) CreateSecret(key string, val framework.SecretEntry) {
	p.backend.CreateSecret(key, val)
}

func (p *ProviderV2) DeleteSecret(key string) {
	p.backend.DeleteSecret(key)
}

func useV2StaticAuth(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareNamespacedProvider(awsV2AuthProfileStatic)
	}
}

func useV2ExternalIDAuth(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareNamespacedProvider(awsV2AuthProfileExternalID)
	}
}

func useV2SessionTagsAuth(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareNamespacedProvider(awsV2AuthProfileSessionTags)
	}
}

func useV2ReferencedIRSA(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			configName := prov.providerConfigName(awsV2AuthProfileReferencedIRSA)
			clusterProviderName := referencedIRSAClusterProviderName(prov.framework.Namespace.Name)

			createParameterStoreV2Config(prov.framework, prov.framework.Namespace.Name, configName, prov.access, awsV2AuthProfileReferencedIRSA)
			frameworkv2.CreateClusterProviderConnection(
				prov.framework,
				clusterProviderName,
				frameworkv2.ProviderAddress("aws"),
				awsProviderAPIVersion,
				awsv2alpha1.ParameterStoreKind,
				configName,
				prov.framework.Namespace.Name,
				esv1.AuthenticationScopeManifestNamespace,
				nil,
			)
			frameworkv2.WaitForClusterProviderReady(prov.framework, clusterProviderName, defaultV2WaitTimeout)
			configureV2ReferencedIRSAStoreRef(tc, clusterProviderName)
		}
	}
}

func useV2MountedIRSA(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareNamespacedProviderAtAddress(
			awsV2AuthProfileMountedIRSA,
			frameworkv2.ProviderAddressInNamespace("aws", prov.access.SANamespace),
		)
	}
}

func referencedIRSAClusterProviderName(namespace string) string {
	return namespace + "-referenced-irsa"
}

func configureV2ReferencedIRSAStoreRef(tc *framework.TestCase, clusterProviderName string) {
	tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1.ClusterProviderKindStr
	tc.ExternalSecret.Spec.SecretStoreRef.Name = clusterProviderName
}

func (p *ProviderV2) prepareNamespacedProvider(profile ...awsV2AuthProfile) func(*framework.TestCase, framework.SecretStoreProvider) {
	authProfile := awsV2AuthProfileStatic
	if len(profile) > 0 {
		authProfile = profile[0]
	}
	return p.prepareNamespacedProviderAtAddress(authProfile, frameworkv2.ProviderAddress("aws"))
}

func (p *ProviderV2) prepareNamespacedProviderAtAddress(profile awsV2AuthProfile, address string) func(*framework.TestCase, framework.SecretStoreProvider) {
	return func(_ *framework.TestCase, _ framework.SecretStoreProvider) {
		skipIfAWSAssumeRoleProbeDenied(p.access, profile)

		configName := p.providerConfigName(profile)
		createParameterStoreV2Config(p.framework, p.framework.Namespace.Name, configName, p.access, profile)
		frameworkv2.CreateProviderConnection(
			p.framework,
			p.framework.Namespace.Name,
			p.framework.Namespace.Name,
			address,
			awsProviderAPIVersion,
			awsv2alpha1.ParameterStoreKind,
			configName,
			p.framework.Namespace.Name,
		)
		frameworkv2.WaitForProviderConnectionReady(p.framework, p.framework.Namespace.Name, p.framework.Namespace.Name, defaultV2WaitTimeout)
	}
}

func (p *ProviderV2) providerConfigName(profile ...awsV2AuthProfile) string {
	authProfile := awsV2AuthProfileStatic
	if len(profile) > 0 {
		authProfile = profile[0]
	}
	return fmt.Sprintf("%s-%s", p.framework.Namespace.Name, authProfile)
}

func createParameterStoreV2ProviderConnection(f *framework.Framework, namespace, name, providerName, providerNamespace string) {
	frameworkv2.CreateProviderConnection(
		f,
		namespace,
		name,
		frameworkv2.ProviderAddress("aws"),
		awsProviderAPIVersion,
		awsv2alpha1.ParameterStoreKind,
		providerName,
		providerNamespace,
	)
	log.Logf("created ParameterStore Provider connection: %s/%s", namespace, name)
}
