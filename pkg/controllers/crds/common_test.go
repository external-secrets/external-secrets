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

// import (
// 	"context"
// 	"time"

// 	. "github.com/onsi/ginkgo/v2"
// 	. "github.com/onsi/gomega"
// 	corev1 "k8s.io/api/core/v1"
// 	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/types"

// 	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
// )

// type testCase struct {
// 	crd    apiextensions.CustomResourceDefinition
// 	assert func()
// }

// var _ = Describe("CRD reconcile", func() {
// 	var test *testCase

// 	BeforeEach(func() {
// 		test = makeDefaultTestcase()
// 	})

// 	AfterEach(func() {
// 		Expect(k8sClient.Delete(context.Background(), &test.crd)).ToNot(HaveOccurred())
// 	})

// 	// a invalid provider config should be reflected
// 	// in the store status condition
// 	PatchesCRD := func(tc *testCase) {
// 		tc.assert = func() {
// 			Eventually(func() bool {
// 				ss := apiextensions.CustomResourceDefinition{}
// 				err := k8sClient.Get(context.Background(), types.NamespacedName{
// 					Name: "secrestores.external-secrets.io",
// 				}, &ss)
// 				if err != nil {
// 					return false
// 				}
// 				return ss.Spec.Conversion.Strategy == "Webhook"
// 			}).
// 				WithTimeout(time.Second * 10).
// 				WithPolling(time.Second).
// 				Should(BeTrue())
// 		}
// 	}

// 	// if controllerClass does not match the controller
// 	// should not touch this store
// 	ignoreNonTargetCRDs := func(tc *testCase) {
// 		tc.assert = func() {
// 			Consistently(func() bool {
// 				ss := apiextensions.CustomResourceDefinition{}
// 				err := k8sClient.Get(context.Background(), types.NamespacedName{
// 					Name: defaultStoreName,
// 				}, &ss)
// 				if err != nil {
// 					return false
// 				}
// 				return ss.Spec.Conversion == tc.crd.Spec.Conversion
// 			}).
// 				WithTimeout(time.Second * 3).
// 				WithPolling(time.Millisecond * 500).
// 				Should(BeTrue())
// 		}
// 	}

// 	DescribeTable("Controller Reconcile logic", func(muts ...func(tc *testCase)) {
// 		for _, mut := range muts {
// 			mut(test)
// 		}
// 		err := k8sClient.Create(context.Background(), test.store.Copy())
// 		Expect(err).ToNot(HaveOccurred())
// 		test.assert()
// 	},
// 		// namespaced store
// 		Entry("[namespace] invalid provider with secretStore should set InvalidStore condition", invalidProvider),
// 		Entry("[namespace] ignore stores with non-matching class", ignoreControllerClass),
// 		Entry("[namespace] valid provider has status=ready", validProvider),

// 		// cluster store
// 		Entry("[cluster] invalid provider with secretStore should set InvalidStore condition", invalidProvider, useClusterStore),
// 		Entry("[cluster] ignore stores with non-matching class", ignoreControllerClass, useClusterStore),
// 		Entry("[cluster] valid provider has status=ready", validProvider, useClusterStore),
// 	)

// })

// const (
// 	defaultStoreName       = "default-store"
// 	defaultControllerClass = "test-ctrl"
// )

// func makeDefaultTestcase() *testCase {
// 	return &testCase{
// 		assert: func() {
// 			// this is a noop by default
// 		},
// 		crd: &apiextensions.CustomResourceDefinition{
// 			TypeMeta: metav1.TypeMeta{
// 				Kind:       esapi.SecretStoreKind,
// 				APIVersion: esapi.SecretStoreKindAPIVersion,
// 			},
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      defaultStoreName,
// 				Namespace: "default",
// 			},
// 			Spec: esapi.SecretStoreSpec{
// 				Controller: defaultControllerClass,
// 				// empty provider
// 				// a testCase mutator must fill in the concrete provider
// 				Provider: &esapi.SecretStoreProvider{
// 					Vault: &esapi.VaultProvider{
// 						Version: esapi.VaultKVStoreV1,
// 					},
// 				},
// 			},
// 		},
// 	}
// }

// func hasEvent(involvedKind, name, reason string) bool {
// 	el := &corev1.EventList{}
// 	err := k8sClient.List(context.Background(), el)
// 	if err != nil {
// 		return false
// 	}
// 	for i := range el.Items {
// 		ev := el.Items[i]
// 		if ev.InvolvedObject.Kind == involvedKind && ev.InvolvedObject.Name == name {
// 			if ev.Reason == reason {
// 				return true
// 			}
// 		}
// 	}
// 	return false
// }
