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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/onsi/gomega/format"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	ctest "github.com/external-secrets/external-secrets/pkg/controllers/commontest"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret/esmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
	"github.com/external-secrets/external-secrets/pkg/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	labelKey           = "label-key"
	labelValue         = "label-value"
	annotationKey      = "annotation-key"
	annotationValue    = "annotation-value"
	existingLabelKey   = "existing-label-key"
	existingLabelValue = "existing-label-value"
)

var (
	fakeProvider   *fake.Client
	metric         dto.Metric
	metricDuration dto.Metric
	timeout        = time.Second * 10
	interval       = time.Millisecond * 250
)

var (
	testSyncCallsTotal *prometheus.CounterVec
	testSyncCallsError *prometheus.CounterVec

	testExternalSecretCondition         *prometheus.GaugeVec
	testExternalSecretReconcileDuration *prometheus.GaugeVec
)

type testCase struct {
	secretStore    esv1beta1.GenericStore
	externalSecret *esv1beta1.ExternalSecret

	// checkCondition should succeed if the externalSecret has the expected condition
	checkCondition func(Gomega, *esv1beta1.ExternalSecret)

	// checkExternalSecret should succeed if the externalSecret is a expected
	// NOTE: this is always called after checkCondition has succeeded
	checkExternalSecret func(*esv1beta1.ExternalSecret)

	// checkSecret should succeed if the target secret is as expected
	// NOTE: this is always called after checkExternalSecret has succeeded
	checkSecret func(*esv1beta1.ExternalSecret, *v1.Secret)
}

type testTweaks func(*testCase)

var _ = Describe("Kind=secret existence logic", func() {
	validData := map[string][]byte{
		"foo": []byte("value1"),
		"bar": []byte("value2"),
	}
	type testCase struct {
		Name           string
		Input          *v1.Secret
		ExpectedOutput bool
	}
	tests := []testCase{
		{
			Name:           "Should not be valid in case of missing uid",
			Input:          &v1.Secret{},
			ExpectedOutput: false,
		},
		{
			Name: "A nil annotation should not be valid",
			Input: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					UID: "xxx",
					Labels: map[string]string{
						esv1beta1.LabelManaged: esv1beta1.LabelManagedValue,
					},
					Annotations: map[string]string{},
				},
			},
			ExpectedOutput: false,
		},
		{
			Name: "An invalid annotation hash should not be valid",
			Input: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					UID: "xxx",
					Labels: map[string]string{
						esv1beta1.LabelManaged: esv1beta1.LabelManagedValue,
					},
					Annotations: map[string]string{
						esv1beta1.AnnotationDataHash: "xxxxxx",
					},
				},
			},
			ExpectedOutput: false,
		},
		{
			Name: "A valid secret should return true",
			Input: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					UID: "xxx",
					Labels: map[string]string{
						esv1beta1.LabelManaged: esv1beta1.LabelManagedValue,
					},
					Annotations: map[string]string{
						esv1beta1.AnnotationDataHash: utils.ObjectHash(validData),
					},
				},
				Data: validData,
			},
			ExpectedOutput: true,
		},
	}

	for _, tt := range tests {
		It(tt.Name, func() {
			Expect(isSecretValid(tt.Input)).To(BeEquivalentTo(tt.ExpectedOutput))
		})
	}
})

