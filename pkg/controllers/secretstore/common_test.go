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
	"errors"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
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

	// an invalid provider config should be reflected
	// in the store status condition
	invalidProvider := func(tc *testCase) {
		tc.assert = func() {
			Eventually(func(g Gomega) error {
				ss := tc.store.Copy()
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      defaultStoreName,
					Namespace: ss.GetObjectMeta().Namespace,
				}, ss)
				if err != nil {
					return err
				}
				status := ss.GetStatus()
				expected := []esapi.SecretStoreStatusCondition{
					{
						Type:    esapi.SecretStoreReady,
						Status:  corev1.ConditionFalse,
						Reason:  esapi.ReasonInvalidProviderConfig,
						Message: fmt.Errorf(errStoreClient, errors.New("cannot initialize Vault client: no valid auth method specified")).Error(),
					},
				}

				opts := cmpopts.IgnoreFields(esapi.SecretStoreStatusCondition{}, "LastTransitionTime")
				g.Expect(status.Conditions).To(BeComparableTo(expected, opts))
				return nil
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(Succeed())
		}
	}

	// if controllerClass does not match the controller
	// should not touch this store
	ignoreControllerClass := func(tc *testCase) {
		spc := tc.store.GetSpec()
		spc.Controller = "something-else"
		tc.assert = func() {
			Consistently(func(g Gomega) error {
				ss := tc.store.Copy()
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      defaultStoreName,
					Namespace: ss.GetObjectMeta().Namespace,
				}, ss)
				if err != nil {
					return err
				}

				status := ss.GetStatus()
				g.Expect(status.Conditions).To(BeEmpty())
				return nil
			}).
				WithTimeout(time.Second * 3).
				WithPolling(time.Millisecond * 500).
				Should(Succeed())
		}
	}

	validProvider := func(tc *testCase) {
		spc := tc.store.GetSpec()
		spc.Provider.Vault = nil
		spc.Provider.Fake = &esapi.FakeProvider{
			Data: []esapi.FakeProviderData{},
		}

		tc.assert = func() {
			Eventually(func(g Gomega) error {
				ss := tc.store.Copy()
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      defaultStoreName,
					Namespace: ss.GetNamespace(),
				}, ss)
				if err != nil {
					return err
				}

				status := ss.GetStatus()
				expected := []esapi.SecretStoreStatusCondition{
					{
						Type:    esapi.SecretStoreReady,
						Status:  corev1.ConditionTrue,
						Reason:  esapi.ReasonStoreValid,
						Message: msgValid,
					},
				}

				opts := cmpopts.IgnoreFields(esapi.SecretStoreStatusCondition{}, "LastTransitionTime")
				g.Expect(status.Conditions).To(BeComparableTo(expected, opts))
				return nil
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(Succeed())
		}

	}

	readWrite := func(tc *testCase) {
		spc := tc.store.GetSpec()
		spc.Provider.Vault = nil
		spc.Provider.Fake = &esapi.FakeProvider{
			Data: []esapi.FakeProviderData{},
		}

		tc.assert = func() {
			Eventually(func(g Gomega) error {
				ss := tc.store.Copy()
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      defaultStoreName,
					Namespace: ss.GetNamespace(),
				}, ss)
				if err != nil {
					return err
				}

				status := ss.GetStatus()
				g.Expect(status.Capabilities).To(Equal(esapi.SecretStoreReadWrite))
				return nil
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(Succeed())
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
