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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	ctest "github.com/external-secrets/external-secrets/pkg/controllers/commontest"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret/esmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
	"github.com/external-secrets/external-secrets/pkg/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	secretStore      esv1beta1.GenericStore
	externalSecret   *esv1beta1.ExternalSecret
	targetSecretName string

	// checkCondition should return true if the externalSecret
	// has the expected condition
	checkCondition func(*esv1beta1.ExternalSecret) bool

	// checkExternalSecret is called after the condition has been verified
	// use this to verify the externalSecret
	checkExternalSecret func(*esv1beta1.ExternalSecret)

	// optional. use this to test the secret value
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
		Input          v1.Secret
		ExpectedOutput bool
	}
	tests := []testCase{
		{
			Name:           "Should not be valid in case of missing uid",
			Input:          v1.Secret{},
			ExpectedOutput: false,
		},
		{
			Name: "A nil annotation should not be valid",
			Input: v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					UID:         "xxx",
					Annotations: map[string]string{},
				},
			},
			ExpectedOutput: false,
		},
		{
			Name: "A nil annotation should not be valid",
			Input: v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					UID:         "xxx",
					Annotations: map[string]string{},
				},
			},
			ExpectedOutput: false,
		},
		{
			Name: "An invalid annotation hash should not be valid",
			Input: v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					UID: "xxx",
					Annotations: map[string]string{
						esv1beta1.AnnotationDataHash: "xxxxxx",
					},
				},
			},
			ExpectedOutput: false,
		},
		{
			Name: "A valid config map should return true",
			Input: v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					UID: "xxx",
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
			// default condition: es should be ready
			targetSecretName: ExternalSecretTargetSecretName,
			checkCondition: func(es *esv1beta1.ExternalSecret) bool {
				cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
				if cond == nil || cond.Status != v1.ConditionTrue {
					return false
				}
				return true
			},
			checkExternalSecret: func(es *esv1beta1.ExternalSecret) {
				// noop by default
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
			// check secret name
			Expect(secret.ObjectMeta.Name).To(Equal(ExternalSecretName))

			// check binding secret on external secret
			Expect(es.Status.Binding.Name).To(Equal(secret.ObjectMeta.Name))
		}
	}
	// if target Secret name is not specified it should use the ExternalSecret name.
	syncBigNames := func(tc *testCase) {
		tc.targetSecretName = "this-is-a-very-big-secret-name-that-wouldnt-be-generated-due-to-label-limits"
		tc.externalSecret.Spec.Target.Name = "this-is-a-very-big-secret-name-that-wouldnt-be-generated-due-to-label-limits"
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			// check binding secret on external secret
			Expect(es.Status.Binding.Name).To(Equal(tc.externalSecret.Spec.Target.Name))
		}
	}
	// the secret name is reflected on the external secret's status as the binding secret
	syncBindingSecret := func(tc *testCase) {
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			// check binding secret on external secret
			Expect(es.Status.Binding.Name).To(Equal(secret.ObjectMeta.Name))
		}
	}

	// their is no binding secret when a secret is not synced
	skipBindingSecret := func(tc *testCase) {
		tc.externalSecret.Spec.Target.CreationPolicy = esv1beta1.CreatePolicyNone
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			// check binding secret is not set
			Expect(es.Status.Binding.Name).To(BeEmpty())
		}
	}

	// labels and annotations from the Kind=ExternalSecret
	// should be copied over to the Kind=Secret
	syncLabelsAnnotations := func(tc *testCase) {
		tc.externalSecret.ObjectMeta.Labels = map[string]string{
			"label-key": "label-value",
		}
		tc.externalSecret.ObjectMeta.Annotations = map[string]string{
			"annotation-key": "annotation-value",
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue("label-key", "label-value"))
			Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue("annotation-key", "annotation-value"))

			// ownerRef must not be set!
			Expect(ctest.HasOwnerRef(secret.ObjectMeta, "ExternalSecret", ExternalSecretName)).To(BeTrue())
		}
	}

	// labels and annotations from the ExternalSecret
	// should be merged to the Secret if exists
	mergeLabelsAnnotations := func(tc *testCase) {
		tc.externalSecret.ObjectMeta.Labels = map[string]string{
			"label-key": "label-value",
		}
		tc.externalSecret.ObjectMeta.Annotations = map[string]string{
			"annotation-key": "annotation-value",
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		// Create a secret owned by another entity to test if the pre-existing metadata is preserved
		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
				Labels: map[string]string{
					"existing-label-key": "existing-label-value",
				},
				Annotations: map[string]string{
					"existing-annotation-key": "existing-annotation-value",
				},
			},
		}, client.FieldOwner(FakeManager))).To(Succeed())

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue("label-key", "label-value"))
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue("existing-label-key", "existing-label-value"))
			Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue("annotation-key", "annotation-value"))
			Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue("existing-annotation-key", "existing-annotation-value"))
		}
	}

	removeOutdatedLabelsAnnotations := func(tc *testCase) {
		tc.externalSecret.ObjectMeta.Labels = map[string]string{
			"label-key": "label-value",
		}
		tc.externalSecret.ObjectMeta.Annotations = map[string]string{
			"annotation-key": "annotation-value",
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		// Create a secret owned by the operator to test if the outdated pre-existing metadata is removed
		Expect(k8sClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
				Labels: map[string]string{
					"existing-label-key": "existing-label-value",
				},
				Annotations: map[string]string{
					"existing-annotation-key": "existing-annotation-value",
				},
			},
		}, client.FieldOwner(ExternalSecretFQDN))).To(Succeed())

		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue("label-key", "label-value"))
			Expect(secret.ObjectMeta.Labels).NotTo(HaveKeyWithValue("existing-label-key", "existing-label-value"))
			Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue("annotation-key", "annotation-value"))
			Expect(secret.ObjectMeta.Annotations).NotTo(HaveKeyWithValue("existing-annotation-key", "existing-annotation-value"))
		}
	}

	checkPrometheusCounters := func(tc *testCase) {
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 0.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 1.0)).To(BeTrue())
			Eventually(func() bool {
				Expect(testSyncCallsTotal.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				// three reconciliations: initial sync, status update, secret update
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())
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
					"existing-label-key": "existing-label-value",
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
			// check value
			Expect(string(secret.Data[existingKey])).To(Equal(existingVal))
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))

			Expect(secret.ObjectMeta.Labels).To(HaveLen(2))
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue("existing-label-key", "existing-label-value"))
			Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue("es-label-key", "es-label-value"))

			Expect(secret.ObjectMeta.Annotations).To(HaveLen(3))
			Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue("existing-annotation-key", "existing-annotation-value"))
			Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue("es-annotation-key", "es-annotation-value"))
			Expect(secret.ObjectMeta.Annotations).To(HaveKey(esv1beta1.AnnotationDataHash))

			Expect(ctest.HasOwnerRef(secret.ObjectMeta, "ExternalSecret", ExternalSecretFQDN)).To(BeFalse())
			Expect(secret.ObjectMeta.ManagedFields).To(HaveLen(2))
			Expect(ctest.HasFieldOwnership(
				secret.ObjectMeta,
				ExternalSecretFQDN,
				fmt.Sprintf(`{"f:data":{"f:targetProperty":{}},"f:immutable":{},"f:metadata":{"f:annotations":{"f:es-annotation-key":{},"f:%s":{}},"f:labels":{"f:es-label-key":{}}}}`, esv1beta1.AnnotationDataHash)),
			).To(BeEmpty())
			Expect(ctest.HasFieldOwnership(secret.ObjectMeta, FakeManager, `{"f:data":{".":{},"f:pre-existing-key":{}},"f:metadata":{"f:annotations":{".":{},"f:existing-annotation-key":{}},"f:labels":{".":{},"f:existing-label-key":{}}},"f:type":{}}`)).To(BeEmpty())
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
			// Overwrite the secret value to check if the change kicks reconciliation and overwrites it again
			Expect(k8sClient.Update(context.Background(), &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretTargetSecretName,
					Namespace: ExternalSecretNamespace,
				},
				Data: map[string][]byte{
					existingKey: []byte("differentValue"),
				},
			}, client.FieldOwner(FakeManager))).To(Succeed())

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

			cleanSecret := secret.DeepCopy()
			Expect(k8sClient.Patch(context.Background(), secret, client.MergeFrom(cleanSecret))).To(Succeed())

			newSecret := &v1.Secret{}

			Eventually(func() bool {
				secretLookupKey := types.NamespacedName{
					Name:      ExternalSecretTargetSecretName,
					Namespace: ExternalSecretNamespace,
				}

				err := k8sClient.Get(context.Background(), secretLookupKey, newSecret)
				if err != nil {
					return false
				}
				return oldResourceVersion == newSecret.ResourceVersion
			}, timeout, interval).Should(Equal(true))

		}
	}

	// should not merge with secret if it doesn't exist
	mergeWithSecretErr := func(tc *testCase) {
		const secretVal = "someValue"
		tc.externalSecret.Spec.Target.CreationPolicy = esv1beta1.CreatePolicyMerge

		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func() bool {
				Expect(testSyncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
		}
	}

	// controller should force ownership
	mergeWithConflict := func(tc *testCase) {
		const secretVal = "someValue"
		// this should confict
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
			// check that value stays the same
			Expect(string(secret.Data[existingKey])).To(Equal(secretVal))

			// check owner/managedFields
			Expect(ctest.HasOwnerRef(secret.ObjectMeta, "ExternalSecret", ExternalSecretFQDN)).To(BeFalse())
			Expect(secret.ObjectMeta.ManagedFields).To(HaveLen(2))
			Expect(ctest.HasFieldOwnership(
				secret.ObjectMeta,
				ExternalSecretFQDN,
				fmt.Sprintf("{\"f:data\":{\"f:targetProperty\":{}},\"f:immutable\":{},\"f:metadata\":{\"f:annotations\":{\"f:%s\":{}}}}", esv1beta1.AnnotationDataHash)),
			).To(BeEmpty())
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
			// check values
			Expect(string(secret.Data[secretKey])).To(Equal(secretVal))
		}
	}

	deleteOrphanedSecrets := func(tc *testCase) {
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			cleanEs := es.DeepCopy()
			oldSecret := v1.Secret{}
			oldSecretName := types.NamespacedName{
				Name:      "test-secret",
				Namespace: secret.Namespace,
			}
			newSecret := v1.Secret{}
			secretName := types.NamespacedName{
				Name:      "new-foo",
				Namespace: secret.Namespace,
			}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), oldSecretName, &oldSecret)
				return err == nil
			}, time.Second*10, time.Millisecond*200).Should(BeTrue())
			es.Spec.Target.Name = "new-foo"
			Expect(k8sClient.Patch(context.Background(), es, client.MergeFrom(cleanEs))).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretName, &newSecret)
				return err == nil
			}, time.Second*10, time.Millisecond*200).Should(BeTrue())
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), oldSecretName, &oldSecret)
				return apierrors.IsNotFound(err)
			}, time.Second*10, time.Millisecond*200).Should(BeTrue())
		}
	}

	ignoreMismatchControllerForGeneratorRef := func(tc *testCase) {
		const secretKey = "somekey"
		const secretVal = "someValue"

		fakeGenerator := &genv1alpha1.Fake{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mytestfake2",
				Namespace: ExternalSecretNamespace,
			},
			Spec: genv1alpha1.FakeSpec{
				Data: map[string]string{
					secretKey: secretVal,
				},
				Controller: "fakeControllerClass",
			},
		}

		fakeGeneratorJSON, _ := json.Marshal(fakeGenerator)

		Expect(shouldSkipGenerator(
			&Reconciler{
				ControllerClass: "default",
			},
			&apiextensions.JSON{Raw: fakeGeneratorJSON},
		)).To(BeTrue())
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
			// check values
			Expect(string(secret.Data["foo"])).To(Equal("bar"))
			Expect(string(secret.Data["foo2"])).To(Equal("bar2"))
			Expect(string(secret.Data["baz"])).To(Equal("bang"))
			Expect(string(secret.Data["baz2"])).To(Equal("bang2"))
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
			// check values
			Expect(string(secret.Data[targetProp])).To(Equal(expectedSecretVal))
			Expect(string(secret.Data[tplStaticKey])).To(Equal(tplStaticVal))

			// labels/annotations should be taken from the template
			for k, v := range es.Spec.Target.Template.Metadata.Labels {
				Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(k, v))

			}
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
			// check values
			Expect(string(secret.Data[targetProp])).To(Equal(expectedSecretVal))
		}
	}
	// // secret should be synced with correct value precedence:
	// // * fromString
	// // * template data
	// // * templateFrom
	// // * data
	// // * dataFrom
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
			// check values
			Expect(string(secret.Data[targetProp])).To(Equal(expectedSecretVal))
			Expect(string(secret.Data[tplStaticKey])).To(Equal(tplStaticVal))
			Expect(string(secret.Data["bar"])).To(Equal("value from map: map-bar-value"))
			Expect(string(secret.Data[tplFromKey])).To(Equal("tpl-from-value: someValue // map-bar-value"))
			Expect(string(secret.Data[tplFromSecKey])).To(Equal("tpl-from-sec-value: someValue // map-bar-value"))
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
			// check values
			Expect(string(secret.Data["map-foo-value-cm"])).To(Equal(BarValue))
			Expect(string(secret.Data["map-foo-value-sec"])).To(Equal(BarValue))
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
			// check values
			Expect(string(secret.Data["map-foo-value-literal"])).To(Equal(BarValue))
			Expect(string(secret.Data["nested"])).To(Equal("json"))
			Expect(string(secret.Data["can"])).To(Equal("be"))
			Expect(string(secret.Data["templated"])).To(Equal("successfully"))
			Expect(secret.ObjectMeta.Annotations["map-foo-value-annotations"]).To(Equal(BarValue))
			Expect(secret.ObjectMeta.Labels["map-foo-value-labels"]).To(Equal(BarValue))
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
			// check values
			Expect(string(secret.Data[targetProp])).To(Equal(expectedSecretVal))
			Expect(string(secret.Data[tplStaticKey])).To(Equal(tplStaticVal))

			// labels/annotations should be taken from the template
			for k, v := range es.Spec.Target.Template.Metadata.Labels {
				Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(k, v))

			}

			// a secret will always have some extra annotations (i.e. hashmap check), so we only check for specific
			// source annotations
			for k, v := range es.Spec.Target.Template.Metadata.Annotations {
				Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue(k, v))
			}

			cleanEs := tc.externalSecret.DeepCopy()

			// now update ExternalSecret
			tc.externalSecret.Spec.Target.Template.Metadata.Annotations["fuzz"] = "buzz"
			tc.externalSecret.Spec.Target.Template.Metadata.Labels["fuzz"] = "buzz"
			tc.externalSecret.Spec.Target.Template.Data["new"] = "value"
			Expect(k8sClient.Patch(context.Background(), tc.externalSecret, client.MergeFrom(cleanEs))).To(Succeed())

			// wait for secret
			sec := &v1.Secret{}
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretLookupKey, sec)
				if err != nil {
					return false
				}
				// ensure new data value exist
				return string(sec.Data["new"]) == "value"
			}, time.Second*10, time.Millisecond*200).Should(BeTrue())

			// also check labels/annotations have been updated
			for k, v := range es.Spec.Target.Template.Metadata.Labels {
				Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(k, v))

			}
			for k, v := range es.Spec.Target.Template.Metadata.Annotations {
				Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue(k, v))
			}
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
			// check values
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))

			// labels/annotations should be taken from the template
			for k, v := range es.Spec.Target.Template.Metadata.Labels {
				Expect(secret.ObjectMeta.Labels).To(HaveKeyWithValue(k, v))
			}

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
			// check values
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))

			// update provider secret
			newValue := "NEW VALUE"
			sec := &v1.Secret{}
			fakeProvider.WithGetSecret([]byte(newValue), nil)
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretLookupKey, sec)
				if err != nil {
					return false
				}
				v := sec.Data[targetProp]
				return string(v) == newValue
			}, timeout, interval).Should(BeTrue())
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
			// check values
			Expect(string(secret.Data["foo"])).To(Equal("1111"))
			Expect(string(secret.Data["bar"])).To(Equal("2222"))

			// update provider secret
			sec := &v1.Secret{}
			fakeProvider.WithGetSecretMap(map[string][]byte{
				"foo": []byte("1111"),
			}, nil)
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretLookupKey, sec)
				if err != nil {
					return false
				}
				return string(sec.Data["foo"]) == "1111" &&
					sec.Data["bar"] == nil // must not be defined, it was deleted
			}, timeout, interval).Should(BeTrue())
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
			// check values
			Expect(string(secret.Data["foo"])).To(Equal("1111"))
			Expect(string(secret.Data["bar"])).To(Equal("2222"))

			// update provider secret
			sec := &v1.Secret{}
			fakeProvider.WithGetSecretMap(map[string][]byte{
				"foo": []byte("1111"),
			}, nil)
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretLookupKey, sec)
				if err != nil {
					return false
				}
				return string(sec.Data["foo"]) == "1111" &&
					sec.Data["bar"] == nil // must not be defined, it was deleted
			}, timeout, interval).Should(BeTrue())
		}
	}

	refreshintervalZero := func(tc *testCase) {
		const targetProp = "targetProperty"
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: 0}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			// check values
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))

			// update provider secret
			newValue := "NEW VALUE"
			sec := &v1.Secret{}
			fakeProvider.WithGetSecret([]byte(newValue), nil)
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			}
			Consistently(func() bool {
				err := k8sClient.Get(context.Background(), secretLookupKey, sec)
				if err != nil {
					return false
				}
				v := sec.Data[targetProp]
				return string(v) == secretVal
			}, time.Second*10, time.Second).Should(BeTrue())
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
			Expect(secret.Data["foo"]).To(Equal(expVal))

			// update provider secret
			fakeProvider.WithGetAllSecrets(map[string][]byte{
				"foo": expVal,
			}, nil)
			sec := &v1.Secret{}
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			}
			Eventually(func() bool {
				By("checking secret value for foo=1234 and bar=nil")
				err := k8sClient.Get(context.Background(), secretLookupKey, sec)
				if err != nil {
					return false
				}
				return bytes.Equal(sec.Data["foo"], expVal) && sec.Data["bar"] == nil
			}, time.Second*10, time.Second).Should(BeTrue())

			// return specific delete err to indicate deletion
			fakeProvider.WithGetAllSecrets(map[string][]byte{}, esv1beta1.NoSecretErr)
			Eventually(func() bool {
				By("checking that secret has been deleted")
				err := k8sClient.Get(context.Background(), secretLookupKey, sec)
				return apierrors.IsNotFound(err)
			}, time.Second*10, time.Second).Should(BeTrue())
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
			Expect(secret.Data["foo"]).To(Equal(expVal))

			sec := &v1.Secret{}
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			}
			// return specific delete err to indicate deletion
			// however this should not trigger a delete
			fakeProvider.WithGetAllSecrets(map[string][]byte{}, esv1beta1.NoSecretErr)
			Consistently(func() bool {
				By("checking that secret has not been deleted")
				err := k8sClient.Get(context.Background(), secretLookupKey, sec)
				if err != nil {
					GinkgoLogr.Error(err, "failed getting a secret")
					return false
				}
				if got := sec.Data["foo"]; !bytes.Equal(got, expVal) {
					GinkgoLogr.Info("received an unexpected secret value", "got", got, "expected", expVal)
					return false
				}
				return true
			}, time.Second*10, time.Second).Should(BeTrue())
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
		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			expected := []esv1beta1.ExternalSecretStatusCondition{
				{
					Type:    esv1beta1.ExternalSecretReady,
					Status:  v1.ConditionTrue,
					Reason:  esv1beta1.ConditionReasonSecretSynced,
					Message: "Secret was synced",
				},
			}

			opts := cmpopts.IgnoreFields(esv1beta1.ExternalSecretStatusCondition{}, "LastTransitionTime")
			if diff := cmp.Diff(expected, es.Status.Conditions, opts); diff != "" {
				GinkgoLogr.Info("(-got, +want)\n%s", "diff", diff)
				return false
			}
			return true
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
			// check value
			Expect(string(secret.Data[existingKey])).To(Equal(existingVal))
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))

			sec := &v1.Secret{}
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			}
			// return specific delete err to indicate deletion
			// however this should not trigger a delete
			// instead expect that only the pre-existing value exists
			fakeProvider.WithGetSecret(nil, esv1beta1.NoSecretErr)
			Eventually(func() bool {
				By("checking that secret has not been deleted and pre-existing key exists")
				err := k8sClient.Get(context.Background(), secretLookupKey, sec)
				return !apierrors.IsNotFound(err) &&
					len(sec.Data) == 1 &&
					bytes.Equal(sec.Data[existingKey], []byte(existingVal))
			}, time.Second*30, time.Second).Should(BeTrue())

		}
	}

	// orphan the secret after the external secret has been deleted
	createSecretPolicyOrphan := func(tc *testCase) {
		const secretVal = "someValue"
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.externalSecret.Spec.Target.CreationPolicy = esv1beta1.CreatePolicyOrphan

		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			// check value
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))

			sec := &v1.Secret{}
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			}
			err := k8sClient.Delete(context.Background(), tc.externalSecret)
			Expect(err).ToNot(HaveOccurred())
			Consistently(func() bool {
				By("checking that secret has not been deleted")
				err := k8sClient.Get(context.Background(), secretLookupKey, sec)
				return !apierrors.IsNotFound(err)
			}, time.Second*15, time.Second).Should(BeTrue())
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
			// check values
			Expect(string(secret.Data["new-foo"])).To(Equal(FooValue))
			Expect(string(secret.Data["new-bar"])).To(Equal(BarValue))
			Expect(string(secret.Data["old-foo"])).To(Equal(FooValue))
			Expect(string(secret.Data["old-bar"])).To(Equal(BarValue))
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
		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func() bool {
				Expect(testSyncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
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
		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func() bool {
				Expect(testSyncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
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
			// check values
			Expect(string(secret.Data["foo"])).To(Equal(FooValue))
			Expect(string(secret.Data["bar"])).To(Equal(BarValue))
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
						RegExp: "foobar",
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
			// check values
			Expect(string(secret.Data["new-foo"])).To(Equal(FooValue))
			Expect(string(secret.Data["new-bar"])).To(Equal(BarValue))
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
						RegExp: "foobar",
					},
				},
			},
		}
		fakeProvider.WithGetAllSecrets(map[string][]byte{
			"foo": []byte(FooValue),
			"bar": []byte(BarValue),
		}, nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			// check values
			Expect(string(secret.Data["foo"])).To(Equal(FooValue))
			Expect(string(secret.Data["bar"])).To(Equal(BarValue))
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
			Expect(secret.Type).To(Equal(v1.SecretTypeTLS))
			// check values
			Expect(string(secret.Data["tls.crt"])).To(Equal(FooValue))
			Expect(string(secret.Data["tls.key"])).To(Equal(BarValue))
		}
	}

	// when a provider errors in a GetSecret call
	// a error condition must be set.
	providerErrCondition := func(tc *testCase) {
		const secretVal = "foobar"
		fakeProvider.WithGetSecret(nil, fmt.Errorf("boom"))
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Millisecond * 100}
		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func() bool {
				Expect(testSyncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())

			// es condition should reflect recovered provider error
			fakeProvider.WithGetSecret([]byte(secretVal), nil)
			esKey := types.NamespacedName{Name: ExternalSecretName, Namespace: ExternalSecretNamespace}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), esKey, es)
				if err != nil {
					return false
				}
				// condition must now be true!
				cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
				if cond == nil && cond.Status != v1.ConditionTrue {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
		}
	}

	// When a ExternalSecret references an non-existing SecretStore
	// a error condition must be set.
	storeMissingErrCondition := func(tc *testCase) {
		tc.externalSecret.Spec.SecretStoreRef.Name = "nonexistent"
		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func() bool {
				Expect(testSyncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
		}
	}

	// when the provider constructor errors (e.g. invalid configuration)
	// a SecretSyncedError status condition must be set
	storeConstructErrCondition := func(tc *testCase) {
		fakeProvider.WithNew(func(context.Context, esv1beta1.GenericStore, client.Client,
			string) (esv1beta1.SecretsClient, error) {
			return nil, fmt.Errorf("artificial constructor error")
		})
		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			// condition must be false
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			Eventually(func() bool {
				Expect(testSyncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				Expect(testExternalSecretReconcileDuration.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metricDuration)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0 && metricDuration.GetGauge().GetValue() > 0.0
			}, timeout, interval).Should(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
		}
	}

	// when a SecretStore has a controller field set which we don't care about
	// the externalSecret must not be touched
	ignoreMismatchController := func(tc *testCase) {
		tc.secretStore.GetSpec().Controller = "nop"
		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			return cond == nil
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			// Condition True and False should be 0, since the Condition was not created
			Eventually(func() float64 {
				Expect(testExternalSecretCondition.WithLabelValues(ExternalSecretName, ExternalSecretNamespace, string(esv1beta1.ExternalSecretReady), string(v1.ConditionTrue)).Write(&metric)).To(Succeed())
				return metric.GetGauge().GetValue()
			}, timeout, interval).Should(Equal(0.0))

			Eventually(func() float64 {
				Expect(testExternalSecretCondition.WithLabelValues(ExternalSecretName, ExternalSecretNamespace, string(esv1beta1.ExternalSecretReady), string(v1.ConditionFalse)).Write(&metric)).To(Succeed())
				return metric.GetGauge().GetValue()
			}, timeout, interval).Should(Equal(0.0))

			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 0.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
		}
	}

	ignoreClusterSecretStoreWhenDisabled := func(tc *testCase) {
		tc.externalSecret.Spec.SecretStoreRef.Kind = esv1beta1.ClusterSecretStoreKind

		Expect(shouldSkipClusterSecretStore(
			&Reconciler{
				ClusterSecretStoreEnabled: false,
			},
			*tc.externalSecret,
		)).To(BeTrue())

		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			return cond == nil
		}
	}

	// When the ownership is set to owner, and we delete a dependent child kind=secret
	// it should be recreated without waiting for refresh interval
	checkDeletion := func(tc *testCase) {
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Minute * 10}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {

			// check values
			oldUID := secret.UID
			Expect(oldUID).NotTo(BeEmpty())

			// delete the related config
			Expect(k8sClient.Delete(context.TODO(), secret))

			var newSecret v1.Secret
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretLookupKey, &newSecret)
				if err != nil {
					return false
				}
				// new secret should be a new, recreated object with a different UID
				return newSecret.UID != oldUID
			}, timeout, interval).Should(BeTrue())
		}
	}

	// Checks that secret annotation has been written based on the data
	checkSecretDataHashAnnotation := func(tc *testCase) {
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			expectedHash := utils.ObjectHash(map[string][]byte{
				targetProp: []byte(secretVal),
			})
			Expect(secret.Annotations[esv1beta1.AnnotationDataHash]).To(Equal(expectedHash))
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
			expectedHash := utils.ObjectHash(map[string][]byte{
				existingKey: []byte(existingVal),
				targetProp:  []byte(secretVal),
			})
			Expect(secret.Annotations[esv1beta1.AnnotationDataHash]).To(Equal(expectedHash))
		}
	}

	// When we amend the created kind=secret, refresh operation should be run again regardless of refresh interval
	checkSecretDataHashAnnotationChange := func(tc *testCase) {
		fakeData := map[string][]byte{
			"targetProperty": []byte(FooValue),
		}
		fakeProvider.WithGetSecretMap(fakeData, nil)
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Minute * 10}
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			oldHash := secret.Annotations[esv1beta1.AnnotationDataHash]
			oldResourceVersion := secret.ResourceVersion
			Expect(oldHash).NotTo(BeEmpty())

			cleanSecret := secret.DeepCopy()
			secret.Data["new"] = []byte("value")
			secret.ObjectMeta.Annotations[esv1beta1.AnnotationDataHash] = "thisiswronghash"
			Expect(k8sClient.Patch(context.Background(), secret, client.MergeFrom(cleanSecret))).To(Succeed())

			var refreshedSecret v1.Secret
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretLookupKey, &refreshedSecret)
				if err != nil {
					return false
				}
				// refreshed secret should have a different generation (sign that it was updated), but since
				// the secret source is the same (not changed), the hash should be reverted to an old value
				return refreshedSecret.ResourceVersion != oldResourceVersion && refreshedSecret.Annotations[esv1beta1.AnnotationDataHash] == oldHash
			}, timeout, interval).Should(BeTrue())
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
			Expect(secret.Data["key"]).To(Equal([]byte("someValue-foo")))
			newEs := es.DeepCopy()
			newEs.Spec.Target.Template.Data = map[string]string{
				"new": "foo",
			}
			Expect(k8sClient.Patch(context.Background(), newEs, client.MergeFrom(es))).To(Succeed())

			var refreshedSecret v1.Secret
			secretLookupKey := types.NamespacedName{
				Name:      ExternalSecretTargetSecretName,
				Namespace: ExternalSecretNamespace,
			}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretLookupKey, &refreshedSecret)
				if err != nil {
					return false
				}
				_, ok := refreshedSecret.Data["key"]
				return !ok && bytes.Equal(refreshedSecret.Data["new"], []byte("foo"))
			}, timeout, interval).Should(BeTrue())
		}
	}
	// When we update the template, remaining keys should not be preserved
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
			Expect(secret.Data["key"]).To(Equal([]byte("someValue-foo")))
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))
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
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))
		}
	}

	// Secret is not created when ClusterSecretStore has a single non-matching string condition
	noSecretCreatedWhenNamespaceDoesntMatchStringCondition := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				Namespaces: []string{"some-other-ns"},
			},
		}

		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
	}

	// Secret is not created when ClusterSecretStore has a single non-matching string condition with multiple names
	noSecretCreatedWhenNamespaceDoesntMatchStringConditionWithMultipleNames := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				Namespaces: []string{"some-other-ns", "another-ns"},
			},
		}

		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
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

		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
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
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))
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
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))
		}
	}

	// Secret is not created when ClusterSecretStore has a single non-matching label condition
	noSecretCreatedWhenNamespaceDoesntMatchLabelCondition := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"some-label-key": "some-label-value"}},
			},
		}

		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
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
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))
		}
	}

	// Secret is not created when ClusterSecretStore has a partially matching label condition
	noSecretCreatedWhenNamespacePartiallyMatchLabelCondition := func(tc *testCase) {
		tc.secretStore.GetSpec().Conditions = []esv1beta1.ClusterSecretStoreCondition{
			{
				NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{NamespaceLabelKey: NamespaceLabelValue, "some-label-key": "some-label-value"}},
			},
		}

		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
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
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))
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
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))
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

		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1beta1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
	}

	DescribeTable("When reconciling an ExternalSecret",
		func(tweaks ...testTweaks) {
			tc := makeDefaultTestcase()
			for _, tweak := range tweaks {
				tweak(tc)
			}
			ctx := context.Background()
			By("creating a secret store and external secret")
			Expect(k8sClient.Create(ctx, tc.secretStore)).To(Succeed())
			Expect(k8sClient.Create(ctx, tc.externalSecret)).Should(Succeed())
			esKey := types.NamespacedName{Name: ExternalSecretName, Namespace: ExternalSecretNamespace}
			createdES := &esv1beta1.ExternalSecret{}
			By("checking the es condition")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, esKey, createdES)
				if err != nil {
					return false
				}
				return tc.checkCondition(createdES)
			}, timeout, interval).Should(BeTrue())
			tc.checkExternalSecret(createdES)

			// this must be optional so we can test faulty es configuration
			if tc.checkSecret != nil {
				syncedSecret := &v1.Secret{}
				secretLookupKey := types.NamespacedName{
					Name:      tc.targetSecretName,
					Namespace: ExternalSecretNamespace,
				}
				if createdES.Spec.Target.Name == "" {
					secretLookupKey = types.NamespacedName{
						Name:      ExternalSecretName,
						Namespace: ExternalSecretNamespace,
					}
				}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, secretLookupKey, syncedSecret)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				tc.checkSecret(createdES, syncedSecret)
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
		Entry("should sync with generatorRef", syncWithGeneratorRef),
		Entry("should not process generatorRef with mismatching controller field", ignoreMismatchControllerForGeneratorRef),
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
		Entry("should fetch secret using dataFrom", syncWithDataFrom),
		Entry("should rewrite secret using dataFrom", syncAndRewriteWithDataFrom),
		Entry("should not automatically convert from extract if rewrite is used", invalidExtractKeysErrCondition),
		Entry("should fetch secret using dataFrom.find", syncDataFromFind),
		Entry("should rewrite secret using dataFrom.find", syncAndRewriteDataFromFind),
		Entry("should not automatically convert from find if rewrite is used", invalidFindKeysErrCondition),
		Entry("should fetch secret using dataFrom and a template", syncWithDataFromTemplate),
		Entry("should set error condition when provider errors", providerErrCondition),
		Entry("should set an error condition when store does not exist", storeMissingErrCondition),
		Entry("should set an error condition when store provider constructor fails", storeConstructErrCondition),
		Entry("should not process store with mismatching controller field", ignoreMismatchController),
		Entry("should not process cluster secret store when it is disabled", ignoreClusterSecretStoreWhenDisabled),
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
			Expect(shouldRefresh(esv1beta1.ExternalSecret{
				Status: esv1beta1.ExternalSecretStatus{
					SyncedResourceVersion: "some resource version",
				},
			})).To(BeTrue())
		})
		It("should refresh when labels change", func() {
			es := esv1beta1.ExternalSecret{
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
			es := esv1beta1.ExternalSecret{
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
			es := esv1beta1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: esv1beta1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: 0},
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
			es := esv1beta1.ExternalSecret{
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
			es := esv1beta1.ExternalSecret{
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
			es := esv1beta1.ExternalSecret{
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

var _ = Describe("Controller Reconcile logic", func() {
	Context("controller reconcile", func() {
		It("should reconcile when resource is not synced", func() {
			Expect(shouldReconcile(esv1beta1.ExternalSecret{
				Status: esv1beta1.ExternalSecretStatus{
					SyncedResourceVersion: "some resource version",
					Conditions:            []esv1beta1.ExternalSecretStatusCondition{{Reason: "NotASecretSynced"}},
				},
			})).To(BeTrue())
		})

		It("should reconcile when secret isn't immutable", func() {
			Expect(shouldReconcile(esv1beta1.ExternalSecret{
				Spec: esv1beta1.ExternalSecretSpec{
					Target: esv1beta1.ExternalSecretTarget{
						Immutable: false,
					},
				},
			})).To(BeTrue())
		})

		It("should not reconcile if secret is immutable and has synced condition", func() {
			Expect(shouldReconcile(esv1beta1.ExternalSecret{
				Spec: esv1beta1.ExternalSecretSpec{
					Target: esv1beta1.ExternalSecretTarget{
						Immutable: true,
					},
				},
				Status: esv1beta1.ExternalSecretStatus{
					SyncedResourceVersion: "some resource version",
					Conditions:            []esv1beta1.ExternalSecretStatusCondition{{Reason: "SecretSynced"}},
				},
			})).To(BeFalse())
		})
	})
})

func externalSecretConditionShouldBe(name, ns string, ct esv1beta1.ExternalSecretConditionType, cs v1.ConditionStatus, v float64) bool {
	return Eventually(func() float64 {
		Expect(testExternalSecretCondition.WithLabelValues(name, ns, string(ct), string(cs)).Write(&metric)).To(Succeed())
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