// NOTE: because the tests mutate `fakeProvider`, we can't run these blocks in parallel.
// therefore, we run them using the `Serial` Ginkgo decorator to avoid parallelism.
var _ = Describe("ExternalSecret controller", Serial, func() {

	const (
		ExternalSecretName             = "test-es"
		ExternalSecretFQDN             = "externalsecrets.external-secrets.io/test-es"
		ExternalSecretStore            = "test-store"
		ExternalSecretTargetSecretName = "test-secret"
		FakeManager                    = "fake.manager"
		expectedSecretVal              = "SOMEVALUE was templated"
		targetPropObj                  = "{{ .targetProperty | toString | upper }} was templated"
		FooValue                       = "map-foo-value"
		BarValue                       = "map-bar-value"
		NamespaceLabelKey              = "css-test-label-key"
		NamespaceLabelValue            = "css-test-label-value"
	)

	var ExternalSecretNamespace string

	// if we are in debug and need to increase the timeout for testing, we can do so by using an env var
	if customTimeout := os.Getenv("TEST_CUSTOM_TIMEOUT_SEC"); customTimeout != "" {
		if t, err := strconv.Atoi(customTimeout); err == nil {
			timeout = time.Second * time.Duration(t)
		}
	}

	BeforeEach(func() {
		var err error
		ExternalSecretNamespace, err = ctest.CreateNamespaceWithLabels("test-ns", k8sClient, map[string]string{NamespaceLabelKey: NamespaceLabelValue})
		Expect(err).ToNot(HaveOccurred())
		metric.Reset()
		testSyncCallsTotal.Reset()
		testSyncCallsError.Reset()
		testExternalSecretCondition.Reset()
		testExternalSecretReconcileDuration.Reset()
		fakeProvider.Reset()
	})

	AfterEach(
		func() {
			secretStore := &esv1beta1.SecretStore{}
			secretStoreLookupKey := types.NamespacedName{
				Name:      ExternalSecretStore,
				Namespace: ExternalSecretNamespace,
			}

			if err := k8sClient.Get(context.Background(), secretStoreLookupKey, secretStore); err == nil {
				Expect(k8sClient.Delete(context.Background(), secretStore)).To(Succeed())
			}

			clusterSecretStore := &esv1beta1.ClusterSecretStore{}
			clusterSecretStoreLookupKey := types.NamespacedName{
				Name: ExternalSecretStore,
			}

			if err := k8sClient.Get(context.Background(), clusterSecretStoreLookupKey, clusterSecretStore); err == nil {
				Expect(k8sClient.Delete(context.Background(), clusterSecretStore)).To(Succeed())
			}

			Expect(k8sClient.Delete(context.Background(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ExternalSecretNamespace,
				},
			})).To(Succeed())
		},
	)

	const (
		secretVal      = "some-value"
		targetProp     = "targetProperty"
		remoteKey      = "barz"
		remoteProperty = "bang"
		existingKey    = "pre-existing-key"
		existingVal    = "pre-existing-value"
	)

	makeDefaultTestcase := func() *testCase {
		return &testCase{
			// default condition: ES should be ready with reason "SecretSynced"
			checkCondition: func(g Gomega, es *esv1beta1.ExternalSecret) {
				expected := []esv1beta1.ExternalSecretStatusCondition{
					{
						Type:    esv1beta1.ExternalSecretReady,
						Status:  v1.ConditionTrue,
						Reason:  esv1beta1.ConditionReasonSecretSynced,
						Message: msgSynced,
					},
				}
				opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
				g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
			},
			secretStore: &esv1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretStore,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						AWS: &esv1beta1.AWSProvider{
							Service: esv1beta1.AWSServiceSecretsManager,
						},
					},
				},
			},
			externalSecret: &esv1beta1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretName,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1beta1.ExternalSecretSpec{
					SecretStoreRef: esv1beta1.SecretStoreRef{
						Name: ExternalSecretStore,
					},
					Target: esv1beta1.ExternalSecretTarget{
						Name: ExternalSecretTargetSecretName,
					},
					Data: []esv1beta1.ExternalSecretData{
						{
							SecretKey: targetProp,
							RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
								Key:      remoteKey,
								Property: remoteProperty,
							},
						},
					},
				},
			},
		}
	}

	// if target Secret name is not specified it should use the ExternalSecret name.
	syncWithoutTargetName := func(tc *testCase) {
		tc.externalSecret.Spec.Target.Name = ""
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the target secret name")
			Expect(secret.ObjectMeta.Name).To(Equal(ExternalSecretName))

			By("checking the binding secret name")
			Expect(es.Status.Binding.Name).To(Equal(secret.ObjectMeta.Name))
		}
	}

	syncBigNames := func(tc *testCase) {
		tc.externalSecret.Spec.Target.Name = "this-is-a-very-big-secret-name-that-wouldnt-be-generated-due-to-label-limits"
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the target secret name")
			Expect(es.Status.Binding.Name).To(Equal(tc.externalSecret.Spec.Target.Name))
		}
	}
	// the secret name is reflected on the external secret's status as the binding secret
	syncBindingSecret := func(tc *testCase) {
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the binding secret name")
			Expect(es.Status.Binding.Name).To(Equal(secret.ObjectMeta.Name))
		}
	}

	// there is no binding secret when a secret is not synced
	skipBindingSecret := func(tc *testCase) {
		tc.externalSecret.Spec.Target.CreationPolicy = esv1beta1.CreatePolicyNone
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			By("checking the binding secret name is empty")
			Expect(es.Status.Binding.Name).To(BeEmpty())
		}
	}

	// labels and annotations from the Kind=ExternalSecret
	// should be copied over to the Kind=Secret
	syncLabelsAnnotations := func(tc *testCase) {
		tc.externalSecret.ObjectMeta.Labels = map[string]string{
			labelKey: labelValue,
		}
		tc.externalSecret.ObjectMeta.Annotations = map[string]string{
			annotationKey: annotationValue,
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the labels and annotations")
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(labelKey, labelValue))
			Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue(annotationKey, annotationValue))

			By("ensuring the the target secret is owned by the ExternalSecret")
			Expect(ctest.HasOwnerRef(secret.ObjectMeta, "ExternalSecret", ExternalSecretName)).To(BeTrue())
		}
	}

	// labels and annotations from the ExternalSecret
	// should be merged to the Secret if exists
	mergeLabelsAnnotations := func(tc *testCase) {
		tc.externalSecret.ObjectMeta.Labels = map[string]string{
			labelKey: labelValue,
		}
		tc.externalSecret.ObjectMeta.Annotations = map[string]string{
			annotationKey: annotationValue,
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		// Create a secret owned by another entity to test if the pre-existing metadata is preserved
		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
				Labels: map[string]string{
					existingLabelKey: existingLabelValue,
				},
				Annotations: map[string]string{
					"existing-annotation-key": "existing-annotation-value",
				},
			},
		}, client.FieldOwner(FakeManager))).To(Succeed())

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the target secret labels")
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(labelKey, labelValue))
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(existingLabelKey, existingLabelValue))

			By("checking the target secret annotations")
			Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue(annotationKey, annotationValue))
			Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue("existing-annotation-key", "existing-annotation-value"))
		}
	}

	removeOutdatedLabelsAnnotations := func(tc *testCase) {
		tc.externalSecret.ObjectMeta.Labels = map[string]string{
			labelKey: labelValue,
		}
		tc.externalSecret.ObjectMeta.Annotations = map[string]string{
			annotationKey: annotationValue,
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		// Create a secret owned by the operator to test if the outdated pre-existing metadata is removed
		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
				Labels: map[string]string{
					existingLabelKey: existingLabelValue,
				},
				Annotations: map[string]string{
					"existing-annotation-key": "existing-annotation-value",
				},
			},
		}, client.FieldOwner(ExternalSecretFQDN))).To(Succeed())

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the target secret labels")
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(labelKey, labelValue))
			Expect(secret.ObjectMeta.Labels).NotTo(HaveKeyWithValue(existingLabelKey, existingLabelValue))

			By("checking the target secret annotations")
			Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue(annotationKey, annotationValue))
			Expect(secret.ObjectMeta.Annotations).NotTo(HaveKeyWithValue("existing-annotation-key", "existing-annotation-value"))
		}
	}

	checkPrometheusCounters := func(tc *testCase) {
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func(g Gomega) bool {
				g.Expect(testSyncCallsTotal.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				g.Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				// three reconciliations: initial sync, status update, secret update
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())

			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 0.0)
			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 1.0)
		}
	}

	// merge with existing secret using creationPolicy=Merge
	// it should NOT have a ownerReference
	// metadata.managedFields with the correct owner should be added to the secret
	mergeWithSecret := func(tc *testCase) {
		const secretVal = "someValue"
		tc.externalSecret.Spec.Target.CreationPolicy = esv1beta1.CreatePolicyMerge
		tc.externalSecret.Labels = map[string]string{
			"es-label-key": "es-label-value",
		}
		tc.externalSecret.Annotations = map[string]string{
			"es-annotation-key": "es-annotation-value",
		}

		// create secret beforehand
		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
				Labels: map[string]string{
					existingLabelKey: existingLabelValue,
				},
				Annotations: map[string]string{
					"existing-annotation-key": "existing-annotation-value",
				},
			},
			Data: map[string][]byte{
				existingKey: []byte(existingVal),
			},
		}, client.FieldOwner(FakeManager))).To(Succeed())

		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(existingKey, []byte(existingVal)))
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))

			By("checking the secret labels")
			Expect(secret.ObjectMeta.Labels).To(HaveLen(3))
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(existingLabelKey, existingLabelValue))
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue("es-label-key", "es-label-value"))
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(esv1beta1.LabelManaged, esv1beta1.LabelManagedValue))

			By("checking the secret annotations")
			Expect(secret.ObjectMeta.Annotations).To(HaveLen(3))
			Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue("existing-annotation-key", "existing-annotation-value"))
			Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue("es-annotation-key", "es-annotation-value"))
			Expect(secret.ObjectMeta.Annotations).To(HaveKey(esv1beta1.AnnotationDataHash))

			By("ensuring the the secret is not owned by the ExternalSecret")
			Expect(ctest.HasOwnerRef(secret.ObjectMeta, "ExternalSecret", ExternalSecretFQDN)).To(BeFalse())
			Expect(secret.ObjectMeta.ManagedFields).To(HaveLen(2))

			By("ensuring the secret field ownership is correct")
			oldCharactersAroundMismatchToInclude := format.CharactersAroundMismatchToInclude
			format.CharactersAroundMismatchToInclude = 10
			Expect(ctest.FirstManagedFieldForManager(secret.ObjectMeta, ExternalSecretFQDN)).To(
				Equal(fmt.Sprintf(`{"f:data":{"f:targetProperty":{}},"f:metadata":{"f:annotations":{"f:es-annotation-key":{},"f:%s":{}},"f:labels":{"f:es-label-key":{},"f:%s":{}}}}`, esv1beta1.AnnotationDataHash, esv1beta1.LabelManaged)),
			)
			Expect(ctest.FirstManagedFieldForManager(secret.ObjectMeta, FakeManager)).To(
				Equal(`{"f:data":{".":{},"f:pre-existing-key":{}},"f:metadata":{"f:annotations":{".":{},"f:existing-annotation-key":{}},"f:labels":{".":{},"f:existing-label-key":{}}},"f:type":{}}`),
			)
			format.CharactersAroundMismatchToInclude = oldCharactersAroundMismatchToInclude
		}
	}

	mergeWithSecretUpdate := func(tc *testCase) {
		const secretVal = "someValue"
		tc.externalSecret.Spec.Target.CreationPolicy = esv1beta1.CreatePolicyMerge
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Hour}

		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			},
			Data: map[string][]byte{
				existingKey: []byte(existingVal),
			},
		}, client.FieldOwner(FakeManager))).To(Succeed())

		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("updating the target secret")
			Expect(k8sClient.Update(context.Background(), &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretTargetSecretName,
					Namespace: ExternalSecretNamespace,
				},
				Data: map[string][]byte{
					existingKey: []byte("differentValue"),
				},
			}, client.FieldOwner(FakeManager))).To(Succeed())

			By("ensuring the target secret is reverted to the ExternalSecret value")
			Expect(string(secret.Data[existingKey])).To(Equal(existingVal))
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))
		}
	}

	// should not update if no changes
	mergeWithSecretNoChange := func(tc *testCase) {
		tc.externalSecret.Spec.Target.CreationPolicy = esv1beta1.CreatePolicyMerge

		// create secret beforehand
		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			},
			Data: map[string][]byte{
				existingKey: []byte(existingVal),
			},
		}, client.FieldOwner(FakeManager))).To(Succeed())

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			oldResourceVersion := secret.ResourceVersion

			By("patching the secret with no changes")
			cleanSecret := secret.DeepCopy()
			Expect(k8sClient.Patch(context.Background(), secret, client.MergeFrom(cleanSecret))).To(Succeed())

			Consistently(func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{
					Name:      ExternalSecretTargetSecretName,
					Namespace: ExternalSecretNamespace,
				}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())

				By("checking that the target secret was not updated")
				g.Expect(oldResourceVersion).To(Equal(newSecret.ResourceVersion))
			}, time.Second*10, time.Second).Should(Succeed())
		}
	}

	// should not merge with secret if it doesn't exist
	mergeWithSecretErr := func(tc *testCase) {
		const secretVal = "someValue"
		tc.externalSecret.Spec.Target.CreationPolicy = esv1beta1.CreatePolicyMerge

		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:    esv1beta1.ExternalSecretReady,
					Status:  v1.ConditionTrue,
					Reason:  esv1beta1.ConditionReasonSecretMissing,
					Message: msgMissing,
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func(g Gomega) bool {
				g.Expect(testSyncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				g.Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				return metric.GetCounter().GetValue() == 0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())

			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 0.0)
			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 1.0)
		}
	}

	// controller should force ownership
	mergeWithConflict := func(tc *testCase) {
		const secretVal = "someValue"
		// this should conflict with the existing secret data key
		const existingKey = targetProp
		tc.externalSecret.Spec.Target.CreationPolicy = esv1beta1.CreatePolicyMerge

		// create secret beforehand
		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			},
			Data: map[string][]byte{
				existingKey: []byte(existingVal),
			},
		}, client.FieldOwner(FakeManager))).To(Succeed())

		fakeProvider.WithGetSecret([]byte(secretVal), nil)

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("ensuring the existing key was overwritten")
			Expect(secret.Data).To(HaveKeyWithValue(existingKey, []byte(secretVal)))

			By("ensuring the the secret is not owned by the ExternalSecret")
			Expect(ctest.HasOwnerRef(secret.ObjectMeta, "ExternalSecret", ExternalSecretFQDN)).To(BeFalse())

			By("ensuring the secret field ownership is correct")
			Expect(secret.ObjectMeta.ManagedFields).To(HaveLen(2))
			oldCharactersAroundMismatchToInclude := format.CharactersAroundMismatchToInclude
			format.CharactersAroundMismatchToInclude = 10
			Expect(ctest.FirstManagedFieldForManager(secret.ObjectMeta, ExternalSecretFQDN)).To(
				Equal(fmt.Sprintf(`{"f:data":{"f:targetProperty":{}},"f:metadata":{"f:annotations":{".":{},"f:%s":{}},"f:labels":{".":{},"f:%s":{}}}}`, esv1beta1.AnnotationDataHash, esv1beta1.LabelManaged)),
			)
			format.CharactersAroundMismatchToInclude = oldCharactersAroundMismatchToInclude
		}
	}

	syncWithGeneratorRef := func(tc *testCase) {
		const secretKey = "somekey"
		const secretVal = "someValue"

		Expect(k8sClient.Create(context.Background(), &genv1alpha1.Fake{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mytestfake",
				Namespace: ExternalSecretNamespace,
			},
			Spec: genv1alpha1.FakeSpec{
				Data: map[string]string{
					secretKey: secretVal,
				},
			},
		})).To(Succeed())

		// reset secretStoreRef
		tc.externalSecret.Spec.SecretStoreRef = esv1beta1.SecretStoreRef{}
		tc.externalSecret.Spec.Data = nil
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				SourceRef: &esv1beta1.StoreGeneratorSourceRef{
					GeneratorRef: &esv1beta1.GeneratorRef{
						APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version,
						Kind:       "Fake",
						Name:       "mytestfake",
					},
				},
			},
		}

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the target secret data")
			Expect(secret.Data).To(HaveKeyWithValue(secretKey, []byte(secretVal)))
		}
	}
	syncWithClusterGeneratorRef := func(tc *testCase) {
		const secretKey = "somekey2"
		const secretVal = "someValue2"
		Expect(k8sClient.Create(context.Background(), &genv1alpha1.ClusterGenerator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mytestfake",
			},
			Spec: genv1alpha1.ClusterGeneratorSpec{
				Kind: "Fake",
				Generator: genv1alpha1.GeneratorSpec{
					FakeSpec: &genv1alpha1.FakeSpec{
						Data: map[string]string{
							secretKey: secretVal,
						},
					},
				},
			},
		})).To(Succeed())

		// reset secretStoreRef
		tc.externalSecret.Spec.SecretStoreRef = esv1beta1.SecretStoreRef{}
		tc.externalSecret.Spec.Data = nil
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				SourceRef: &esv1beta1.StoreGeneratorSourceRef{
					GeneratorRef: &esv1beta1.GeneratorRef{
						APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version,
						Kind:       "ClusterGenerator",
						Name:       "mytestfake",
					},
				},
			},
		}

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the target secret data")
			Expect(secret.Data).To(HaveKeyWithValue(secretKey, []byte(secretVal)))
		}
	}

	deleteOrphanedSecrets := func(tc *testCase) {
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("ensuring the old target secret exists")
			oldSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: secret.Namespace}
			oldSecret := &v1.Secret{}
			Expect(k8sClient.Get(context.Background(), oldSecretKey, oldSecret)).To(Succeed())

			By("changing the ES target secret name")
			patch := client.MergeFrom(es.DeepCopy())
			es.Spec.Target.Name = "new-foo"
			Expect(k8sClient.Patch(context.Background(), es, patch)).To(Succeed())

			By("waiting for the new target secret to be created")
			Eventually(func(g Gomega) {
				newSecretKey := types.NamespacedName{Name: "new-foo", Namespace: secret.Namespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("ensuring the old target secret is deleted")
			Eventually(func(g Gomega) bool {
				err := k8sClient.Get(context.Background(), oldSecretKey, oldSecret)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		}
	}

	syncWithMultipleSecretStores := func(tc *testCase) {
		Expect(k8sClient.Create(context.Background(), &esv1beta1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: ExternalSecretNamespace,
			},
			Spec: esv1beta1.SecretStoreSpec{
				Provider: &esv1beta1.SecretStoreProvider{
					Fake: &esv1beta1.FakeProvider{
						Data: []esv1beta1.FakeProviderData{
							{
								Key:     "foo",
								Version: "",
								ValueMap: map[string]string{
									"foo":  "bar",
									"foo2": "bar2",
								},
							},
						},
					},
				},
			},
		})).To(Succeed())

		Expect(k8sClient.Create(context.Background(), &esv1beta1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "baz",
				Namespace: ExternalSecretNamespace,
			},
			Spec: esv1beta1.SecretStoreSpec{
				Provider: &esv1beta1.SecretStoreProvider{
					Fake: &esv1beta1.FakeProvider{
						Data: []esv1beta1.FakeProviderData{
							{
								Key:     "baz",
								Version: "",
								ValueMap: map[string]string{
									"baz":  "bang",
									"baz2": "bang2",
								},
							},
						},
					},
				},
			},
		})).To(Succeed())

		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: "foo",
				},
				SourceRef: &esv1beta1.StoreGeneratorSourceRef{
					SecretStoreRef: &esv1beta1.SecretStoreRef{
						Name: "foo",
						Kind: esv1beta1.SecretStoreKind,
					},
				},
			},
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: "baz",
				},
				SourceRef: &esv1beta1.StoreGeneratorSourceRef{
					SecretStoreRef: &esv1beta1.SecretStoreRef{
						Name: "baz",
						Kind: esv1beta1.SecretStoreKind,
					},
				},
			},
		}

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the target secret data")
			Expect(secret.Data).To(HaveKeyWithValue("foo", []byte("bar")))
			Expect(secret.Data).To(HaveKeyWithValue("foo2", []byte("bar2")))
			Expect(secret.Data).To(HaveKeyWithValue("baz", []byte("bang")))
			Expect(secret.Data).To(HaveKeyWithValue("baz2", []byte("bang2")))
		}
	}

	// when using a template it should be used as a blueprint
	// to construct a new secret: labels, annotations and type
	syncWithTemplate := func(tc *testCase) {
		const secretVal = "someValue"
		const tplStaticKey = "tplstatickey"
		const tplStaticVal = "tplstaticvalue"
		tc.externalSecret.ObjectMeta.Labels = map[string]string{
			"fooobar": "bazz",
		}
		tc.externalSecret.ObjectMeta.Annotations = map[string]string{
			"hihihih": "hehehe",
		}
		tc.externalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Metadata: esv1beta1.ExternalSecretTemplateMetadata{
				Labels: map[string]string{
					"foos": "ball",
				},
				Annotations: map[string]string{
					"hihi": "ga",
				},
			},
			Type:          v1.SecretTypeOpaque,
			EngineVersion: esv1beta1.TemplateEngineV1,
			Data: map[string]string{
				targetProp:   targetPropObj,
				tplStaticKey: tplStaticVal,
			},
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret values")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(expectedSecretVal)))
			Expect(secret.Data).To(HaveKeyWithValue(tplStaticKey, []byte(tplStaticVal)))

			By("ensuring the labels from the ES template are applied")
			for k, v := range es.Spec.Target.Template.Metadata.Labels {
				Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(k, v))
			}

			By("ensuring the annotations from the ES template are applied")
			for k, v := range es.Spec.Target.Template.Metadata.Annotations {
				Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue(k, v))
			}
		}
	}

	// when using a v2 template it should use the v2 engine version
	syncWithTemplateV2 := func(tc *testCase) {
		const secretVal = "someValue"
		tc.externalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Type: v1.SecretTypeOpaque,
			// it should default to v2 for beta11
			// EngineVersion: esv1beta1.TemplateEngineV2,
			Data: map[string]string{
				targetProp: "{{ .targetProperty | upper }} was templated",
			},
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(expectedSecretVal)))
		}
	}

	// secret should be synced with correct value precedence:
	// * fromString
	// * template data
	// * templateFrom
	// * data
	// * dataFrom
	syncWithTemplatePrecedence := func(tc *testCase) {
		const secretVal = "someValue"
		const tplStaticKey = "tplstatickey"
		const tplStaticVal = "tplstaticvalue"
		const tplFromCMName = "template-cm"
		const tplFromSecretName = "template-secret"
		const tplFromKey = "tpl-from-key"
		const tplFromSecKey = "tpl-from-sec-key"
		const tplFromVal = "tpl-from-value: {{ .targetProperty | toString }} // {{ .bar | toString }}"
		const tplFromSecVal = "tpl-from-sec-value: {{ .targetProperty | toString }} // {{ .bar | toString }}"
		Expect(k8sClient.Create(context.Background(), &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tplFromCMName,
				Namespace: ExternalSecretNamespace,
			},
			Data: map[string]string{
				tplFromKey: tplFromVal,
			},
		})).To(Succeed())
		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tplFromSecretName,
				Namespace: ExternalSecretNamespace,
			},
			Data: map[string][]byte{
				tplFromSecKey: []byte(tplFromSecVal),
			},
		})).To(Succeed())
		tc.externalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Metadata: esv1beta1.ExternalSecretTemplateMetadata{},
			Type:     v1.SecretTypeOpaque,
			TemplateFrom: []esv1beta1.TemplateFrom{
				{
					ConfigMap: &esv1beta1.TemplateRef{
						Name: tplFromCMName,
						Items: []esv1beta1.TemplateRefItem{
							{
								Key: tplFromKey,
							},
						},
					},
				},
				{
					Secret: &esv1beta1.TemplateRef{
						Name: tplFromSecretName,
						Items: []esv1beta1.TemplateRefItem{
							{
								Key: tplFromSecKey,
							},
						},
					},
				},
			},
			Data: map[string]string{
				// this should be the data value, not dataFrom
				targetProp: targetPropObj,
				// this should use the value from the map
				"bar": "value from map: {{ .bar | toString }}",
				// just a static value
				tplStaticKey: tplStaticVal,
			},
		}
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: "datamap",
				},
			},
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		fakeProvider.WithGetSecretMap(map[string][]byte{
			"targetProperty": []byte(FooValue),
			"bar":            []byte(BarValue),
		}, nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(expectedSecretVal)))
			Expect(secret.Data).To(HaveKeyWithValue(tplStaticKey, []byte(tplStaticVal)))
			Expect(secret.Data).To(HaveKeyWithValue("bar", []byte("value from map: map-bar-value")))
			Expect(secret.Data).To(HaveKeyWithValue(tplFromKey, []byte("tpl-from-value: someValue // map-bar-value")))
			Expect(secret.Data).To(HaveKeyWithValue(tplFromSecKey, []byte("tpl-from-sec-value: someValue // map-bar-value")))
		}
	}
	syncTemplateFromKeysAndValues := func(tc *testCase) {
		const tplFromCMName = "template-cm"
		const tplFromSecretName = "template-secret"
		const tplFromKey = "tpl-from-key"
		const tplFromSecKey = "tpl-from-sec-key"
		const tplFromVal = "{{ .targetKey }}-cm: {{ .targetValue }}"
		const tplFromSecVal = "{{ .targetKey }}-sec: {{ .targetValue }}"
		Expect(k8sClient.Create(context.Background(), &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tplFromCMName,
				Namespace: ExternalSecretNamespace,
			},
			Data: map[string]string{
				tplFromKey: tplFromVal,
			},
		})).To(Succeed())
		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tplFromSecretName,
				Namespace: ExternalSecretNamespace,
			},
			Data: map[string][]byte{
				tplFromSecKey: []byte(tplFromSecVal),
			},
		})).To(Succeed())
		tc.externalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Metadata: esv1beta1.ExternalSecretTemplateMetadata{},
			Type:     v1.SecretTypeOpaque,
			TemplateFrom: []esv1beta1.TemplateFrom{
				{
					ConfigMap: &esv1beta1.TemplateRef{
						Name: tplFromCMName,
						Items: []esv1beta1.TemplateRefItem{
							{
								Key:        tplFromKey,
								TemplateAs: esv1beta1.TemplateScopeKeysAndValues,
							},
						},
					},
				},
				{
					Secret: &esv1beta1.TemplateRef{
						Name: tplFromSecretName,
						Items: []esv1beta1.TemplateRefItem{
							{
								Key:        tplFromSecKey,
								TemplateAs: esv1beta1.TemplateScopeKeysAndValues,
							},
						},
					},
				},
			},
		}
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: "datamap",
				},
			},
		}
		fakeProvider.WithGetSecretMap(map[string][]byte{
			"targetKey":   []byte(FooValue),
			"targetValue": []byte(BarValue),
		}, nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue("map-foo-value-cm", []byte(BarValue)))
			Expect(secret.Data).To(HaveKeyWithValue("map-foo-value-sec", []byte(BarValue)))
		}
	}

	syncTemplateFromLiteral := func(tc *testCase) {
		tplDataVal := "{{ .targetKey }}-literal: {{ .targetValue }}"
		tplAnnotationsVal := "{{ .targetKey }}-annotations: {{ .targetValue }}"
		tplLabelsVal := "{{ .targetKey }}-labels: {{ .targetValue }}"
		tplComplexVal := `
{{- range $k, $v := ( .complex | fromJson )}}
{{ $k }}: {{ $v }}
{{- end }}
`
		tc.externalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Metadata: esv1beta1.ExternalSecretTemplateMetadata{},
			Type:     v1.SecretTypeOpaque,
			TemplateFrom: []esv1beta1.TemplateFrom{
				{
					Literal: &tplDataVal,
				},
				{
					Literal: &tplComplexVal,
				},
				{
					Target:  esv1beta1.TemplateTargetAnnotations,
					Literal: &tplAnnotationsVal,
				},
				{
					Target:  esv1beta1.TemplateTargetLabels,
					Literal: &tplLabelsVal,
				},
			},
		}
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: "datamap",
				},
			},
		}
		fakeProvider.WithGetSecretMap(map[string][]byte{
			"targetKey":   []byte(FooValue),
			"targetValue": []byte(BarValue),
			"complex":     []byte("{\"nested\":\"json\",\"can\":\"be\",\"templated\":\"successfully\"}"),
		}, nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret metadata")
			Expect(secret.Annotations).To(HaveKeyWithValue("map-foo-value-annotations", BarValue))
			Expect(secret.Labels).To(HaveKeyWithValue("map-foo-value-labels", BarValue))

			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue("map-foo-value-literal", []byte(BarValue)))
			Expect(secret.Data).To(HaveKeyWithValue("nested", []byte("json")))
			Expect(secret.Data).To(HaveKeyWithValue("can", []byte("be")))
			Expect(secret.Data).To(HaveKeyWithValue("templated", []byte("successfully")))
		}
	}

	refreshWithTemplate := func(tc *testCase) {
		const secretVal = "someValue"
		const tplStaticKey = "tplstatickey"
		const tplStaticVal = "tplstaticvalue"
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.externalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Metadata: esv1beta1.ExternalSecretTemplateMetadata{
				Labels:      map[string]string{"foo": "bar"},
				Annotations: map[string]string{"foo": "bar"},
			},
			Type: v1.SecretTypeOpaque,
			Data: map[string]string{
				targetProp:   targetPropObj,
				tplStaticKey: tplStaticVal,
			},
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(expectedSecretVal)))
			Expect(secret.Data).To(HaveKeyWithValue(tplStaticKey, []byte(tplStaticVal)))

			By("ensuring the labels from the ES template are applied")
			for k, v := range es.Spec.Target.Template.Metadata.Labels {
				Expect(secret.Labels).To(HaveKeyWithValue(k, v))
			}

			By("ensuring the annotations from the ES template are applied")
			for k, v := range es.Spec.Target.Template.Metadata.Annotations {
				Expect(secret.Annotations).To(HaveKeyWithValue(k, v))
			}

			By("patching the ExternalSecret with a new template")
			patch := client.MergeFrom(es.DeepCopy())
			es.Spec.Target.Template.Metadata.Annotations["fuzz"] = "buzz"
			es.Spec.Target.Template.Metadata.Labels["fuzz"] = "buzz"
			es.Spec.Target.Template.Data["new"] = "value"
			Expect(k8sClient.Patch(context.Background(), es, patch)).To(Succeed())

			Eventually(func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())

				By("ensuring the new data is present")
				g.Expect(newSecret.Data).To(HaveKeyWithValue("new", []byte("value")))

				By("ensuring the new labels are present")
				for k, v := range es.Spec.Target.Template.Metadata.Labels {
					g.Expect(newSecret.ObjectMeta.Labels).To(HaveKeyWithValue(k, v))
				}

				By("ensuring the new annotations are present")
				for k, v := range es.Spec.Target.Template.Metadata.Annotations {
					g.Expect(newSecret.ObjectMeta.Annotations).To(HaveKeyWithValue(k, v))
				}
			}, timeout, interval).Should(Succeed())
		}
	}

	onlyMetadataFromTemplate := func(tc *testCase) {
		const secretVal = "someValue"
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.externalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Metadata: esv1beta1.ExternalSecretTemplateMetadata{
				Labels:      map[string]string{"foo": "bar"},
				Annotations: map[string]string{"foo": "bar"},
			},
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))

			By("ensuring the labels from the ES template are applied")
			for k, v := range es.Spec.Target.Template.Metadata.Labels {
				Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(k, v))
			}

			By("ensuring the annotations from the ES template are applied")
			for k, v := range es.Spec.Target.Template.Metadata.Annotations {
				Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue(k, v))
			}
		}
	}

	// when the provider secret changes the Kind=Secret value
	// must change, too.
	refreshSecretValue := func(tc *testCase) {
		const targetProp = "targetProperty"
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))

			By("updating the provider secret")
			newValue := "NEW VALUE"
			fakeProvider.WithGetSecret([]byte(newValue), nil)

			Eventually(func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())

				By("ensuring the new data is present")
				g.Expect(newSecret.Data).To(HaveKeyWithValue(targetProp, []byte(newValue)))
			}, timeout, interval).Should(Succeed())
		}
	}

	// when a provider secret was deleted it must be deleted from
	// the secret aswell
	refreshSecretValueMap := func(tc *testCase) {
		fakeProvider.WithGetSecretMap(map[string][]byte{
			"foo": []byte("1111"),
			"bar": []byte("2222"),
		}, nil)
		tc.externalSecret.Spec.Data = []esv1beta1.ExternalSecretData{}
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: remoteKey,
				},
			},
		}
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue("foo", []byte("1111")))
			Expect(secret.Data).To(HaveKeyWithValue("bar", []byte("2222")))

			By("removing the bar key from the provider secret")
			fakeProvider.WithGetSecretMap(map[string][]byte{
				"foo": []byte("1111"),
			}, nil)

			Eventually(func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())

				By("ensuring the bar key is removed from the target secret")
				g.Expect(newSecret.Data).To(HaveKeyWithValue("foo", []byte("1111")))
				g.Expect(newSecret.Data).ToNot(HaveKey("bar"))
			}, timeout, interval).Should(Succeed())
		}
	}

	// when a provider secret was deleted it must be deleted from
	// the secret aswell when using a template
	refreshSecretValueMapTemplate := func(tc *testCase) {
		fakeProvider.WithGetSecretMap(map[string][]byte{
			"foo": []byte("1111"),
			"bar": []byte("2222"),
		}, nil)
		tc.externalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{}
		tc.externalSecret.Spec.Data = []esv1beta1.ExternalSecretData{}
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: remoteKey,
				},
			},
		}
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue("foo", []byte("1111")))
			Expect(secret.Data).To(HaveKeyWithValue("bar", []byte("2222")))

			By("removing the bar key from the provider secret")
			fakeProvider.WithGetSecretMap(map[string][]byte{
				"foo": []byte("1111"),
			}, nil)

			Eventually(func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())

				By("ensuring the bar key is removed from the target secret")
				g.Expect(newSecret.Data).To(HaveKeyWithValue("foo", []byte("1111")))
				g.Expect(newSecret.Data).ToNot(HaveKey("bar"))
			}, timeout, interval).Should(Succeed())
		}
	}

	refreshintervalZero := func(tc *testCase) {
		const targetProp = "targetProperty"
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: 0}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))

			By("updating the provider secret")
			newValue := "NEW VALUE"
			fakeProvider.WithGetSecret([]byte(newValue), nil)

			Consistently(func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())

				By("ensuring the target secret is not updated")
				g.Expect(newSecret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))
			}, time.Second*10, time.Second).Should(Succeed())
		}
	}

	refreshInClusterK8sProvider := func(tc *testCase) {
		const sourceSecret1Name = "source-secret-1"
		const sourceSecret2Name = "source-secret-2"
		const serviceAccountName = "k8s-provider-sa"

		tc.secretStore = &esv1beta1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretStore,
				Namespace: ExternalSecretNamespace,
			},
			Spec: esv1beta1.SecretStoreSpec{
				Provider: &esv1beta1.SecretStoreProvider{
					Kubernetes: &esv1beta1.KubernetesProvider{
						RemoteNamespace: ExternalSecretNamespace,
						Server: esv1beta1.KubernetesServer{
							URL:      testEnv.ControlPlane.APIServer.ListenAddr.URL("https", "").String(),
							CABundle: testEnv.ControlPlane.APIServer.CA,
						},
						Auth: esv1beta1.KubernetesAuth{
							ServiceAccount: &esmeta.ServiceAccountSelector{
								Name: serviceAccountName,
							},
						},
					},
				},
			},
		}

		tc.externalSecret.Spec.Target.DeletionPolicy = esv1beta1.DeletionPolicyDelete         // because some source secrets will not be created initially
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Minute * 10} // needs to be longer than the test will run
		tc.externalSecret.Spec.Data = nil
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: sourceSecret1Name,
				},
			},
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: sourceSecret2Name,
				},
			},
		}

		// create the service account beforehand
		By("creating the k8s provider service account")
		Expect(k8sClient.Create(context.Background(), &v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: ExternalSecretNamespace,
			},
		})).To(Succeed())

		// create RBAC rules for the service account
		By("creating the k8s provider RBAC rules")
		Expect(k8sClient.Create(context.Background(), &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: ExternalSecretNamespace,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"authorization.k8s.io"},
					Resources: []string{"selfsubjectrulesreviews"},
					Verbs:     []string{"create"},
				},
			},
		})).To(Succeed())
		Expect(k8sClient.Create(context.Background(), &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: ExternalSecretNamespace,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccountName,
					Namespace: ExternalSecretNamespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     serviceAccountName,
			},
		})).To(Succeed())

		// create source secret 1
		// NOTE: we do not create source secret 2, as we will create it later to test trigger on create
		By("creating source secret 1")
		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sourceSecret1Name,
				Namespace: ExternalSecretNamespace,
			},
			Data: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			},
		})).To(Succeed())

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(BeComparableTo(map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			}))

			By("ensuring the ES does not have a trigger annotation")
			Expect(es.Annotations).ToNot(HaveKey(esv1beta1.AnnotationRefreshTrigger))

			By("checking the status.triggers.inClusterSecrets")
			expectedTriggers := esv1beta1.ExternalSecretStatusTriggers{
				InClusterSecrets: []esv1beta1.TriggerInClusterSecret{
					{
						Name:      sourceSecret1Name,
						Namespace: ExternalSecretNamespace,
					},
					{
						Name:      sourceSecret2Name,
						Namespace: ExternalSecretNamespace,
					},
				},
			}
			Expect(es.Status.Triggers).To(BeComparableTo(expectedTriggers))

			By("getting source secret 1")
			sourceSecret1Key := types.NamespacedName{Name: sourceSecret1Name, Namespace: ExternalSecretNamespace}
			sourceSecret1 := &v1.Secret{}
			Expect(k8sClient.Get(context.Background(), sourceSecret1Key, sourceSecret1)).To(Succeed())

			By("updating source secret 1")
			const newValue = "new-value"
			patch := client.MergeFrom(sourceSecret1.DeepCopy())
			sourceSecret1.Data["key1"] = []byte(newValue)
			Expect(k8sClient.Patch(context.Background(), sourceSecret1, patch)).To(Succeed())

			By("ensuring the target secret is updated")
			Eventually(func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())
				g.Expect(newSecret.Data).To(HaveKeyWithValue("key1", []byte(newValue)))
			}, timeout, interval).Should(Succeed())

			By("getting the updated ES")
			updatedESKey := types.NamespacedName{Name: ExternalSecretName, Namespace: ExternalSecretNamespace}
			updatedES := &esv1beta1.ExternalSecret{}
			Expect(k8sClient.Get(context.Background(), updatedESKey, updatedES)).To(Succeed())

			By("checking the ES trigger annotation")
			expectedTrigger := RefreshTriggerCause{
				InClusterSecret: &RefreshTriggerInClusterSecret{
					Name:            sourceSecret1Name,
					Namespace:       ExternalSecretNamespace,
					ResourceVersion: sourceSecret1.ResourceVersion,
				},
			}
			expectedTriggerJSON, err := json.Marshal(expectedTrigger)
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedES.Annotations).To(HaveKeyWithValue(esv1beta1.AnnotationRefreshTrigger, string(expectedTriggerJSON)))

			By("creating source secret 2")
			sourceSecret2 := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sourceSecret2Name,
					Namespace: ExternalSecretNamespace,
				},
				Data: map[string][]byte{
					"key3": []byte("value3"),
					"key4": []byte("value4"),
				},
			}
			Expect(k8sClient.Create(context.Background(), sourceSecret2)).To(Succeed())

			By("ensuring the target secret is updated")
			Eventually(func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())
				g.Expect(newSecret.Data).To(BeComparableTo(map[string][]byte{
					"key1": []byte(newValue),
					"key2": []byte("value2"),
					"key3": []byte("value3"),
					"key4": []byte("value4"),
				}))
			}, timeout, interval).Should(Succeed())

			By("getting the updated ES")
			Expect(k8sClient.Get(context.Background(), updatedESKey, updatedES)).To(Succeed())

			By("checking the ES trigger annotation")
			expectedTrigger = RefreshTriggerCause{
				InClusterSecret: &RefreshTriggerInClusterSecret{
					Name:            sourceSecret2Name,
					Namespace:       ExternalSecretNamespace,
					ResourceVersion: sourceSecret2.ResourceVersion,
				},
			}
			expectedTriggerJSON, err = json.Marshal(expectedTrigger)
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedES.Annotations).To(HaveKeyWithValue(esv1beta1.AnnotationRefreshTrigger, string(expectedTriggerJSON)))
		}
	}

	deletionPolicyDelete := func(tc *testCase) {
		expVal := []byte("1234")
		// set initial value
		fakeProvider.WithGetAllSecrets(map[string][]byte{
			"foo": expVal,
			"bar": expVal,
		}, nil)
		tc.externalSecret.Spec.Data = nil
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Find: &esv1beta1.ExternalSecretFind{
					Tags: map[string]string{},
				},
			},
		}
		tc.externalSecret.Spec.Target.DeletionPolicy = esv1beta1.DeletionPolicyDelete
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue("foo", expVal))
			Expect(secret.Data).To(HaveKeyWithValue("bar", expVal))

			By("removing the bar key from the provider secret")
			fakeProvider.WithGetAllSecrets(map[string][]byte{
				"foo": expVal,
			}, nil)

			Eventually(func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())

				By("ensuring the bar key is removed from the target secret")
				g.Expect(newSecret.Data).To(HaveKeyWithValue("foo", expVal))
				g.Expect(newSecret.Data).ToNot(HaveKey("bar"))
			}, timeout, interval).Should(Succeed())

			By("removing all keys from the provider secret")
			fakeProvider.WithGetAllSecrets(map[string][]byte{}, esv1beta1.NoSecretErr)

			By("ensuring the target secret is deleted")
			Eventually(func(g Gomega) bool {
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				err := k8sClient.Get(context.Background(), newSecretKey, newSecret)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		}
	}

	deletionPolicyRetain := func(tc *testCase) {
		expVal := []byte("1234")
		// set initial value
		fakeProvider.WithGetAllSecrets(map[string][]byte{
			"foo": expVal,
			"bar": expVal,
		}, nil)
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Find: &esv1beta1.ExternalSecretFind{
					Tags: map[string]string{},
				},
			},
		}
		tc.externalSecret.Spec.Target.DeletionPolicy = esv1beta1.DeletionPolicyRetain
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue("foo", expVal))
			Expect(secret.Data).To(HaveKeyWithValue("bar", expVal))

			By("removing all keys from the provider secret")
			fakeProvider.WithGetAllSecrets(map[string][]byte{}, esv1beta1.NoSecretErr)

			Consistently(func(g Gomega) {
				By("ensuring the target secret is not deleted")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())

				By("ensuring the target secret still has the keys")
				g.Expect(newSecret.Data).To(HaveKeyWithValue("foo", expVal))
				g.Expect(newSecret.Data).To(HaveKeyWithValue("bar", expVal))
			}, time.Second*10, time.Second).Should(Succeed())
		}
	}

	deletionPolicyRetainEmptyData := func(tc *testCase) {
		// set initial value
		fakeProvider.WithGetAllSecrets(make(map[string][]byte), nil)
		tc.externalSecret.Spec.Data = make([]esv1beta1.ExternalSecretData, 0)
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Find: &esv1beta1.ExternalSecretFind{
					Tags: map[string]string{
						"non-existing-key": "non-existing-value",
					},
				},
			},
		}
		tc.externalSecret.Spec.Target.DeletionPolicy = esv1beta1.DeletionPolicyRetain
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:    esv1beta1.ExternalSecretReady,
					Status:  v1.ConditionTrue,
					Reason:  esv1beta1.ConditionReasonSecretSynced,
					Message: msgSyncedRetain,
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
	}

	// merge with existing secret using creationPolicy=Merge
	// if provider secret gets deleted only the managed field should get deleted
	deletionPolicyMerge := func(tc *testCase) {
		const secretVal = "someValue"
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.externalSecret.Spec.Target.CreationPolicy = esv1beta1.CreatePolicyMerge
		tc.externalSecret.Spec.Target.DeletionPolicy = esv1beta1.DeletionPolicyMerge

		// create secret beforehand
		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			},
			Data: map[string][]byte{
				existingKey: []byte(existingVal),
			},
		}, client.FieldOwner(FakeManager))).To(Succeed())

		fakeProvider.WithGetSecret([]byte(secretVal), nil)

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(existingKey, []byte(existingVal)))
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))

			By("removing the provider secret")
			fakeProvider.WithGetSecret(nil, esv1beta1.NoSecretErr)

			// secretIsValid checks if the target secret is in the correct state
			// we define it here so we can call it in both Eventually and Consistently
			secretIsValid := func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())

				By("ensuring the existing key is still present")
				g.Expect(newSecret.Data).To(HaveKeyWithValue(existingKey, []byte(existingVal)))

				By("ensuring the managed data key is removed")
				g.Expect(newSecret.Data).NotTo(HaveKey(targetProp))
			}

			By("waiting for the secret to be in the correct state")
			Eventually(secretIsValid, timeout, interval).Should(Succeed())

			By("ensuring the secret stays in the correct state")
			Consistently(secretIsValid, time.Second*15, time.Second).Should(Succeed())
		}
	}

	// orphan the secret after the external secret has been deleted
	createSecretPolicyOrphan := func(tc *testCase) {
		const secretVal = "someValue"
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.externalSecret.Spec.Target.CreationPolicy = esv1beta1.CreatePolicyOrphan

		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))

			By("deleting the external secret")
			Expect(k8sClient.Delete(context.Background(), es)).To(Succeed())

			By("ensuring the target secret is not deleted")
			Consistently(func(g Gomega) {
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())
			}, time.Second*10, time.Second).Should(Succeed())
		}
	}

	// with rewrite all keys from a dataFrom operation
	// should be put with new rewriting into the secret
	syncAndRewriteWithDataFrom := func(tc *testCase) {
		tc.externalSecret.Spec.Data = nil
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: remoteKey,
				},
				Rewrite: []esv1beta1.ExternalSecretRewrite{{
					Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
						Source: "(.*)",
						Target: "new-$1",
					},
				}},
			},
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: remoteKey,
				},
				Rewrite: []esv1beta1.ExternalSecretRewrite{{
					Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
						Source: "(.*)",
						Target: "old-$1",
					},
				}},
			},
		}
		fakeProvider.WithGetSecretMap(map[string][]byte{
			"foo": []byte(FooValue),
			"bar": []byte(BarValue),
		}, nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue("new-foo", []byte(FooValue)))
			Expect(secret.Data).To(HaveKeyWithValue("new-bar", []byte(BarValue)))
			Expect(secret.Data).To(HaveKeyWithValue("old-foo", []byte(FooValue)))
			Expect(secret.Data).To(HaveKeyWithValue("old-bar", []byte(BarValue)))
		}
	}

	// with rewrite keys from dataFrom
	// should error if keys are not compliant
	invalidExtractKeysErrCondition := func(tc *testCase) {
		tc.externalSecret.Spec.Data = nil
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: remoteKey,
				},
				Rewrite: []esv1beta1.ExternalSecretRewrite{{
					Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
						Source: "(.*)",
						Target: "$1",
					},
				}},
			},
		}
		fakeProvider.WithGetSecretMap(map[string][]byte{
			"foo/bar": []byte(FooValue),
			"bar/foo": []byte(BarValue),
		}, nil)
		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:    esv1beta1.ExternalSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  esv1beta1.ConditionReasonSecretSyncedError,
					Message: fmt.Errorf(errProviderData, fmt.Errorf(errSpecDataFromExtract, 0, remoteKey, fmt.Errorf(errInvalidKeys, errors.New("key has invalid character /, only alphanumeric, '-', '.' and '_' are allowed: foo/bar")))).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func(g Gomega) bool {
				g.Expect(testSyncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				g.Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())

			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 1.0)
			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)
		}

	}

	// with rewrite keys from dataFrom
	// should error if keys are not compliant
	invalidFindKeysErrCondition := func(tc *testCase) {
		tc.externalSecret.Spec.Data = nil
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Find: &esv1beta1.ExternalSecretFind{
					Name: &esv1beta1.FindName{
						RegExp: ".*",
					},
				},
				Rewrite: []esv1beta1.ExternalSecretRewrite{{
					Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
						Source: "(.*)",
						Target: "$1",
					},
				}},
			},
		}
		fakeProvider.WithGetAllSecrets(map[string][]byte{
			"foo/bar": []byte(FooValue),
			"bar/foo": []byte(BarValue),
		}, nil)
		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:    esv1beta1.ExternalSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  esv1beta1.ConditionReasonSecretSyncedError,
					Message: fmt.Errorf(errProviderData, fmt.Errorf(errSpecDataFromFind, 0, fmt.Errorf(errInvalidKeys, errors.New("key has invalid character /, only alphanumeric, '-', '.' and '_' are allowed: foo/bar")))).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func(g Gomega) bool {
				g.Expect(testSyncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				g.Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())

			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 1.0)
			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)
		}

	}

	// with dataFrom all properties from the specified secret
	// should be put into the secret
	syncWithDataFrom := func(tc *testCase) {
		tc.externalSecret.Spec.Data = nil
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: remoteKey,
				},
			},
		}
		fakeProvider.WithGetSecretMap(map[string][]byte{
			"foo": []byte(FooValue),
			"bar": []byte(BarValue),
		}, nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue("foo", []byte(FooValue)))
			Expect(secret.Data).To(HaveKeyWithValue("bar", []byte(BarValue)))
		}
	}
	// with dataFrom.Find the change is on the called method GetAllSecrets
	// all keys should be put into the secret
	syncAndRewriteDataFromFind := func(tc *testCase) {
		tc.externalSecret.Spec.Data = nil
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Find: &esv1beta1.ExternalSecretFind{
					Name: &esv1beta1.FindName{
						RegExp: ".*",
					},
				},
				Rewrite: []esv1beta1.ExternalSecretRewrite{
					{
						Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
							Source: "(.*)",
							Target: "new-$1",
						},
					},
				},
			},
		}
		fakeProvider.WithGetAllSecrets(map[string][]byte{
			"foo": []byte(FooValue),
			"bar": []byte(BarValue),
		}, nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue("new-foo", []byte(FooValue)))
			Expect(secret.Data).To(HaveKeyWithValue("new-bar", []byte(BarValue)))
		}
	}

	// with dataFrom.Find the change is on the called method GetAllSecrets
	// all keys should be put into the secret
	syncDataFromFind := func(tc *testCase) {
		tc.externalSecret.Spec.Data = nil
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Find: &esv1beta1.ExternalSecretFind{
					Name: &esv1beta1.FindName{
						RegExp: ".*",
					},
				},
			},
		}
		fakeProvider.WithGetAllSecrets(map[string][]byte{
			"foo": []byte(FooValue),
			"bar": []byte(BarValue),
		}, nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue("foo", []byte(FooValue)))
			Expect(secret.Data).To(HaveKeyWithValue("bar", []byte(BarValue)))
		}
	}

	// with dataFrom and using a template
	// should be put into the secret
	syncWithDataFromTemplate := func(tc *testCase) {
		tc.externalSecret.Spec.Data = nil
		tc.externalSecret.Spec.Target = esv1beta1.ExternalSecretTarget{
			Name: ExternalSecretTargetSecretName,
			Template: &esv1beta1.ExternalSecretTemplate{
				Type: v1.SecretTypeTLS,
			},
		}

		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: remoteKey,
				},
			},
		}
		fakeProvider.WithGetSecretMap(map[string][]byte{
			"tls.crt": []byte(FooValue),
			"tls.key": []byte(BarValue),
		}, nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret type")
			Expect(secret.Type).To(Equal(v1.SecretTypeTLS))

			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue("tls.crt", []byte(FooValue)))
			Expect(secret.Data).To(HaveKeyWithValue("tls.key", []byte(BarValue)))
		}
	}

	esStatusSourcesSecretStores := func(tc *testCase) {
		remoteKey1 := "remoteKey1"
		remoteKey2 := "remoteKey2"
		remoteKey3 := "remoteKey3"

		providerData := make(map[string]map[string][]byte)
		providerData[remoteKey1] = map[string][]byte{
			"1foo": []byte(FooValue),
			"1bar": []byte(BarValue),
		}
		providerData[remoteKey2] = map[string][]byte{
			"2foo": []byte(FooValue),
			"2bar": []byte(BarValue),
		}
		providerData[remoteKey3] = map[string][]byte{
			"3foo": []byte(FooValue),
			"3bar": []byte(BarValue),
		}

		fakeProvider.ResetWithStructuredData(providerData)

		// the deletion policy must not be Retain (default) because we are referencing a non-existing key
		tc.externalSecret.Spec.Target.DeletionPolicy = esv1beta1.DeletionPolicyDelete
		tc.externalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: "1foo",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key:      remoteKey1,
					Property: "1foo",
				},
			},
			{
				SecretKey: "xxxx",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key:      "missingKey1",
					Property: "xxxx",
				},
			},
		}
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: remoteKey2,
				},
			},
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: "missingKey2",
				},
			},
			{
				Find: &esv1beta1.ExternalSecretFind{
					Name: &esv1beta1.FindName{
						RegExp: remoteKey3,
					},
				},
			},
		}

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(BeComparableTo(map[string][]byte{
				"1foo": []byte(FooValue),
				// NOTE: "1bar" is not present because the deletion policy is set to Delete
				"2foo":     []byte(FooValue),
				"2bar":     []byte(BarValue),
				remoteKey3: []byte(fmt.Sprintf(`{"3bar":%q,"3foo":%q}`, BarValue, FooValue)),
			}))

			By("checking the status.sources")
			Expect(es.Status.Sources).To(BeComparableTo(esv1beta1.ExternalSecretStatusSources{
				SecretStores: []esv1beta1.ProviderSourceInfo{
					{
						Name:      "test-store",
						NotReady:  true, // this is a mock provider, so the controller will not see it as ready
						NotExists: false,
						ListedKeys: []esv1beta1.SourceListedKey{
							{Key: "missingKey1", NotExists: true},
							{Key: "missingKey2", NotExists: true},
							{Key: remoteKey1, NotExists: false},
							{Key: remoteKey2, NotExists: false},
						},
						FoundKeys: []esv1beta1.SourceFoundKey{
							{Key: remoteKey3},
						},
					},
				},
				ClusterSecretStores: []esv1beta1.ProviderSourceInfo{},
			}))
		}
	}

	esStatusSourcesClusterSecretStores := func(tc *testCase) {
		remoteKey1 := "remoteKey1"
		remoteKey2 := "remoteKey2"
		remoteKey3 := "remoteKey3"

		providerData := make(map[string]map[string][]byte)
		providerData[remoteKey1] = map[string][]byte{
			"1foo": []byte(FooValue),
			"1bar": []byte(BarValue),
		}
		providerData[remoteKey2] = map[string][]byte{
			"2foo": []byte(FooValue),
			"2bar": []byte(BarValue),
		}
		providerData[remoteKey3] = map[string][]byte{
			"3foo": []byte(FooValue),
			"3bar": []byte(BarValue),
		}

		fakeProvider.ResetWithStructuredData(providerData)

		// the deletion policy must not be Retain (default) because we are referencing a non-existing key
		tc.externalSecret.Spec.Target.DeletionPolicy = esv1beta1.DeletionPolicyDelete
		tc.externalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: "1foo",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key:      remoteKey1,
					Property: "1foo",
				},
			},
			{
				SecretKey: "xxxx",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key:      "missingKey1",
					Property: "xxxx",
				},
			},
		}
		tc.externalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: remoteKey2,
				},
			},
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: "missingKey2",
				},
			},
			{
				Find: &esv1beta1.ExternalSecretFind{
					Name: &esv1beta1.FindName{
						RegExp: remoteKey3,
					},
				},
			},
		}

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(BeComparableTo(map[string][]byte{
				"1foo": []byte(FooValue),
				// NOTE: "1bar" is not present because the deletion policy is set to Delete
				"2foo":     []byte(FooValue),
				"2bar":     []byte(BarValue),
				remoteKey3: []byte(fmt.Sprintf(`{"3bar":%q,"3foo":%q}`, BarValue, FooValue)),
			}))

			By("checking the status.sources")
			Expect(es.Status.Sources).To(BeComparableTo(esv1beta1.ExternalSecretStatusSources{
				SecretStores: []esv1beta1.ProviderSourceInfo{},
				ClusterSecretStores: []esv1beta1.ProviderSourceInfo{
					{
						Name:      "test-store",
						NotReady:  true, // this is a mock provider, so the controller will not see it as ready
						NotExists: false,
						ListedKeys: []esv1beta1.SourceListedKey{
							{Key: "missingKey1", NotExists: true},
							{Key: "missingKey2", NotExists: true},
							{Key: remoteKey1, NotExists: false},
							{Key: remoteKey2, NotExists: false},
						},
						FoundKeys: []esv1beta1.SourceFoundKey{
							{Key: remoteKey3},
						},
					},
				},
			}))
		}
	}

	// when a provider errors in a GetSecret call
	// a error condition must be set.
	providerErrCondition := func(tc *testCase) {
		const secretVal = "foobar"
		fakeProvider.WithGetSecret(nil, errors.New("boom"))
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Millisecond * 100}
		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:    esv1beta1.ExternalSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  esv1beta1.ConditionReasonSecretSyncedError,
					Message: fmt.Errorf(errProviderData, fmt.Errorf(errSpecData, 0, remoteKey, errors.New("boom"))).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func(g Gomega) bool {
				g.Expect(testSyncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				g.Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())

			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 1.0)
			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)

			By("removing the provider error")
			fakeProvider.WithGetSecret([]byte(secretVal), nil)

			Eventually(func(g Gomega) {
				By("getting the ExternalSecret")
				esKey := types.NamespacedName{Name: ExternalSecretName, Namespace: ExternalSecretNamespace}
				g.Expect(k8sClient.Get(context.Background(), esKey, es)).To(Succeed())

				By("checking the ExternalSecret is now ready")
				expected := []esv1beta1.ExternalSecretStatusCondition{
					{
						Type:    esv1beta1.ExternalSecretReady,
						Status:  v1.ConditionTrue,
						Reason:  esv1beta1.ConditionReasonSecretSynced,
						Message: msgSynced,
					},
				}
				opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
				g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
			}, timeout, interval).Should(Succeed())
		}
	}

	// When a ExternalSecret references an non-existing SecretStore
	// a error condition must be set.
	storeMissingErrCondition := func(tc *testCase) {
		tc.externalSecret.Spec.SecretStoreRef.Name = "nonexistent"
		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:    esv1beta1.ExternalSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  esv1beta1.ConditionReasonSecretSyncedError,
					Message: fmt.Errorf(errSourcesNotExists, []string{"SecretStore/nonexistent"}).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func(g Gomega) bool {
				g.Expect(testSyncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				g.Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())

			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 1.0)
			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)
		}
	}

	// when the provider constructor errors (e.g. invalid configuration)
	// a SecretSyncedError status condition must be set
	storeConstructErrCondition := func(tc *testCase) {
		fakeProvider.WithNew(func(context.Context, esv1beta1.GenericStore, client.Client,
			string) (esv1beta1.SecretsClient, error) {
			return nil, errors.New("artificial constructor error")
		})
		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:    esv1beta1.ExternalSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  esv1beta1.ConditionReasonSecretSyncedError,
					Message: fmt.Errorf(errProviderData, fmt.Errorf(errSpecData, 0, remoteKey, errors.New("artificial constructor error"))).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func(g Gomega) bool {
				g.Expect(testSyncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				g.Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())

			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 1.0)
			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)
		}
	}

	// when a SecretStore has a controller field set which we don't care about
	// the externalSecret must not be touched
	ignoreMismatchController := func(tc *testCase) {
		tc.secretStore.GetSpec().Controller = "nop"
		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			g.Expect(es.Status.Conditions).To(BeEmpty())
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			// Condition True and False should be 0, since the Condition was not created
			Eventually(func(g Gomega) float64 {
				g.Expect(testExternalSecretCondition.WithLabelValues(ExternalSecretName, ExternalSecretNamespace, string(esv1beta1.ExternalSecretReady), string(v1.ConditionTrue)).Write(&metric)).To(Succeed())
				return metric.GetGauge().GetValue()
			}, timeout, interval).Should(Equal(0.0))

			Eventually(func(g Gomega) float64 {
				g.Expect(testExternalSecretCondition.WithLabelValues(ExternalSecretName, ExternalSecretNamespace, string(esv1beta1.ExternalSecretReady), string(v1.ConditionFalse)).Write(&metric)).To(Succeed())
				return metric.GetGauge().GetValue()
			}, timeout, interval).Should(Equal(0.0))

			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 0.0)
			externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)
		}
	}

	// When the ownership is set to owner, and we delete a dependent child kind=secret
	// it should be recreated without waiting for refresh interval
	checkDeletion := func(tc *testCase) {
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Minute * 10}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("ensuring the secret exists")
			oldUID := secret.UID
			Expect(oldUID).NotTo(BeEmpty())

			By("deleting the secret")
			Expect(k8sClient.Delete(context.Background(), secret)).To(Succeed())

			Eventually(func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())

				By("ensuring the target secret was recreated")
				g.Expect(newSecret.UID).NotTo(Equal(oldUID))
			}, timeout, interval).Should(Succeed())
		}
	}

	// Checks that secret annotation has been written based on the data
	checkSecretDataHashAnnotation := func(tc *testCase) {
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("ensuring the secret has the correct data hash annotation")
			expectedHash := utils.ObjectHash(map[string][]byte{
				targetProp: []byte(secretVal),
			})
			Expect(secret.Annotations).To(HaveKeyWithValue(esv1beta1.AnnotationDataHash, expectedHash))
		}
	}

	// Checks that secret annotation has been written based on the all the data for merge keys
	checkMergeSecretDataHashAnnotation := func(tc *testCase) {
		const secretVal = "someValue"
		tc.externalSecret.Spec.Target.CreationPolicy = esv1beta1.CreatePolicyMerge

		// create secret beforehand
		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			},
			Data: map[string][]byte{
				existingKey: []byte(existingVal),
			},
		}, client.FieldOwner(FakeManager))).To(Succeed())

		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("ensuring the secret has the correct data hash annotation")
			expectedHash := utils.ObjectHash(map[string][]byte{
				existingKey: []byte(existingVal),
				targetProp:  []byte(secretVal),
			})
			Expect(secret.Annotations).To(HaveKeyWithValue(esv1beta1.AnnotationDataHash, expectedHash))
		}
	}

	// When we amend the created kind=secret, refresh operation should be run again regardless of refresh interval
	checkSecretDataHashAnnotationChange := func(tc *testCase) {
		fakeData := map[string][]byte{
			targetProp: []byte(secretVal),
		}
		fakeProvider.WithGetSecretMap(fakeData, nil)
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Minute * 10}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			oldResourceVersion := secret.ResourceVersion

			By("ensuring the secret has the correct data hash annotation")
			expectedHash := utils.ObjectHash(secret.Data)
			Expect(secret.Annotations).To(HaveKeyWithValue(esv1beta1.AnnotationDataHash, expectedHash))

			By("patching the secret with the wrong data annotation")
			patch := client.MergeFrom(secret.DeepCopy())
			secret.Annotations[esv1beta1.AnnotationDataHash] = "wrong_hash"
			Expect(k8sClient.Patch(context.Background(), secret, patch)).To(Succeed())

			Eventually(func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())

				By("ensuring the secret has been refreshed")
				g.Expect(newSecret.ResourceVersion).NotTo(Equal(oldResourceVersion))
				g.Expect(newSecret.Annotations).To(HaveKeyWithValue(esv1beta1.AnnotationDataHash, expectedHash))
			}, timeout, interval).Should(Succeed())
		}
	}

	// When we update the template, remaining keys should not be preserved
	templateShouldRewrite := func(tc *testCase) {
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Minute * 10}
		tc.externalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Data: map[string]string{
				"key": `{{.targetProperty}}-foo`,
			},
		}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue("key", []byte("someValue-foo")))

			By("patching the ExternalSecret to update the template")
			patch := client.MergeFrom(es.DeepCopy())
			es.Spec.Target.Template.Data = map[string]string{
				"new": "foo",
			}
			Expect(k8sClient.Patch(context.Background(), es, patch)).To(Succeed())

			Eventually(func(g Gomega) {
				By("getting the target secret")
				newSecretKey := types.NamespacedName{Name: ExternalSecretTargetSecretName, Namespace: ExternalSecretNamespace}
				newSecret := &v1.Secret{}
				g.Expect(k8sClient.Get(context.Background(), newSecretKey, newSecret)).To(Succeed())

				By("ensuring the secret has the new data")
				g.Expect(newSecret.Data).To(HaveKeyWithValue("new", []byte("foo")))

				By("ensuring the secret does not have the old data")
				g.Expect(newSecret.Data).NotTo(HaveKey("key"))
			}, timeout, interval).Should(Succeed())
		}
	}

	// should keep spec.data with data from template if MergePolicy=Merge
	templateShouldMerge := func(tc *testCase) {
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Minute * 10}
		tc.externalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			MergePolicy: esv1beta1.MergePolicyMerge,
			Data: map[string]string{
				"key": `{{.targetProperty}}-foo`,
			},
		}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("ensuring target secret has keys from spec.target.template.data")
			Expect(secret.Data).To(HaveKeyWithValue("key", []byte("someValue-foo")))

			By("ensuring target secret has keys from spec.data[]")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))
		}
	}

	useClusterSecretStore := func(tc *testCase) {
		tc.secretStore = &esv1beta1.ClusterSecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name: ExternalSecretStore,
			},
			Spec: esv1beta1.SecretStoreSpec{
				Provider: &esv1beta1.SecretStoreProvider{
					AWS: &esv1beta1.AWSProvider{
						Service: esv1beta1.AWSServiceSecretsManager,
					},
				},
			},
		}
		tc.externalSecret.Spec.SecretStoreRef.Kind = esv1beta1.ClusterSecretStoreKind
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
	}

	// Secret is created when ClusterSecretStore has no conditions
	noConditionsSecretCreated := func(tc *testCase) {
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))
		}
	}

	// Secret is not created when ClusterSecretStore has a single non-matching string condition
	noSecretCreatedWhenNamespaceDoesntMatchStringCondition := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				Namespaces: []string{"some-other-ns"},
			},
		}

		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:   esv1beta1.ExternalSecretReady,
					Status: v1.ConditionFalse,
					Reason: esv1beta1.ConditionReasonSecretSyncedError,
					Message: fmt.Errorf(errProviderData,
						fmt.Errorf(errSpecData, 0, remoteKey,
							fmt.Errorf("using cluster store %q is not allowed from namespace %q: denied by spec.condition", ExternalSecretStore, ExternalSecretNamespace),
						),
					).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
	}

	// Secret is not created when ClusterSecretStore has a single non-matching string condition with multiple names
	noSecretCreatedWhenNamespaceDoesntMatchStringConditionWithMultipleNames := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				Namespaces: []string{"some-other-ns", "another-ns"},
			},
		}

		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:   esv1beta1.ExternalSecretReady,
					Status: v1.ConditionFalse,
					Reason: esv1beta1.ConditionReasonSecretSyncedError,
					Message: fmt.Errorf(errProviderData,
						fmt.Errorf(errSpecData, 0, remoteKey,
							fmt.Errorf("using cluster store %q is not allowed from namespace %q: denied by spec.condition", ExternalSecretStore, ExternalSecretNamespace),
						),
					).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
	}

	// Secret is not created when ClusterSecretStore has a multiple non-matching string condition
	noSecretCreatedWhenNamespaceDoesntMatchMultipleStringCondition := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				Namespaces: []string{"some-other-ns"},
			},
			{
				Namespaces: []string{"another-ns"},
			},
		}

		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:   esv1beta1.ExternalSecretReady,
					Status: v1.ConditionFalse,
					Reason: esv1beta1.ConditionReasonSecretSyncedError,
					Message: fmt.Errorf(errProviderData,
						fmt.Errorf(errSpecData, 0, remoteKey,
							fmt.Errorf("using cluster store %q is not allowed from namespace %q: denied by spec.condition", ExternalSecretStore, ExternalSecretNamespace),
						),
					).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
	}

	// Secret is created when ClusterSecretStore has a single matching string condition
	secretCreatedWhenNamespaceMatchesSingleStringCondition := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				Namespaces: []string{ExternalSecretNamespace},
			},
		}

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))
		}
	}

	// Secret is created when ClusterSecretStore has a multiple string conditions, one matching
	secretCreatedWhenNamespaceMatchesMultipleStringConditions := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				Namespaces: []string{ExternalSecretNamespace, "some-other-ns"},
			},
		}

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))
		}
	}

	// Secret is not created when ClusterSecretStore has a single non-matching label condition
	noSecretCreatedWhenNamespaceDoesntMatchLabelCondition := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"some-label-key": "some-label-value"}},
			},
		}

		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:   esv1beta1.ExternalSecretReady,
					Status: v1.ConditionFalse,
					Reason: esv1beta1.ConditionReasonSecretSyncedError,
					Message: fmt.Errorf(errProviderData,
						fmt.Errorf(errSpecData, 0, remoteKey,
							fmt.Errorf("using cluster store %q is not allowed from namespace %q: denied by spec.condition", ExternalSecretStore, ExternalSecretNamespace),
						),
					).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
	}

	// Secret is created when ClusterSecretStore has a single matching label condition
	secretCreatedWhenNamespaceMatchOnlyLabelCondition := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{NamespaceLabelKey: NamespaceLabelValue}},
			},
		}

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))
		}
	}

	// Secret is not created when ClusterSecretStore has a partially matching label condition
	noSecretCreatedWhenNamespacePartiallyMatchLabelCondition := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{NamespaceLabelKey: NamespaceLabelValue, "some-label-key": "some-label-value"}},
			},
		}

		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:   esv1beta1.ExternalSecretReady,
					Status: v1.ConditionFalse,
					Reason: esv1beta1.ConditionReasonSecretSyncedError,
					Message: fmt.Errorf(errProviderData,
						fmt.Errorf(errSpecData, 0, remoteKey,
							fmt.Errorf("using cluster store %q is not allowed from namespace %q: denied by spec.condition", ExternalSecretStore, ExternalSecretNamespace),
						),
					).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
	}

	// Secret is created when ClusterSecretStore has at least one matching label condition
	secretCreatedWhenNamespaceMatchOneLabelCondition := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{NamespaceLabelKey: NamespaceLabelValue}},
			},
			{
				NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"some-label-key": "some-label-value"}},
			},
		}

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))
		}
	}

	// Secret is created when ClusterSecretStore has multiple matching conditions
	secretCreatedWhenNamespaceMatchMultipleConditions := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{NamespaceLabelKey: NamespaceLabelValue}},
			},
			{
				Namespaces: []string{ExternalSecretNamespace},
			},
		}

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			By("checking the secret data")
			Expect(secret.Data).To(HaveKeyWithValue(targetProp, []byte(secretVal)))
		}
	}

	// Secret is not created when ClusterSecretStore has multiple non-matching conditions
	noSecretCreatedWhenNamespaceMatchMultipleNonMatchingConditions := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"some-label-key": "some-label-value"}},
			},
			{
				Namespaces: []string{"some-other-ns"},
			},
		}

		tc.checkCondition = func(g Gomega, es *esv1beta1.ExternalSecret) {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:   esv1beta1.ExternalSecretReady,
					Status: v1.ConditionFalse,
					Reason: esv1beta1.ConditionReasonSecretSyncedError,
					Message: fmt.Errorf(errProviderData,
						fmt.Errorf(errSpecData, 0, remoteKey,
							fmt.Errorf("using cluster store %q is not allowed from namespace %q: denied by spec.condition", ExternalSecretStore, ExternalSecretNamespace),
						),
					).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(es.Status.Conditions).To(BeComparableTo(expected, opts))
		}
	}

	DescribeTable("When reconciling an ExternalSecret",
		func(tweaks ...testTweaks) {
			tc := makeDefaultTestcase()
			for _, tweak := range tweaks {
				tweak(tc)
			}
			ctx := context.Background()

			By("creating the secret store")
			Expect(k8sClient.Create(ctx, tc.secretStore)).To(Succeed())

			if tc.externalSecret != nil {
				By("creating the external secret")
				Expect(k8sClient.Create(ctx, tc.externalSecret)).To(Succeed())
			}

			esKey := types.NamespacedName{Name: ExternalSecretName, Namespace: ExternalSecretNamespace}
			createdES := &esv1beta1.ExternalSecret{}

			Eventually(func(g Gomega) {
				By("checking the external secret condition")
				g.Expect(k8sClient.Get(ctx, esKey, createdES)).To(Succeed())
				tc.checkCondition(g, createdES)
			}, timeout, interval).Should(Succeed())

			// this must be optional as we don't always want to check the external secret
			if tc.checkExternalSecret != nil {
				By("checking the external secret")
				tc.checkExternalSecret(createdES)
			}

			// this must be optional, as the target secret is not always created (e.g. when the ES config is invalid)
			if tc.checkSecret != nil {

				// the target secret name defaults to the ExternalSecret name, if not explicitly set
				targetSecretName := createdES.Spec.Target.Name
				if targetSecretName == "" {
					targetSecretName = createdES.Name
				}

				targetSecretKey := types.NamespacedName{Name: targetSecretName, Namespace: ExternalSecretNamespace}
				targetSecret := &v1.Secret{}

				Eventually(func(g Gomega) {
					By("waiting for the target secret to be created")
					g.Expect(k8sClient.Get(ctx, targetSecretKey, targetSecret)).To(Succeed())
				}, timeout, interval).Should(Succeed())

				By("checking the target secret")
				tc.checkSecret(createdES, targetSecret)
			}
		},
		Entry("should recreate deleted secret", checkDeletion),
		Entry("should create proper hash annotation for the external secret", checkSecretDataHashAnnotation),
		Entry("should create proper hash annotation for the external secret with creationPolicy=Merge", checkMergeSecretDataHashAnnotation),
		Entry("es deletes orphaned secrets", deleteOrphanedSecrets),
		Entry("should refresh when the hash annotation doesn't correspond to secret data", checkSecretDataHashAnnotationChange),
		Entry("should use external secret name if target secret name isn't defined", syncWithoutTargetName),
		Entry("should sync to target secrets with naming bigger than 63 characters", syncBigNames),
		Entry("should expose the secret as a provisioned service binding secret", syncBindingSecret),
		Entry("should not expose a provisioned service when no secret is synced", skipBindingSecret),
		Entry("should set labels and annotations from the ExternalSecret", syncLabelsAnnotations),
		Entry("should merge labels and annotations to the ones owned by other entity", mergeLabelsAnnotations),
		Entry("should removed outdated labels and annotations", removeOutdatedLabelsAnnotations),
		Entry("should set prometheus counters", checkPrometheusCounters),
		Entry("should merge with existing secret using creationPolicy=Merge", mergeWithSecret),
		Entry("should kick reconciliation when secret changes using creationPolicy=Merge", mergeWithSecretUpdate),
		Entry("should error if secret doesn't exist when using creationPolicy=Merge", mergeWithSecretErr),
		Entry("should not resolve conflicts with creationPolicy=Merge", mergeWithConflict),
		Entry("should not update unchanged secret using creationPolicy=Merge", mergeWithSecretNoChange),
		Entry("should not delete pre-existing secret with creationPolicy=Orphan", createSecretPolicyOrphan),
		Entry("should sync cluster generator ref", syncWithClusterGeneratorRef),
		Entry("should sync with generatorRef", syncWithGeneratorRef),
		Entry("should sync with multiple secret stores via sourceRef", syncWithMultipleSecretStores),
		Entry("should sync with template", syncWithTemplate),
		Entry("should sync with template engine v2", syncWithTemplateV2),
		Entry("should sync template with correct value precedence", syncWithTemplatePrecedence),
		Entry("should sync template from keys and values", syncTemplateFromKeysAndValues),
		Entry("should sync template from literal", syncTemplateFromLiteral),
		Entry("should update template if ExternalSecret is updated", templateShouldRewrite),
		Entry("should keep data with templates if MergePolicy=Merge", templateShouldMerge),
		Entry("should refresh secret from template", refreshWithTemplate),
		Entry("should be able to use only metadata from template", onlyMetadataFromTemplate),
		Entry("should refresh secret value when provider secret changes", refreshSecretValue),
		Entry("should refresh secret map when provider secret changes", refreshSecretValueMap),
		Entry("should refresh secret map when provider secret changes when using a template", refreshSecretValueMapTemplate),
		Entry("should not refresh secret value when provider secret changes but refreshInterval is zero", refreshintervalZero),
		Entry("should trigger refresh for in-cluster kubernetes provider", refreshInClusterK8sProvider),
		Entry("should fetch secret using dataFrom", syncWithDataFrom),
		Entry("should rewrite secret using dataFrom", syncAndRewriteWithDataFrom),
		Entry("should not automatically convert from extract if rewrite is used", invalidExtractKeysErrCondition),
		Entry("should fetch secret using dataFrom.find", syncDataFromFind),
		Entry("should rewrite secret using dataFrom.find", syncAndRewriteDataFromFind),
		Entry("should not automatically convert from find if rewrite is used", invalidFindKeysErrCondition),
		Entry("should fetch secret using dataFrom and a template", syncWithDataFromTemplate),
		Entry("should populate ES status.sources.secretStores[] for a secret store", esStatusSourcesSecretStores),
		Entry("should populate ES status.sources.clusterSecretStores[] for a cluster secret store", useClusterSecretStore, esStatusSourcesClusterSecretStores),
		Entry("should set error condition when provider errors", providerErrCondition),
		Entry("should set an error condition when store does not exist", storeMissingErrCondition),
		Entry("should set an error condition when store provider constructor fails", storeConstructErrCondition),
		Entry("should not process store with mismatching controller field", ignoreMismatchController),
		Entry("should eventually delete target secret with deletionPolicy=Delete", deletionPolicyDelete),
		Entry("should not delete target secret with deletionPolicy=Retain", deletionPolicyRetain),
		Entry("should update the status properly even if the deletionPolicy is Retain and the data is empty", deletionPolicyRetainEmptyData),
		Entry("should not delete pre-existing secret with deletionPolicy=Merge", deletionPolicyMerge),
		Entry("secret is created when there are no conditions for the cluster secret store", useClusterSecretStore, noConditionsSecretCreated),
		Entry("secret is not created when the condition for the cluster secret store states a different namespace single string condition", useClusterSecretStore, noSecretCreatedWhenNamespaceDoesntMatchStringCondition),
		Entry("secret is not created when the condition for the cluster secret store states a different namespace single string condition with multiple names", useClusterSecretStore, noSecretCreatedWhenNamespaceDoesntMatchStringConditionWithMultipleNames),
		Entry("secret is not created when the condition for the cluster secret store states a different namespace multiple string conditions", useClusterSecretStore, noSecretCreatedWhenNamespaceDoesntMatchMultipleStringCondition),
		Entry("secret is created when the condition for the cluster secret store has only one matching namespace by string condition", useClusterSecretStore, secretCreatedWhenNamespaceMatchesSingleStringCondition),
		Entry("secret is created when the condition for the cluster secret store has one matching namespace of multiple namespaces by string condition", useClusterSecretStore, secretCreatedWhenNamespaceMatchesMultipleStringConditions),
		Entry("secret is not created when the condition for the cluster secret store states a non-matching label condition", useClusterSecretStore, noSecretCreatedWhenNamespaceDoesntMatchLabelCondition),
		Entry("secret is created when the condition for the cluster secret store states a single matching label condition", useClusterSecretStore, secretCreatedWhenNamespaceMatchOnlyLabelCondition),
		Entry("secret is not created when the condition for the cluster secret store states a partially-matching label condition", useClusterSecretStore, noSecretCreatedWhenNamespacePartiallyMatchLabelCondition),
		Entry("secret is created when one of the label conditions for the cluster secret store matches", useClusterSecretStore, secretCreatedWhenNamespaceMatchOneLabelCondition),
		Entry("secret is created when the namespaces matches multiple cluster secret store conditions", useClusterSecretStore, secretCreatedWhenNamespaceMatchMultipleConditions),
		Entry("secret is not created when the namespaces doesn't match any of multiple cluster secret store conditions", useClusterSecretStore, noSecretCreatedWhenNamespaceMatchMultipleNonMatchingConditions),
	)
})

