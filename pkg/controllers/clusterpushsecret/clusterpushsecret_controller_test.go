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

package clusterpushsecret

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

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/clusterpushsecret/cpsmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/provider/testing/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	ctrlmetrics.SetUpLabelNames(false)
	cpsmetrics.SetUpMetrics()
	fakeProvider = fake.New()
	v1beta1.ForceRegister(fakeProvider, &v1beta1.SecretStoreProvider{
		Fake: &v1beta1.FakeProvider{},
	})
}

var (
	secretName                = "test-secret"
	testPushSecret            = "test-ps"
	newPushSecret             = "new-ps-name"
	defaultKey                = "default-key"
	defaultVal                = "default-value"
	testLabelKey              = "test-label-key"
	testLabelValue            = "test-label-value"
	testAnnotationKey         = "test-annotation-key"
	testAnnotationValue       = "test-annotation-value"
	updateStoreName           = "updated-test-store"
	kubernetesMetadataLabel   = "kubernetes.io/metadata.name"
	noneMatchingAnnotationKey = "no-longer-match-label-key"
	noneMatchingAnnotationVal = "no-longer-match-annotation-value"
	fakeProvider              *fake.Client
	timeout                   = time.Second * 10
	interval                  = time.Millisecond * 250
)

type clusterPushSecretTestCase struct {
	namespaces                []v1.Namespace
	clusterPushSecret         func(namespaces []v1.Namespace) v1alpha1.ClusterPushSecret
	sourceSecret              func(namespaces []v1.Namespace) []v1.Secret
	beforeCheck               func(ctx context.Context, namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret)
	expectedClusterPushSecret func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) v1alpha1.ClusterPushSecret
	expectedPushSecrets       func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) []v1alpha1.PushSecret
}

