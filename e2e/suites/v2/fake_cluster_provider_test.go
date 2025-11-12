/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var _ = Describe("[v2] Fake ClusterProvider", Label("v2", "fake", "cluster-provider"), func() {
	f := framework.New("v2-fake-cluster-provider")

	Context("GetSecret with ClusterProvider", func() {
		var testNamespace *corev1.Namespace

		BeforeEach(func() {
			testNamespace = SetupTestNamespace(f, "v2-fake-cluster-")

			// Create Fake provider with test data
			CreateFakeProvider(f, testNamespace.Name, "fake-provider-cluster", []v1.FakeProviderData{
				{Key: "cluster-username", Value: "cluster-user"},
				{Key: "cluster-password", Value: "cluster-password"},
				{Key: "cluster-token", Value: "cluster-token-12345"},
			})
		})

		AfterEach(func() {
			if testNamespace != nil {
				Expect(f.CRClient.Delete(context.Background(), testNamespace)).To(Succeed())
			}
		})

		It("should sync secrets from ClusterProvider", func() {
			By("creating a ClusterProvider pointing to Fake provider")
			CreateClusterProvider(f, "cluster-fake-provider",
				"provider-fake.external-secrets-system.svc:8080",
				"provider.external-secrets.io/v2alpha1",
				"Fake",
				"fake-provider-cluster",
				testNamespace.Name,
				esv1.AuthenticationScopeProviderNamespace,
				nil)

			By("waiting for ClusterProvider to be ready")
			WaitForClusterProviderReady(f, "cluster-fake-provider", 30*time.Second)

			By("creating an ExternalSecret")
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-es-cluster",
					Namespace: testNamespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "cluster-fake-provider",
						Kind: "ClusterProvider",
					},
					Target: esv1.ExternalSecretTarget{
						Name: "synced-cluster-secret",
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "username",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "cluster-username",
							},
						},
						{
							SecretKey: "password",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "cluster-password",
							},
						},
						{
							SecretKey: "token",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "cluster-token",
							},
						},
					},
				},
			}
			Expect(f.CRClient.Create(context.Background(), es)).To(Succeed())

			By("waiting for secret to be synced")
			var syncedSecret corev1.Secret
			Eventually(func() bool {
				err := f.CRClient.Get(context.Background(),
					types.NamespacedName{Name: "synced-cluster-secret", Namespace: testNamespace.Name},
					&syncedSecret)
				return err == nil
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			By("verifying the synced secret data")
			Expect(syncedSecret.Data["username"]).To(Equal([]byte("cluster-user")))
			Expect(syncedSecret.Data["password"]).To(Equal([]byte("cluster-password")))
			Expect(syncedSecret.Data["token"]).To(Equal([]byte("cluster-token-12345")))
		})

		It("should work from multiple namespaces", func() {
			testNamespace2 := SetupTestNamespace(f, "v2-fake-cluster-2-")
			defer func() {
				Expect(f.CRClient.Delete(context.Background(), testNamespace2)).To(Succeed())
			}()

			By("creating a ClusterProvider")
			CreateClusterProvider(f, "cluster-fake-multi-ns",
				"provider-fake.external-secrets-system.svc:8080",
				"provider.external-secrets.io/v2alpha1",
				"Fake",
				"fake-provider-cluster",
				testNamespace.Name,
				esv1.AuthenticationScopeProviderNamespace,
				nil)

			WaitForClusterProviderReady(f, "cluster-fake-multi-ns", 30*time.Second)

			By("creating ExternalSecrets in both namespaces")
			for _, ns := range []string{testNamespace.Name, testNamespace2.Name} {
				es := &esv1.ExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-es-multi",
						Namespace: ns,
					},
					Spec: esv1.ExternalSecretSpec{
						SecretStoreRef: esv1.SecretStoreRef{
							Name: "cluster-fake-multi-ns",
							Kind: "ClusterProvider",
						},
						Target: esv1.ExternalSecretTarget{
							Name: "multi-ns-secret",
						},
						Data: []esv1.ExternalSecretData{
							{
								SecretKey: "username",
								RemoteRef: esv1.ExternalSecretDataRemoteRef{
									Key: "cluster-username",
								},
							},
						},
					},
				}
				Expect(f.CRClient.Create(context.Background(), es)).To(Succeed())
			}

			By("verifying secrets are synced in both namespaces")
			for _, ns := range []string{testNamespace.Name, testNamespace2.Name} {
				var syncedSecret corev1.Secret
				Eventually(func() bool {
					err := f.CRClient.Get(context.Background(),
						types.NamespacedName{Name: "multi-ns-secret", Namespace: ns},
						&syncedSecret)
					return err == nil
				}, 30*time.Second, 1*time.Second).Should(BeTrue(), "Secret should be synced in namespace "+ns)

				Expect(syncedSecret.Data["username"]).To(Equal([]byte("cluster-user")))
			}
		})
	})

	Context("Generator Support with ClusterProvider", func() {
		var testNamespace *corev1.Namespace

		BeforeEach(func() {
			testNamespace = SetupTestNamespace(f, "v2-fake-cluster-gen-")
			CreateFakeProvider(f, testNamespace.Name, "fake-provider-gen", []v1.FakeProviderData{})
		})

		AfterEach(func() {
			if testNamespace != nil {
				Expect(f.CRClient.Delete(context.Background(), testNamespace)).To(Succeed())
			}
		})

		It("should generate secrets from Fake generator with ClusterProvider", func() {
			By("creating a Fake generator")
			CreateFakeGenerator(f, testNamespace.Name, "test-cluster-generator", map[string]string{
				"gen-username": "generated-cluster-user",
				"gen-password": "generated-cluster-password",
				"gen-api-key":  "generated-cluster-api-key",
			})

			By("creating a ClusterProvider for generator support")
			CreateClusterProvider(f, "cluster-fake-generator",
				"provider-fake.external-secrets-system.svc:8080",
				"provider.external-secrets.io/v2alpha1",
				"Fake",
				"fake-provider-gen",
				testNamespace.Name,
				esv1.AuthenticationScopeProviderNamespace,
				nil)

			WaitForClusterProviderReady(f, "cluster-fake-generator", 30*time.Second)

			By("creating an ExternalSecret with dataFrom referencing the generator")
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-es-cluster-generator",
					Namespace: testNamespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "cluster-fake-generator",
						Kind: "ClusterProvider",
					},
					Target: esv1.ExternalSecretTarget{
						Name: "cluster-generated-secret",
					},
					DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &esv1.StoreGeneratorSourceRef{
								GeneratorRef: &esv1.GeneratorRef{
									APIVersion: "generators.external-secrets.io/v1alpha1",
									Kind:       "Fake",
									Name:       "test-cluster-generator",
								},
							},
						},
					},
				},
			}
			Expect(f.CRClient.Create(context.Background(), es)).To(Succeed())

			By("waiting for secret to be synced")
			var syncedSecret corev1.Secret
			Eventually(func() bool {
				err := f.CRClient.Get(context.Background(),
					types.NamespacedName{Name: "cluster-generated-secret", Namespace: testNamespace.Name},
					&syncedSecret)
				return err == nil
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			By("verifying the generated secret data")
			Expect(syncedSecret.Data["gen-username"]).To(Equal([]byte("generated-cluster-user")))
			Expect(syncedSecret.Data["gen-password"]).To(Equal([]byte("generated-cluster-password")))
			Expect(syncedSecret.Data["gen-api-key"]).To(Equal([]byte("generated-cluster-api-key")))
		})
	})

	Context("Namespace Conditions", func() {
		var (
			testNamespaceAllowed *corev1.Namespace
			testNamespaceDenied  *corev1.Namespace
		)

		BeforeEach(func() {
			testNamespaceAllowed = SetupTestNamespace(f, "v2-fake-cluster-allowed-")
			testNamespaceDenied = SetupTestNamespace(f, "v2-fake-cluster-denied-")

			// Label the allowed namespace
			testNamespaceAllowed.Labels = map[string]string{"team": "platform"}
			Expect(f.CRClient.Update(context.Background(), testNamespaceAllowed)).To(Succeed())

			// Create Fake provider
			CreateFakeProvider(f, testNamespaceAllowed.Name, "fake-provider-conditions", []v1.FakeProviderData{
				{Key: "test-key", Value: "test-value"},
			})
		})

		AfterEach(func() {
			if testNamespaceAllowed != nil {
				Expect(f.CRClient.Delete(context.Background(), testNamespaceAllowed)).To(Succeed())
			}
			if testNamespaceDenied != nil {
				Expect(f.CRClient.Delete(context.Background(), testNamespaceDenied)).To(Succeed())
			}
		})

		It("should enforce namespace label selectors", func() {
			By("creating a ClusterProvider with namespace selector")
			conditions := []esv1.ClusterSecretStoreCondition{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"team": "platform"},
					},
				},
			}
			CreateClusterProvider(f, "cluster-fake-labeled",
				"provider-fake.external-secrets-system.svc:8080",
				"provider.external-secrets.io/v2alpha1",
				"Fake",
				"fake-provider-conditions",
				testNamespaceAllowed.Name,
				esv1.AuthenticationScopeProviderNamespace,
				conditions)

			WaitForClusterProviderReady(f, "cluster-fake-labeled", 30*time.Second)

			By("creating ExternalSecret in allowed namespace")
			esAllowed := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-es-allowed",
					Namespace: testNamespaceAllowed.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "cluster-fake-labeled",
						Kind: "ClusterProvider",
					},
					Target: esv1.ExternalSecretTarget{
						Name: "allowed-secret",
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "test-key",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "test-key",
							},
						},
					},
				},
			}
			Expect(f.CRClient.Create(context.Background(), esAllowed)).To(Succeed())

			By("verifying ExternalSecret in allowed namespace succeeds")
			var allowedSecret corev1.Secret
			Eventually(func() bool {
				err := f.CRClient.Get(context.Background(),
					types.NamespacedName{Name: "allowed-secret", Namespace: testNamespaceAllowed.Name},
					&allowedSecret)
				return err == nil
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			Expect(allowedSecret.Data["test-key"]).To(Equal([]byte("test-value")))

			By("creating ExternalSecret in denied namespace")
			esDenied := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-es-denied",
					Namespace: testNamespaceDenied.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "cluster-fake-labeled",
						Kind: "ClusterProvider",
					},
					Target: esv1.ExternalSecretTarget{
						Name: "denied-secret",
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "test-key",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "test-key",
							},
						},
					},
				},
			}
			Expect(f.CRClient.Create(context.Background(), esDenied)).To(Succeed())

			By("verifying ExternalSecret in denied namespace fails")
			// First wait for the ExternalSecret to be reconciled and have a condition
			Eventually(func() bool {
				var es esv1.ExternalSecret
				err := f.CRClient.Get(context.Background(),
					types.NamespacedName{Name: "test-es-denied", Namespace: testNamespaceDenied.Name},
					&es)
				if err != nil {
					return false
				}
				return len(es.Status.Conditions) > 0
			}, 10*time.Second, 1*time.Second).Should(BeTrue(), "ExternalSecret should have conditions")

			// Then verify it stays in error state
			Consistently(func() bool {
				var es esv1.ExternalSecret
				err := f.CRClient.Get(context.Background(),
					types.NamespacedName{Name: "test-es-denied", Namespace: testNamespaceDenied.Name},
					&es)
				if err != nil {
					return false
				}
				// Check for error condition
				for _, condition := range es.Status.Conditions {
					if condition.Type == "Ready" {
						// Should be False (error state)
						return condition.Status == corev1.ConditionFalse
					}
				}
				return false
			}, 5*time.Second, 1*time.Second).Should(BeTrue(), "ExternalSecret should have error condition")
		})
	})
})

