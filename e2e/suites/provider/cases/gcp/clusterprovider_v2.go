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
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	gcpsmv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/gcp/v2alpha1"
)

var _ = Describe("[gcp] v2 cluster provider", Label("gcp", "secretsmanager", "v2", "cluster-provider"), func() {
	f := framework.New("eso-gcp-v2-clusterprovider")
	prov := NewProviderV2(f)
	harness := newGCPClusterProviderExternalSecretHarness(f, prov)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
	})

	DescribeTable("cluster provider external secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		Entry(common.ClusterProviderManifestNamespace(f, harness)),
		Entry(common.ClusterProviderProviderNamespace(f, harness)),
		Entry(common.ClusterProviderDeniedByConditions(f, harness)),
	)
})

type clusterProviderScenario struct {
	common  v2ClusterProviderScenario
	access  gcpAccessConfig
	backend *GcpProvider
	f       *framework.Framework
}

func newClusterProviderScenario(f *framework.Framework, prefix string, authScope esv1.AuthenticationScope, access gcpAccessConfig, backend *GcpProvider) *clusterProviderScenario {
	shared := newV2ClusterProviderScenario(f.Namespace.Name, prefix, authScope, func(prefix string) string {
		return common.CreateProviderCaseNamespace(f, prefix, defaultV2PollInterval)
	})
	s := &clusterProviderScenario{
		common:  shared,
		access:  access,
		backend: backend,
		f:       f,
	}
	createSecretManagerV2StaticConfig(s.f, s.common.ConfigNamespace, s.common.ConfigName, s.access)
	return s
}

func (s *clusterProviderScenario) createClusterProvider(conditions []esv1.ClusterSecretStoreCondition) string {
	clusterProviderName := s.common.ClusterProviderName()
	frameworkv2.CreateClusterProviderConnection(
		s.f,
		clusterProviderName,
		frameworkv2.ProviderAddress("gcp"),
		gcpsmv2alpha1.GroupVersion.String(),
		gcpsmv2alpha1.SecretManagerKind,
		s.common.ConfigName,
		s.common.ProviderRefNamespace,
		s.common.AuthScope,
		conditions,
	)
	return clusterProviderName
}

func (s *clusterProviderScenario) CreateSecret(key string, val framework.SecretEntry) {
	s.backend.CreateSecret(key, val)
}

func (s *clusterProviderScenario) DeleteSecret(key string) {
	s.backend.DeleteSecret(key)
}

func newGCPClusterProviderExternalSecretHarness(f *framework.Framework, prov *ProviderV2) common.ClusterProviderExternalSecretHarness {
	return common.ClusterProviderExternalSecretHarness{
		Prepare: func(_ *framework.TestCase, cfg common.ClusterProviderConfig) *common.ClusterProviderExternalSecretRuntime {
			s := newClusterProviderScenario(f, cfg.Name, cfg.AuthScope, prov.access, prov.backend)
			clusterProviderName := s.createClusterProvider(cfg.Conditions)
			frameworkv2.WaitForClusterProviderReady(f, clusterProviderName, defaultV2WaitTimeout)

			return &common.ClusterProviderExternalSecretRuntime{
				ClusterProviderName: clusterProviderName,
				Provider:            s,
			}
		},
	}
}
