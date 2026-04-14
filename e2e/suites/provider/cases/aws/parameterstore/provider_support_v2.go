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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	"github.com/external-secrets/external-secrets-e2e/framework/log"
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmetav1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	awsv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/aws/v2alpha1"
)

const (
	awsProviderAPIVersion = "provider.external-secrets.io/v2alpha1"
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
	access    awsV2AccessConfig
	client    *ssm.Client
	clientErr error
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

func newParameterStoreV2Config(namespace, name string, access awsV2AccessConfig) *awsv2alpha1.ParameterStore {
	return &awsv2alpha1.ParameterStore{
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
			Auth:   staticAWSV2Auth(awscommon.CredentialsSecretName(name)),
		},
	}
}

func createParameterStoreV2Config(f *framework.Framework, namespace, name string, access awsV2AccessConfig) *awsv2alpha1.ParameterStore {
	createStaticCredentialsSecret(f, namespace, awscommon.CredentialsSecretName(name), access)
	cfg := newParameterStoreV2Config(namespace, name, access)
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
		tc.Prepare = prov.prepareNamespacedProvider()
	}
}

func (p *ProviderV2) prepareNamespacedProvider() func(*framework.TestCase, framework.SecretStoreProvider) {
	return func(_ *framework.TestCase, _ framework.SecretStoreProvider) {
		configName := p.providerConfigName()
		createParameterStoreV2Config(p.framework, p.framework.Namespace.Name, configName, p.access)
		frameworkv2.CreateProviderConnection(
			p.framework,
			p.framework.Namespace.Name,
			p.framework.Namespace.Name,
			frameworkv2.ProviderAddress("aws"),
			awsProviderAPIVersion,
			awsv2alpha1.ParameterStoreKind,
			configName,
			p.framework.Namespace.Name,
		)
		frameworkv2.WaitForProviderConnectionReady(p.framework, p.framework.Namespace.Name, p.framework.Namespace.Name, defaultV2WaitTimeout)
	}
}

func (p *ProviderV2) providerConfigName() string {
	return fmt.Sprintf("%s-parameterstore", p.framework.Namespace.Name)
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
