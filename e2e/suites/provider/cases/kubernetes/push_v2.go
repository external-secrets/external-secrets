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
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
)

var _ = Describe("[kubernetes] v2 push secret", Label("kubernetes", "v2", "push-secret"), func() {
	f := framework.New("eso-kubernetes-v2-push")
	prov := NewProvider(f)
	harness := newKubernetesClusterProviderPushHarness(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
		frameworkv2.WaitForProviderConnectionReady(f, f.Namespace.Name, f.Namespace.Name, defaultV2WaitTimeout)
	})

	DescribeTable("push secret",
		framework.TableFuncWithPushSecret(f, prov, nil),
		Entry(common.PushSecretPreservesSourceMetadata(f)),
		Entry(common.PushSecretImplicitProviderKind(f)),
		Entry(common.PushSecretRejectsNamespacedRemoteNamespaceOverride(f)),
		Entry(common.ClusterProviderPushManifestNamespace(f, harness)),
		Entry(common.ClusterProviderPushProviderNamespace(f, harness)),
		Entry(common.ClusterProviderPushManifestNamespaceRecovery(f, harness)),
		Entry(common.ClusterProviderPushProviderNamespaceRecovery(f, harness)),
		Entry(common.ClusterProviderPushAllowsRemoteNamespaceOverride(f, harness)),
		Entry(common.ClusterProviderPushDeniedByConditions(f, harness)),
	)
})

func newKubernetesClusterProviderPushHarness(f *framework.Framework) common.ClusterProviderPushHarness {
	return common.ClusterProviderPushHarness{
		Prepare: func(tc *framework.TestCase, cfg common.ClusterProviderConfig) *common.ClusterProviderPushRuntime {
			s := newClusterProviderV2Scenario(f, cfg.Name)
			s.allowRemoteAccessForScope(cfg.AuthScope, cfg.Name)

			clusterProviderName := s.createClusterProvider(cfg.Name, cfg.AuthScope, cfg.Conditions)
			frameworkv2.WaitForClusterProviderReady(f, clusterProviderName, defaultV2WaitTimeout)

			// Kubernetes push harness supports all optional ClusterProvider push capabilities.
			return &common.ClusterProviderPushRuntime{
				ClusterProviderName:    clusterProviderName,
				DefaultRemoteNamespace: s.remoteNamespace,
				BreakAuth: func() {
					updateKubernetesProviderServiceAccount(f, s.providerNamespace, s.providerConfigName(cfg.Name), "missing-service-account")
				},
				RepairAuth: func() {
					updateKubernetesProviderServiceAccount(f, s.providerNamespace, s.providerConfigName(cfg.Name), s.serviceAccount)
				},
				WaitForRemoteSecretValue: func(namespace, name, key, expectedValue string) {
					waitForSecretValueInNamespace(f, namespace, name, key, expectedValue)
				},
				ExpectNoRemoteSecret: func(namespace, name string) {
					expectNoSecretInNamespace(f, namespace, name)
				},
				CreateWritableRemoteScope: func(prefix string) string {
					namespace := createE2ENamespace(f, prefix)
					frameworkv2.CreateKubernetesAccessRole(
						f,
						fmt.Sprintf("%s-access-%s", s.namePrefix, prefix),
						s.serviceAccount,
						s.workloadNamespace,
						namespace,
					)
					return namespace
				},
			}
		},
	}
}

func waitForSecretValueInNamespace(f *framework.Framework, namespace, name, key, expectedValue string) {
	Eventually(func(g Gomega) {
		var secret corev1.Secret
		g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, &secret)).To(Succeed())
		g.Expect(secret.Data).To(HaveKeyWithValue(key, []byte(expectedValue)))
	}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())
}

func expectNoSecretInNamespace(f *framework.Framework, namespace, name string) {
	Consistently(func() bool {
		var secret corev1.Secret
		err := f.CRClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, &secret)
		return apierrors.IsNotFound(err)
	}, 10*time.Second, defaultV2PollInterval).Should(BeTrue())
}