var _ = Describe("ExternalSecret refresh logic", func() {
	Context("secret refresh", func() {
		It("should refresh when resource version does not match", func() {
			Expect(shouldRefresh(&esv1beta1.ExternalSecret{
				Spec: esv1beta1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Minute},
				},
				Status: esv1beta1.ExternalSecretStatus{
					SyncedResourceVersion: "some resource version",
				},
			})).To(BeTrue())
		})
		It("should refresh when labels change", func() {
			es := &esv1beta1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: esv1beta1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Minute},
				},
				Status: esv1beta1.ExternalSecretStatus{
					RefreshTime: metav1.Now(),
				},
			}
			es.Status.SyncedResourceVersion = getResourceVersion(es)
			// this should not refresh, rv matches object
			Expect(shouldRefresh(es)).To(BeFalse())

			// change labels without changing the syncedResourceVersion and expect refresh
			es.ObjectMeta.Labels["new"] = "w00t"
			Expect(shouldRefresh(es)).To(BeTrue())
		})

		It("should refresh when annotations change", func() {
			es := &esv1beta1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Spec: esv1beta1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Minute},
				},
				Status: esv1beta1.ExternalSecretStatus{
					RefreshTime: metav1.Now(),
				},
			}
			es.Status.SyncedResourceVersion = getResourceVersion(es)
			// this should not refresh, rv matches object
			Expect(shouldRefresh(es)).To(BeFalse())

			// change annotations without changing the syncedResourceVersion and expect refresh
			es.ObjectMeta.Annotations["new"] = "w00t"
			Expect(shouldRefresh(es)).To(BeTrue())
		})

		It("should refresh when generation has changed", func() {
			es := &esv1beta1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: esv1beta1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Minute},
				},
				Status: esv1beta1.ExternalSecretStatus{
					RefreshTime: metav1.Now(),
				},
			}
			es.Status.SyncedResourceVersion = getResourceVersion(es)
			Expect(shouldRefresh(es)).To(BeFalse())

			// update gen -> refresh
			es.ObjectMeta.Generation = 2
			Expect(shouldRefresh(es)).To(BeTrue())
		})

		It("should skip refresh when refreshInterval is 0", func() {
			es := &esv1beta1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: esv1beta1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: 0},
				},
				Status: esv1beta1.ExternalSecretStatus{},
			}
			// resource version matches
			es.Status.SyncedResourceVersion = getResourceVersion(es)
			Expect(shouldRefresh(es)).To(BeFalse())
		})

		It("should refresh when refresh interval has passed", func() {
			es := &esv1beta1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: esv1beta1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Second},
				},
				Status: esv1beta1.ExternalSecretStatus{
					RefreshTime: metav1.NewTime(metav1.Now().Add(-time.Second * 5)),
				},
			}
			// resource version matches
			es.Status.SyncedResourceVersion = getResourceVersion(es)
			Expect(shouldRefresh(es)).To(BeTrue())
		})

		It("should refresh when no refresh time was set", func() {
			es := &esv1beta1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: esv1beta1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Second},
				},
				Status: esv1beta1.ExternalSecretStatus{},
			}
			// resource version matches
			es.Status.SyncedResourceVersion = getResourceVersion(es)
			Expect(shouldRefresh(es)).To(BeTrue())
		})

	})
	Context("objectmeta hash", func() {
		It("should produce different hashes for different k/v pairs", func() {
			h1 := hashMeta(metav1.ObjectMeta{
				Generation: 1,
				Annotations: map[string]string{
					"foo": "bar",
				},
			})
			h2 := hashMeta(metav1.ObjectMeta{
				Generation: 1,
				Annotations: map[string]string{
					"foo": "bing",
				},
			})
			Expect(h1).ToNot(Equal(h2))
		})

		It("should produce different hashes for different generations but same label/annotations", func() {
			h1 := hashMeta(metav1.ObjectMeta{
				Generation: 1,
				Annotations: map[string]string{
					"foo": "bar",
				},
				Labels: map[string]string{
					"foo": "bar",
				},
			})
			h2 := hashMeta(metav1.ObjectMeta{
				Generation: 2,
				Annotations: map[string]string{
					"foo": "bar",
				},
				Labels: map[string]string{
					"foo": "bar",
				},
			})
			Expect(h1).To(Equal(h2))
		})

		It("should produce the same hash for the same k/v pairs", func() {
			h1 := hashMeta(metav1.ObjectMeta{
				Generation: 1,
			})
			h2 := hashMeta(metav1.ObjectMeta{
				Generation: 1,
			})
			Expect(h1).To(Equal(h2))

			h1 = hashMeta(metav1.ObjectMeta{
				Generation: 1,
				Annotations: map[string]string{
					"foo": "bar",
				},
			})
			h2 = hashMeta(metav1.ObjectMeta{
				Generation: 1,
				Annotations: map[string]string{
					"foo": "bar",
				},
			})
			Expect(h1).To(Equal(h2))
		})
	})
})

