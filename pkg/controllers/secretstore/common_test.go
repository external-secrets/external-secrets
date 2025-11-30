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

package secretstore

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testCase struct {
	store  esapi.GenericStore
	ps     *esv1alpha1.PushSecret
	assert func()
}

const (
	defaultStoreName       = "default-store"
	defaultControllerClass = "test-ctrl"
)

var _ = Describe("SecretStore Controller", func() {
	var test *testCase

	BeforeEach(func() {
		test = makeDefaultTestcase()
	})

	Context("Reconcile Logic", func() {
		AfterEach(func() {
			Expect(k8sClient.Delete(context.Background(), test.store)).ToNot(HaveOccurred())
		})

		// an invalid provider config should be reflected
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

					conditionLen := len(ss.GetStatus().Conditions) == 0
					if !conditionLen {
						GinkgoLogr.Info("store conditions is NOT empty but should have been", "conditions", ss.GetStatus().Conditions)
					}

					return conditionLen
				}).
					WithTimeout(time.Second*3).
					WithPolling(time.Millisecond*500).
					Should(BeTrue(), "condition should have been empty")
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

		// an unknown store validation result should be reflected
		// in the store status condition
		validationUnknown := func(tc *testCase) {
			spc := tc.store.GetSpec()
			spc.Provider.Vault = nil
			validationResultUnknown := esapi.ValidationResultUnknown
			spc.Provider.Fake = &esapi.FakeProvider{
				Data:             []esapi.FakeProviderData{},
				ValidationResult: &validationResultUnknown,
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

					return ss.GetStatus().Conditions[0].Reason == esapi.ReasonValidationUnknown &&
						ss.GetStatus().Conditions[0].Type == esapi.SecretStoreReady &&
						ss.GetStatus().Conditions[0].Status == corev1.ConditionTrue &&
						hasEvent(tc.store.GetTypeMeta().Kind, ss.GetName(), esapi.ReasonValidationUnknown)
				}).
					WithTimeout(time.Second * 5).
					WithPolling(time.Second).
					Should(BeTrue())
			}
		}

		DescribeTable("Provider Configuration", func(muts ...func(tc *testCase)) {
			for _, mut := range muts {
				mut(test)
			}
			err := k8sClient.Create(context.Background(), test.store.Copy())
			Expect(err).ToNot(HaveOccurred())
			test.assert()
		},
			// Namespaced store tests
			Entry("[namespace] invalid provider should set InvalidStore condition", invalidProvider),
			Entry("[namespace] should ignore stores with non-matching controller class", ignoreControllerClass),
			Entry("[namespace] valid provider should have status=ready", validProvider),
			Entry("[namespace] valid provider should have capabilities=ReadWrite", readWrite),
			Entry("[cluster] validation unknown status should set ValidationUnknown condition", validationUnknown),

			// Cluster store tests
			Entry("[cluster] invalid provider should set InvalidStore condition", invalidProvider, useClusterStore),
			Entry("[cluster] should ignore stores with non-matching controller class", ignoreControllerClass, useClusterStore),
			Entry("[cluster] valid provider should have status=ready", validProvider, useClusterStore),
			Entry("[cluster] valid provider should have capabilities=ReadWrite", readWrite, useClusterStore),
			Entry("[cluster] validation unknown status should set ValidationUnknown condition", validationUnknown, useClusterStore),
		)
	})

	Context("Finalizer Management", func() {
		BeforeEach(func() {
			// Setup valid provider for finalizer tests
			spc := test.store.GetSpec()
			spc.Provider.Vault = nil
			spc.Provider.Fake = &esapi.FakeProvider{
				Data: []esapi.FakeProviderData{},
			}
		})

		AfterEach(func() {
			cleanupResources(test)
		})

		DescribeTable("Finalizer Addition", func(muts ...func(tc *testCase)) {
			for _, mut := range muts {
				mut(test)
			}

			Expect(k8sClient.Create(context.Background(), test.store)).ToNot(HaveOccurred())
			Expect(k8sClient.Create(context.Background(), test.ps)).ToNot(HaveOccurred())

			Eventually(func() []string {
				return getStoreFinalizers(test.store)
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(ContainElement(secretStoreFinalizer))
		},
			Entry("[namespace] should add finalizer when PushSecret with DeletionPolicy=Delete is created", usePushSecret),
			Entry("[cluster] should add finalizer when PushSecret with DeletionPolicy=Delete is created", usePushSecret, useClusterStore),
		)

		DescribeTable("Finalizer Removal on PushSecret Deletion", func(muts ...func(tc *testCase)) {
			for _, mut := range muts {
				mut(test)
			}

			test.store.SetFinalizers([]string{secretStoreFinalizer})
			Expect(k8sClient.Create(context.Background(), test.store)).ToNot(HaveOccurred())
			Expect(k8sClient.Create(context.Background(), test.ps)).ToNot(HaveOccurred())
			Expect(k8sClient.Delete(context.Background(), test.ps)).ToNot(HaveOccurred())

			Eventually(func() []string {
				return getStoreFinalizers(test.store)
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				ShouldNot(ContainElement(secretStoreFinalizer))
		},
			Entry("[namespace] should remove finalizer when PushSecret is deleted", usePushSecret),
			Entry("[cluster] should remove finalizer when PushSecret is deleted", usePushSecret, useClusterStore),
		)

		DescribeTable("Store Deletion Prevention", func(muts ...func(tc *testCase)) {
			for _, mut := range muts {
				mut(test)
			}

			Expect(k8sClient.Create(context.Background(), test.store)).ToNot(HaveOccurred())
			Expect(k8sClient.Create(context.Background(), test.ps)).ToNot(HaveOccurred())

			// Wait for finalizer to be added
			Eventually(func() []string {
				return getStoreFinalizers(test.store)
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(ContainElement(secretStoreFinalizer))

			Expect(k8sClient.Delete(context.Background(), test.store)).ToNot(HaveOccurred())

			Consistently(func() []string {
				return getStoreFinalizers(test.store)
			}).
				WithTimeout(time.Second * 3).
				WithPolling(time.Millisecond * 500).
				Should(ContainElement(secretStoreFinalizer))
		},
			Entry("[namespace] should prevent deletion when finalizer exists", usePushSecret),
			Entry("[cluster] should prevent deletion when finalizer exists", usePushSecret, useClusterStore),
		)

		DescribeTable("Complete Deletion Flow", func(muts ...func(tc *testCase)) {
			for _, mut := range muts {
				mut(test)
			}

			Expect(k8sClient.Create(context.Background(), test.store)).ToNot(HaveOccurred())
			Expect(k8sClient.Create(context.Background(), test.ps)).ToNot(HaveOccurred())
			Expect(k8sClient.Delete(context.Background(), test.store)).ToNot(HaveOccurred())
			Expect(k8sClient.Delete(context.Background(), test.ps)).ToNot(HaveOccurred())

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      test.store.GetName(),
					Namespace: test.store.GetNamespace(),
				}, test.store)
				return apierrors.IsNotFound(err)
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(BeTrue())
		},
			Entry("[namespace] should allow deletion when both Store and PushSecret are deleted", usePushSecret),
			Entry("[cluster] should allow deletion when both Store and PushSecret are deleted", usePushSecret, useClusterStore),
		)

		DescribeTable("Multiple PushSecrets Scenario", func(muts ...func(tc *testCase)) {
			for _, mut := range muts {
				mut(test)
			}

			ps2 := test.ps.DeepCopy()
			ps2.Name = "push-secret-2"

			Expect(k8sClient.Create(context.Background(), test.store)).ToNot(HaveOccurred())
			Expect(k8sClient.Create(context.Background(), test.ps)).ToNot(HaveOccurred())
			Expect(k8sClient.Create(context.Background(), ps2)).ToNot(HaveOccurred())

			// Wait for finalizer to be added
			Eventually(func() []string {
				return getStoreFinalizers(test.store)
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(ContainElement(secretStoreFinalizer))

			Expect(k8sClient.Delete(context.Background(), test.ps)).ToNot(HaveOccurred())

			// Finalizer should remain because ps2 still exists
			Consistently(func() []string {
				return getStoreFinalizers(test.store)
			}).
				WithTimeout(time.Second * 3).
				WithPolling(time.Millisecond * 500).
				Should(ContainElement(secretStoreFinalizer))

			// Cleanup
			Expect(k8sClient.Delete(context.Background(), ps2)).ToNot(HaveOccurred())
		},
			Entry("[namespace] finalizer should remain when other PushSecrets exist", usePushSecret),
			Entry("[cluster] finalizer should remain when other PushSecrets exist", usePushSecret, useClusterStore),
		)

		DescribeTable("DeletionPolicy Change", func(muts ...func(tc *testCase)) {
			for _, mut := range muts {
				mut(test)
			}

			Expect(k8sClient.Create(context.Background(), test.store)).ToNot(HaveOccurred())
			Expect(k8sClient.Create(context.Background(), test.ps)).ToNot(HaveOccurred())

			// Wait for finalizer to be added
			Eventually(func() []string {
				return getStoreFinalizers(test.store)
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(ContainElement(secretStoreFinalizer))

			// Update PushSecret to DeletionPolicy=None
			Eventually(func() error {
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      test.ps.Name,
					Namespace: test.ps.Namespace,
				}, test.ps)
				Expect(err).ToNot(HaveOccurred())
				test.ps.Spec.DeletionPolicy = esv1alpha1.PushSecretDeletionPolicyNone
				return k8sClient.Update(context.Background(), test.ps)
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				Should(Succeed())

			Eventually(func() []string {
				return getStoreFinalizers(test.store)
			}).
				WithTimeout(time.Second * 10).
				WithPolling(time.Second).
				ShouldNot(ContainElement(secretStoreFinalizer))
		},
			Entry("[namespace] should remove finalizer when DeletionPolicy changes to None", usePushSecret),
			Entry("[cluster] should remove finalizer when DeletionPolicy changes to None", usePushSecret, useClusterStore),
		)
	})

})

func cleanupResources(test *testCase) {
	if test.ps != nil {
		err := k8sClient.Delete(context.Background(), test.ps)
		if err != nil && !apierrors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	}

	err := k8sClient.Delete(context.Background(), test.store)
	if err != nil && !apierrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	Eventually(func() bool {
		err := k8sClient.Get(context.Background(), types.NamespacedName{
			Name:      test.store.GetName(),
			Namespace: test.store.GetNamespace(),
		}, test.store)
		return apierrors.IsNotFound(err)
	}).
		WithTimeout(time.Second * 10).
		WithPolling(time.Second).
		Should(BeTrue())
}

func getStoreFinalizers(store esapi.GenericStore) []string {
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      store.GetName(),
		Namespace: store.GetNamespace(),
	}, store)
	if err != nil {
		return []string{}
	}
	return store.GetFinalizers()
}

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

	if tc.ps != nil {
		tc.ps.Spec.SecretStoreRefs[0].Kind = esapi.ClusterSecretStoreKind
	}
}

func usePushSecret(tc *testCase) {
	tc.ps = &esv1alpha1.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "push-secret",
			Namespace: "default",
		},
		Spec: esv1alpha1.PushSecretSpec{
			DeletionPolicy: esv1alpha1.PushSecretDeletionPolicyDelete,
			Selector: esv1alpha1.PushSecretSelector{
				Secret: &esv1alpha1.PushSecretSecret{
					Name: "foo",
				},
			},
			SecretStoreRefs: []esv1alpha1.PushSecretStoreRef{
				{
					Name: defaultStoreName,
				},
			},
		},
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
