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
package externalsecret

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

var _ = Describe("ExternalSecret controller", func() {
	const (
		ExternalSecretName             = "test-es"
		ExternalSecretNamespace        = "test-ns"
		ExternalSecretStore            = "test-store"
		ExternalSecretTargetSecretName = "test-secret"
		timeout                        = time.Second * 5
		interval                       = time.Millisecond * 250
	)

	BeforeEach(func() {
		Expect(k8sClient.Create(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ExternalSecretNamespace,
			},
		})).To(Succeed())

		Expect(k8sClient.Create(context.Background(), &esv1alpha1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretStore,
				Namespace: ExternalSecretNamespace,
			},
			Spec: esv1alpha1.SecretStoreSpec{
				Provider: &esv1alpha1.SecretStoreProvider{
					NOOP: &esv1alpha1.NOOPProvider{},
				},
			},
		})).To(Succeed())

	})
	AfterEach(func() {
		Expect(k8sClient.Delete(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ExternalSecretNamespace,
			},
		})).To(Succeed())
		Expect(k8sClient.Delete(context.Background(), &esv1alpha1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretStore,
				Namespace: ExternalSecretNamespace,
			},
		})).To(Succeed())
	})

	Context("When updating ExternalSecret Status", func() {
		It("should set the condition eventually", func() {
			By("creating an ExternalSecret")
			ctx := context.Background()
			es := &esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					SecretStoreRef: esv1alpha1.SecretStoreRef{
						Name: ExternalSecretStore,
					},
					Target: esv1alpha1.ExternalSecretTarget{
						Name: ExternalSecretTargetSecretName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, es)).Should(Succeed())
			esLookupKey := types.NamespacedName{Name: ExternalSecretName, Namespace: ExternalSecretNamespace}
			createdES := &esv1alpha1.ExternalSecret{}

			// We'll need to retry getting this newly created CronJob, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, esLookupKey, createdES)
				if err != nil {
					return false
				}

				// check status
				cond := GetExternalSecretCondition(createdES.Status, esv1alpha1.ExternalSecretReady)
				if cond == nil || cond.Status != v1.ConditionTrue {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

		})
	})
})
