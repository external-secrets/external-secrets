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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/log"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmetav1 "github.com/external-secrets/external-secrets/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	defaultV2WaitTimeout  = 60 * time.Second
	defaultV2PollInterval = 2 * time.Second
)

type awsV2AccessConfig struct {
	KID    string
	SAK    string
	ST     string
	Region string
}

type parameterStoreBackend struct {
	access     awsV2AccessConfig
	client     *ssm.Client
	clientErr  error
	clientOnce sync.Once
}

func loadAWSV2AccessConfigFromEnv() awsV2AccessConfig {
	return awsV2AccessConfig{
		KID:    os.Getenv("AWS_ACCESS_KEY_ID"),
		SAK:    os.Getenv("AWS_SECRET_ACCESS_KEY"),
		ST:     os.Getenv("AWS_SESSION_TOKEN"),
		Region: os.Getenv("AWS_REGION"),
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

func newParameterStoreV2StoreProvider(secretName string, access awsV2AccessConfig, authNamespace *string) *esv1.SecretStoreProvider {
	provider := &esv1.SecretStoreProvider{
		AWS: &esv1.AWSProvider{
			Service: esv1.AWSServiceParameterStore,
			Region:  access.Region,
			Auth:    staticAWSV2Auth(secretName),
		},
	}
	if authNamespace != nil && provider.AWS.Auth.SecretRef != nil {
		provider.AWS.Auth.SecretRef.AccessKeyID.Namespace = authNamespace
		provider.AWS.Auth.SecretRef.SecretAccessKey.Namespace = authNamespace
		if provider.AWS.Auth.SecretRef.SessionToken != nil {
			provider.AWS.Auth.SecretRef.SessionToken.Namespace = authNamespace
		}
	}
	return provider
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
		tc.Prepare = prov.prepareNamespacedProvider()
	}
}

func (p *ProviderV2) prepareNamespacedProvider() func(*framework.TestCase, framework.SecretStoreProvider) {
	return func(_ *framework.TestCase, _ framework.SecretStoreProvider) {
		configName := p.providerConfigName()
		createStaticCredentialsSecret(p.framework, p.framework.Namespace.Name, awscommon.CredentialsSecretName(configName), p.access)
		frameworkv2.CreateRuntimeSecretStore(
			p.framework,
			p.framework.Namespace.Name,
			p.framework.Namespace.Name,
			frameworkv2.ProviderAddress("aws"),
			newParameterStoreV2StoreProvider(awscommon.CredentialsSecretName(configName), p.access, nil),
		)
		frameworkv2.WaitForSecretStoreReady(p.framework, p.framework.Namespace.Name, p.framework.Namespace.Name, defaultV2WaitTimeout)
	}
}

func (p *ProviderV2) providerConfigName() string {
	return fmt.Sprintf("%s-parameterstore", p.framework.Namespace.Name)
}

func createParameterStoreV2RuntimeClusterSecretStore(f *framework.Framework, name, secretNamespace, secretName string, access awsV2AccessConfig, authNamespace *string, conditions []esv1.ClusterSecretStoreCondition) {
	createStaticCredentialsSecret(f, secretNamespace, secretName, access)
	frameworkv2.CreateRuntimeClusterSecretStore(
		f,
		name,
		frameworkv2.ProviderAddress("aws"),
		newParameterStoreV2StoreProvider(secretName, access, authNamespace),
		conditions,
	)
	log.Logf("created ParameterStore ClusterSecretStore: %s", name)
}
