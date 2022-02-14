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
package crds

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

type testCase struct {
	crd     unstructured.Unstructured
	crd2    unstructured.Unstructured
	service corev1.Service
	secret  corev1.Secret
	assert  func()
}

var _ = Describe("CRD reconcile", func() {
	var test *testCase

	BeforeEach(func() {
		test = makeDefaultTestcase()
	})

	AfterEach(func() {
	})

	// a invalid provider config should be reflected
	// in the store status condition
	PatchesCRD := func(tc *testCase) {
		tc.assert = func() {
			Consistently(func() bool {
				ss := unstructured.Unstructured{}
				ss.SetGroupVersionKind(schema.GroupVersionKind{Kind: "CustomResourceDefinition", Version: "v1", Group: "apiextensions.k8s.io"})
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name: "secretstores.test.io",
				}, &ss)
				if err != nil {
					return false
				}
				val, ok, err := unstructured.NestedString(ss.Object, "spec", "conversion", "webhook", "clientConfig", "service", "name")
				if err != nil || !ok {
					return false
				}
				want, ok, err := unstructured.NestedString(tc.crd.Object, "spec", "conversion", "webhook", "clientConfig", "service", "name")
				if err != nil || !ok {
					return false
				}
				return want != val
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(BeTrue())
		}
	}

	// if controllerClass does not match the controller
	// should not touch this store
	ignoreNonTargetCRDs := func(tc *testCase) {
		tc.assert = func() {
			Consistently(func() bool {
				ss := unstructured.Unstructured{}
				ss.SetGroupVersionKind(schema.GroupVersionKind{Kind: "CustomResourceDefinition", Version: "v1", Group: "apiextensions.k8s.io"})
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name: "some-other.test.io",
				}, &ss)
				if err != nil {
					return false
				}
				got, ok, err := unstructured.NestedString(ss.Object, "spec", "conversion", "webhook", "clientConfig", "service", "name")
				if !ok || err != nil {
					return false
				}
				want, ok, err := unstructured.NestedString(tc.crd2.Object, "spec", "conversion", "webhook", "clientConfig", "service", "name")
				if !ok || err != nil {
					return false
				}
				return got == want
			}).
				WithTimeout(time.Second * 3).
				WithPolling(time.Millisecond * 500).
				Should(BeTrue())
		}
	}

	DescribeTable("Controller Reconcile logic", func(muts ...func(tc *testCase)) {
		for _, mut := range muts {
			mut(test)
		}
		ctx := context.Background()
		k8sClient.Create(ctx, &test.secret)
		k8sClient.Create(ctx, &test.service)
		k8sClient.Create(ctx, &test.crd)
		k8sClient.Create(ctx, &test.crd2)
		test.assert()
	},

		Entry("[namespace] Ignore non Target CRDs", ignoreNonTargetCRDs),
		Entry("[namespace] Patch target CRDs", PatchesCRD),
	)

})

func makeUnstructuredCRD(plural, group string) unstructured.Unstructured {
	crd := apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: plural + "." + group,
		},
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Versions: []apiextensions.CustomResourceDefinitionVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &apiextensions.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensions.JSONSchemaProps{
							Type: "object",
						},
					},
				},
			},
			Group: group,
			Scope: apiextensions.NamespaceScoped,
			Names: apiextensions.CustomResourceDefinitionNames{
				Plural:   plural,
				Singular: "idc",
				Kind:     "IDC",
				ListKind: "IDCList",
			},
			Conversion: &apiextensions.CustomResourceConversion{
				Strategy: "Webhook",
				Webhook: &apiextensions.WebhookConversion{
					ConversionReviewVersions: []string{"v1"},
					ClientConfig: &apiextensions.WebhookClientConfig{
						CABundle: []byte("foobar"),
						Service: &apiextensions.ServiceReference{
							Name:      "webhook",
							Namespace: "default",
						},
					},
				},
			},
		},
	}
	marshal, _ := json.Marshal(crd)
	unmarshal := make(map[string]interface{})
	json.Unmarshal(marshal, &unmarshal)
	u := unstructured.Unstructured{
		Object: unmarshal,
	}
	u.SetGroupVersionKind(schema.GroupVersionKind{Kind: "CustomResourceDefinition", Version: "v1", Group: "apiextensions.k8s.io"})
	return u
}

func makeSecret() corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}
}

func makeService() corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
}

func makeDefaultTestcase() *testCase {
	return &testCase{
		assert: func() {
			// this is a noop by default
		},
		crd:     makeUnstructuredCRD("secretstores", "test.io"),
		crd2:    makeUnstructuredCRD("some-other", "test.io"),
		secret:  makeSecret(),
		service: makeService(),
	}
}