var _ = Describe("ClusterPushSecret controller", func() {
	defaultClusterPushSecret := func() *v1alpha1.ClusterPushSecret {
		return &v1alpha1.ClusterPushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("test-pes-%s", randString(10)),
			},
			Spec: v1alpha1.ClusterPushSecretSpec{
				PushSecretSpec: v1alpha1.PushSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Hour},
					SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
						{
							Name: "test-store",
							Kind: "SecretStore",
						},
					},
					Selector: v1alpha1.PushSecretSelector{
						Secret: &v1alpha1.PushSecretSecret{
							Name: secretName,
						},
					},
					Data: []v1alpha1.PushSecretData{
						{
							Match:    v1alpha1.PushSecretMatch{},
							Metadata: nil,
						},
					},
				},
			},
		}
	}

	defaultSourceSecret := func(namespaces []v1.Namespace) []v1.Secret {
		var result []v1.Secret
		for _, s := range namespaces {
			result = append(result, v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: s.Name,
				},
				Data: map[string][]byte{
					defaultKey: []byte(defaultVal),
				},
			})
		}

		return result
	}

	DescribeTable("When reconciling a ClusterPush Secret",
		func(tc clusterPushSecretTestCase) {
			ctx := context.Background()
			By("creating namespaces")
			var namespaces []v1.Namespace
			for _, ns := range tc.namespaces {
				err := k8sClient.Create(ctx, &ns)
				Expect(err).ShouldNot(HaveOccurred())
				namespaces = append(namespaces, ns)
			}

			for _, s := range tc.sourceSecret(namespaces) {
				By("creating a source secret")
				err := k8sClient.Create(ctx, &s)
				Expect(err).ShouldNot(HaveOccurred())
			}

			By("creating a cluster push secret")
			pes := tc.clusterPushSecret(tc.namespaces)
			err := k8sClient.Create(ctx, &pes)
			Expect(err).ShouldNot(HaveOccurred())

			By("running before check")
			if tc.beforeCheck != nil {
				tc.beforeCheck(ctx, namespaces, pes)
			}

			// the before check above may have updated the namespaces, so refresh them
			for i, ns := range namespaces {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: ns.Name}, &ns)
				Expect(err).ShouldNot(HaveOccurred())
				namespaces[i] = ns
			}

			By("checking the cluster push secret")
			expectedCPS := tc.expectedClusterPushSecret(namespaces, pes)

			Eventually(func(g Gomega) {
				key := types.NamespacedName{Name: expectedCPS.Name}
				var gotCes v1alpha1.ClusterPushSecret
				err = k8sClient.Get(ctx, key, &gotCes)
				g.Expect(err).ShouldNot(HaveOccurred())

				g.Expect(gotCes.Labels).To(Equal(expectedCPS.Labels))
				g.Expect(gotCes.Annotations).To(Equal(expectedCPS.Annotations))
				g.Expect(gotCes.Spec).To(Equal(expectedCPS.Spec))
				g.Expect(gotCes.Status).To(Equal(expectedCPS.Status))
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("checking the push secrets")
			expectedPSs := tc.expectedPushSecrets(namespaces, pes)

			Eventually(func(g Gomega) {
				var gotESs []v1alpha1.PushSecret
				for _, ns := range namespaces {
					var pushSecrets v1alpha1.PushSecretList
					err := k8sClient.List(ctx, &pushSecrets, crclient.InNamespace(ns.Name))
					g.Expect(err).ShouldNot(HaveOccurred())

					gotESs = append(gotESs, pushSecrets.Items...)
				}

				g.Expect(len(gotESs)).Should(Equal(len(expectedPSs)))
				for _, gotES := range gotESs {
					found := false
					for _, expectedPS := range expectedPSs {
						if gotES.Namespace == expectedPS.Namespace && gotES.Name == expectedPS.Name {
							found = true
							g.Expect(gotES.Labels).To(Equal(expectedPS.Labels))
							g.Expect(gotES.Annotations).To(Equal(expectedPS.Annotations))
							g.Expect(gotES.Spec).To(Equal(expectedPS.Spec))
						}
					}
					g.Expect(found).To(Equal(true))
				}
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		},

		Entry("Should use cluster push secret name if push secret name isn't defined", clusterPushSecretTestCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterPushSecret: func(namespaces []v1.Namespace) v1alpha1.ClusterPushSecret {
				pes := defaultClusterPushSecret()
				pes.Spec.NamespaceSelectors = []*metav1.LabelSelector{
					{
						MatchLabels: map[string]string{kubernetesMetadataLabel: namespaces[0].Name},
					},
				}
				return *pes
			},
			sourceSecret: defaultSourceSecret,
			expectedClusterPushSecret: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) v1alpha1.ClusterPushSecret {
				return v1alpha1.ClusterPushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: v1alpha1.ClusterPushSecretStatus{
						PushSecretName:        created.Name,
						ProvisionedNamespaces: []string{namespaces[0].Name},
						Conditions: []v1alpha1.PushSecretStatusCondition{
							{
								Type:   v1alpha1.PushSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedPushSecrets: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) []v1alpha1.PushSecret {
				return []v1alpha1.PushSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[0].Name,
							Name:      created.Name,
						},
						Spec: created.Spec.PushSecretSpec,
					},
				}
			},
		}),
		Entry("Should set push secret name and metadata if the fields are set", clusterPushSecretTestCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterPushSecret: func(namespaces []v1.Namespace) v1alpha1.ClusterPushSecret {
				pes := defaultClusterPushSecret()
				pes.Spec.NamespaceSelectors = []*metav1.LabelSelector{
					{
						MatchLabels: map[string]string{kubernetesMetadataLabel: namespaces[0].Name},
					},
				}
				pes.Spec.PushSecretName = testPushSecret
				pes.Spec.PushSecretMetadata = v1alpha1.PushSecretMetadata{
					Labels:      map[string]string{"test-label-key1": "test-label-value1", "test-label-key2": "test-label-value2"},
					Annotations: map[string]string{"test-annotation-key1": "test-annotation-value1", "test-annotation-key2": "test-annotation-value2"},
				}
				return *pes
			},
			sourceSecret: defaultSourceSecret,
			expectedClusterPushSecret: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) v1alpha1.ClusterPushSecret {
				return v1alpha1.ClusterPushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: v1alpha1.ClusterPushSecretStatus{
						PushSecretName:        testPushSecret,
						ProvisionedNamespaces: []string{namespaces[0].Name},
						Conditions: []v1alpha1.PushSecretStatusCondition{
							{
								Type:   v1alpha1.PushSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedPushSecrets: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) []v1alpha1.PushSecret {
				return []v1alpha1.PushSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:   namespaces[0].Name,
							Name:        testPushSecret,
							Labels:      map[string]string{"test-label-key1": "test-label-value1", "test-label-key2": "test-label-value2"},
							Annotations: map[string]string{"test-annotation-key1": "test-annotation-value1", "test-annotation-key2": "test-annotation-value2"},
						},
						Spec: created.Spec.PushSecretSpec,
					},
				}
			},
		}),
		Entry("Should delete old push secrets if name has changed", clusterPushSecretTestCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterPushSecret: func(namespaces []v1.Namespace) v1alpha1.ClusterPushSecret {
				pes := defaultClusterPushSecret()
				pes.Spec.NamespaceSelectors = []*metav1.LabelSelector{
					{
						MatchLabels: map[string]string{kubernetesMetadataLabel: namespaces[0].Name},
					},
				}
				pes.Spec.PushSecretName = "old-es-name"
				return *pes
			},
			sourceSecret: defaultSourceSecret,
			beforeCheck: func(ctx context.Context, namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) {
				// Wait until the push secret is provisioned
				var es v1alpha1.PushSecret
				Eventually(func(g Gomega) {
					key := types.NamespacedName{Namespace: namespaces[0].Name, Name: "old-es-name"}
					g.Expect(k8sClient.Get(ctx, key, &es)).ShouldNot(HaveOccurred())
				}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

				copied := created.DeepCopy()
				copied.Spec.PushSecretName = newPushSecret
				Expect(k8sClient.Patch(ctx, copied, crclient.MergeFrom(created.DeepCopy()))).ShouldNot(HaveOccurred())
			},
			expectedClusterPushSecret: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) v1alpha1.ClusterPushSecret {
				updatedSpec := created.Spec.DeepCopy()
				updatedSpec.PushSecretName = newPushSecret

				return v1alpha1.ClusterPushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: *updatedSpec,
					Status: v1alpha1.ClusterPushSecretStatus{
						PushSecretName:        newPushSecret,
						ProvisionedNamespaces: []string{namespaces[0].Name},
						Conditions: []v1alpha1.PushSecretStatusCondition{
							{
								Type:   v1alpha1.PushSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedPushSecrets: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) []v1alpha1.PushSecret {
				return []v1alpha1.PushSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[0].Name,
							Name:      newPushSecret,
						},
						Spec: created.Spec.PushSecretSpec,
					},
				}
			},
		}),
		Entry("Should update push secret if the fields change", clusterPushSecretTestCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterPushSecret: func(namespaces []v1.Namespace) v1alpha1.ClusterPushSecret {
				pes := defaultClusterPushSecret()
				pes.Spec.NamespaceSelectors = []*metav1.LabelSelector{
					{
						MatchLabels: map[string]string{kubernetesMetadataLabel: namespaces[0].Name},
					},
				}
				return *pes
			},
			sourceSecret: defaultSourceSecret,
			beforeCheck: func(ctx context.Context, namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) {
				// Wait until the push secret is provisioned
				var es v1alpha1.PushSecret
				Eventually(func(g Gomega) {
					key := types.NamespacedName{Namespace: namespaces[0].Name, Name: created.Name}
					g.Expect(k8sClient.Get(ctx, key, &es)).ShouldNot(HaveOccurred())
					g.Expect(len(es.Labels)).Should(Equal(0))
					g.Expect(len(es.Annotations)).Should(Equal(0))
					g.Expect(es.Spec).Should(Equal(created.Spec.PushSecretSpec))
				}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

				copied := created.DeepCopy()
				copied.Spec.PushSecretMetadata = v1alpha1.PushSecretMetadata{
					Labels:      map[string]string{testLabelKey: testLabelValue},
					Annotations: map[string]string{testAnnotationKey: testAnnotationValue},
				}
				copied.Spec.PushSecretSpec.SecretStoreRefs = []v1alpha1.PushSecretStoreRef{
					{
						Name: updateStoreName,
						Kind: "SecretStore",
					},
				}
				Expect(k8sClient.Patch(ctx, copied, crclient.MergeFrom(created.DeepCopy()))).ShouldNot(HaveOccurred())
			},
			expectedClusterPushSecret: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) v1alpha1.ClusterPushSecret {
				updatedSpec := created.Spec.DeepCopy()
				updatedSpec.PushSecretMetadata = v1alpha1.PushSecretMetadata{
					Labels:      map[string]string{testLabelKey: testLabelValue},
					Annotations: map[string]string{testAnnotationKey: testAnnotationValue},
				}
				updatedSpec.PushSecretSpec.SecretStoreRefs = []v1alpha1.PushSecretStoreRef{
					{
						Name: updateStoreName,
						Kind: "SecretStore",
					},
				}

				return v1alpha1.ClusterPushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: *updatedSpec,
					Status: v1alpha1.ClusterPushSecretStatus{
						PushSecretName:        created.Name,
						ProvisionedNamespaces: []string{namespaces[0].Name},
						Conditions: []v1alpha1.PushSecretStatusCondition{
							{
								Type:   v1alpha1.PushSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedPushSecrets: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) []v1alpha1.PushSecret {
				updatedSpec := created.Spec.PushSecretSpec.DeepCopy()
				updatedSpec.SecretStoreRefs = []v1alpha1.PushSecretStoreRef{
					{
						Name: updateStoreName,
						Kind: "SecretStore",
					},
				}

				return []v1alpha1.PushSecret{
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
		Entry("Should not overwrite existing push secrets and error out if one is present", clusterPushSecretTestCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterPushSecret: func(namespaces []v1.Namespace) v1alpha1.ClusterPushSecret {
				pes := defaultClusterPushSecret()
				pes.Spec.NamespaceSelectors = []*metav1.LabelSelector{
					{
						MatchLabels: map[string]string{kubernetesMetadataLabel: namespaces[0].Name},
					},
				}

				es := &v1alpha1.PushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pes.Name,
						Namespace: namespaces[0].Name,
					},
					Spec: v1alpha1.PushSecretSpec{
						RefreshInterval: &metav1.Duration{Duration: time.Hour},
						SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
							{
								Name: updateStoreName,
							},
						},
						Selector: v1alpha1.PushSecretSelector{
							Secret: &v1alpha1.PushSecretSecret{
								Name: secretName,
							},
						},
					},
				}
				Expect(k8sClient.Create(context.Background(), es)).ShouldNot(HaveOccurred())

				return *pes
			},
			sourceSecret: defaultSourceSecret,
			expectedClusterPushSecret: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) v1alpha1.ClusterPushSecret {
				return v1alpha1.ClusterPushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: v1alpha1.ClusterPushSecretStatus{
						PushSecretName: created.Name,
						FailedNamespaces: []v1alpha1.ClusterPushSecretNamespaceFailure{
							{
								Namespace: namespaces[0].Name,
								Reason:    "push secret already exists in namespace",
							},
						},
						Conditions: []v1alpha1.PushSecretStatusCondition{
							{
								Type:    v1alpha1.PushSecretReady,
								Status:  v1.ConditionFalse,
								Message: "one or more namespaces failed",
							},
						},
					},
				}
			},
			expectedPushSecrets: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) []v1alpha1.PushSecret {
				return []v1alpha1.PushSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[0].Name,
							Name:      created.Name,
						},
						Spec: v1alpha1.PushSecretSpec{
							RefreshInterval: &metav1.Duration{Duration: time.Hour},
							SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
								{
									Name: updateStoreName,
									Kind: "SecretStore",
								},
							},
							UpdatePolicy: "Replace",
							Selector: v1alpha1.PushSecretSelector{
								Secret: &v1alpha1.PushSecretSecret{
									Name: secretName,
								},
							},
							DeletionPolicy: "None",
						},
					},
				}
			},
		}),
		Entry("Should crate an push secret if one with the same name has been deleted", clusterPushSecretTestCase{
			namespaces: []v1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: randomNamespaceName()}},
			},
			clusterPushSecret: func(namespaces []v1.Namespace) v1alpha1.ClusterPushSecret {
				pes := defaultClusterPushSecret()
				pes.Spec.NamespaceSelectors = []*metav1.LabelSelector{
					{
						MatchLabels: map[string]string{kubernetesMetadataLabel: namespaces[0].Name},
					},
				}

				es := &v1alpha1.PushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pes.Name,
						Namespace: namespaces[0].Name,
					},
					Spec: v1alpha1.PushSecretSpec{
						RefreshInterval: &metav1.Duration{Duration: time.Hour},
						SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
							{
								Name: updateStoreName,
							},
						},
						Selector: v1alpha1.PushSecretSelector{
							Secret: &v1alpha1.PushSecretSecret{
								Name: secretName,
							},
						},
					},
				}
				Expect(k8sClient.Create(context.Background(), es)).ShouldNot(HaveOccurred())
				return *pes
			},
			sourceSecret: defaultSourceSecret,
			beforeCheck: func(ctx context.Context, namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) {
				pes := v1alpha1.ClusterPushSecret{}
				Eventually(func(g Gomega) {
					key := types.NamespacedName{Namespace: created.Namespace, Name: created.Name}
					g.Expect(k8sClient.Get(ctx, key, &pes)).ShouldNot(HaveOccurred())
					g.Expect(len(pes.Status.FailedNamespaces)).Should(Equal(1))
				}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

				es := &v1alpha1.PushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pes.Name,
						Namespace: namespaces[0].Name,
					},
				}
				Expect(k8sClient.Delete(ctx, es)).ShouldNot(HaveOccurred())
			},
			expectedClusterPushSecret: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) v1alpha1.ClusterPushSecret {
				return v1alpha1.ClusterPushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: v1alpha1.ClusterPushSecretStatus{
						PushSecretName:        created.Name,
						ProvisionedNamespaces: []string{namespaces[0].Name},
						Conditions: []v1alpha1.PushSecretStatusCondition{
							{
								Type:   v1alpha1.PushSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedPushSecrets: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) []v1alpha1.PushSecret {
				return []v1alpha1.PushSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[0].Name,
							Name:      created.Name,
						},
						Spec: created.Spec.PushSecretSpec,
					},
				}
			},
		}),
		Entry("Should delete push secrets when namespaces no longer match", clusterPushSecretTestCase{
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   randomNamespaceName(),
						Labels: map[string]string{noneMatchingAnnotationKey: noneMatchingAnnotationVal},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   randomNamespaceName(),
						Labels: map[string]string{noneMatchingAnnotationKey: noneMatchingAnnotationVal},
					},
				},
			},
			sourceSecret: defaultSourceSecret,
			clusterPushSecret: func(namespaces []v1.Namespace) v1alpha1.ClusterPushSecret {
				pes := defaultClusterPushSecret()
				pes.Spec.RefreshInterval = &metav1.Duration{Duration: 100 * time.Millisecond}
				pes.Spec.NamespaceSelectors = []*metav1.LabelSelector{
					{
						MatchLabels: map[string]string{noneMatchingAnnotationKey: noneMatchingAnnotationVal},
					},
				}
				return *pes
			},
			beforeCheck: func(ctx context.Context, namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) {
				// Wait until the target ESs have been created
				Eventually(func(g Gomega) {
					for _, ns := range namespaces {
						key := types.NamespacedName{Namespace: ns.Name, Name: created.Name}
						g.Expect(k8sClient.Get(ctx, key, &v1alpha1.PushSecret{})).ShouldNot(HaveOccurred())
					}
				}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

				namespaces[0].Labels = map[string]string{}
				Expect(k8sClient.Update(ctx, &namespaces[0])).ShouldNot(HaveOccurred())
			},
			expectedClusterPushSecret: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) v1alpha1.ClusterPushSecret {
				return v1alpha1.ClusterPushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: v1alpha1.ClusterPushSecretStatus{
						PushSecretName:        created.Name,
						ProvisionedNamespaces: []string{namespaces[1].Name},
						Conditions: []v1alpha1.PushSecretStatusCondition{
							{
								Type:   v1alpha1.PushSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedPushSecrets: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) []v1alpha1.PushSecret {
				return []v1alpha1.PushSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[1].Name,
							Name:      created.Name,
						},
						Spec: created.Spec.PushSecretSpec,
					},
				}
			},
		}),
		Entry("Should sync with match expression", clusterPushSecretTestCase{
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
			sourceSecret: defaultSourceSecret,
			clusterPushSecret: func(namespaces []v1.Namespace) v1alpha1.ClusterPushSecret {
				pes := defaultClusterPushSecret()
				pes.Spec.RefreshInterval = &metav1.Duration{Duration: 100 * time.Millisecond}
				pes.Spec.NamespaceSelectors = []*metav1.LabelSelector{
					{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "prefix",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{"foo", "bar"}, // "baz" is excluded
							},
						},
					},
				}
				return *pes
			},
			expectedClusterPushSecret: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) v1alpha1.ClusterPushSecret {
				provisionedNamespaces := []string{namespaces[0].Name, namespaces[1].Name}
				sort.Strings(provisionedNamespaces)
				return v1alpha1.ClusterPushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: v1alpha1.ClusterPushSecretStatus{
						PushSecretName:        created.Name,
						ProvisionedNamespaces: provisionedNamespaces,
						Conditions: []v1alpha1.PushSecretStatusCondition{
							{
								Type:   v1alpha1.PushSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedPushSecrets: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) []v1alpha1.PushSecret {
				return []v1alpha1.PushSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[0].Name,
							Name:      created.Name,
						},
						Spec: created.Spec.PushSecretSpec,
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: namespaces[1].Name,
							Name:      created.Name,
						},
						Spec: created.Spec.PushSecretSpec,
					},
				}
			},
		}),
		Entry("Should be ready if no namespace matches", clusterPushSecretTestCase{
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: randomNamespaceName(),
					},
				},
			},
			clusterPushSecret: func(namespaces []v1.Namespace) v1alpha1.ClusterPushSecret {
				pes := defaultClusterPushSecret()
				pes.Spec.NamespaceSelectors = []*metav1.LabelSelector{
					{
						MatchLabels: map[string]string{kubernetesMetadataLabel: "no-namespace-matches"},
					},
				}
				return *pes
			},
			sourceSecret: defaultSourceSecret,
			expectedClusterPushSecret: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) v1alpha1.ClusterPushSecret {
				return v1alpha1.ClusterPushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: v1alpha1.ClusterPushSecretStatus{
						PushSecretName: created.Name,
						Conditions: []v1alpha1.PushSecretStatusCondition{
							{
								Type:   v1alpha1.PushSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedPushSecrets: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) []v1alpha1.PushSecret {
				return []v1alpha1.PushSecret{}
			},
		}),
		Entry("Should be ready if namespace is selected via the namespace selectors", clusterPushSecretTestCase{
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
			clusterPushSecret: func(namespaces []v1.Namespace) v1alpha1.ClusterPushSecret {
				pes := defaultClusterPushSecret()
				pes.Spec.NamespaceSelectors = []*metav1.LabelSelector{
					{
						MatchLabels: map[string]string{"key": "value1"},
					},
					{
						MatchLabels: map[string]string{"key": "value2"},
					},
				}
				return *pes
			},
			sourceSecret: defaultSourceSecret,
			expectedClusterPushSecret: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) v1alpha1.ClusterPushSecret {
				return v1alpha1.ClusterPushSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name: created.Name,
					},
					Spec: created.Spec,
					Status: v1alpha1.ClusterPushSecretStatus{
						PushSecretName: created.Name,
						ProvisionedNamespaces: []string{
							"namespace1",
							"namespace2",
						},
						Conditions: []v1alpha1.PushSecretStatusCondition{
							{
								Type:   v1alpha1.PushSecretReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}
			},
			expectedPushSecrets: func(namespaces []v1.Namespace, created v1alpha1.ClusterPushSecret) []v1alpha1.PushSecret {
				return []v1alpha1.PushSecret{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "namespace1",
							Name:      created.Name,
						},
						Spec: created.Spec.PushSecretSpec,
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "namespace2",
							Name:      created.Name,
						},
						Spec: created.Spec.PushSecretSpec,
					},
				}
			},
		}))
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
