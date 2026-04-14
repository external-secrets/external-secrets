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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	awsv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/aws/v2alpha1"
)

var _ = Describe("[aws] v2 namespaced provider", Label("aws", "secretsmanager", "v2", "namespaced-provider"), func() {
	f := framework.New("eso-aws-sm-v2")
	prov := NewProviderV2(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
	})

	DescribeTable("namespaced provider",
		framework.TableFuncWithExternalSecret(f, prov),
		framework.Compose(withStaticAuth, f, func(f *framework.Framework) (string, func(*framework.TestCase)) {
			return common.NamespacedProviderSync(f, common.NamespacedProviderSyncConfig{
				Description:        "[aws] should sync an ExternalSecret through a namespaced Provider using static credentials",
				ExternalSecretName: "aws-v2-static-es",
				TargetSecretName:   "aws-v2-static-target",
				RemoteKey:          f.MakeRemoteRefKey("aws-v2-static-remote"),
				RemoteSecretValue:  `{"value":"aws-v2-static-value"}`,
				RemoteProperty:     "value",
				SecretKey:          "value",
				ExpectedValue:      "aws-v2-static-value",
			})
		}, useV2StaticAuth(prov)),
		framework.Compose(withStaticAuth, f, func(f *framework.Framework) (string, func(*framework.TestCase)) {
			return common.NamespacedProviderRefresh(f, common.NamespacedProviderRefreshConfig{
				Description:         "[aws] should refresh synced secrets after the remote AWS secret changes",
				ExternalSecretName:  "aws-v2-refresh-es",
				TargetSecretName:    "aws-v2-refresh-target",
				RemoteKey:           f.MakeRemoteRefKey("aws-v2-refresh-remote"),
				InitialSecretValue:  `{"value":"aws-v2-initial"}`,
				UpdatedSecretValue:  `{"value":"aws-v2-updated"}`,
				RemoteProperty:      "value",
				SecretKey:           "value",
				InitialExpectedData: "aws-v2-initial",
				UpdatedExpectedData: "aws-v2-updated",
				RefreshInterval:     10 * time.Second,
				WaitTimeout:         30 * time.Second,
			})
		}, useV2StaticAuth(prov)),
		framework.Compose(withStaticAuth, f, func(f *framework.Framework) (string, func(*framework.TestCase)) {
			return common.NamespacedProviderFind(f, common.NamespacedProviderFindConfig{
				Description:        "[aws] should sync ExternalSecret dataFrom.find through a namespaced Provider",
				ExternalSecretName: "aws-v2-find-es",
				TargetSecretName:   "aws-v2-find-target",
				MatchRegExp:        "^aws-v2-find-(one|two)$",
				MatchingSecrets: map[string]string{
					"aws-v2-find-one": "aws-v2-one",
					"aws-v2-find-two": "aws-v2-two",
				},
				IgnoredSecrets: map[string]string{
					"aws-v2-ignore": "aws-v2-ignore",
				},
			})
		}, useV2StaticAuth(prov)),
		framework.Compose(withStaticAuth, f, common.StatusNotUpdatedAfterSuccessfulSync, useV2StaticAuth(prov)),
		framework.Compose(withExtID, f, SimpleSyncWithNamespaceTags(nil), useV2ExternalIDAuth(prov)),
		framework.Compose(withSessionTags, f, SimpleSyncWithNamespaceTags(nil), useV2SessionTagsAuth(prov)),
	)
})

type ProviderV2 struct {
	access    awsAccessConfig
	backend   *secretsManagerBackend
	framework *framework.Framework
}

func NewProviderV2(f *framework.Framework) *ProviderV2 {
	access := loadAWSAccessConfigFromEnv()
	f.MakeRemoteRefKey = func(base string) string {
		if f.Namespace == nil {
			return base
		}
		suffix := f.Namespace.Name
		if len(suffix) > 8 {
			suffix = suffix[len(suffix)-8:]
		}
		if suffix == "" {
			return base
		}
		return fmt.Sprintf("%s-%s", base, suffix)
	}
	prov := &ProviderV2{
		access:    access,
		backend:   newSecretsManagerBackend(f, access),
		framework: f,
	}

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			return
		}
		skipIfAWSStaticCredentialsMissing(access)
	})

	return prov
}

func (p *ProviderV2) CreateSecret(key string, val framework.SecretEntry) {
	p.backend.CreateSecret(key, val)
}

func (p *ProviderV2) DeleteSecret(key string) {
	p.backend.DeleteSecret(key)
}

func useV2StaticAuth(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareNamespacedProvider(awsAuthProfileStatic)
	}
}

func useV2ExternalIDAuth(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareNamespacedProvider(awsAuthProfileExternalID)
	}
}

func useV2SessionTagsAuth(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareNamespacedProvider(awsAuthProfileSessionTags)
	}
}

func (p *ProviderV2) prepareNamespacedProvider(profile awsAuthProfile) func(*framework.TestCase, framework.SecretStoreProvider) {
	return p.prepareNamespacedProviderAtAddress(profile, frameworkv2.ProviderAddress("aws"))
}

func (p *ProviderV2) prepareNamespacedProviderAtAddress(profile awsAuthProfile, address string) func(*framework.TestCase, framework.SecretStoreProvider) {
	return func(_ *framework.TestCase, _ framework.SecretStoreProvider) {
		skipIfAWSAssumeRoleProbeDenied(p.access, profile)

		configName := p.providerConfigName(profile)
		createSecretsManagerV2Config(p.framework, p.framework.Namespace.Name, configName, p.access, profile)
		frameworkv2.CreateProviderConnection(
			p.framework,
			p.framework.Namespace.Name,
			p.framework.Namespace.Name,
			address,
			awsProviderAPIVersion,
			awsv2alpha1.SecretsManagerKind,
			configName,
			p.framework.Namespace.Name,
		)
		frameworkv2.WaitForProviderConnectionReady(p.framework, p.framework.Namespace.Name, p.framework.Namespace.Name, defaultV2WaitTimeout)
	}
}

func (p *ProviderV2) providerConfigName(profile awsAuthProfile) string {
	return fmt.Sprintf("%s-%s", p.framework.Namespace.Name, profile)
}
