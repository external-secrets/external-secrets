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

var _ = Describe("[v2] GetAllSecrets", Label("v2", "get-all-secrets"), func() {
	f := framework.New("v2-get-all-secrets")

	var (
		sourceNamespace *corev1.Namespace
		targetNamespace *corev1.Namespace
	)

	BeforeEach(func() {
		sourceNamespace = SetupTestNamespace(f, "v2-get-all-source-")
		targetNamespace = SetupTestNamespace(f, "v2-get-all-target-")
		CreateProviderSecretWriterRole(f, targetNamespace.Name, sourceNamespace.Name)

		// Create test secrets with different labels
		secrets := []struct {
			name   string
			labels map[string]string
			data   map[string][]byte
		}{
			{
				name:   "app-secret-1",
				labels: map[string]string{"app": "myapp", "env": "prod"},
				data:   map[string][]byte{"key1": []byte("value1")},
			},
			{
				name:   "app-secret-2",
				labels: map[string]string{"app": "myapp", "env": "dev"},
				data:   map[string][]byte{"key2": []byte("value2")},
			},
			{
				name:   "db-secret-1",
				labels: map[string]string{"app": "database", "env": "prod"},
				data:   map[string][]byte{"password": []byte("dbpass")},
			},
			{
				name:   "other-secret",
				labels: map[string]string{"type": "config"},
				data:   map[string][]byte{"config": []byte("data")},
			},
		}

		for _, s := range secrets {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      s.name,
					Namespace: sourceNamespace.Name,
					Labels:    s.labels,
				},
				Type: corev1.SecretTypeOpaque,
				Data: s.data,
			}
			Expect(f.CRClient.Create(context.Background(), secret)).To(Succeed())
		}

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

	It("should find secrets by tags (labels)", func() {
		caBundle := GetClusterCABundle(f)
		CreateKubernetes(f, targetNamespace.Name, "k8s-provider", sourceNamespace.Name, caBundle)
		CreateProvider(f, targetNamespace.Name, "test-secretstore", "k8s-provider", targetNamespace.Name)
		WaitForProviderConnectionReady(f, targetNamespace.Name, "test-secretstore", 5*time.Second)

		By("creating an ExternalSecret with dataFrom using tags")
		externalSecret := &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-external-secret-tags",
				Namespace: targetNamespace.Name,
			},
			Spec: esv1.ExternalSecretSpec{
				SecretStoreRef: esv1.SecretStoreRef{
					Kind: "Provider",
					Name: "test-secretstore",
				},
				Target: esv1.ExternalSecretTarget{
					Name:           "synced-secret-tags",
					CreationPolicy: esv1.CreatePolicyOwner,
				},
				RefreshInterval: &metav1.Duration{Duration: 1 * time.Hour},
				DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
					{
						Find: &esv1.ExternalSecretFind{
							Tags: map[string]string{
								"app": "myapp",
							},
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
				types.NamespacedName{Name: "test-external-secret-tags", Namespace: targetNamespace.Name},
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

		By("verifying the synced secret contains data from secrets with matching tags")
		var syncedSecret corev1.Secret
		Expect(f.CRClient.Get(context.Background(),
			types.NamespacedName{Name: "synced-secret-tags", Namespace: targetNamespace.Name},
			&syncedSecret)).To(Succeed())

		// GetAllSecrets returns secret name -> JSON data
		// Should contain keys for app-secret-1 and app-secret-2 (both have app=myapp)
		Expect(syncedSecret.Data).To(HaveKey("app-secret-1"))
		Expect(syncedSecret.Data).To(HaveKey("app-secret-2"))
		// Should NOT contain data from db-secret-1 or other-secret
		Expect(syncedSecret.Data).NotTo(HaveKey("db-secret-1"))
		Expect(syncedSecret.Data).NotTo(HaveKey("other-secret"))

		// Verify the values are JSON-encoded secret data
		Expect(string(syncedSecret.Data["app-secret-1"])).To(ContainSubstring("key1"))
		Expect(string(syncedSecret.Data["app-secret-2"])).To(ContainSubstring("key2"))
	})

	It("should find secrets by name regexp", func() {
		caBundle := GetClusterCABundle(f)
		CreateKubernetes(f, targetNamespace.Name, "k8s-provider-regex", sourceNamespace.Name, caBundle)
		CreateProvider(f, targetNamespace.Name, "test-secretstore-regex", "k8s-provider-regex", targetNamespace.Name)
		WaitForProviderConnectionReady(f, targetNamespace.Name, "test-secretstore-regex", 5*time.Second)

		By("creating an ExternalSecret with dataFrom using name regexp")
		externalSecret := &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-external-secret-regex",
				Namespace: targetNamespace.Name,
			},
			Spec: esv1.ExternalSecretSpec{
				SecretStoreRef: esv1.SecretStoreRef{
					Kind: "Provider",
					Name: "test-secretstore-regex",
				},
				Target: esv1.ExternalSecretTarget{
					Name:           "synced-secret-regex",
					CreationPolicy: esv1.CreatePolicyOwner,
				},
				RefreshInterval: &metav1.Duration{Duration: 1 * time.Hour},
				DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
					{
						Find: &esv1.ExternalSecretFind{
							Name: &esv1.FindName{
								RegExp: "^app-secret-.*",
							},
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
				types.NamespacedName{Name: "test-external-secret-regex", Namespace: targetNamespace.Name},
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

		By("verifying the synced secret contains data from secrets matching the regexp")
		var syncedSecret corev1.Secret
		Expect(f.CRClient.Get(context.Background(),
			types.NamespacedName{Name: "synced-secret-regex", Namespace: targetNamespace.Name},
			&syncedSecret)).To(Succeed())

		// GetAllSecrets returns secret name -> JSON data
		// Should contain keys for app-secret-1 and app-secret-2 (match ^app-secret-.*)
		Expect(syncedSecret.Data).To(HaveKey("app-secret-1"))
		Expect(syncedSecret.Data).To(HaveKey("app-secret-2"))
		// Should NOT contain data from db-secret-1 or other-secret
		Expect(syncedSecret.Data).NotTo(HaveKey("db-secret-1"))
		Expect(syncedSecret.Data).NotTo(HaveKey("other-secret"))

		// Verify the values are JSON-encoded secret data
		Expect(string(syncedSecret.Data["app-secret-1"])).To(ContainSubstring("key1"))
		Expect(string(syncedSecret.Data["app-secret-2"])).To(ContainSubstring("key2"))
	})
})
