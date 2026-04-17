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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	k8sv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1"
)

var _ = Describe("[kubernetes] v2 namespaced provider", Label("kubernetes", "v2", "namespaced-provider"), func() {
	f := framework.New("eso-kubernetes-v2-provider")
	prov := NewProvider(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
		frameworkv2.WaitForProviderConnectionReady(f, f.Namespace.Name, f.Namespace.Name, defaultV2WaitTimeout)
	})

	DescribeTable("namespaced provider read paths",
		framework.TableFuncWithExternalSecret(f, prov),
		Entry(common.NamespacedProviderSync(f, common.NamespacedProviderSyncConfig{
			Description:        "[kubernetes] should sync an ExternalSecret through a namespaced Provider",
			ExternalSecretName: "provider-v2-es",
			TargetSecretName:   "provider-v2-target",
			RemoteKey:          "provider-v2-remote",
			RemoteSecretValue:  `{"value":"provider-v2-value"}`,
			RemoteProperty:     "value",
			SecretKey:          "value",
			ExpectedValue:      "provider-v2-value",
		})),
		Entry(common.NamespacedProviderRefresh(f, common.NamespacedProviderRefreshConfig{
			Description:         "[kubernetes] should refresh synced secrets after the remote Kubernetes secret changes",
			ExternalSecretName:  "provider-v2-refresh-es",
			TargetSecretName:    "provider-v2-refresh-target",
			RemoteKey:           "provider-v2-refresh-remote",
			InitialSecretValue:  `{"value":"provider-v2-initial"}`,
			UpdatedSecretValue:  `{"value":"provider-v2-updated"}`,
			RemoteProperty:      "value",
			SecretKey:           "value",
			InitialExpectedData: "provider-v2-initial",
			UpdatedExpectedData: "provider-v2-updated",
			RefreshInterval:     defaultV2RefreshInterval,
			WaitTimeout:         30 * time.Second,
			UpdateRemoteSecret: func(_ *framework.TestCase, _ framework.SecretStoreProvider) {
				updateRemoteSecretValue(f, f.Namespace.Name, "provider-v2-refresh-remote", "provider-v2-updated")
			},
		})),
		Entry(common.NamespacedProviderFind(f, common.NamespacedProviderFindConfig{
			Description:        "[kubernetes] should sync ExternalSecret dataFrom.find through a namespaced Provider",
			ExternalSecretName: "provider-v2-find-es",
			TargetSecretName:   "provider-v2-find-target",
			MatchRegExp:        "provider-v2-find-(one|two)",
			MatchingSecrets: map[string]string{
				"provider-v2-find-one": `{"value":"provider-v2-one"}`,
				"provider-v2-find-two": `{"value":"provider-v2-two"}`,
			},
			IgnoredSecrets: map[string]string{
				"provider-v2-ignore": `{"value":"provider-v2-ignore"}`,
			},
		})),
	)

	It("recovers after repairing namespaced Provider auth", func() {
		prov.CreateSecret("provider-v2-recovery-remote", framework.SecretEntry{
			Value: `{"value":"provider-v2-recovered"}`,
		})

		updateKubernetesProviderServiceAccount(f, f.Namespace.Name, providerConfigName(f.Namespace.Name, ""), "missing-service-account")

		brokenProvider := frameworkv2.WaitForProviderConnectionNotReady(f, f.Namespace.Name, f.Namespace.Name, defaultV2WaitTimeout)
		Expect(brokenProvider.Status.Capabilities).To(BeEmpty())

		externalSecret := &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "provider-v2-recovery-es",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1.ExternalSecretSpec{
				SecretStoreRef: esv1.SecretStoreRef{
					Name: f.Namespace.Name,
					Kind: esv1.ProviderKindStr,
				},
				RefreshInterval: &metav1.Duration{Duration: time.Hour},
				Target: esv1.ExternalSecretTarget{
					Name: "provider-v2-recovery-target",
				},
				Data: []esv1.ExternalSecretData{{
					SecretKey: "value",
					RemoteRef: esv1.ExternalSecretDataRemoteRef{
						Key:      "provider-v2-recovery-remote",
						Property: "value",
					},
				}},
			},
		}
		Expect(f.CRClient.Create(context.Background(), externalSecret)).To(Succeed())

		waitForExternalSecretReadyStatus(f, f.Namespace.Name, externalSecret.Name, corev1.ConditionFalse)
		expectSecretToBeAbsent(f, f.Namespace.Name, "provider-v2-recovery-target")

		updateKubernetesProviderServiceAccount(f, f.Namespace.Name, providerConfigName(f.Namespace.Name, ""), frameworkv2.DefaultSAName)

		repairedProvider := frameworkv2.WaitForProviderConnectionReady(f, f.Namespace.Name, f.Namespace.Name, defaultV2WaitTimeout)
		Expect(repairedProvider.Status.Capabilities).To(Equal(esv1.ProviderReadWrite))

		_, err := waitForSecretValueWithin(f, 30*time.Second, f.Namespace.Name, "provider-v2-recovery-target", &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"value": []byte("provider-v2-recovered"),
			},
		})
		Expect(err).NotTo(HaveOccurred())
	})
})

func updateRemoteSecretValue(f *framework.Framework, namespace, name, value string) {
	var secret corev1.Secret
	Expect(f.CRClient.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &secret)).To(Succeed())

	secret.Data["value"] = []byte(value)
	Expect(f.CRClient.Update(context.Background(), &secret)).To(Succeed())
}

func updateKubernetesProviderServiceAccount(f *framework.Framework, namespace, name, serviceAccountName string) {
	var providerConfig k8sv2alpha1.Kubernetes
	Expect(f.CRClient.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &providerConfig)).To(Succeed())

	if providerConfig.Spec.Auth == nil || providerConfig.Spec.Auth.ServiceAccount == nil {
		providerConfig.Spec.Auth = &esv1.KubernetesAuth{
			ServiceAccount: &esmeta.ServiceAccountSelector{},
		}
	}
	providerConfig.Spec.Auth.ServiceAccount.Name = serviceAccountName
	Expect(f.CRClient.Update(context.Background(), &providerConfig)).To(Succeed())
}

func waitForExternalSecretReadyStatus(f *framework.Framework, namespace, name string, expectedStatus corev1.ConditionStatus) {
	Eventually(func(g Gomega) {
		var externalSecret esv1.ExternalSecret
		g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, &externalSecret)).To(Succeed())

		condition := esv1.GetExternalSecretCondition(externalSecret.Status, esv1.ExternalSecretReady)
		g.Expect(condition).NotTo(BeNil())
		g.Expect(condition.Status).To(Equal(expectedStatus))
	}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())
}

func waitForSecretValueWithin(f *framework.Framework, timeout time.Duration, namespace, name string, expected *corev1.Secret) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		err := f.CRClient.Get(context.Background(), types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, secret)
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		match, matchErr := Equal(expected.Data).Match(secret.Data)
		if matchErr != nil {
			return false, matchErr
		}
		return secret.Type == expected.Type && match, nil
	})
	return secret, err
}

func expectSecretToBeAbsent(f *framework.Framework, namespace, name string) {
	Consistently(func() bool {
		var secret corev1.Secret
		err := f.CRClient.Get(context.Background(), types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, &secret)
		return apierrors.IsNotFound(err)
	}, defaultV2RefreshInterval, defaultV2PollInterval).Should(BeTrue())
}
