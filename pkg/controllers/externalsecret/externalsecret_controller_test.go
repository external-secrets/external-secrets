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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	dto "github.com/prometheus/client_model/go"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

var (
	fakeProvider *fake.Client
	metric       dto.Metric
	timeout      = time.Second * 10
	interval     = time.Millisecond * 250
)

type testCase struct {
	secretStore    *esv1alpha1.SecretStore
	externalSecret *esv1alpha1.ExternalSecret

	// checkCondition should return true if the externalSecret
	// has the expected condition
	checkCondition func(*esv1alpha1.ExternalSecret) bool

	// checkExternalSecret is called after the condition has been verified
	// use this to verify the externalSecret
	checkExternalSecret func(*esv1alpha1.ExternalSecret)

	// optional. use this to test the secret value
	checkSecret func(*esv1alpha1.ExternalSecret, *v1.Secret)
}

type testTweaks func(*testCase)

var _ = Describe("ExternalSecret controller", func() {
	const (
		ExternalSecretName             = "test-es"
		ExternalSecretStore            = "test-store"
		ExternalSecretTargetSecretName = "test-secret"
	)

	var ExternalSecretNamespace string

	BeforeEach(func() {
		var err error
		ExternalSecretNamespace, err = CreateNamespace("test-ns", k8sClient)
		Expect(err).ToNot(HaveOccurred())
		metric.Reset()
		syncCallsTotal.Reset()
		syncCallsError.Reset()
		externalSecretCondition.Reset()
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ExternalSecretNamespace,
			},
		}, client.PropagationPolicy(metav1.DeletePropagationBackground)), client.GracePeriodSeconds(0)).To(Succeed())
		Expect(k8sClient.Delete(context.Background(), &esv1alpha1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretStore,
				Namespace: ExternalSecretNamespace,
			},
		}, client.PropagationPolicy(metav1.DeletePropagationBackground)), client.GracePeriodSeconds(0)).To(Succeed())
	})

	const targetProp = "targetProperty"
	const remoteKey = "barz"
	const remoteProperty = "bang"

	makeDefaultTestcase := func() *testCase {
		return &testCase{
			// default condition: es should be ready
			checkCondition: func(es *esv1alpha1.ExternalSecret) bool {
				cond := GetExternalSecretCondition(es.Status, esv1alpha1.ExternalSecretReady)
				if cond == nil || cond.Status != v1.ConditionTrue {
					return false
				}
				return true
			},
			checkExternalSecret: func(es *esv1alpha1.ExternalSecret) {},
			secretStore: &esv1alpha1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ExternalSecretStore,
					Namespace: ExternalSecretNamespace,
				},
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Service: esv1alpha1.AWSServiceSecretsManager,
						},
					},
				},
			},
			externalSecret: &esv1alpha1.ExternalSecret{
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
					Data: []esv1alpha1.ExternalSecretData{
						{
							SecretKey: targetProp,
							RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
								Key:      remoteKey,
								Property: remoteProperty,
							},
						},
					},
				},
			},
		}
	}

	// labels and annotations from the Kind=ExternalSecret
	// should be copied over to the Kind=Secret
	syncLabelsAnnotations := func(tc *testCase) {
		const secretVal = "someValue"
		tc.externalSecret.ObjectMeta.Labels = map[string]string{
			"fooobar": "bazz",
		}
		tc.externalSecret.ObjectMeta.Annotations = map[string]string{
			"hihihih": "hehehe",
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1alpha1.ExternalSecret, secret *v1.Secret) {
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 0.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 1.0)).To(BeTrue())
			Eventually(func() bool {
				Expect(syncCallsTotal.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue() == 1.0
			}, timeout, interval).Should(BeTrue())

			// check value
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))

			// check labels & annotations
			Expect(secret.ObjectMeta.Labels).To(BeEquivalentTo(es.ObjectMeta.Labels))
			Expect(secret.ObjectMeta.Annotations).To(BeEquivalentTo(es.ObjectMeta.Annotations))
		}
	}

	// when using a template it should be used as a blueprint
	// to construct a new secret: labels, annotations and type
	syncWithTemplate := func(tc *testCase) {
		const secretVal = "someValue"
		const expectedSecretVal = "SOMEVALUE was templated"
		const tplStaticKey = "tplstatickey"
		const tplStaticVal = "tplstaticvalue"
		tc.externalSecret.ObjectMeta.Labels = map[string]string{
			"fooobar": "bazz",
		}
		tc.externalSecret.ObjectMeta.Annotations = map[string]string{
			"hihihih": "hehehe",
		}
		tc.externalSecret.Spec.Target.Template = &esv1alpha1.ExternalSecretTemplate{
			Metadata: esv1alpha1.ExternalSecretTemplateMetadata{
				Labels: map[string]string{
					"foos": "ball",
				},
				Annotations: map[string]string{
					"hihi": "ga",
				},
			},
			Type: v1.SecretTypeOpaque,
			Data: map[string]string{
				targetProp:   "{{ .targetProperty | toString | upper }} was templated",
				tplStaticKey: tplStaticVal,
			},
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1alpha1.ExternalSecret, secret *v1.Secret) {
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 0.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 1.0)).To(BeTrue())
			Eventually(func() bool {
				Expect(syncCallsTotal.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue() == 1.0
			}, timeout, interval).Should(BeTrue())

			// check values
			Expect(string(secret.Data[targetProp])).To(Equal(expectedSecretVal))
			Expect(string(secret.Data[tplStaticKey])).To(Equal(tplStaticVal))

			// labels/annotations should be taken from the template
			Expect(secret.ObjectMeta.Labels).To(BeEquivalentTo(es.Spec.Target.Template.Metadata.Labels))
			Expect(secret.ObjectMeta.Annotations).To(BeEquivalentTo(es.Spec.Target.Template.Metadata.Annotations))
		}
	}

	// secret should be synced with correct value precedence:
	// * template
	// * templateFrom (not implemented)
	// * data
	// * dataFrom
	syncWithTemplatePrecedence := func(tc *testCase) {
		const secretVal = "someValue"
		const expectedSecretVal = "SOMEVALUE was templated"
		const tplStaticKey = "tplstatickey"
		const tplStaticVal = "tplstaticvalue"
		tc.externalSecret.Spec.Target.Template = &esv1alpha1.ExternalSecretTemplate{
			Metadata: esv1alpha1.ExternalSecretTemplateMetadata{},
			Type:     v1.SecretTypeOpaque,
			Data: map[string]string{
				// this should be the data value, not dataFrom
				targetProp: "{{ .targetProperty | toString | upper }} was templated",
				// this should use the value from the map
				"bar": "value from map: {{ .bar | toString }}",
				// just a static value
				tplStaticKey: tplStaticVal,
			},
		}
		tc.externalSecret.Spec.DataFrom = []esv1alpha1.ExternalSecretDataRemoteRef{
			{
				Key: "datamap",
			},
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		fakeProvider.WithGetSecretMap(map[string][]byte{
			"targetProperty": []byte("map-foo-value"),
			"bar":            []byte("map-bar-value"),
		}, nil)
		tc.checkSecret = func(es *esv1alpha1.ExternalSecret, secret *v1.Secret) {
			// check values
			Expect(string(secret.Data[targetProp])).To(Equal(expectedSecretVal))
			Expect(string(secret.Data[tplStaticKey])).To(Equal(tplStaticVal))
			Expect(string(secret.Data["bar"])).To(Equal("value from map: map-bar-value"))
		}
	}

	refreshWithTemplate := func(tc *testCase) {
		const secretVal = "someValue"
		const expectedSecretVal = "SOMEVALUE was templated"
		const tplStaticKey = "tplstatickey"
		const tplStaticVal = "tplstaticvalue"
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.externalSecret.Spec.Target.Template = &esv1alpha1.ExternalSecretTemplate{
			Metadata: esv1alpha1.ExternalSecretTemplateMetadata{
				Labels:      map[string]string{"foo": "bar"},
				Annotations: map[string]string{"foo": "bar"},
			},
			Type: v1.SecretTypeOpaque,
			Data: map[string]string{
				targetProp:   "{{ .targetProperty | toString | upper }} was templated",
				tplStaticKey: tplStaticVal,
			},
		}
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1alpha1.ExternalSecret, secret *v1.Secret) {
			// check values
			Expect(string(secret.Data[targetProp])).To(Equal(expectedSecretVal))
			Expect(string(secret.Data[tplStaticKey])).To(Equal(tplStaticVal))

			// labels/annotations should be taken from the template
			Expect(secret.ObjectMeta.Labels).To(BeEquivalentTo(es.Spec.Target.Template.Metadata.Labels))
			Expect(secret.ObjectMeta.Annotations).To(BeEquivalentTo(es.Spec.Target.Template.Metadata.Annotations))

			cleanEs := tc.externalSecret.DeepCopyObject()

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
				fmt.Fprintf(GinkgoWriter, "data: %#v\n", sec.Data)
				// ensure new data value exist
				return string(sec.Data["new"]) == "value"
			}, time.Second*10, time.Millisecond*200).Should(BeTrue())

			// also check labels/annotations have been updated
			Expect(secret.ObjectMeta.Labels).To(BeEquivalentTo(es.Spec.Target.Template.Metadata.Labels))
			Expect(secret.ObjectMeta.Annotations).To(BeEquivalentTo(es.Spec.Target.Template.Metadata.Annotations))
		}
	}

	// when the provider secret changes the Kind=Secret value
	// must change, too.
	refreshSecretValue := func(tc *testCase) {
		const targetProp = "targetProperty"
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
		tc.checkSecret = func(es *esv1alpha1.ExternalSecret, secret *v1.Secret) {
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 0.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 1.0)).To(BeTrue())
			Eventually(func() bool {
				Expect(syncCallsTotal.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue() == 1.0
			}, timeout, interval).Should(BeTrue())

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

	// with dataFrom all properties from the specified secret
	// should be put into the secret
	syncWithDataFrom := func(tc *testCase) {
		tc.externalSecret.Spec.Data = nil
		tc.externalSecret.Spec.DataFrom = []esv1alpha1.ExternalSecretDataRemoteRef{
			{
				Key: remoteKey,
			},
		}
		fakeProvider.WithGetSecretMap(map[string][]byte{
			"foo": []byte("map-foo-value"),
			"bar": []byte("map-bar-value"),
		}, nil)
		tc.checkSecret = func(es *esv1alpha1.ExternalSecret, secret *v1.Secret) {
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 0.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 1.0)).To(BeTrue())
			Eventually(func() bool {
				Expect(syncCallsTotal.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue() == 1.0
			}, timeout, interval).Should(BeTrue())

			// check values
			Expect(string(secret.Data["foo"])).To(Equal("map-foo-value"))
			Expect(string(secret.Data["bar"])).To(Equal("map-bar-value"))
		}
	}

	// when a provider errors in a GetSecret call
	// a error condition must be set.
	providerErrCondition := func(tc *testCase) {
		const secretVal = "foobar"
		fakeProvider.WithGetSecret(nil, fmt.Errorf("boom"))
		tc.externalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Millisecond * 100}
		tc.checkCondition = func(es *esv1alpha1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1alpha1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1alpha1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
		tc.checkExternalSecret = func(es *esv1alpha1.ExternalSecret) {
			Eventually(func() bool {
				Expect(syncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0
			}, timeout, interval).Should(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())

			// es condition should reflect recovered provider error
			fakeProvider.WithGetSecret([]byte(secretVal), nil)
			esKey := types.NamespacedName{Name: ExternalSecretName, Namespace: ExternalSecretNamespace}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), esKey, es)
				if err != nil {
					return false
				}
				// condition must now be true!
				cond := GetExternalSecretCondition(es.Status, esv1alpha1.ExternalSecretReady)
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
		tc.checkCondition = func(es *esv1alpha1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1alpha1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1alpha1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
		tc.checkExternalSecret = func(es *esv1alpha1.ExternalSecret) {
			Eventually(func() bool {
				Expect(syncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0
			}, timeout, interval).Should(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
		}
	}

	// when the provider constructor errors (e.g. invalid configuration)
	// a SecretSyncedError status condition must be set
	storeConstructErrCondition := func(tc *testCase) {
		fakeProvider.WithNew(func(context.Context, esv1alpha1.GenericStore, client.Client,
			string) (provider.SecretsClient, error) {
			return nil, fmt.Errorf("artificial constructor error")
		})
		tc.checkCondition = func(es *esv1alpha1.ExternalSecret) bool {
			// condition must be false
			cond := GetExternalSecretCondition(es.Status, esv1alpha1.ExternalSecretReady)
			if cond == nil || cond.Status != v1.ConditionFalse || cond.Reason != esv1alpha1.ConditionReasonSecretSyncedError {
				return false
			}
			return true
		}
		tc.checkExternalSecret = func(es *esv1alpha1.ExternalSecret) {
			Eventually(func() bool {
				Expect(syncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0
			}, timeout, interval).Should(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
		}
	}

	// when a SecretStore has a controller field set which we don't care about
	// the externalSecret must not be touched
	ignoreMismatchController := func(tc *testCase) {
		tc.secretStore.Spec.Controller = "nop"
		tc.checkCondition = func(es *esv1alpha1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1alpha1.ExternalSecretReady)
			return cond == nil
		}
		tc.checkExternalSecret = func(es *esv1alpha1.ExternalSecret) {
			// Condition True and False should be 0, since the Condition was not created
			Eventually(func() float64 {
				Expect(externalSecretCondition.WithLabelValues(ExternalSecretName, ExternalSecretNamespace, string(esv1alpha1.ExternalSecretReady), string(v1.ConditionTrue)).Write(&metric)).To(Succeed())
				return metric.GetGauge().GetValue()
			}, timeout, interval).Should(Equal(0.0))

			Eventually(func() float64 {
				Expect(externalSecretCondition.WithLabelValues(ExternalSecretName, ExternalSecretNamespace, string(esv1alpha1.ExternalSecretReady), string(v1.ConditionFalse)).Write(&metric)).To(Succeed())
				return metric.GetGauge().GetValue()
			}, timeout, interval).Should(Equal(0.0))

			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionFalse, 0.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1alpha1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
		}
	}

	DescribeTable("When reconciling an ExternalSecret",
		func(tweaks ...testTweaks) {
			tc := makeDefaultTestcase()
			for _, tweak := range tweaks {
				tweak(tc)
			}
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, tc.secretStore)).To(Succeed())
			Expect(k8sClient.Create(ctx, tc.externalSecret)).Should(Succeed())
			esKey := types.NamespacedName{Name: ExternalSecretName, Namespace: ExternalSecretNamespace}
			createdES := &esv1alpha1.ExternalSecret{}
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
					Name:      ExternalSecretTargetSecretName,
					Namespace: ExternalSecretNamespace,
				}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, secretLookupKey, syncedSecret)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				tc.checkSecret(createdES, syncedSecret)
			}
		},
		Entry("should set the condition eventually", syncLabelsAnnotations),
		Entry("should sync with template", syncWithTemplate),
		Entry("should sync template with correct value precedence", syncWithTemplatePrecedence),
		Entry("should refresh secret from template", refreshWithTemplate),
		Entry("should refresh secret value when provider secret changes", refreshSecretValue),
		Entry("should fetch secret using dataFrom", syncWithDataFrom),
		Entry("should set error condition when provider errors", providerErrCondition),
		Entry("should set an error condition when store does not exist", storeMissingErrCondition),
		Entry("should set an error condition when store provider constructor fails", storeConstructErrCondition),
		Entry("should not process store with mismatching controller field", ignoreMismatchController),
	)
})

