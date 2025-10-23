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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/log"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

var _ = Describe("[v2] PushSecret", Label("v2", "kubernetes", "push-secret"), func() {
	f := framework.New("eso-v2-push-secret")

	var (
		testNamespace *corev1.Namespace
	)

	BeforeEach(func() {
		testNamespace = SetupTestNamespace(f, "v2-push-secret-")
		CreateProviderSecretWriterRole(f, testNamespace.Name, testNamespace.Name)
	})

	AfterEach(func() {
		// Cleanup namespace
		if testNamespace != nil {
			Expect(f.CRClient.Delete(context.Background(), testNamespace)).To(Succeed())
		}
	})

	It("should push secret to Kubernetes provider", func() {
		caBundle := GetClusterCABundle(f)
		CreateKubernetes(f, testNamespace.Name, "k8s-provider", testNamespace.Name, caBundle)
		CreateProvider(f, testNamespace.Name, "test-secretstore", "k8s-provider", testNamespace.Name)
		WaitForProviderConnectionReady(f, testNamespace.Name, "test-secretstore", 5*time.Second)

		By("creating source secret")
		sourceSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-secret",
				Namespace: testNamespace.Name,
			},
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret123"),
			},
		}
		Expect(f.CRClient.Create(context.Background(), sourceSecret)).To(Succeed())
		log.Logf("created source secret: %s/%s", testNamespace.Name, "source-secret")

		VerifyProviderConnectionCapabilities(f, testNamespace.Name, "test-secretstore", esv1.ProviderReadWrite)

		By("creating PushSecret")
		pushSecret := &esv1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pushsecret",
				Namespace: testNamespace.Name,
			},
			Spec: esv1alpha1.PushSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: 10 * time.Second},
				SecretStoreRefs: []esv1alpha1.PushSecretStoreRef{
					{
						Name:       "test-secretstore",
						Kind:       "Provider",
						APIVersion: "external-secrets.io/v1",
					},
				},
				Selector: esv1alpha1.PushSecretSelector{
					Secret: &esv1alpha1.PushSecretSecret{
						Name: "source-secret",
					},
				},
				Data: []esv1alpha1.PushSecretData{
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "username",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "pushed-secret",
								Property:  "username",
							},
						},
					},
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "password",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "pushed-secret",
								Property:  "password",
							},
						},
					},
				},
			},
		}
		Expect(f.CRClient.Create(context.Background(), pushSecret)).To(Succeed())
		log.Logf("created PushSecret: %s/%s", testNamespace.Name, "test-pushsecret")

		By("verifying PushSecret is synced")
		Eventually(func() bool {
			var ps esv1alpha1.PushSecret
			err := f.CRClient.Get(context.Background(),
				types.NamespacedName{Name: "test-pushsecret", Namespace: testNamespace.Name},
				&ps)
			if err != nil {
				log.Logf("failed to get PushSecret: %v", err)
				return false
			}

			for _, condition := range ps.Status.Conditions {
				if condition.Type == esv1alpha1.PushSecretReady && condition.Status == corev1.ConditionTrue {
					log.Logf("PushSecret is ready with status: %s", condition.Reason)
					return true
				}
			}
			log.Logf("PushSecret not ready yet, conditions: %+v", ps.Status.Conditions)
			return false
		}, 10*time.Second, 2*time.Second).Should(BeTrue(), "PushSecret should become ready")

		By("verifying pushed secret exists in target namespace")
		var pushedSecret corev1.Secret
		Eventually(func() bool {
			err := f.CRClient.Get(context.Background(),
				types.NamespacedName{Name: "pushed-secret", Namespace: testNamespace.Name},
				&pushedSecret)
			if err != nil {
				log.Logf("pushed secret not found yet: %v", err)
				return false
			}
			return true
		}, 10*time.Second, 2*time.Second).Should(BeTrue(), "pushed secret should exist")

		By("verifying pushed secret has correct data")
		Expect(pushedSecret.Data).To(HaveKey("username"))
		Expect(pushedSecret.Data).To(HaveKey("password"))
		Expect(string(pushedSecret.Data["username"])).To(Equal("admin"))
		Expect(string(pushedSecret.Data["password"])).To(Equal("secret123"))
		log.Logf("successfully verified pushed secret data")
	})

	It("should delete secrets when DeletionPolicy=Delete", func() {
		caBundle := GetClusterCABundle(f)
		CreateKubernetes(f, testNamespace.Name, "k8s-provider", testNamespace.Name, caBundle)
		CreateProvider(f, testNamespace.Name, "test-secretstore", "k8s-provider", testNamespace.Name)
		WaitForProviderConnectionReady(f, testNamespace.Name, "test-secretstore", 5*time.Second)

		By("creating source secret")
		sourceSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-secret-delete",
				Namespace: testNamespace.Name,
			},
			Data: map[string][]byte{
				"key1": []byte("value1"),
			},
		}
		Expect(f.CRClient.Create(context.Background(), sourceSecret)).To(Succeed())

		By("creating PushSecret with DeletionPolicy=Delete")
		pushSecret := &esv1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pushsecret-delete",
				Namespace: testNamespace.Name,
			},
			Spec: esv1alpha1.PushSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: 10 * time.Second},
				DeletionPolicy:  esv1alpha1.PushSecretDeletionPolicyDelete,
				SecretStoreRefs: []esv1alpha1.PushSecretStoreRef{
					{
						Name:       "test-secretstore",
						Kind:       "Provider",
						APIVersion: "external-secrets.io/v1",
					},
				},
				Selector: esv1alpha1.PushSecretSelector{
					Secret: &esv1alpha1.PushSecretSecret{
						Name: "source-secret-delete",
					},
				},
				Data: []esv1alpha1.PushSecretData{
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "key1",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "pushed-secret-delete",
								Property:  "key1",
							},
						},
					},
				},
			},
		}
		Expect(f.CRClient.Create(context.Background(), pushSecret)).To(Succeed())
		log.Logf("created PushSecret with Delete policy")

		By("waiting for PushSecret to sync")
		Eventually(func() bool {
			var ps esv1alpha1.PushSecret
			err := f.CRClient.Get(context.Background(),
				types.NamespacedName{Name: "test-pushsecret-delete", Namespace: testNamespace.Name},
				&ps)
			if err != nil {
				return false
			}
			for _, condition := range ps.Status.Conditions {
				if condition.Type == esv1alpha1.PushSecretReady && condition.Status == corev1.ConditionTrue {
					return true
				}
			}
			return false
		}, 30*time.Second, 2*time.Second).Should(BeTrue())

		By("verifying pushed secret was created")
		var pushedSecret corev1.Secret
		Expect(f.CRClient.Get(context.Background(),
			types.NamespacedName{Name: "pushed-secret-delete", Namespace: testNamespace.Name},
			&pushedSecret)).To(Succeed())

		By("deleting PushSecret")
		Expect(f.CRClient.Delete(context.Background(), pushSecret)).To(Succeed())

		By("verifying pushed secret is deleted due to DeletionPolicy=Delete")
		Eventually(func() bool {
			err := f.CRClient.Get(context.Background(),
				types.NamespacedName{Name: "pushed-secret-delete", Namespace: testNamespace.Name},
				&pushedSecret)
			return apierrors.IsNotFound(err)
		}, 30*time.Second, 2*time.Second).Should(BeTrue(), "pushed secret should be deleted when PushSecret is deleted")

		log.Logf("successfully verified Delete policy removes secrets")
	})
})