func externalSecretConditionShouldBe(name, ns string, ct esv1beta1.ExternalSecretConditionType, cs v1.ConditionStatus, v float64) {
	Eventually(func(g Gomega) float64 {
		g.Expect(testExternalSecretCondition.WithLabelValues(name, ns, string(ct), string(cs)).Write(&metric)).To(Succeed())
		return metric.GetGauge().GetValue()
	}, timeout, interval).Should(Equal(v))
}

func init() {
	fakeProvider = fake.New()
	esv1beta1.ForceRegister(fakeProvider, &esv1beta1.SecretStoreProvider{
		AWS: &esv1beta1.AWSProvider{
			Service: esv1beta1.AWSServiceSecretsManager,
		},
	})

	ctrlmetrics.SetUpLabelNames(false)
	esmetrics.SetUpMetrics()
	testSyncCallsTotal = esmetrics.GetCounterVec(esmetrics.SyncCallsKey)
	testSyncCallsError = esmetrics.GetCounterVec(esmetrics.SyncCallsErrorKey)
	testExternalSecretCondition = esmetrics.GetGaugeVec(esmetrics.ExternalSecretStatusConditionKey)
	testExternalSecretReconcileDuration = esmetrics.GetGaugeVec(esmetrics.ExternalSecretReconcileDurationKey)
}
