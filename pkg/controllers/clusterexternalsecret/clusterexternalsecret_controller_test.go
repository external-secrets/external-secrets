/*
Copyright © 2025 ESO Maintainer Team

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

package clusterexternalsecret

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/controllers/clusterexternalsecret/cesmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	ctrlmetrics.SetUpLabelNames(false)
	cesmetrics.SetUpMetrics()
}

const (
	metadataLabelName       = "kubernetes.io/metadata.name"
	testLabelKey            = "test-label-key"
	testAnnotationKey       = "test-annotation-key"
	testLabelValue          = "test-label-value"
	testAnnotationValue     = "test-annotation-value"
	updatedTestStore        = "updated-test-store"
	noLongerMatchLabelKey   = "no-longer-match-label-key"
	noLongerMatchLabelValue = "no-longer-match-label-value"
)

var (
	timeout  = time.Second * 10
	interval = time.Millisecond * 250
)

type testCase struct {
	namespaces                    []v1.Namespace
	clusterExternalSecret         func(namespaces []v1.Namespace) esv1.ClusterExternalSecret
	beforeCheck                   func(ctx context.Context, namespaces []v1.Namespace, created esv1.ClusterExternalSecret)
	expectedClusterExternalSecret func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret
	expectedExternalSecrets       func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret
}

var _ = Describe("ClusterExternalSecret controller", func() {
	defaultClusterExternalSecret := func() *esv1.ClusterExternalSecret {
		return &esv1.ClusterExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("test-ces-%s", randString(10)),
			},
			Spec: esv1.ClusterExternalSecretSpec{
				ExternalSecretSpec: esv1.ExternalSecretSpec{
					SecretStoreRef: esv1.SecretStoreRef{
						Name: "test-store",
					},
					Target: esv1.ExternalSecretTarget{
						Name: "test-secret",
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "test-secret-key",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "test-remote-key",
							},
						},
					},
				},
			},
		}
	}

	DescribeTable("When reconciling a ClusterExternal Secret",
		func(tc testCase) {
			ctx := context.Background()
			By("creating namespaces")
			var namespaces []v1.Namespace
			for _, ns := range tc.namespaces {
				err := k8sClient.Create(ctx, &ns)
				Expect(err).ShouldNot(HaveOccurred())
				namespaces = append(namespaces, ns)
			}

			By("creating a cluster external secret")
			ces := tc.clusterExternalSecret(tc.namespaces)
			err := k8sClient.Create(ctx, &ces)
			Expect(err).ShouldNot(HaveOccurred())

			By("running before check")
			if tc.beforeCheck != nil {
				tc.beforeCheck(ctx, namespaces, ces)
			}

			// the before check above may have updated the namespaces, so refresh them
			for i, ns := range namespaces {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: ns.Name}, &ns)
				Expect(err).ShouldNot(HaveOccurred())
				namespaces[i] = ns
			}

			By("checking the cluster external secret")
			expectedCES := tc.expectedClusterExternalSecret(namespaces, ces)

			Eventually(func(g Gomega) {
				key := types.NamespacedName{Name: expectedCES.Name}
				var gotCes esv1.ClusterExternalSecret
				err = k8sClient.Get(ctx, key, &gotCes)
				g.Expect(err).ShouldNot(HaveOccurred())

				g.Expect(gotCes.Labels).To(Equal(expectedCES.Labels))
				g.Expect(gotCes.Annotations).To(Equal(expectedCES.Annotations))
				g.Expect(gotCes.Spec).To(Equal(expectedCES.Spec))
				g.Expect(gotCes.Status).To(Equal(expectedCES.Status))
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("checking the external secrets")
			expectedESs := tc.expectedExternalSecrets(namespaces, ces)

			Eventually(func(g Gomega) {
				var gotESs []esv1.ExternalSecret
				for _, ns := range namespaces {
					var externalSecrets esv1.ExternalSecretList
					err := k8sClient.List(ctx, &externalSecrets, crclient.InNamespace(ns.Name))
					g.Expect(err).ShouldNot(HaveOccurred())

					gotESs = append(gotESs, externalSecrets.Items...)
				}

				g.Expect(len(gotESs)).Should(Equal(len(expectedESs)))
				for _, gotES := range gotESs {
					found := false
					for _, expectedES := range expectedESs {
						if gotES.Namespace == expectedES.Namespace && gotES.Name == expectedES.Name {
							found = true
							g.Expect(gotES.Labels).To(Equal(expectedES.Labels))
							g.Expect(gotES.Annotations).To(Equal(expectedES.Annotations))
							g.Expect(gotES.Spec).To(Equal(expectedES.Spec))
						}
					}
					g.Expect(found).To(Equal(true))
				}
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		},

		Entry("Should use cluster external secret name if external secret name isn't defined", testCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterExternalSecret: func(namespaces []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				ces.Spec.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{metadataLabelName: namespaces[0].Name},
				}
				return *ces
			},
			expectedClusterExternalSecret: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName:    created.Name,
						ProvisionedNamespaces: []string{namespaces[0].Name},
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:   esv1.ClusterExternalSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				return []esv1.ExternalSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[0].Name,
							Name:      created.Name,
						},
						Spec: created.Spec.ExternalSecretSpec,
					},
				}
			},
		}),
		Entry("Should set external secret name and metadata if the fields are set", testCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterExternalSecret: func(namespaces []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				ces.Spec.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{metadataLabelName: namespaces[0].Name},
				}
				ces.Spec.ExternalSecretName = "test-es"
				ces.Spec.ExternalSecretMetadata = esv1.ExternalSecretMetadata{
					Labels:      map[string]string{"test-label-key1": "test-label-value1", "test-label-key2": "test-label-value2"},
					Annotations: map[string]string{"test-annotation-key1": "test-annotation-value1", "test-annotation-key2": "test-annotation-value2"},
				}
				return *ces
			},
			expectedClusterExternalSecret: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName:    "test-es",
						ProvisionedNamespaces: []string{namespaces[0].Name},
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:   esv1.ClusterExternalSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				return []esv1.ExternalSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:   namespaces[0].Name,
							Name:        "test-es",
							Labels:      map[string]string{"test-label-key1": "test-label-value1", "test-label-key2": "test-label-value2"},
							Annotations: map[string]string{"test-annotation-key1": "test-annotation-value1", "test-annotation-key2": "test-annotation-value2"},
						},
						Spec: created.Spec.ExternalSecretSpec,
					},
				}
			},
		}),
		Entry("Should delete old external secrets if name has changed", testCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterExternalSecret: func(namespaces []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				ces.Spec.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{metadataLabelName: namespaces[0].Name},
				}
				ces.Spec.ExternalSecretName = "old-es-name"
				return *ces
			},
			beforeCheck: func(ctx context.Context, namespaces []v1.Namespace, created esv1.ClusterExternalSecret) {
				// Wait until the external secret is provisioned
				var es esv1.ExternalSecret
				Eventually(func(g Gomega) {
					key := types.NamespacedName{Namespace: namespaces[0].Name, Name: "old-es-name"}
					g.Expect(k8sClient.Get(ctx, key, &es)).ShouldNot(HaveOccurred())
				}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

				copied := created.DeepCopy()
				copied.Spec.ExternalSecretName = "new-es-name"
				Expect(k8sClient.Patch(ctx, copied, crclient.MergeFrom(created.DeepCopy()))).ShouldNot(HaveOccurred())
			},
			expectedClusterExternalSecret: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				updatedSpec := created.Spec.DeepCopy()
				updatedSpec.ExternalSecretName = "new-es-name"

				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: *updatedSpec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName:    "new-es-name",
						ProvisionedNamespaces: []string{namespaces[0].Name},
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:   esv1.ClusterExternalSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				return []esv1.ExternalSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[0].Name,
							Name:      "new-es-name",
						},
						Spec: created.Spec.ExternalSecretSpec,
					},
				}
			},
		}),
		Entry("Should update external secret if the fields change", testCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterExternalSecret: func(namespaces []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				ces.Spec.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{metadataLabelName: namespaces[0].Name},
				}
				return *ces
			},
			beforeCheck: func(ctx context.Context, namespaces []v1.Namespace, created esv1.ClusterExternalSecret) {
				// Wait until the external secret is provisioned
				var es esv1.ExternalSecret
				Eventually(func(g Gomega) {
					key := types.NamespacedName{Namespace: namespaces[0].Name, Name: created.Name}
					g.Expect(k8sClient.Get(ctx, key, &es)).ShouldNot(HaveOccurred())
					g.Expect(len(es.Labels)).Should(Equal(0))
					g.Expect(len(es.Annotations)).Should(Equal(0))
					g.Expect(es.Spec).Should(Equal(created.Spec.ExternalSecretSpec))
				}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

				copied := created.DeepCopy()
				copied.Spec.ExternalSecretMetadata = esv1.ExternalSecretMetadata{
					Labels:      map[string]string{testLabelKey: testLabelValue},
					Annotations: map[string]string{testAnnotationKey: testAnnotationValue},
				}
				copied.Spec.ExternalSecretSpec.SecretStoreRef.Name = updatedTestStore
				Expect(k8sClient.Patch(ctx, copied, crclient.MergeFrom(created.DeepCopy()))).ShouldNot(HaveOccurred())
			},
			expectedClusterExternalSecret: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				updatedSpec := created.Spec.DeepCopy()
				updatedSpec.ExternalSecretMetadata = esv1.ExternalSecretMetadata{
					Labels:      map[string]string{testLabelKey: testLabelValue},
					Annotations: map[string]string{testAnnotationKey: testAnnotationValue},
				}
				updatedSpec.ExternalSecretSpec.SecretStoreRef.Name = updatedTestStore

				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: *updatedSpec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName:    created.Name,
						ProvisionedNamespaces: []string{namespaces[0].Name},
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:   esv1.ClusterExternalSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				updatedSpec := created.Spec.ExternalSecretSpec.DeepCopy()
				updatedSpec.SecretStoreRef.Name = updatedTestStore

				return []esv1.ExternalSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:   namespaces[0].Name,
							Name:        created.Name,
							Labels:      map[string]string{testLabelKey: testLabelValue},
							Annotations: map[string]string{testAnnotationKey: testAnnotationValue},
						},
						Spec: *updatedSpec,
					},
				}
			},
		}),
		Entry("Should not overwrite existing external secrets and error out if one is present", testCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterExternalSecret: func(namespaces []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				ces.Spec.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{metadataLabelName: namespaces[0].Name},
				}

				es := &esv1.ExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ces.Name,
						Namespace: namespaces[0].Name,
					},
				}
				Expect(k8sClient.Create(context.Background(), es)).ShouldNot(HaveOccurred())

				return *ces
			},
			expectedClusterExternalSecret: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName: created.Name,
						FailedNamespaces: []esv1.ClusterExternalSecretNamespaceFailure{
							{
								Namespace: namespaces[0].Name,
								Reason:    "external secret already exists in namespace",
							},
						},
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:    esv1.ClusterExternalSecretReady,
								Status:  v1.ConditionFalse,
								Message: "one or more namespaces failed",
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				return []esv1.ExternalSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[0].Name,
							Name:      created.Name,
						},
						Spec: esv1.ExternalSecretSpec{
							Target: esv1.ExternalSecretTarget{
								CreationPolicy: "Owner",
								DeletionPolicy: "Retain",
							},
							RefreshInterval: &metav1.Duration{Duration: time.Hour},
						},
					},
				}
			},
		}),
		Entry("Should crate an external secret if one with the same name has been deleted", testCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterExternalSecret: func(namespaces []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				ces.Spec.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{metadataLabelName: namespaces[0].Name},
				}

				es := &esv1.ExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ces.Name,
						Namespace: namespaces[0].Name,
					},
				}
				Expect(k8sClient.Create(context.Background(), es)).ShouldNot(HaveOccurred())
				return *ces
			},
			beforeCheck: func(ctx context.Context, namespaces []v1.Namespace, created esv1.ClusterExternalSecret) {
				ces := esv1.ClusterExternalSecret{}
				Eventually(func(g Gomega) {
					key := types.NamespacedName{Namespace: created.Namespace, Name: created.Name}
					g.Expect(k8sClient.Get(ctx, key, &ces)).ShouldNot(HaveOccurred())
					g.Expect(len(ces.Status.FailedNamespaces)).Should(Equal(1))
				}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

				es := &esv1.ExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ces.Name,
						Namespace: namespaces[0].Name,
					},
				}
				Expect(k8sClient.Delete(ctx, es)).ShouldNot(HaveOccurred())
			},
			expectedClusterExternalSecret: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName:    created.Name,
						ProvisionedNamespaces: []string{namespaces[0].Name},
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:   esv1.ClusterExternalSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				return []esv1.ExternalSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[0].Name,
							Name:      created.Name,
						},
						Spec: created.Spec.ExternalSecretSpec,
					},
				}
			},
		}),
		Entry("Should delete external secrets when namespaces no longer match", testCase{
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   randomNamespaceName(),
						Labels: map[string]string{noLongerMatchLabelKey: noLongerMatchLabelValue},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   randomNamespaceName(),
						Labels: map[string]string{noLongerMatchLabelKey: noLongerMatchLabelValue},
					},
				},
			},
			clusterExternalSecret: func(_ []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				ces.Spec.RefreshInterval = &metav1.Duration{Duration: 100 * time.Millisecond}
				ces.Spec.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{noLongerMatchLabelKey: noLongerMatchLabelValue},
				}
				return *ces
			},
			beforeCheck: func(ctx context.Context, namespaces []v1.Namespace, created esv1.ClusterExternalSecret) {
				// Wait until the target ESs have been created
				Eventually(func(g Gomega) {
					for _, ns := range namespaces {
						key := types.NamespacedName{Namespace: ns.Name, Name: created.Name}
						g.Expect(k8sClient.Get(ctx, key, &esv1.ExternalSecret{})).ShouldNot(HaveOccurred())
					}
				}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

				// Retry on conflict since controller may be updating namespace with finalizers
				Eventually(func() error {
					var ns v1.Namespace
					if err := k8sClient.Get(ctx, types.NamespacedName{Name: namespaces[0].Name}, &ns); err != nil {
						return err
					}
					ns.Labels = map[string]string{}
					return k8sClient.Update(ctx, &ns)
				}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
			},
			expectedClusterExternalSecret: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName:    created.Name,
						ProvisionedNamespaces: []string{namespaces[1].Name},
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:   esv1.ClusterExternalSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				return []esv1.ExternalSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[1].Name,
							Name:      created.Name,
						},
						Spec: created.Spec.ExternalSecretSpec,
					},
				}
			},
		}),
		Entry("Should sync with match expression", testCase{
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   randomNamespaceName(),
						Labels: map[string]string{"prefix": "foo"},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   randomNamespaceName(),
						Labels: map[string]string{"prefix": "bar"},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   randomNamespaceName(),
						Labels: map[string]string{"prefix": "baz"},
					},
				},
			},
			clusterExternalSecret: func(_ []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				ces.Spec.RefreshInterval = &metav1.Duration{Duration: 100 * time.Millisecond}
				ces.Spec.NamespaceSelector = &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "prefix",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"foo", "bar"}, // "baz" is excluded
						},
					},
				}
				return *ces
			},
			expectedClusterExternalSecret: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				provisionedNamespaces := []string{namespaces[0].Name, namespaces[1].Name}
				sort.Strings(provisionedNamespaces)
				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName:    created.Name,
						ProvisionedNamespaces: provisionedNamespaces,
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:   esv1.ClusterExternalSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				return []esv1.ExternalSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[0].Name,
							Name:      created.Name,
						},
						Spec: created.Spec.ExternalSecretSpec,
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[1].Name,
							Name:      created.Name,
						},
						Spec: created.Spec.ExternalSecretSpec,
					},
				}
			},
		}),
		Entry("Should be ready if no namespace matches", testCase{
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: randomNamespaceName(),
					},
				},
			},
			clusterExternalSecret: func(_ []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				ces.Spec.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{metadataLabelName: "no-namespace-matches"},
				}
				return *ces
			},
			expectedClusterExternalSecret: func(_ []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName: created.Name,
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:   esv1.ClusterExternalSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(_ []v1.Namespace, _ esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				return []esv1.ExternalSecret{}
			},
		}),
		Entry("Should be ready if namespace is selected via the namespace selectors", testCase{
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace1",
						Labels: map[string]string{
							"key": "value1",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace2",
						Labels: map[string]string{
							"key": "value2",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace3",
						Labels: map[string]string{
							"key": "value3",
						},
					},
				},
			},
			clusterExternalSecret: func(_ []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				ces.Spec.NamespaceSelectors = []*metav1.LabelSelector{
					{
						MatchLabels: map[string]string{"key": "value1"},
					},
					{
						MatchLabels: map[string]string{"key": "value2"},
					},
				}
				return *ces
			},
			expectedClusterExternalSecret: func(_ []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName: created.Name,
						ProvisionedNamespaces: []string{
							"namespace1",
							"namespace2",
						},
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:   esv1.ClusterExternalSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(_ []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				return []esv1.ExternalSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "namespace1",
							Name:      created.Name,
						},
						Spec: created.Spec.ExternalSecretSpec,
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "namespace2",
							Name:      created.Name,
						},
						Spec: created.Spec.ExternalSecretSpec,
					},
				}
			},
		}),
		Entry("Should be ready if namespace is selected via namespaces", testCase{
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "not-matching-namespace",
					},
				},
			},
			clusterExternalSecret: func(_ []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				// does-not-exists tests that we would continue on to the next and not stop if the
				// namespace hasn't been created yet.
				ces.Spec.Namespaces = []string{"does-not-exist", "not-matching-namespace"}
				return *ces
			},
			expectedClusterExternalSecret: func(_ []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName: created.Name,
						ProvisionedNamespaces: []string{
							"not-matching-namespace",
						},
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:   esv1.ClusterExternalSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(_ []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				return []esv1.ExternalSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "not-matching-namespace",
							Name:      created.Name,
						},
						Spec: created.Spec.ExternalSecretSpec,
					},
				}
			},
		}),
		Entry("Should propagate the force-sync annotation", testCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterExternalSecret: func(namespaces []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				ces.Annotations = map[string]string{esv1.AnnotationForceSync: "true"}
				ces.Spec.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{metadataLabelName: namespaces[0].Name},
				}
				return *ces
			},
			expectedClusterExternalSecret: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:        created.Name,
						Annotations: map[string]string{esv1.AnnotationForceSync: "true"},
					},
					Spec: created.Spec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName:    created.Name,
						ProvisionedNamespaces: []string{namespaces[0].Name},
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:   esv1.ClusterExternalSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				return []esv1.ExternalSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:   namespaces[0].Name,
							Name:        created.Name,
							Annotations: map[string]string{esv1.AnnotationForceSync: "true"},
						},
						Spec: created.Spec.ExternalSecretSpec,
					},
				}
			},
		}),
		Entry("Should prune the force-sync annotation", testCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterExternalSecret: func(namespaces []v1.Namespace) esv1.ClusterExternalSecret {
				ces := defaultClusterExternalSecret()
				ces.Annotations = map[string]string{esv1.AnnotationForceSync: "true"}
				ces.Spec.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{metadataLabelName: namespaces[0].Name},
				}
				return *ces
			},
			beforeCheck: func(ctx context.Context, namespaces []v1.Namespace, created esv1.ClusterExternalSecret) {
				// Wait until the external secret is provisioned and has
				// the force-sync annotation
				var es esv1.ExternalSecret
				Eventually(func(g Gomega) {
					key := types.NamespacedName{Namespace: namespaces[0].Name, Name: created.Name}
					g.Expect(k8sClient.Get(ctx, key, &es)).ShouldNot(HaveOccurred())
					g.Expect(len(es.Annotations)).Should(Equal(1))
					g.Expect(es.Spec).Should(Equal(created.Spec.ExternalSecretSpec))
				}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

				// Prune the force-sync annotation
				copied := created.DeepCopy()
				delete(copied.Annotations, esv1.AnnotationForceSync)
				Expect(k8sClient.Patch(ctx, copied, crclient.MergeFrom(created.DeepCopy()))).ShouldNot(HaveOccurred())
			},
			expectedClusterExternalSecret: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) esv1.ClusterExternalSecret {
				return esv1.ClusterExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: esv1.ClusterExternalSecretStatus{
						ExternalSecretName:    created.Name,
						ProvisionedNamespaces: []string{namespaces[0].Name},
						Conditions: []esv1.ClusterExternalSecretStatusCondition{
							{
								Type:   esv1.ClusterExternalSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedExternalSecrets: func(namespaces []v1.Namespace, created esv1.ClusterExternalSecret) []esv1.ExternalSecret {
				return []esv1.ExternalSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[0].Name,
							Name:      created.Name,
						},
						Spec: created.Spec.ExternalSecretSpec,
					},
				}
			},
		}),
	)
})

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func randomNamespaceName() string {
	return fmt.Sprintf("testns-%s", randString(10))
}
