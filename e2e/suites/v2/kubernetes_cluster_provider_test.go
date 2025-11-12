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
)

var _ = Describe("V2 ClusterProvider Tests", Label("v2", "cluster-provider", "e2e"), func() {
	f := framework.New("v2-cluster-provider")

	Describe("Kubernetes ClusterProvider", func() {
		const (
			sourceSecretName = "source-secret-cluster"
			targetSecretName = "target-secret-cluster"
		)

		var (
			sourceNamespace *corev1.Namespace
			targetNamespaceA *corev1.Namespace
			targetNamespaceB *corev1.Namespace
		)

		BeforeEach(func() {
			sourceNamespace = SetupTestNamespace(f, "v2-cluster-source-")
			targetNamespaceA = SetupTestNamespace(f, "v2-cluster-target-a-")
			targetNamespaceB = SetupTestNamespace(f, "v2-cluster-target-b-")

			// Create RBAC roles for provider access
			// For ClusterProvider with ProviderNamespace scope, the service account
			// in sourceNamespace needs to access secrets in sourceNamespace
			CreateProviderSecretWriterRole(f, sourceNamespace.Name, sourceNamespace.Name)

			// Create source secret
			sourceSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sourceSecretName,
					Namespace: sourceNamespace.Name,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"username": []byte("cluster-admin"),
					"password": []byte("cluster-secret-password"),
				},
			}
			Expect(f.CRClient.Create(context.Background(), sourceSecret)).To(Succeed())
		})

		AfterEach(func() {
			if sourceNamespace != nil {
				Expect(f.CRClient.Delete(context.Background(), sourceNamespace)).To(Succeed())
			}
			if targetNamespaceA != nil {
				Expect(f.CRClient.Delete(context.Background(), targetNamespaceA)).To(Succeed())
			}
			if targetNamespaceB != nil {
				Expect(f.CRClient.Delete(context.Background(), targetNamespaceB)).To(Succeed())
			}
		})

		It("should sync secrets with ProviderNamespace authentication scope", func() {
			caBundle := GetClusterCABundle(f)
			k8sStore := CreateKubernetes(f, sourceNamespace.Name, "k8s-store-cluster", sourceNamespace.Name, caBundle)

			By("creating a ClusterProvider with ProviderNamespace authentication scope")
			CreateClusterProvider(f, "cluster-k8s-provider",
				"provider-kubernetes.external-secrets-system.svc:8080",
				"provider.external-secrets.io/v2alpha1",
				"Kubernetes",
				k8sStore.Name,
				sourceNamespace.Name,
				esv1.AuthenticationScopeProviderNamespace,
				nil)

			WaitForClusterProviderReady(f, "cluster-k8s-provider", 10*time.Second)

			By("creating an ExternalSecret in target namespace A")
			externalSecretA := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-es-cluster-a",
					Namespace: targetNamespaceA.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Kind: "ClusterProvider",
						Name: "cluster-k8s-provider",
					},
					Target: esv1.ExternalSecretTarget{
						Name:           targetSecretName,
						CreationPolicy: esv1.CreatePolicyOwner,
					},
					RefreshInterval: &metav1.Duration{Duration: 1 * time.Hour},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "username",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key:      sourceSecretName,
								Property: "username",
							},
						},
						{
							SecretKey: "password",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key:      sourceSecretName,
								Property: "password",
							},
						},
					},
				},
			}
			Expect(f.CRClient.Create(context.Background(), externalSecretA)).To(Succeed())

			By("waiting for ExternalSecret A to sync")
			Eventually(func() bool {
				var es esv1.ExternalSecret
				err := f.CRClient.Get(context.Background(),
					types.NamespacedName{Name: "test-es-cluster-a", Namespace: targetNamespaceA.Name},
					&es)
				if err != nil {
					return false
				}

				for _, condition := range es.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == corev1.ConditionTrue {
						return true
					}
				}
				return false
			}, 15*time.Second, 2*time.Second).Should(BeTrue(), "ExternalSecret should become ready")

			By("verifying the synced secret in namespace A")
			var targetSecret corev1.Secret
			Expect(f.CRClient.Get(context.Background(),
				types.NamespacedName{Name: targetSecretName, Namespace: targetNamespaceA.Name},
				&targetSecret)).To(Succeed())

			Expect(targetSecret.Data).To(HaveKeyWithValue("username", []byte("cluster-admin")))
			Expect(targetSecret.Data).To(HaveKeyWithValue("password", []byte("cluster-secret-password")))
		})

		It("should sync secrets with ManifestNamespace authentication scope", func() {
			caBundle := GetClusterCABundle(f)

			// For ManifestNamespace scope, each namespace authenticates as itself
			// Create RBAC for target namespace B
			CreateProviderSecretWriterRole(f, targetNamespaceB.Name, targetNamespaceB.Name)

			// Create a Kubernetes provider in any namespace (we'll use target A) with remoteNamespace set to B
			CreateKubernetes(f, targetNamespaceA.Name, "k8s-store", targetNamespaceB.Name, caBundle)

			// Create secret in namespace B (where we'll read from)
			secretB := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-b",
					Namespace: targetNamespaceB.Name,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"data": []byte("from-namespace-b"),
				},
			}
			Expect(f.CRClient.Create(context.Background(), secretB)).To(Succeed())

			By("creating a ClusterProvider with ManifestNamespace authentication scope")
			// Point to Kubernetes provider in namespace A, but use ManifestNamespace auth
			// This means auth will use namespace B's service account, which has RBAC in namespace B
			CreateClusterProvider(f, "cluster-k8s-manifest-scope",
				"provider-kubernetes.external-secrets-system.svc:8080",
				"provider.external-secrets.io/v2alpha1",
				"Kubernetes",
				"k8s-store",
				targetNamespaceA.Name,
				esv1.AuthenticationScopeManifestNamespace,
				nil)

			WaitForClusterProviderReady(f, "cluster-k8s-manifest-scope", 10*time.Second)

			By("creating ExternalSecret in namespace B")
			// Should authenticate using namespace B's credentials and access secrets in namespace B
			externalSecretB := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-es-manifest-scope",
					Namespace: targetNamespaceB.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Kind: "ClusterProvider",
						Name: "cluster-k8s-manifest-scope",
					},
					Target: esv1.ExternalSecretTarget{
						Name:           "synced-secret-b",
						CreationPolicy: esv1.CreatePolicyOwner,
					},
					RefreshInterval: &metav1.Duration{Duration: 1 * time.Hour},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "data",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key:      "secret-b",
								Property: "data",
							},
						},
					},
				},
			}
			Expect(f.CRClient.Create(context.Background(), externalSecretB)).To(Succeed())

			By("waiting for ExternalSecret B to sync")
			Eventually(func() bool {
				var es esv1.ExternalSecret
				err := f.CRClient.Get(context.Background(),
					types.NamespacedName{Name: "test-es-manifest-scope", Namespace: targetNamespaceB.Name},
					&es)
				if err != nil {
					return false
				}

				for _, condition := range es.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == corev1.ConditionTrue {
						return true
					}
				}
				return false
			}, 15*time.Second, 2*time.Second).Should(BeTrue(), "ExternalSecret should become ready")

			By("verifying the synced secret has data from namespace B")
			var syncedSecret corev1.Secret
			Expect(f.CRClient.Get(context.Background(),
				types.NamespacedName{Name: "synced-secret-b", Namespace: targetNamespaceB.Name},
				&syncedSecret)).To(Succeed())

			Expect(syncedSecret.Data).To(HaveKeyWithValue("data", []byte("from-namespace-b")))
		})

		It("should enforce namespace conditions", func() {
			caBundle := GetClusterCABundle(f)
			k8sStore := CreateKubernetes(f, sourceNamespace.Name, "k8s-store-conditions", sourceNamespace.Name, caBundle)

			// Add label to namespace A
			targetNamespaceA.Labels = map[string]string{"env": "prod"}
			Expect(f.CRClient.Update(context.Background(), targetNamespaceA)).To(Succeed())

			By("creating a ClusterProvider with namespace selector")
			conditions := []esv1.ClusterSecretStoreCondition{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"env": "prod"},
					},
				},
			}
			CreateClusterProvider(f, "cluster-k8s-conditions",
				"provider-kubernetes.external-secrets-system.svc:8080",
				"provider.external-secrets.io/v2alpha1",
				"Kubernetes",
				k8sStore.Name,
				sourceNamespace.Name,
				esv1.AuthenticationScopeProviderNamespace,
				conditions)

			WaitForClusterProviderReady(f, "cluster-k8s-conditions", 10*time.Second)

			By("creating ExternalSecret in matching namespace A")
			esA := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-es-allowed",
					Namespace: targetNamespaceA.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Kind: "ClusterProvider",
						Name: "cluster-k8s-conditions",
					},
					Target: esv1.ExternalSecretTarget{
						Name:           "allowed-secret",
						CreationPolicy: esv1.CreatePolicyOwner,
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "username",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key:      sourceSecretName,
								Property: "username",
							},
						},
					},
				},
			}
			Expect(f.CRClient.Create(context.Background(), esA)).To(Succeed())

			By("verifying ExternalSecret in namespace A succeeds")
			Eventually(func() bool {
				var es esv1.ExternalSecret
				err := f.CRClient.Get(context.Background(),
					types.NamespacedName{Name: "test-es-allowed", Namespace: targetNamespaceA.Name},
					&es)
				if err != nil {
					return false
				}
				for _, condition := range es.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == corev1.ConditionTrue {
						return true
					}
				}
				return false
			}, 15*time.Second, 2*time.Second).Should(BeTrue())

			By("creating ExternalSecret in non-matching namespace B")
			esB := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-es-denied",
					Namespace: targetNamespaceB.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Kind: "ClusterProvider",
						Name: "cluster-k8s-conditions",
					},
					Target: esv1.ExternalSecretTarget{
						Name: "denied-secret",
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "username",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key:      sourceSecretName,
								Property: "username",
							},
						},
					},
				},
			}
			Expect(f.CRClient.Create(context.Background(), esB)).To(Succeed())

			By("verifying ExternalSecret in namespace B fails with condition error")
			// First wait for the ExternalSecret to be reconciled and have a condition
			Eventually(func() bool {
				var es esv1.ExternalSecret
				err := f.CRClient.Get(context.Background(),
					types.NamespacedName{Name: "test-es-denied", Namespace: targetNamespaceB.Name},
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
					types.NamespacedName{Name: "test-es-denied", Namespace: targetNamespaceB.Name},
					&es)
				if err != nil {
					return false
				}
				// Check if it has an error condition
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

