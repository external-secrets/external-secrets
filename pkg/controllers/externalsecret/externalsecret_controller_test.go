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
	"fmt"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	dto "github.com/prometheus/client_model/go"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	ctest "github.com/external-secrets/external-secrets/pkg/controllers/commontest"
	"github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
)

var (
	fakeProvider *fake.Client
	metric       dto.Metric
	timeout      = time.Second * 10
	interval     = time.Millisecond * 250
)

type testCase struct {
	secretStore    *esv1beta1.SecretStore
	externalSecret *esv1beta1.ExternalSecret

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
						esv1beta1.AnnotationDataHash: "caa0155759a6a9b3b6ada5a6883ee2bb",
					},
				},
				Data: map[string][]byte{
					"foo": []byte("value1"),
					"bar": []byte("value2"),
				},
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
var _ = Describe("ExternalSecret controller", func() {
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
		ExternalSecretNamespace, err = ctest.CreateNamespace("test-ns", k8sClient)
		Expect(err).ToNot(HaveOccurred())
		metric.Reset()
		syncCallsTotal.Reset()
		syncCallsError.Reset()
		externalSecretCondition.Reset()
		fakeProvider.Reset()
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ExternalSecretNamespace,
			},
		})).To(Succeed())
		Expect(k8sClient.Delete(context.Background(), &esv1beta1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalSecretStore,
				Namespace: ExternalSecretNamespace,
			},
		})).To(Succeed())
	})

	const targetProp = "targetProperty"
	const remoteKey = "barz"
	const remoteProperty = "bang"

	makeDefaultTestcase := func() *testCase {
		return &testCase{
			// default condition: es should be ready
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
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			// check value
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))

			// check labels & annotations
			Expect(secret.ObjectMeta.Labels).To(BeEquivalentTo(es.ObjectMeta.Labels))
			for k, v := range es.ObjectMeta.Annotations {
				Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue(k, v))
			}
			// ownerRef must not not be set!
			Expect(ctest.HasOwnerRef(secret.ObjectMeta, "ExternalSecret", ExternalSecretName)).To(BeTrue())
		}
	}

	checkPrometheusCounters := func(tc *testCase) {
		const secretVal = "someValue"
		fakeProvider.WithGetSecret([]byte(secretVal), nil)
		tc.checkSecret = func(es *esv1beta1.ExternalSecret, secret *v1.Secret) {
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 0.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 1.0)).To(BeTrue())
			Eventually(func() bool {
				Expect(syncCallsTotal.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue() == 1.0
			}, timeout, interval).Should(BeTrue())
		}
	}

	// merge with existing secret using creationPolicy=Merge
	// it should NOT have a ownerReference
	// metadata.managedFields with the correct owner should be added to the secret
	mergeWithSecret := func(tc *testCase) {
		const secretVal = "someValue"
		const existingKey = "pre-existing-key"
		existingVal := "pre-existing-value"
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
			// check value
			Expect(string(secret.Data[existingKey])).To(Equal(existingVal))
			Expect(string(secret.Data[targetProp])).To(Equal(secretVal))

			// check labels & annotations
			Expect(secret.ObjectMeta.Labels).To(BeEquivalentTo(es.ObjectMeta.Labels))
			for k, v := range es.ObjectMeta.Annotations {
				Expect(secret.ObjectMeta.Annotations).To(HaveKeyWithValue(k, v))
			}
			Expect(ctest.HasOwnerRef(secret.ObjectMeta, "ExternalSecret", ExternalSecretFQDN)).To(BeFalse())
			Expect(secret.ObjectMeta.ManagedFields).To(HaveLen(2))
			Expect(ctest.HasFieldOwnership(
				secret.ObjectMeta,
				ExternalSecretFQDN,
				fmt.Sprintf("{\"f:data\":{\"f:targetProperty\":{}},\"f:immutable\":{},\"f:metadata\":{\"f:annotations\":{\"f:%s\":{}}}}", esv1beta1.AnnotationDataHash)),
			).To(BeTrue())
			Expect(ctest.HasFieldOwnership(secret.ObjectMeta, FakeManager, "{\"f:data\":{\".\":{},\"f:pre-existing-key\":{}},\"f:type\":{}}")).To(BeTrue())
		}
	}

	// should not update if no changes
	mergeWithSecretNoChange := func(tc *testCase) {
		const existingKey = "pre-existing-key"
		existingVal := "someValue"
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
				Expect(syncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0
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
		existingVal := "pre-existing-value"
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
			Expect(ctest.HasFieldOwnership(secret.ObjectMeta, ExternalSecretFQDN, "{\"f:data\":{\"f:targetProperty\":{}},\"f:immutable\":{},\"f:metadata\":{\"f:annotations\":{\"f:reconcile.external-secrets.io/data-hash\":{}}}}")).To(BeTrue())
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
			Expect(secret.ObjectMeta.Labels).To(BeEquivalentTo(es.Spec.Target.Template.Metadata.Labels))
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

	// secret should be synced with correct value precedence:
	// * template
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
			// check values
			Expect(string(secret.Data[targetProp])).To(Equal(expectedSecretVal))
			Expect(string(secret.Data[tplStaticKey])).To(Equal(tplStaticVal))
			Expect(string(secret.Data["bar"])).To(Equal("value from map: map-bar-value"))
			Expect(string(secret.Data[tplFromKey])).To(Equal("tpl-from-value: someValue // map-bar-value"))
			Expect(string(secret.Data[tplFromSecKey])).To(Equal("tpl-from-sec-value: someValue // map-bar-value"))
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
			Expect(secret.ObjectMeta.Labels).To(BeEquivalentTo(es.Spec.Target.Template.Metadata.Labels))

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
			Expect(secret.ObjectMeta.Labels).To(BeEquivalentTo(es.Spec.Target.Template.Metadata.Labels))
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
			Expect(secret.ObjectMeta.Labels).To(BeEquivalentTo(es.Spec.Target.Template.Metadata.Labels))
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

	deleteSecretPolicy := func(tc *testCase) {
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

	deleteSecretPolicyRetain := func(tc *testCase) {
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
				return apierrors.IsNotFound(err) && bytes.Equal(sec.Data["foo"], expVal)
			}, time.Second*10, time.Second).Should(BeFalse())
		}
	}

	// merge with existing secret using creationPolicy=Merge
	// if provider secret gets deleted only the managed field should get deleted
	deleteSecretPolicyMerge := func(tc *testCase) {
		const secretVal = "someValue"
		const existingKey = "some-existing-key"
		existingVal := "some-existing-value"
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
				Expect(syncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0
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
				Expect(syncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0
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
				Expect(syncCallsError.WithLabelValues(ExternalSecretName, ExternalSecretNamespace).Write(&metric)).To(Succeed())
				return metric.GetCounter().GetValue() >= 2.0
			}, timeout, interval).Should(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionFalse, 1.0)).To(BeTrue())
			Expect(externalSecretConditionShouldBe(ExternalSecretName, ExternalSecretNamespace, esv1beta1.ExternalSecretReady, v1.ConditionTrue, 0.0)).To(BeTrue())
		}
	}

	// when a SecretStore has a controller field set which we don't care about
	// the externalSecret must not be touched
	ignoreMismatchController := func(tc *testCase) {
		tc.secretStore.Spec.Controller = "nop"
		tc.checkCondition = func(es *esv1beta1.ExternalSecret) bool {
			cond := GetExternalSecretCondition(es.Status, esv1beta1.ExternalSecretReady)
			return cond == nil
		}
		tc.checkExternalSecret = func(es *esv1beta1.ExternalSecret) {
			// Condition True and False should be 0, since the Condition was not created
			Eventually(func() float64 {
				Expect(externalSecretCondition.WithLabelValues(ExternalSecretName, ExternalSecretNamespace, string(esv1beta1.ExternalSecretReady), string(v1.ConditionTrue)).Write(&metric)).To(Succeed())
				return metric.GetGauge().GetValue()
			}, timeout, interval).Should(Equal(0.0))

			Eventually(func() float64 {
				Expect(externalSecretCondition.WithLabelValues(ExternalSecretName, ExternalSecretNamespace, string(esv1beta1.ExternalSecretReady), string(v1.ConditionFalse)).Write(&metric)).To(Succeed())
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
			Expect(secret.Annotations[esv1beta1.AnnotationDataHash]).To(Equal("9d30b95ca81e156f9454b5ef3bfcc6ee"))
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
					Name:      ExternalSecretTargetSecretName,
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
		Entry("should refresh when the hash annotation doesn't correspond to secret data", checkSecretDataHashAnnotationChange),
		Entry("should use external secret name if target secret name isn't defined", syncWithoutTargetName),
		Entry("should set the condition eventually", syncLabelsAnnotations),
		Entry("should set prometheus counters", checkPrometheusCounters),
		Entry("should merge with existing secret using creationPolicy=Merge", mergeWithSecret),
		Entry("should error if secret doesn't exist when using creationPolicy=Merge", mergeWithSecretErr),
		Entry("should not resolve conflicts with creationPolicy=Merge", mergeWithConflict),
		Entry("should not update unchanged secret using creationPolicy=Merge", mergeWithSecretNoChange),
		Entry("should not delete pre-existing secret with creationPolicy=Orphan", createSecretPolicyOrphan),
		Entry("should sync with template", syncWithTemplate),
		Entry("should sync with template engine v2", syncWithTemplateV2),
		Entry("should sync template with correct value precedence", syncWithTemplatePrecedence),
		Entry("should refresh secret from template", refreshWithTemplate),
		Entry("should be able to use only metadata from template", onlyMetadataFromTemplate),
		Entry("should refresh secret value when provider secret changes", refreshSecretValue),
		Entry("should refresh secret map when provider secret changes", refreshSecretValueMap),
		Entry("should refresh secret map when provider secret changes when using a template", refreshSecretValueMapTemplate),
		Entry("should not refresh secret value when provider secret changes but refreshInterval is zero", refreshintervalZero),
		Entry("should fetch secret using dataFrom", syncWithDataFrom),
		Entry("should fetch secret using dataFrom.find", syncDataFromFind),
		Entry("should fetch secret using dataFrom and a template", syncWithDataFromTemplate),
		Entry("should set error condition when provider errors", providerErrCondition),
		Entry("should set an error condition when store does not exist", storeMissingErrCondition),
		Entry("should set an error condition when store provider constructor fails", storeConstructErrCondition),
		Entry("should not process store with mismatching controller field", ignoreMismatchController),
		Entry("should not process cluster secret store when it is disabled", ignoreClusterSecretStoreWhenDisabled),
		Entry("should eventually delete target secret with deletionPolicy=Delete", deleteSecretPolicy),
		Entry("should not delete target secret with deletionPolicy=Retain", deleteSecretPolicyRetain),
		Entry("should not delete pre-existing secret with deletionPolicy=Merge", deleteSecretPolicyMerge),
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
		Expect(externalSecretCondition.WithLabelValues(name, ns, string(ct), string(cs)).Write(&metric)).To(Succeed())
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
}
