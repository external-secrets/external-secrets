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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
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
})
