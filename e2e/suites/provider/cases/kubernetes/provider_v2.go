/*
Copyright © 2026 ESO Maintainer Team

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
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

	It("syncs an ExternalSecret through a namespaced Provider", func() {
		prov.CreateSecret("provider-v2-remote", framework.SecretEntry{
			Value: `{"value":"provider-v2-value"}`,
		})

		externalSecret := &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "provider-v2-es",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1.ExternalSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
				SecretStoreRef: esv1.SecretStoreRef{
					Name: f.Namespace.Name,
					Kind: esv1.ProviderKindStr,
				},
				Target: esv1.ExternalSecretTarget{
					Name: "provider-v2-target",
				},
				Data: []esv1.ExternalSecretData{{
					SecretKey: "value",
					RemoteRef: esv1.ExternalSecretDataRemoteRef{
						Key:      "provider-v2-remote",
						Property: "value",
					},
				}},
			},
		}
		Expect(f.CRClient.Create(context.Background(), externalSecret)).To(Succeed())

		expected := &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"value": []byte("provider-v2-value"),
			},
		}

		_, err := f.WaitForSecretValue(f.Namespace.Name, "provider-v2-target", expected)
		Expect(err).NotTo(HaveOccurred())
	})

	It("refreshes synced secrets after the remote Kubernetes secret changes", func() {
		prov.CreateSecret("provider-v2-refresh-remote", framework.SecretEntry{
			Value: `{"value":"provider-v2-initial"}`,
		})

		externalSecret := &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "provider-v2-refresh-es",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1.ExternalSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
				SecretStoreRef: esv1.SecretStoreRef{
					Name: f.Namespace.Name,
					Kind: esv1.ProviderKindStr,
				},
				Target: esv1.ExternalSecretTarget{
					Name: "provider-v2-refresh-target",
				},
				Data: []esv1.ExternalSecretData{{
					SecretKey: "value",
					RemoteRef: esv1.ExternalSecretDataRemoteRef{
						Key:      "provider-v2-refresh-remote",
						Property: "value",
					},
				}},
			},
		}
		Expect(f.CRClient.Create(context.Background(), externalSecret)).To(Succeed())

		_, err := f.WaitForSecretValue(f.Namespace.Name, "provider-v2-refresh-target", &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"value": []byte("provider-v2-initial"),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		updateRemoteSecretValue(f, f.Namespace.Name, "provider-v2-refresh-remote", "provider-v2-updated")

		_, err = f.WaitForSecretValue(f.Namespace.Name, "provider-v2-refresh-target", &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"value": []byte("provider-v2-updated"),
			},
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("syncs ExternalSecret dataFrom.find through a namespaced Provider", func() {
		secretOne := "provider-v2-find-one"
		secretTwo := "provider-v2-find-two"
		secretThree := "provider-v2-ignore"

		prov.CreateSecret(secretOne, framework.SecretEntry{
			Value: `{"value":"provider-v2-one"}`,
		})
		prov.CreateSecret(secretTwo, framework.SecretEntry{
			Value: `{"value":"provider-v2-two"}`,
		})
		prov.CreateSecret(secretThree, framework.SecretEntry{
			Value: `{"value":"provider-v2-ignore"}`,
		})

		externalSecret := &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "provider-v2-find-es",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1.ExternalSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
				SecretStoreRef: esv1.SecretStoreRef{
					Name: f.Namespace.Name,
					Kind: esv1.ProviderKindStr,
				},
				Target: esv1.ExternalSecretTarget{
					Name: "provider-v2-find-target",
				},
				DataFrom: []esv1.ExternalSecretDataFromRemoteRef{{
					Find: &esv1.ExternalSecretFind{
						Name: &esv1.FindName{
							RegExp: "provider-v2-find-(one|two)",
						},
					},
				}},
			},
		}
		Expect(f.CRClient.Create(context.Background(), externalSecret)).To(Succeed())

		expected := &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretOne: []byte(`{"value":"provider-v2-one"}`),
				secretTwo: []byte(`{"value":"provider-v2-two"}`),
			},
		}

		_, err := f.WaitForSecretValue(f.Namespace.Name, "provider-v2-find-target", expected)
		Expect(err).NotTo(HaveOccurred())
	})

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
				RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
				SecretStoreRef: esv1.SecretStoreRef{
					Name: f.Namespace.Name,
					Kind: esv1.ProviderKindStr,
				},
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

		_, err := f.WaitForSecretValue(f.Namespace.Name, "provider-v2-recovery-target", &corev1.Secret{
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
