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

package secretstore

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testCase struct {
	store  esapi.GenericStore
	assert func()
}

var _ = Describe("SecretStore reconcile", func() {
	var test *testCase

	BeforeEach(func() {
		test = makeDefaultTestcase()
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.Background(), test.store)).ToNot(HaveOccurred())
	})

	// a invalid provider config should be reflected
	// in the store status condition
	invalidProvider := func(tc *testCase) {
		tc.assert = func() {
			Eventually(func() bool {
				ss := tc.store.Copy()
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      defaultStoreName,
					Namespace: ss.GetObjectMeta().Namespace,
				}, ss)
				if err != nil {
					return false
				}
				status := ss.GetStatus()
				if len(status.Conditions) != 1 {
					return false
				}
				return status.Conditions[0].Reason == esapi.ReasonInvalidProviderConfig &&
					hasEvent(tc.store.GetTypeMeta().Kind, ss.GetName(), esapi.ReasonInvalidProviderConfig)
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(BeTrue())
		}
	}

	// if controllerClass does not match the controller
	// should not touch this store
	ignoreControllerClass := func(tc *testCase) {
		spc := tc.store.GetSpec()
		spc.Controller = "something-else"
		tc.assert = func() {
			Consistently(func() bool {
				ss := tc.store.Copy()
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      defaultStoreName,
					Namespace: ss.GetObjectMeta().Namespace,
				}, ss)
				if err != nil {
					return true
				}
				return len(ss.GetStatus().Conditions) == 0
			}).
				WithTimeout(time.Second * 3).
				WithPolling(time.Millisecond * 500).
				Should(BeTrue())
		}
	}

	validProvider := func(tc *testCase) {
		spc := tc.store.GetSpec()
		spc.Provider.Vault = nil
		spc.Provider.Fake = &esapi.FakeProvider{
			Data: []esapi.FakeProviderData{},
		}

		tc.assert = func() {
			Eventually(func() bool {
				ss := tc.store.Copy()
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      defaultStoreName,
					Namespace: ss.GetNamespace(),
				}, ss)
				if err != nil {
					return false
				}

				if len(ss.GetStatus().Conditions) != 1 {
					return false
				}

				return ss.GetStatus().Conditions[0].Reason == esapi.ReasonStoreValid &&
					ss.GetStatus().Conditions[0].Type == esapi.SecretStoreReady &&
					ss.GetStatus().Conditions[0].Status == corev1.ConditionTrue &&
					hasEvent(tc.store.GetTypeMeta().Kind, ss.GetName(), esapi.ReasonStoreValid)
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(BeTrue())
		}

	}

	readWrite := func(tc *testCase) {
		spc := tc.store.GetSpec()
		spc.Provider.Vault = nil
		spc.Provider.Fake = &esapi.FakeProvider{
			Data: []esapi.FakeProviderData{},
		}

		tc.assert = func() {
			Eventually(func() bool {
				ss := tc.store.Copy()
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      defaultStoreName,
					Namespace: ss.GetNamespace(),
				}, ss)
				if err != nil {
					return false
				}

				if ss.GetStatus().Capabilities != esapi.SecretStoreReadWrite {
					return false
				}

				return true
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(BeTrue())
		}

	}

	DescribeTable("Controller Reconcile logic", func(muts ...func(tc *testCase)) {
		for _, mut := range muts {
			mut(test)
		}
		err := k8sClient.Create(context.Background(), test.store.Copy())
		Expect(err).ToNot(HaveOccurred())
		test.assert()
	},
		// namespaced store
		Entry("[namespace] invalid provider with secretStore should set InvalidStore condition", invalidProvider),
		Entry("[namespace] ignore stores with non-matching class", ignoreControllerClass),
		Entry("[namespace] valid provider has status=ready", validProvider),
		Entry("[namespace] valid provider has capabilities=ReadWrite", readWrite),

		// cluster store
		Entry("[cluster] invalid provider with secretStore should set InvalidStore condition", invalidProvider, useClusterStore),
		Entry("[cluster] ignore stores with non-matching class", ignoreControllerClass, useClusterStore),
		Entry("[cluster] valid provider has status=ready", validProvider, useClusterStore),
		Entry("[cluster] valid provider has capabilities=ReadWrite", readWrite, useClusterStore),
	)

})

const (
	defaultStoreName       = "default-store"
	defaultControllerClass = "test-ctrl"
)

func makeDefaultTestcase() *testCase {
	return &testCase{
		assert: func() {
			// this is a noop by default
		},
		store: &esapi.SecretStore{
			TypeMeta: metav1.TypeMeta{
				Kind:       esapi.SecretStoreKind,
				APIVersion: esapi.SecretStoreKindAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      defaultStoreName,
				Namespace: "default",
			},
			Spec: esapi.SecretStoreSpec{
				Controller: defaultControllerClass,
				// empty provider
				// a testCase mutator must fill in the concrete provider
				Provider: &esapi.SecretStoreProvider{
					Vault: &esapi.VaultProvider{
						Version: esapi.VaultKVStoreV1,
					},
				},
			},
		},
	}
}

func useClusterStore(tc *testCase) {
	spc := tc.store.GetSpec()
	meta := tc.store.GetObjectMeta()

	tc.store = &esapi.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind:       esapi.ClusterSecretStoreKind,
			APIVersion: esapi.ClusterSecretStoreKindAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: meta.Name,
		},
		Spec: *spc,
	}
}

func hasEvent(involvedKind, name, reason string) bool {
	el := &corev1.EventList{}
	err := k8sClient.List(context.Background(), el)
	if err != nil {
		return false
	}
	for i := range el.Items {
		ev := el.Items[i]
		if ev.InvolvedObject.Kind == involvedKind && ev.InvolvedObject.Name == name {
			if ev.Reason == reason {
				return true
			}
		}
	}
	return false
}