var _ = Describe("ExternalSecret refresh logic", func() {
	Context("secret refresh", func() {
		It("should refresh when resource version does not match", func() {
			Expect(shouldRefresh(esv1alpha1.ExternalSecret{
				Status: esv1alpha1.ExternalSecretStatus{
					SyncedResourceVersion: "some resource version",
				},
			})).To(BeTrue())
		})

		It("should refresh when labels change", func() {
			es := esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Status: esv1alpha1.ExternalSecretStatus{},
			}
			es.Status.SyncedResourceVersion = getResourceVersion(es)
			// this should not refresh, rv matches object
			Expect(shouldRefresh(es)).To(BeFalse())

			// change labels without changing the syncedResourceVersion and expect refresh
			es.ObjectMeta.Labels["new"] = "w00t"
			Expect(shouldRefresh(es)).To(BeTrue())
		})

		It("should refresh when annotations change", func() {
			es := esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
				Status: esv1alpha1.ExternalSecretStatus{},
			}
			es.Status.SyncedResourceVersion = getResourceVersion(es)
			// this should not refresh, rv matches object
			Expect(shouldRefresh(es)).To(BeFalse())

			// change annotations without changing the syncedResourceVersion and expect refresh
			es.ObjectMeta.Annotations["new"] = "w00t"
			Expect(shouldRefresh(es)).To(BeTrue())
		})

		It("should refresh when generation has changed", func() {
			es := esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Status: esv1alpha1.ExternalSecretStatus{},
			}
			es.Status.SyncedResourceVersion = getResourceVersion(es)
			Expect(shouldRefresh(es)).To(BeFalse())

			// update gen -> refresh
			es.ObjectMeta.Generation = 2
			Expect(shouldRefresh(es)).To(BeTrue())
		})

		It("should skip refresh when refreshInterval is nil", func() {
			es := esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					RefreshInterval: nil,
				},
				Status: esv1alpha1.ExternalSecretStatus{},
			}
			// resource version matches
			es.Status.SyncedResourceVersion = getResourceVersion(es)
			Expect(shouldRefresh(es)).To(BeFalse())
		})

		It("should refresh when refresh interval has passed", func() {
			es := esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Second},
				},
				Status: esv1alpha1.ExternalSecretStatus{
					RefreshTime: metav1.NewTime(metav1.Now().Add(-time.Second * 5)),
				},
			}
			// resource version matches
			es.Status.SyncedResourceVersion = getResourceVersion(es)
			Expect(shouldRefresh(es)).To(BeTrue())
		})

		It("should refresh when no refresh time was set", func() {
			es := esv1alpha1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: esv1alpha1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Second},
				},
				Status: esv1alpha1.ExternalSecretStatus{},
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

// CreateNamespace creates a new namespace in the cluster.
func CreateNamespace(baseName string, c client.Client) (string, error) {
	genName := fmt.Sprintf("ctrl-test-%v", baseName)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: genName,
		},
	}
	var err error
	err = wait.Poll(time.Second, 10*time.Second, func() (bool, error) {
		err = c.Create(context.Background(), ns)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", err
	}
	return ns.Name, nil
}

func externalSecretConditionShouldBe(name, ns string, ct esv1alpha1.ExternalSecretConditionType, cs v1.ConditionStatus, v float64) bool {
	return Eventually(func() float64 {
		Expect(externalSecretCondition.WithLabelValues(name, ns, string(ct), string(cs)).Write(&metric)).To(Succeed())
		return metric.GetGauge().GetValue()
	}, timeout, interval).Should(Equal(v))
}

func init() {
	fakeProvider = fake.New()
	schema.ForceRegister(fakeProvider, &esv1alpha1.SecretStoreProvider{
		AWS: &esv1alpha1.AWSProvider{
			Service: esv1alpha1.AWSServiceSecretsManager,
		},
	})
}
