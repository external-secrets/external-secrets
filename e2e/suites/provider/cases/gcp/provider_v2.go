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

package gcp

import (
	"time"

	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
)

var _ = Describe("[gcp] v2 namespaced provider", Label("gcp", "secretsmanager", "v2", "namespaced-provider"), func() {
	f := framework.New("eso-gcp-v2")
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
				Description:        "[gcp] should sync through a namespaced Provider using static credentials",
				ExternalSecretName: "gcp-v2-static-es",
				TargetSecretName:   "gcp-v2-static-target",
				RemoteKey:          f.MakeRemoteRefKey("gcp-v2-static-remote"),
				RemoteSecretValue:  `{"value":"gcp-v2-static-value"}`,
				RemoteProperty:     "value",
				SecretKey:          "value",
				ExpectedValue:      "gcp-v2-static-value",
			})
		}, useV2StaticAuth(prov)),
		framework.Compose(withStaticAuth, f, func(f *framework.Framework) (string, func(*framework.TestCase)) {
			return common.NamespacedProviderRefresh(f, common.NamespacedProviderRefreshConfig{
				Description:         "[gcp] should refresh synced secrets after the remote secret changes",
				ExternalSecretName:  "gcp-v2-refresh-es",
				TargetSecretName:    "gcp-v2-refresh-target",
				RemoteKey:           f.MakeRemoteRefKey("gcp-v2-refresh-remote"),
				InitialSecretValue:  `{"value":"gcp-v2-initial"}`,
				UpdatedSecretValue:  `{"value":"gcp-v2-updated"}`,
				RemoteProperty:      "value",
				SecretKey:           "value",
				InitialExpectedData: "gcp-v2-initial",
				UpdatedExpectedData: "gcp-v2-updated",
				RefreshInterval:     10 * time.Second,
				WaitTimeout:         30 * time.Second,
				UpdateRemoteSecret: func(_ *framework.TestCase, _ framework.SecretStoreProvider) {
					prov.UpdateSecret(f.MakeRemoteRefKey("gcp-v2-refresh-remote"), framework.SecretEntry{
						Value: `{"value":"gcp-v2-updated"}`,
					})
				},
			})
		}, useV2StaticAuth(prov)),
		framework.Compose(withStaticAuth, f, func(f *framework.Framework) (string, func(*framework.TestCase)) {
			return common.NamespacedProviderFind(f, common.NamespacedProviderFindConfig{
				Description:        "[gcp] should sync dataFrom.find through a namespaced Provider",
				ExternalSecretName: "gcp-v2-find-es",
				TargetSecretName:   "gcp-v2-find-target",
				MatchRegExp:        "^gcp-v2-find-(one|two)$",
				MatchingSecrets: map[string]string{
					"gcp-v2-find-one": "gcp-v2-one",
					"gcp-v2-find-two": "gcp-v2-two",
				},
				IgnoredSecrets: map[string]string{
					"gcp-v2-ignore": "gcp-v2-ignore",
				},
			})
		}, useV2StaticAuth(prov)),
	)
})
