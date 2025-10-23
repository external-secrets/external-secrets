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

var _ = Describe("V2 End-to-End Tests", Label("v2", "e2e"), func() {
	f := framework.New("v2-e2e")

	Describe("Kubernetes Provider", func() {
		const (
			sourceSecretName = "source-secret"
			targetSecretName = "target-secret"
			secretStoreName  = "kubernetes-secretstore"
		)

		var (
			sourceNamespace *corev1.Namespace
			targetNamespace *corev1.Namespace
		)

		BeforeEach(func() {
			sourceNamespace = SetupTestNamespace(f, "v2-source-")
			targetNamespace = SetupTestNamespace(f, "v2-target-")
			CreateProviderSecretWriterRole(f, targetNamespace.Name, sourceNamespace.Name)

			// Create source secret
			sourceSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sourceSecretName,
					Namespace: sourceNamespace.Name,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"username": []byte("admin"),
					"password": []byte("super-secret-password"),
					"api-key":  []byte("abc123xyz789"),
				},
			}
			Expect(f.CRClient.Create(context.Background(), sourceSecret)).To(Succeed())
		})

		AfterEach(func() {
			// Cleanup namespaces
			if sourceNamespace != nil {
				Expect(f.CRClient.Delete(context.Background(), sourceNamespace)).To(Succeed())
			}
			if targetNamespace != nil {
				Expect(f.CRClient.Delete(context.Background(), targetNamespace)).To(Succeed())
			}
		})

		It("should sync secrets across namespaces", func() {
			caBundle := GetClusterCABundle(f)
			CreateKubernetes(f, targetNamespace.Name, "k8s-store", sourceNamespace.Name, caBundle)
			CreateProvider(f, targetNamespace.Name, secretStoreName, "k8s-store", targetNamespace.Name)
			WaitForProviderConnectionReady(f, targetNamespace.Name, secretStoreName, 5*time.Second)

			By("creating an ExternalSecret")
			externalSecret := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-external-secret",
					Namespace: targetNamespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Kind: "Provider",
						Name: secretStoreName,
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
			Expect(f.CRClient.Create(context.Background(), externalSecret)).To(Succeed())

			By("waiting for ExternalSecret to sync")
			Eventually(func() bool {
				var es esv1.ExternalSecret
				err := f.CRClient.Get(context.Background(),
					types.NamespacedName{Name: "test-external-secret", Namespace: targetNamespace.Name},
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
			}, 10*time.Second, 2*time.Second).Should(BeTrue(), "ExternalSecret should become ready")

			By("verifying the synced secret")
			var targetSecret corev1.Secret
			Expect(f.CRClient.Get(context.Background(),
				types.NamespacedName{Name: targetSecretName, Namespace: targetNamespace.Name},
				&targetSecret)).To(Succeed())

			Expect(targetSecret.Data).To(HaveKeyWithValue("username", []byte("admin")))
			Expect(targetSecret.Data).To(HaveKeyWithValue("password", []byte("super-secret-password")))

			By("verifying ExternalSecret status")
			var es esv1.ExternalSecret
			Expect(f.CRClient.Get(context.Background(),
				types.NamespacedName{Name: "test-external-secret", Namespace: targetNamespace.Name},
				&es)).To(Succeed())

			Expect(es.Status.SyncedResourceVersion).NotTo(BeEmpty())
			Expect(es.Status.RefreshTime).NotTo(BeNil())
			Expect(es.Status.Conditions).NotTo(BeEmpty())
		})

	})
})
