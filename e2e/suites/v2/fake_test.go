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

var _ = Describe("[v2] Fake Provider", Label("v2", "fake"), func() {
	f := framework.New("v2-fake-provider")

	Context("GetSecret", func() {
		var testNamespace *corev1.Namespace

		BeforeEach(func() {
			testNamespace = SetupTestNamespace(f, "v2-fake-")

			// Create Fake provider with test data
			CreateFakeProvider(f, testNamespace.Name, "fake-provider", []v1.FakeProviderData{
				{Key: "username", Value: "test-user"},
				{Key: "password", Value: "test-password"},
			})

			// Create ProviderConnection
			CreateFakeProviderConnection(f, testNamespace.Name, "test-secretstore", "fake-provider", testNamespace.Name)
		})

		AfterEach(func() {
			if testNamespace != nil {
				Expect(f.CRClient.Delete(context.Background(), testNamespace)).To(Succeed())
			}
		})

		It("should sync secrets from Fake provider", func() {
			By("waiting for ProviderConnection to be ready")
			WaitForProviderConnectionReady(f, testNamespace.Name, "test-secretstore", 30*time.Second)

			By("creating an ExternalSecret")
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-es",
					Namespace: testNamespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "test-secretstore",
						Kind: "Provider",
					},
					Target: esv1.ExternalSecretTarget{
						Name: "synced-secret",
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "username",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "username",
							},
						},
						{
							SecretKey: "password",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "password",
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
					types.NamespacedName{Name: "synced-secret", Namespace: testNamespace.Name},
					&syncedSecret)
				return err == nil
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			By("verifying the synced secret data")
			Expect(syncedSecret.Data["username"]).To(Equal([]byte("test-user")))
			Expect(syncedSecret.Data["password"]).To(Equal([]byte("test-password")))
		})
	})

	Context("Capabilities", func() {
		var testNamespace *corev1.Namespace

		BeforeEach(func() {
			testNamespace = SetupTestNamespace(f, "v2-fake-capabilities-")

			CreateFakeProvider(f, testNamespace.Name, "fake-provider", []v1.FakeProviderData{
				{Key: "test-key", Value: "test-value"},
			})

			CreateFakeProviderConnection(f, testNamespace.Name, "test-secretstore", "fake-provider", testNamespace.Name)
		})

		AfterEach(func() {
			if testNamespace != nil {
				Expect(f.CRClient.Delete(context.Background(), testNamespace)).To(Succeed())
			}
		})

		It("should report READ_WRITE capabilities", func() {
			By("waiting for ProviderConnection to be ready")
			WaitForProviderConnectionReady(f, testNamespace.Name, "test-secretstore", 30*time.Second)

			By("verifying capabilities")
			VerifyProviderConnectionCapabilities(f, testNamespace.Name, "test-secretstore", esv1.ProviderReadWrite)
		})
	})

	Context("Generator Support", func() {
		var testNamespace *corev1.Namespace

		BeforeEach(func() {
			testNamespace = SetupTestNamespace(f, "v2-fake-generator-")
			CreateFakeProvider(f, testNamespace.Name, "fake-provider", []v1.FakeProviderData{})
		})

		AfterEach(func() {
			if testNamespace != nil {
				Expect(f.CRClient.Delete(context.Background(), testNamespace)).To(Succeed())
			}
		})

		It("should generate secrets from Fake generator", func() {
			By("creating a Fake generator")
			CreateFakeGenerator(f, testNamespace.Name, "test-generator", map[string]string{
				"username": "generated-user",
				"password": "generated-password",
				"token":    "generated-token",
			})

			By("creating a ProviderConnection to the fake provider for generator support")
			CreateFakeProviderConnection(f, testNamespace.Name, "fake-generator-connection", "fake-provider", testNamespace.Name)

			By("waiting for ProviderConnection to be ready")
			WaitForProviderConnectionReady(f, testNamespace.Name, "fake-generator-connection", 30*time.Second)

			By("creating an ExternalSecret with dataFrom referencing the generator")
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-es-generator",
					Namespace: testNamespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "fake-generator-connection",
						Kind: "Provider",
					},
					Target: esv1.ExternalSecretTarget{
						Name: "generated-secret",
					},
					DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &esv1.StoreGeneratorSourceRef{
								GeneratorRef: &esv1.GeneratorRef{
									APIVersion: "generators.external-secrets.io/v1alpha1",
									Kind:       "Fake",
									Name:       "test-generator",
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
					types.NamespacedName{Name: "generated-secret", Namespace: testNamespace.Name},
					&syncedSecret)
				return err == nil
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			By("verifying the generated secret data")
			Expect(syncedSecret.Data["username"]).To(Equal([]byte("generated-user")))
			Expect(syncedSecret.Data["password"]).To(Equal([]byte("generated-password")))
			Expect(syncedSecret.Data["token"]).To(Equal([]byte("generated-token")))
		})

		It("should generate secrets with rewrite rules", func() {
			By("creating a Fake generator")
			CreateFakeGenerator(f, testNamespace.Name, "test-generator-rewrite", map[string]string{
				"db-host": "localhost",
				"db-port": "5432",
				"db-name": "mydb",
			})

			By("creating a ProviderConnection to the fake provider for generator support")
			CreateFakeProviderConnection(f, testNamespace.Name, "fake-generator-connection-rewrite", "fake-provider", testNamespace.Name)

			By("waiting for ProviderConnection to be ready")
			WaitForProviderConnectionReady(f, testNamespace.Name, "fake-generator-connection-rewrite", 30*time.Second)

			By("creating an ExternalSecret with rewrite rules")
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-es-generator-rewrite",
					Namespace: testNamespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "fake-generator-connection-rewrite",
						Kind: "Provider",
					},
					Target: esv1.ExternalSecretTarget{
						Name: "generated-secret-rewrite",
					},
					DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &esv1.StoreGeneratorSourceRef{
								GeneratorRef: &esv1.GeneratorRef{
									APIVersion: "generators.external-secrets.io/v1alpha1",
									Kind:       "Fake",
									Name:       "test-generator-rewrite",
								},
							},
							Rewrite: []esv1.ExternalSecretRewrite{
								{
									Regexp: &esv1.ExternalSecretRewriteRegexp{
										Source: "db-(.*)",
										Target: "database_$1",
									},
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
					types.NamespacedName{Name: "generated-secret-rewrite", Namespace: testNamespace.Name},
					&syncedSecret)
				return err == nil
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			By("verifying the rewritten secret data")
			Expect(syncedSecret.Data["database_host"]).To(Equal([]byte("localhost")))
			Expect(syncedSecret.Data["database_port"]).To(Equal([]byte("5432")))
			Expect(syncedSecret.Data["database_name"]).To(Equal([]byte("mydb")))
		})
	})
})
