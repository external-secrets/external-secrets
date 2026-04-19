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

package kubernetes

import (
	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("[kubernetes] v2 operational", Serial, Label("kubernetes", "v2", "operational"), func() {
	f := framework.New("eso-kubernetes-v2-operational")
	prov := NewProvider(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
		frameworkv2.WaitForProviderConnectionReady(f, f.Namespace.Name, f.Namespace.Name, defaultV2WaitTimeout)
	})

	DescribeTable("external secret operational behavior",
		framework.TableFuncWithExternalSecret(f, prov),
		Entry(common.NamespacedProviderUnavailable(f, newKubernetesOperationalExternalSecretHarness(f, prov), "kubernetes-operational-unavailable", "recovered")),
		Entry(common.NamespacedProviderRestart(f, newKubernetesOperationalExternalSecretHarness(f, prov), "kubernetes-operational-restart", "restarted")),
		Entry(common.ClusterProviderUnavailable(f, newKubernetesOperationalExternalSecretHarness(f, prov), "kubernetes-operational-cluster-unavailable", "cluster-recovered", esv1.AuthenticationScopeManifestNamespace)),
		Entry(common.ClusterProviderRestart(f, newKubernetesOperationalExternalSecretHarness(f, prov), "kubernetes-operational-cluster-restart", "cluster-restarted", esv1.AuthenticationScopeManifestNamespace)),
	)

	DescribeTable("push secret operational behavior",
		framework.TableFuncWithPushSecret(f, prov, nil),
		Entry(common.NamespacedPushSecretUnavailable(f, newKubernetesOperationalPushHarness(f, prov))),
		Entry(common.ClusterProviderPushUnavailable(f, newKubernetesOperationalPushHarness(f, prov), esv1.AuthenticationScopeManifestNamespace)),
	)
})

func kubernetesBackendTarget() frameworkv2.BackendTarget {
	return frameworkv2.BackendTarget{
		Namespace:        frameworkv2.ProviderNamespace,
		PodLabelSelector: "app.kubernetes.io/name=external-secrets-provider-kubernetes",
	}
}

func newKubernetesOperationalExternalSecretHarness(f *framework.Framework, prov *Provider) common.OperationalExternalSecretHarness {
	return common.OperationalExternalSecretHarness{
		PrepareNamespaced: func(_ *framework.TestCase) *common.OperationalRuntime {
			return &common.OperationalRuntime{
				Provider: prov,
				ProviderRef: esv1.SecretStoreRef{
					Name: f.Namespace.Name,
					Kind: esv1.ProviderStoreKindStr,
				},
				MakeUnavailable: func() {
					frameworkv2.ScaleDeploymentBySelector(f, kubernetesBackendTarget(), 0)
				},
				RestoreAvailability: func() {
					frameworkv2.ScaleDeploymentBySelector(f, kubernetesBackendTarget(), 1)
				},
				RestartBackend: func() {
					frameworkv2.DeleteOneProviderPodBySelector(f, kubernetesBackendTarget())
				},
			}
		},
		PrepareCluster: func(_ *framework.TestCase, cfg common.ClusterProviderConfig) *common.OperationalRuntime {
			s := newClusterProviderV2Scenario(f, cfg.Name, cfg.AuthScope)
			s.allowRemoteAccessForScope(cfg.AuthScope, cfg.Name)

			clusterProviderName := s.createClusterProvider(cfg.Name, cfg.AuthScope, cfg.Conditions)
			frameworkv2.WaitForClusterProviderReady(f, clusterProviderName, defaultV2WaitTimeout)

			return &common.OperationalRuntime{
				Provider: s,
				ProviderRef: esv1.SecretStoreRef{
					Name: clusterProviderName,
					Kind: esv1.ClusterProviderStoreKindStr,
				},
				MakeUnavailable: func() {
					frameworkv2.ScaleDeploymentBySelector(f, kubernetesBackendTarget(), 0)
				},
				RestoreAvailability: func() {
					frameworkv2.ScaleDeploymentBySelector(f, kubernetesBackendTarget(), 1)
				},
				RestartBackend: func() {
					frameworkv2.DeleteOneProviderPodBySelector(f, kubernetesBackendTarget())
				},
			}
		},
	}
}

func newKubernetesOperationalPushHarness(f *framework.Framework, prov *Provider) common.OperationalPushSecretHarness {
	return common.OperationalPushSecretHarness{
		PrepareNamespaced: func(_ *framework.TestCase) *common.OperationalRuntime {
			return &common.OperationalRuntime{
				Provider: prov,
				ProviderRef: esv1.SecretStoreRef{
					Name: f.Namespace.Name,
					Kind: esv1.ProviderStoreKindStr,
				},
				DefaultRemoteNamespace: f.Namespace.Name,
				WaitForRemoteSecret: func(namespace, name, key, expectedValue string) {
					waitForSecretValueInNamespace(f, namespace, name, key, expectedValue)
				},
				ExpectNoRemoteSecret: func(namespace, name string) {
					expectNoSecretInNamespace(f, namespace, name)
				},
				MakeUnavailable: func() {
					frameworkv2.ScaleDeploymentBySelector(f, kubernetesBackendTarget(), 0)
				},
				RestoreAvailability: func() {
					frameworkv2.ScaleDeploymentBySelector(f, kubernetesBackendTarget(), 1)
				},
				RestartBackend: func() {
					frameworkv2.DeleteOneProviderPodBySelector(f, kubernetesBackendTarget())
				},
			}
		},
		PrepareCluster: func(_ *framework.TestCase, cfg common.ClusterProviderConfig) *common.OperationalRuntime {
			s := newClusterProviderV2Scenario(f, cfg.Name, cfg.AuthScope)
			s.allowRemoteAccessForScope(cfg.AuthScope, cfg.Name)

			clusterProviderName := s.createClusterProvider(cfg.Name, cfg.AuthScope, cfg.Conditions)
			frameworkv2.WaitForClusterProviderReady(f, clusterProviderName, defaultV2WaitTimeout)

			return &common.OperationalRuntime{
				Provider: s,
				ProviderRef: esv1.SecretStoreRef{
					Name: clusterProviderName,
					Kind: esv1.ClusterProviderStoreKindStr,
				},
				DefaultRemoteNamespace: s.remoteNamespace,
				WaitForRemoteSecret: func(namespace, name, key, expectedValue string) {
					waitForSecretValueInNamespace(f, namespace, name, key, expectedValue)
				},
				ExpectNoRemoteSecret: func(namespace, name string) {
					expectNoSecretInNamespace(f, namespace, name)
				},
				MakeUnavailable: func() {
					frameworkv2.ScaleDeploymentBySelector(f, kubernetesBackendTarget(), 0)
				},
				RestoreAvailability: func() {
					frameworkv2.ScaleDeploymentBySelector(f, kubernetesBackendTarget(), 1)
				},
				RestartBackend: func() {
					frameworkv2.DeleteOneProviderPodBySelector(f, kubernetesBackendTarget())
				},
			}
		},
	}
}
