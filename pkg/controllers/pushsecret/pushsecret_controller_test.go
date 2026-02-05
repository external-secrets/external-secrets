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

package pushsecret

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	ctest "github.com/external-secrets/external-secrets/pkg/controllers/commontest"
	"github.com/external-secrets/external-secrets/pkg/controllers/pushsecret/psmetrics"
	"github.com/external-secrets/external-secrets/runtime/testing/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	fakeProvider *fake.Client
	timeout      = time.Second * 10
	interval     = time.Millisecond * 250
)

type testCase struct {
	store           esv1.GenericStore
	managedStore1   esv1.GenericStore
	managedStore2   esv1.GenericStore
	unmanagedStore1 esv1.GenericStore
	unmanagedStore2 esv1.GenericStore
	pushsecret      *v1alpha1.PushSecret
	secret          *v1.Secret
	assert          func(pushsecret *v1alpha1.PushSecret, secret *v1.Secret) bool
}

func init() {
	fakeProvider = fake.New()
	esv1.ForceRegister(fakeProvider, &esv1.SecretStoreProvider{
		Fake: &esv1.FakeProvider{},
	}, esv1.MaintenanceStatusMaintained)
	psmetrics.SetUpMetrics()
}

func checkCondition(status v1alpha1.PushSecretStatus, cond v1alpha1.PushSecretStatusCondition) bool {
	for _, condition := range status.Conditions {
		if condition.Message == cond.Message &&
			condition.Reason == cond.Reason &&
			condition.Status == cond.Status &&
			condition.Type == cond.Type {
			return true
		}
	}
	return false
}

type testTweaks func(*testCase)

var _ = Describe("PushSecret controller", func() {
	const (
		PushSecretName  = "test-ps"
		PushSecretStore = "test-store"
		SecretName      = "test-secret"
	)

	var PushSecretNamespace, OtherNamespace string

	// if we are in debug and need to increase the timeout for testing, we can do so by using an env var
	if customTimeout := os.Getenv("TEST_CUSTOM_TIMEOUT_SEC"); customTimeout != "" {
		if t, err := strconv.Atoi(customTimeout); err == nil {
			timeout = time.Second * time.Duration(t)
		}
	}

	BeforeEach(func() {
		var err error
		PushSecretNamespace, err = ctest.CreateNamespace("test-ns", k8sClient)
		Expect(err).ToNot(HaveOccurred())
		OtherNamespace, err = ctest.CreateNamespace("test-ns", k8sClient)
		Expect(err).ToNot(HaveOccurred())
		fakeProvider.Reset()

		Expect(k8sClient.Create(context.Background(), &genv1alpha1.Fake{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Fake",
				APIVersion: "generators.external-secrets.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: PushSecretNamespace,
			},
			Spec: genv1alpha1.FakeSpec{
				Data: map[string]string{
					"key": "foo-bar-from-generator",
				},
			}})).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		k8sClient.Delete(context.Background(), &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
		})
		// give a time for reconciler to remove finalizers before removing SecretStores
		time.Sleep(2 * time.Second)
		k8sClient.Delete(context.Background(), &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretStore,
				Namespace: PushSecretNamespace,
			},
		})
		k8sClient.Delete(context.Background(), &esv1.ClusterSecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name: PushSecretStore,
			},
		})
		k8sClient.Delete(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretName,
				Namespace: PushSecretNamespace,
			},
		})
		Expect(k8sClient.Delete(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: PushSecretNamespace,
			},
		})).To(Succeed())
	})

	const (
		defaultKey          = "key"
		defaultVal          = "value"
		defaultPath         = "path/to/key"
		otherKey            = "other-key"
		otherVal            = "other-value"
		otherPath           = "path/to/other-key"
		newKey              = "new-key"
		newVal              = "new-value"
		storePrefixTemplate = "SecretStore/%v"
	)

	makeDefaultTestcase := func() *testCase {
		return &testCase{
			pushsecret: &v1alpha1.PushSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PushSecretName,
					Namespace: PushSecretNamespace,
				},
				Spec: v1alpha1.PushSecretSpec{
					SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
						{
							Name: PushSecretStore,
							Kind: "SecretStore",
						},
					},
					Selector: v1alpha1.PushSecretSelector{
						Secret: &v1alpha1.PushSecretSecret{
							Name: SecretName,
						},
					},
					Data: []v1alpha1.PushSecretData{
						{
							Match: v1alpha1.PushSecretMatch{
								SecretKey: defaultKey,
								RemoteRef: v1alpha1.PushSecretRemoteRef{
									RemoteKey: defaultPath,
								},
							},
						},
					},
				},
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SecretName,
					Namespace: PushSecretNamespace,
				},
				Data: map[string][]byte{
					defaultKey: []byte(defaultVal),
				},
			},
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PushSecretStore,
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Fake: &esv1.FakeProvider{
							Data: []esv1.FakeProviderData{},
						},
					},
				},
			},
		}
	}

	// if target Secret name is not specified it should use the ExternalSecret name.
	syncSuccessfully := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if Provider value got updated")
				secretValue := secret.Data[defaultKey]
				setSecretArgs := fakeProvider.GetPushSecretData()
				providerValue, ok := setSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey]
				if !ok {
					return false
				}
				got := providerValue.Value
				return bytes.Equal(got, secretValue)
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	updateIfNotExists := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		fakeProvider.SecretExistsFn = func(_ context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
			setSecretArgs := fakeProvider.GetPushSecretData()
			_, ok := setSecretArgs[ref.GetRemoteKey()]
			return ok, nil
		}
		tc.pushsecret.Spec.UpdatePolicy = v1alpha1.PushSecretUpdatePolicyIfNotExists

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Consistently(func() bool {
				By("updating the secret value")
				tc.secret.Data[defaultKey] = []byte(newVal)
				Expect(k8sClient.Update(context.Background(), secret, &client.UpdateOptions{})).Should(Succeed())

				By("checking if Provider value does not get updated")
				setSecretArgs := fakeProvider.GetPushSecretData()
				providerValue, ok := setSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey]
				if !ok {
					return false
				}
				got := providerValue.Value
				return bytes.Equal(got, []byte(defaultVal))
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	updateIfNotExistsPartialSecrets := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		fakeProvider.SecretExistsFn = func(_ context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
			setSecretArgs := fakeProvider.GetPushSecretData()
			_, ok := setSecretArgs[ref.GetRemoteKey()]
			return ok, nil
		}
		tc.pushsecret.Spec.UpdatePolicy = v1alpha1.PushSecretUpdatePolicyIfNotExists
		tc.pushsecret.Spec.Data = append(tc.pushsecret.Spec.Data, v1alpha1.PushSecretData{
			Match: v1alpha1.PushSecretMatch{
				SecretKey: otherKey,
				RemoteRef: v1alpha1.PushSecretRemoteRef{
					RemoteKey: otherPath,
				},
			},
		})

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				tc.secret.Data[defaultKey] = []byte(newVal) // change initial value in secret
				tc.secret.Data[otherKey] = []byte(otherVal)

				By("checking if only not existing Provider value got updated")
				Expect(k8sClient.Update(context.Background(), secret, &client.UpdateOptions{})).Should(Succeed())
				setSecretArgs := fakeProvider.GetPushSecretData()
				providerValue, ok := setSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey]
				if !ok {
					return false
				}
				got := providerValue.Value
				otherProviderValue, ok := setSecretArgs[ps.Spec.Data[1].Match.RemoteRef.RemoteKey]
				if !ok {
					return false
				}
				gotOther := otherProviderValue.Value
				return bytes.Equal(gotOther, tc.secret.Data[otherKey]) && bytes.Equal(got, []byte(defaultVal))
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	updateIfNotExistsSyncStatus := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		fakeProvider.SecretExistsFn = func(_ context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
			setSecretArgs := fakeProvider.GetPushSecretData()
			_, ok := setSecretArgs[ref.GetRemoteKey()]
			return ok, nil
		}
		tc.pushsecret.Spec.UpdatePolicy = v1alpha1.PushSecretUpdatePolicyIfNotExists
		tc.pushsecret.Spec.Data = append(tc.pushsecret.Spec.Data, v1alpha1.PushSecretData{
			Match: v1alpha1.PushSecretMatch{
				SecretKey: otherKey,
				RemoteRef: v1alpha1.PushSecretRemoteRef{
					RemoteKey: otherPath,
				},
			},
		})
		tc.secret.Data[defaultKey] = []byte(newVal)
		tc.secret.Data[otherKey] = []byte(otherVal)
		updatedPS := &v1alpha1.PushSecret{}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if PushSecret status gets updated correctly with UpdatePolicy=IfNotExists")
				Expect(k8sClient.Update(context.Background(), secret, &client.UpdateOptions{})).Should(Succeed())
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				err := k8sClient.Get(context.Background(), psKey, updatedPS)
				if err != nil {
					return false
				}
				_, ok := updatedPS.Status.SyncedPushSecrets[fmt.Sprintf(storePrefixTemplate, PushSecretStore)][defaultPath]
				if !ok {
					return false
				}
				_, ok = updatedPS.Status.SyncedPushSecrets[fmt.Sprintf(storePrefixTemplate, PushSecretStore)][otherPath]
				if !ok {
					return false
				}
				expected := v1alpha1.PushSecretStatusCondition{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionTrue,
					Reason:  v1alpha1.ReasonSynced,
					Message: "PushSecret synced successfully. Existing secrets in providers unchanged.",
				}
				return checkCondition(ps.Status, expected)
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	updateIfNotExistsSyncFailed := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		fakeProvider.SecretExistsFn = func(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
			return false, errors.New("don't know")
		}
		tc.pushsecret.Spec.UpdatePolicy = v1alpha1.PushSecretUpdatePolicyIfNotExists

		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if sync failed if secret existence cannot be verified in Provider")
				expected := v1alpha1.PushSecretStatusCondition{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  v1alpha1.ReasonErrored,
					Message: "set secret failed: could not verify if secret exists in store: don't know",
				}
				return checkCondition(ps.Status, expected)
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}
	syncSuccessfullyReusingKeys := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.pushsecret = &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
			Spec: v1alpha1.PushSecretSpec{
				SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
					{
						Name: PushSecretStore,
						Kind: "SecretStore",
					},
				},
				Selector: v1alpha1.PushSecretSelector{
					Secret: &v1alpha1.PushSecretSecret{
						Name: SecretName,
					},
				},
				Data: []v1alpha1.PushSecretData{
					{
						Match: v1alpha1.PushSecretMatch{
							SecretKey: "otherKey",
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: defaultPath,
							},
						},
					},
				},
				Template: &esv1.ExternalSecretTemplate{
					Metadata: esv1.ExternalSecretTemplateMetadata{
						Labels: map[string]string{
							"foos": "ball",
						},
						Annotations: map[string]string{
							"hihi": "ga",
						},
						Finalizers: []string{"example.com/finalizer"},
					},
					Type:          v1.SecretTypeOpaque,
					EngineVersion: esv1.TemplateEngineV2,
					Data: map[string]string{
						defaultKey: "{{ .key | toString | upper }} was templated",
						"otherKey": "{{ .key | toString | upper }} was also templated",
					},
				},
			},
		}
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if Provider value got updated")
				setSecretArgs := fakeProvider.GetPushSecretData()
				providerValue, ok := setSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey]
				if !ok {
					return false
				}
				got := providerValue.Value
				return bytes.Equal(got, []byte("VALUE was also templated"))
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}
	// if target Secret name is not specified it should use the ExternalSecret name.
	syncSuccessfullyWithTemplate := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.pushsecret = &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
			Spec: v1alpha1.PushSecretSpec{
				SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
					{
						Name: PushSecretStore,
						Kind: "SecretStore",
					},
				},
				Selector: v1alpha1.PushSecretSelector{
					Secret: &v1alpha1.PushSecretSecret{
						Name: SecretName,
					},
				},
				Data: []v1alpha1.PushSecretData{
					{
						Match: v1alpha1.PushSecretMatch{
							SecretKey: defaultKey,
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: defaultPath,
							},
						},
					},
				},
				Template: &esv1.ExternalSecretTemplate{
					Metadata: esv1.ExternalSecretTemplateMetadata{
						Labels: map[string]string{
							"foos": "ball",
						},
						Annotations: map[string]string{
							"hihi": "ga",
						},
						Finalizers: []string{"example.com/finalizer"},
					},
					Type:          v1.SecretTypeOpaque,
					EngineVersion: esv1.TemplateEngineV2,
					Data: map[string]string{
						defaultKey: "{{ .key | toString | upper }} was templated",
					},
				},
			},
		}
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if Provider value got updated")
				setSecretArgs := fakeProvider.GetPushSecretData()
				providerValue, ok := setSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey]
				if !ok {
					return false
				}
				got := providerValue.Value
				return bytes.Equal(got, []byte("VALUE was templated"))
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	// if target Secret name is not specified it should use the ExternalSecret name.
	syncAndDeleteSuccessfully := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.pushsecret = &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
			Spec: v1alpha1.PushSecretSpec{
				DeletionPolicy: v1alpha1.PushSecretDeletionPolicyDelete,
				SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
					{
						Name: PushSecretStore,
						Kind: "SecretStore",
					},
				},
				Selector: v1alpha1.PushSecretSelector{
					Secret: &v1alpha1.PushSecretSecret{
						Name: SecretName,
					},
				},
				Data: []v1alpha1.PushSecretData{
					{
						Match: v1alpha1.PushSecretMatch{
							SecretKey: defaultKey,
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: defaultPath,
							},
						},
					},
				},
			},
		}
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			ps.Spec.Data[0].Match.RemoteRef.RemoteKey = newKey
			updatedPS := &v1alpha1.PushSecret{}
			Expect(k8sClient.Update(context.Background(), ps, &client.UpdateOptions{})).Should(Succeed())
			Eventually(func() bool {
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				By("checking if Provider value got updated")
				err := k8sClient.Get(context.Background(), psKey, updatedPS)
				if err != nil {
					return false
				}
				key, ok := updatedPS.Status.SyncedPushSecrets[fmt.Sprintf(storePrefixTemplate, PushSecretStore)][newKey]
				if !ok {
					return false
				}
				return key.Match.SecretKey == defaultKey
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	// if PushSecret deletes a secret with properties, the status map should be cleaned up correctly
	syncAndDeleteWithProperties := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		fakeProvider.DeleteSecretFn = func() error {
			return nil
		}
		tc.pushsecret = &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
			Spec: v1alpha1.PushSecretSpec{
				DeletionPolicy: v1alpha1.PushSecretDeletionPolicyDelete,
				SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
					{
						Name: PushSecretStore,
						Kind: "SecretStore",
					},
				},
				Selector: v1alpha1.PushSecretSelector{
					Secret: &v1alpha1.PushSecretSecret{
						Name: SecretName,
					},
				},
				Data: []v1alpha1.PushSecretData{
					{
						Match: v1alpha1.PushSecretMatch{
							SecretKey: defaultKey,
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: defaultPath,
								Property:  "field1",
							},
						},
					},
					{
						Match: v1alpha1.PushSecretMatch{
							SecretKey: otherKey,
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: defaultPath,
								Property:  "field2",
							},
						},
					},
				},
			},
		}
		tc.secret.Data[otherKey] = []byte(otherVal)
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			updatedPS := &v1alpha1.PushSecret{}
			// Wait for initial sync
			Eventually(func() bool {
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				err := k8sClient.Get(context.Background(), psKey, updatedPS)
				if err != nil {
					return false
				}
				// Check both properties are in status
				_, ok1 := updatedPS.Status.SyncedPushSecrets[fmt.Sprintf(storePrefixTemplate, PushSecretStore)][defaultPath+"/field1"]
				_, ok2 := updatedPS.Status.SyncedPushSecrets[fmt.Sprintf(storePrefixTemplate, PushSecretStore)][defaultPath+"/field2"]
				return ok1 && ok2
			}, time.Second*10, time.Second).Should(BeTrue())

			// Remove one property
			updatedPS.Spec.Data = []v1alpha1.PushSecretData{
				{
					Match: v1alpha1.PushSecretMatch{
						SecretKey: defaultKey,
						RemoteRef: v1alpha1.PushSecretRemoteRef{
							RemoteKey: defaultPath,
							Property:  "field1",
						},
					},
				},
			}
			Expect(k8sClient.Update(context.Background(), updatedPS, &client.UpdateOptions{})).Should(Succeed())

			// Verify the removed property is deleted from status
			Eventually(func() bool {
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				err := k8sClient.Get(context.Background(), psKey, updatedPS)
				if err != nil {
					return false
				}
				// field1 should still exist
				_, ok1 := updatedPS.Status.SyncedPushSecrets[fmt.Sprintf(storePrefixTemplate, PushSecretStore)][defaultPath+"/field1"]
				// field2 should be removed
				_, ok2 := updatedPS.Status.SyncedPushSecrets[fmt.Sprintf(storePrefixTemplate, PushSecretStore)][defaultPath+"/field2"]
				return ok1 && !ok2
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	// if PushSecret's DeletionPolicy is cleared, it should delete successfully
	syncChangePolicyAndDeleteSuccessfully := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.pushsecret = &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
			Spec: v1alpha1.PushSecretSpec{
				DeletionPolicy: v1alpha1.PushSecretDeletionPolicyDelete,
				SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
					{
						Name: PushSecretStore,
						Kind: "SecretStore",
					},
				},
				Selector: v1alpha1.PushSecretSelector{
					Secret: &v1alpha1.PushSecretSecret{
						Name: SecretName,
					},
				},
				Data: []v1alpha1.PushSecretData{
					{
						Match: v1alpha1.PushSecretMatch{
							SecretKey: defaultKey,
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: defaultPath,
							},
						},
					},
				},
			},
		}
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			ps.Spec.DeletionPolicy = v1alpha1.PushSecretDeletionPolicyNone
			updatedPS := &v1alpha1.PushSecret{}
			Expect(k8sClient.Update(context.Background(), ps, &client.UpdateOptions{})).Should(Succeed())
			Expect(k8sClient.Delete(context.Background(), ps, &client.DeleteOptions{})).Should(Succeed())
			Eventually(func() bool {
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				By("checking if Get PushSecret returns not found")
				err := k8sClient.Get(context.Background(), psKey, updatedPS)
				if err != nil && client.IgnoreNotFound(err) == nil {
					return true
				}
				return false
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	// When source Secret is deleted and DeletionPolicy=Delete, provider secrets should be cleaned up
	deleteProviderSecretsOnSourceSecretDeleted := func(tc *testCase) {
		var deleteCallCount int32
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		fakeProvider.DeleteSecretFn = func() error {
			atomic.AddInt32(&deleteCallCount, 1)
			return nil
		}
		tc.pushsecret = &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
			Spec: v1alpha1.PushSecretSpec{
				DeletionPolicy: v1alpha1.PushSecretDeletionPolicyDelete,
				// Short refresh interval so reconciler detects deleted secret quickly
				RefreshInterval: &metav1.Duration{Duration: time.Second},
				SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
					{
						Name: PushSecretStore,
						Kind: "SecretStore",
					},
				},
				Selector: v1alpha1.PushSecretSelector{
					Secret: &v1alpha1.PushSecretSecret{
						Name: SecretName,
					},
				},
				Data: []v1alpha1.PushSecretData{
					{
						Match: v1alpha1.PushSecretMatch{
							SecretKey: defaultKey,
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: defaultPath,
							},
						},
					},
				},
			},
		}
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			updatedPS := &v1alpha1.PushSecret{}
			psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}

			// Wait for initial sync
			Eventually(func() bool {
				By("waiting for initial sync to complete")
				err := k8sClient.Get(context.Background(), psKey, updatedPS)
				if err != nil {
					return false
				}
				storeKey := fmt.Sprintf(storePrefixTemplate, PushSecretStore)
				_, ok := updatedPS.Status.SyncedPushSecrets[storeKey][defaultPath]
				return ok
			}, time.Second*10, time.Second).Should(BeTrue())

			// Delete the source Secret
			By("deleting source Secret")
			Expect(k8sClient.Delete(context.Background(), secret, &client.DeleteOptions{})).Should(Succeed())

			// Verify provider secrets are cleaned up
			Eventually(func() bool {
				By("checking if provider secrets were deleted")
				return atomic.LoadInt32(&deleteCallCount) > 0
			}, time.Second*10, time.Second).Should(BeTrue())

			// Verify status shows empty synced secrets
			Eventually(func() bool {
				By("checking if SyncedPushSecrets is empty")
				err := k8sClient.Get(context.Background(), psKey, updatedPS)
				if err != nil {
					return false
				}
				storeKey := fmt.Sprintf(storePrefixTemplate, PushSecretStore)
				secrets, exists := updatedPS.Status.SyncedPushSecrets[storeKey]
				return !exists || len(secrets) == 0
			}, time.Second*10, time.Second).Should(BeTrue())

			return true
		}
	}

	failDelete := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		fakeProvider.DeleteSecretFn = func() error {
			return errors.New("Nope")
		}
		tc.pushsecret = &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
			Spec: v1alpha1.PushSecretSpec{
				DeletionPolicy: v1alpha1.PushSecretDeletionPolicyDelete,
				SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
					{
						Name: PushSecretStore,
						Kind: "SecretStore",
					},
				},
				Selector: v1alpha1.PushSecretSelector{
					Secret: &v1alpha1.PushSecretSecret{
						Name: SecretName,
					},
				},
				Data: []v1alpha1.PushSecretData{
					{
						Match: v1alpha1.PushSecretMatch{
							SecretKey: defaultKey,
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: defaultPath,
							},
						},
					},
				},
			},
		}
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			ps.Spec.Data[0].Match.RemoteRef.RemoteKey = newKey
			updatedPS := &v1alpha1.PushSecret{}
			Expect(k8sClient.Update(context.Background(), ps, &client.UpdateOptions{})).Should(Succeed())
			Eventually(func() bool {
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				By("checking if synced secrets correspond to both keys")
				err := k8sClient.Get(context.Background(), psKey, updatedPS)
				if err != nil {
					return false
				}
				_, ok := updatedPS.Status.SyncedPushSecrets[fmt.Sprintf(storePrefixTemplate, PushSecretStore)][newKey]
				if !ok {
					return false
				}
				_, ok = updatedPS.Status.SyncedPushSecrets[fmt.Sprintf(storePrefixTemplate, PushSecretStore)][defaultPath]
				return ok
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}
	failDeleteStore := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		fakeProvider.DeleteSecretFn = func() error {
			return errors.New("boom")
		}
		tc.pushsecret.Spec.DeletionPolicy = v1alpha1.PushSecretDeletionPolicyDelete
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			secondStore := &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-store",
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Fake: &esv1.FakeProvider{
							Data: []esv1.FakeProviderData{},
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), secondStore, &client.CreateOptions{})).Should(Succeed())
			ps.Spec.SecretStoreRefs[0].Name = "new-store"
			updatedPS := &v1alpha1.PushSecret{}
			Expect(k8sClient.Update(context.Background(), ps, &client.UpdateOptions{})).Should(Succeed())
			Eventually(func() bool {
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				By("checking if Provider value got updated")
				err := k8sClient.Get(context.Background(), psKey, updatedPS)
				if err != nil {
					return false
				}
				syncedLen := len(updatedPS.Status.SyncedPushSecrets)
				return syncedLen == 2
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}
	deleteWholeStore := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		fakeProvider.DeleteSecretFn = func() error {
			return nil
		}
		tc.pushsecret.Spec.DeletionPolicy = v1alpha1.PushSecretDeletionPolicyDelete
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			secondStore := &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-store",
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Fake: &esv1.FakeProvider{
							Data: []esv1.FakeProviderData{},
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), secondStore, &client.CreateOptions{})).Should(Succeed())
			ps.Spec.SecretStoreRefs[0].Name = "new-store"
			updatedPS := &v1alpha1.PushSecret{}
			Expect(k8sClient.Update(context.Background(), ps, &client.UpdateOptions{})).Should(Succeed())
			Eventually(func() bool {
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				By("checking if Provider value got updated")
				err := k8sClient.Get(context.Background(), psKey, updatedPS)
				if err != nil {
					return false
				}
				key, ok := updatedPS.Status.SyncedPushSecrets["SecretStore/new-store"][defaultPath]
				if !ok {
					return false
				}
				syncedLen := len(updatedPS.Status.SyncedPushSecrets)
				if syncedLen != 1 {
					return false
				}
				return key.Match.SecretKey == defaultKey
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}
	// if conversion strategy is defined, revert the keys based on the strategy.
	syncSuccessfullyWithConversionStrategy := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.pushsecret = &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
			Spec: v1alpha1.PushSecretSpec{
				SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
					{
						Name: PushSecretStore,
						Kind: "SecretStore",
					},
				},
				Selector: v1alpha1.PushSecretSelector{
					Secret: &v1alpha1.PushSecretSecret{
						Name: SecretName,
					},
				},
				Data: []v1alpha1.PushSecretData{
					{
						ConversionStrategy: v1alpha1.PushSecretConversionReverseUnicode,
						Match: v1alpha1.PushSecretMatch{
							SecretKey: "some-array[0].entity",
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: defaultPath,
							},
						},
					},
				},
			},
		}
		tc.secret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretName,
				Namespace: PushSecretNamespace,
			},
			Data: map[string][]byte{
				"some-array_U005b_0_U005d_.entity": []byte("value"),
			},
		}
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if Provider value got updated")
				secretValue := secret.Data["some-array_U005b_0_U005d_.entity"]
				setSecretArgs := fakeProvider.GetPushSecretData()
				providerValue, ok := setSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey]
				if !ok {
					return false
				}
				got := providerValue.Value
				return bytes.Equal(got, secretValue)
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}
	// if target Secret name is not specified it should use the ExternalSecret name.
	syncMatchingLabels := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		fakeProvider.DeleteSecretFn = func() error {
			return nil
		}
		tc.pushsecret = &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
			Spec: v1alpha1.PushSecretSpec{
				SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"foo": "bar",
							},
						},
						Kind: "SecretStore",
						Name: PushSecretStore,
					},
				},
				Selector: v1alpha1.PushSecretSelector{
					Secret: &v1alpha1.PushSecretSecret{
						Name: SecretName,
					},
				},
				Data: []v1alpha1.PushSecretData{
					{
						Match: v1alpha1.PushSecretMatch{
							SecretKey: defaultKey,
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: defaultPath,
							},
						},
					},
				},
			},
		}
		tc.store = &esv1.SecretStore{
			TypeMeta: metav1.TypeMeta{
				Kind: "SecretStore",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretStore,
				Namespace: PushSecretNamespace,
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Fake: &esv1.FakeProvider{
						Data: []esv1.FakeProviderData{},
					},
				},
			},
		}
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data[defaultKey]
			setSecretArgs := fakeProvider.GetPushSecretData()
			providerValue := setSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey].Value
			expected := v1alpha1.PushSecretStatusCondition{
				Type:    v1alpha1.PushSecretReady,
				Status:  v1.ConditionTrue,
				Reason:  v1alpha1.ReasonSynced,
				Message: "PushSecret synced successfully",
			}
			return bytes.Equal(secretValue, providerValue) && checkCondition(ps.Status, expected)
		}
	}
	syncWithClusterStore := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.store = &esv1.ClusterSecretStore{
			TypeMeta: metav1.TypeMeta{
				Kind: "ClusterSecretStore",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: PushSecretStore,
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Fake: &esv1.FakeProvider{
						Data: []esv1.FakeProviderData{},
					},
				},
			},
		}
		tc.pushsecret.Spec.SecretStoreRefs[0].Kind = "ClusterSecretStore"
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data[defaultKey]
			setSecretArgs := fakeProvider.GetPushSecretData()
			providerValue := setSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey].Value
			expected := v1alpha1.PushSecretStatusCondition{
				Type:    v1alpha1.PushSecretReady,
				Status:  v1.ConditionTrue,
				Reason:  v1alpha1.ReasonSynced,
				Message: "PushSecret synced successfully",
			}
			return bytes.Equal(secretValue, providerValue) && checkCondition(ps.Status, expected)
		}
	}

	syncWithGenerator := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.pushsecret.Spec.Selector.Secret = nil
		tc.pushsecret.Spec.Selector.GeneratorRef = &esv1.GeneratorRef{
			APIVersion: "generators.external-secrets.io/v1alpha1",
			Kind:       "Fake",
			Name:       "test",
		}
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			setSecretArgs := fakeProvider.GetPushSecretData()
			providerValue := setSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey].Value
			expected := v1alpha1.PushSecretStatusCondition{
				Type:    v1alpha1.PushSecretReady,
				Status:  v1.ConditionTrue,
				Reason:  v1alpha1.ReasonSynced,
				Message: "PushSecret synced successfully",
			}
			return bytes.Equal([]byte("foo-bar-from-generator"), providerValue) && checkCondition(ps.Status, expected)
		}
	}
	// if target Secret name is not specified it should use the ExternalSecret name.
	syncWithClusterStoreMatchingLabels := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.pushsecret = &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
			Spec: v1alpha1.PushSecretSpec{
				SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"foo": "bar",
							},
						},
						Kind: "ClusterSecretStore",
						Name: PushSecretStore,
					},
				},
				Selector: v1alpha1.PushSecretSelector{
					Secret: &v1alpha1.PushSecretSecret{
						Name: SecretName,
					},
				},
				Data: []v1alpha1.PushSecretData{
					{
						Match: v1alpha1.PushSecretMatch{
							SecretKey: defaultKey,
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: defaultPath,
							},
						},
					},
				},
			},
		}
		tc.store = &esv1.ClusterSecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name: PushSecretStore,
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Fake: &esv1.FakeProvider{
						Data: []esv1.FakeProviderData{},
					},
				},
			},
		}
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data[defaultKey]
			setSecretArgs := fakeProvider.GetPushSecretData()
			providerValue := setSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey].Value
			expected := v1alpha1.PushSecretStatusCondition{
				Type:    v1alpha1.PushSecretReady,
				Status:  v1.ConditionTrue,
				Reason:  v1alpha1.ReasonSynced,
				Message: "PushSecret synced successfully",
			}
			return bytes.Equal(secretValue, providerValue) && checkCondition(ps.Status, expected)
		}
	}
	// if target Secret name is not specified it should use the ExternalSecret name.
	failNoSecret := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.secret = nil
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			expected := v1alpha1.PushSecretStatusCondition{
				Type:    v1alpha1.PushSecretReady,
				Status:  v1.ConditionFalse,
				Reason:  v1alpha1.ReasonErrored,
				Message: "could not get source secret",
			}
			return checkCondition(ps.Status, expected)
		}
	}
	// if target Secret name is not specified it should use the ExternalSecret name.
	failNoSecretKey := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.pushsecret.Spec.Data[0].Match.SecretKey = "unexisting"
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			expected := v1alpha1.PushSecretStatusCondition{
				Type:    v1alpha1.PushSecretReady,
				Status:  v1.ConditionFalse,
				Reason:  v1alpha1.ReasonErrored,
				Message: "set secret failed: secret key unexisting does not exist",
			}
			return checkCondition(ps.Status, expected)
		}
	}
	// if target Secret name is not specified it should use the ExternalSecret name.
	failNoSecretStore := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.store = nil
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			expected := v1alpha1.PushSecretStatusCondition{
				Type:    v1alpha1.PushSecretReady,
				Status:  v1.ConditionFalse,
				Reason:  v1alpha1.ReasonErrored,
				Message: "could not get SecretStore \"test-store\", secretstores.external-secrets.io \"test-store\" not found",
			}
			return checkCondition(ps.Status, expected)
		}
	}
	// if target Secret name is not specified it should use the ExternalSecret name.
	failNoClusterStore := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.store = nil
		tc.pushsecret.Spec.SecretStoreRefs[0].Kind = "ClusterSecretStore"
		tc.pushsecret.Spec.SecretStoreRefs[0].Name = "unexisting"
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			expected := v1alpha1.PushSecretStatusCondition{
				Type:    v1alpha1.PushSecretReady,
				Status:  v1.ConditionFalse,
				Reason:  v1alpha1.ReasonErrored,
				Message: "could not get ClusterSecretStore \"unexisting\", clustersecretstores.external-secrets.io \"unexisting\" not found",
			}
			return checkCondition(ps.Status, expected)
		}
	}
	// if target Secret name is not specified it should use the ExternalSecret name.
	setSecretFail := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return errors.New("boom")
		}
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			expected := v1alpha1.PushSecretStatusCondition{
				Type:    v1alpha1.PushSecretReady,
				Status:  v1.ConditionFalse,
				Reason:  v1alpha1.ReasonErrored,
				Message: "set secret failed: could not write remote ref key to target secretstore test-store: boom",
			}
			return checkCondition(ps.Status, expected)
		}
	}
	// if target Secret name is not specified it should use the ExternalSecret name.
	newClientFail := func(tc *testCase) {
		fakeProvider.NewFn = func(_ context.Context, _ esv1.GenericStore, _ client.Client, _ string) (esv1.SecretsClient, error) {
			return nil, errors.New("boom")
		}
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			expected := v1alpha1.PushSecretStatusCondition{
				Type:    v1alpha1.PushSecretReady,
				Status:  v1.ConditionFalse,
				Reason:  v1alpha1.ReasonErrored,
				Message: "set secret failed: could not get secrets client for store test-store: boom",
			}
			return checkCondition(ps.Status, expected)
		}
	}
	// SecretStores in different namespace than PushSecret should not be selected.
	secretStoreDifferentNamespace := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		// Create the SecretStore in a different namespace
		tc.store = &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-ns-store",
				Namespace: OtherNamespace,
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			TypeMeta: metav1.TypeMeta{
				Kind: "SecretStore",
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Fake: &esv1.FakeProvider{
						Data: []esv1.FakeProviderData{},
					},
				},
			},
		}
		// Use label selector to select SecretStores
		tc.pushsecret.Spec.SecretStoreRefs = []v1alpha1.PushSecretStoreRef{
			{
				Kind: "SecretStore",
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"foo": "bar",
					},
				},
			},
		}
		// Should not select the SecretStore in a different namespace
		// (if so, it would fail to find it in the same namespace and be reflected in the status)
		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			// Assert that the status is never updated (no SecretStores found)
			Consistently(func() bool {
				err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(ps), ps)
				if err != nil {
					return false
				}
				return len(ps.Status.Conditions) == 0
			}, timeout, interval).Should(BeTrue())
			return true
		}
	}

	// Secrets in different namespace than PushSecret should not be selected.
	secretDifferentNamespace := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		// Create the Secret in a different namespace
		tc.secret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretName,
				Namespace: OtherNamespace,
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Data: map[string][]byte{
				defaultKey: []byte(defaultVal),
			},
		}
		// Use label selector to select Secrets
		tc.pushsecret.Spec.Selector.Secret = &v1alpha1.PushSecretSecret{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"foo": "bar",
				},
			},
		}

		tc.assert = func(_ *v1alpha1.PushSecret, _ *v1.Secret) bool {
			Eventually(func() bool {
				// We should not be able to reference a secret across namespaces,
				// the map should be empty.
				Expect(fakeProvider.GetPushSecretData()).To(BeEmpty())
				return true
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	// dataTo tests
	syncWithDataToMatchAll := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		// Set up secret with multiple keys
		tc.secret.Data = map[string][]byte{
			"db-host":     []byte("localhost"),
			"db-port":     []byte("5432"),
			"db-username": []byte("admin"),
		}
		// Replace data with dataTo that matches all keys
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				// No match pattern means match all
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if all keys were pushed to provider")
				setSecretArgs := fakeProvider.GetPushSecretData()
				// All three keys should be pushed
				if len(setSecretArgs) != 3 {
					return false
				}
				// Check each key was pushed with same name
				for key, expectedValue := range secret.Data {
					providerValue, ok := setSecretArgs[key]
					if !ok {
						return false
					}
					if !bytes.Equal(providerValue.Value, expectedValue) {
						return false
					}
				}
				return true
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	syncWithDataToRegex := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		// Set up secret with multiple keys
		tc.secret.Data = map[string][]byte{
			"db-host":     []byte("localhost"),
			"db-port":     []byte("5432"),
			"app-name":    []byte("myapp"),
			"app-version": []byte("1.0"),
		}
		// Use dataTo with regex to match only db-* keys
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				Match: &v1alpha1.PushSecretDataToMatch{
					RegExp: "^db-.*",
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if only db-* keys were pushed")
				setSecretArgs := fakeProvider.GetPushSecretData()
				// Only two db-* keys should be pushed
				if len(setSecretArgs) != 2 {
					return false
				}
				// Check db-* keys were pushed
				for key := range setSecretArgs {
					if key != "db-host" && key != "db-port" {
						return false
					}
				}
				return true
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	syncWithDataToRegexpRewrite := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		// Set up secret with multiple keys
		tc.secret.Data = map[string][]byte{
			"db-host": []byte("localhost"),
			"db-port": []byte("5432"),
		}
		// Use dataTo with regex rewrite to add prefix
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				Match: &v1alpha1.PushSecretDataToMatch{
					RegExp: "^db-.*",
				},
				Rewrite: []v1alpha1.PushSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
							Source: "^db-",
							Target: "app/database/",
						},
					},
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if keys were rewritten with prefix")
				setSecretArgs := fakeProvider.GetPushSecretData()
				if len(setSecretArgs) != 2 {
					return false
				}
				// Check keys were rewritten
				_, hasHost := setSecretArgs["app/database/host"]
				_, hasPort := setSecretArgs["app/database/port"]
				return hasHost && hasPort
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	syncWithDataToTransformRewrite := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.secret.Data = map[string][]byte{
			"username": []byte("admin"),
		}
		// Use dataTo with template transformation
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				Rewrite: []v1alpha1.PushSecretRewrite{
					{
						Transform: &esv1.ExternalSecretRewriteTransform{
							Template: "app/{{ .value | upper }}",
						},
					},
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if key was transformed using template")
				setSecretArgs := fakeProvider.GetPushSecretData()
				if len(setSecretArgs) != 1 {
					return false
				}
				// Check key was transformed to uppercase with prefix
				providerValue, ok := setSecretArgs["app/USERNAME"]
				if !ok {
					return false
				}
				return bytes.Equal(providerValue.Value, []byte("admin"))
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	syncDataToWithDataOverride := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.secret.Data = map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		}
		// Use both dataTo and explicit data
		// Explicit data should override dataTo for key1
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				// Match all keys, no rewrite
			},
		}
		tc.pushsecret.Spec.Data = []v1alpha1.PushSecretData{
			{
				Match: v1alpha1.PushSecretMatch{
					SecretKey: "key1",
					RemoteRef: v1alpha1.PushSecretRemoteRef{
						RemoteKey: "override-key1", // Different remote key
					},
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if explicit data overrode dataTo")
				setSecretArgs := fakeProvider.GetPushSecretData()
				// Should have 2 keys: override-key1 and key2
				if len(setSecretArgs) != 2 {
					return false
				}
				// key1 should be pushed as override-key1 (from explicit data)
				_, hasOverride := setSecretArgs["override-key1"]
				// key2 should be pushed as key2 (from dataTo)
				_, hasKey2 := setSecretArgs["key2"]
				return hasOverride && hasKey2
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	failDataToInvalidRegex := func(tc *testCase) {
		tc.secret.Data = map[string][]byte{
			"key1": []byte("value1"),
		}
		// Use invalid regex pattern
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				Match: &v1alpha1.PushSecretDataToMatch{
					RegExp: "[invalid(regex",
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if PushSecret has error condition")
				cond := GetPushSecretCondition(ps.Status.Conditions, v1alpha1.PushSecretReady)
				if cond == nil {
					return false
				}
				// Should have error status
				return cond.Status == v1.ConditionFalse && cond.Reason == v1alpha1.ReasonErrored
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	syncWithDataToConversionStrategy := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		// Set up secret with unicode data
		tc.secret.Data = map[string][]byte{
			"unicode-key": []byte("unicode-value-αβγ"),
			"normal-key":  []byte("normal-value"),
		}
		// Use dataTo with ConversionStrategy
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				ConversionStrategy: v1alpha1.PushSecretConversionReverseUnicode,
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if all keys were pushed with unicode conversion")
				setSecretArgs := fakeProvider.GetPushSecretData()
				// Both keys should be pushed
				if len(setSecretArgs) != 2 {
					return false
				}
				// Verify keys exist (actual unicode encoding is tested in provider tests)
				_, hasUnicode := setSecretArgs["unicode-key"]
				_, hasNormal := setSecretArgs["normal-key"]
				return hasUnicode && hasNormal
			}, timeout, time.Second).Should(BeTrue())

			cond := GetPushSecretCondition(ps.Status.Conditions, v1alpha1.PushSecretReady)
			return cond != nil && cond.Status == v1.ConditionTrue && cond.Reason == v1alpha1.ReasonSynced
		}
	}

	// Test dataTo with storeRef targeting specific store
	syncWithDataToStoreRef := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		// Create a second store
		secondStoreName := "second-store"
		secondStore := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secondStoreName,
				Namespace: PushSecretNamespace,
			},
			TypeMeta: metav1.TypeMeta{
				Kind: "SecretStore",
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Fake: &esv1.FakeProvider{
						Data: []esv1.FakeProviderData{},
					},
				},
			},
		}

		tc.secret.Data = map[string][]byte{
			"db-host":  []byte("localhost"),
			"api-key":  []byte("secret123"),
			"app-name": []byte("myapp"),
		}

		// Configure multiple stores
		tc.pushsecret.Spec.SecretStoreRefs = []v1alpha1.PushSecretStoreRef{
			{Name: PushSecretStore, Kind: "SecretStore"},
			{Name: secondStoreName, Kind: "SecretStore"},
		}
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				// Entry targeting first store only
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
					Kind: "SecretStore",
				},
				Match: &v1alpha1.PushSecretDataToMatch{
					RegExp: "^db-.*",
				},
			},
			{
				// Entry targeting second store only
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: secondStoreName,
					Kind: "SecretStore",
				},
				Match: &v1alpha1.PushSecretDataToMatch{
					RegExp: "^api-.*",
				},
			},
			{
				// Entry targeting first store (app-name)
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
					Kind: "SecretStore",
				},
				Match: &v1alpha1.PushSecretDataToMatch{
					RegExp: "^app-.*",
				},
			},
			{
				// Entry targeting second store (app-name)
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: secondStoreName,
					Kind: "SecretStore",
				},
				Match: &v1alpha1.PushSecretDataToMatch{
					RegExp: "^app-.*",
				},
			},
		}

		// Second store is created by test harness via tc.managedStore2 so it exists before PushSecret
		tc.managedStore2 = secondStore

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			updatedPS := &v1alpha1.PushSecret{}
			psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}

			Eventually(func() bool {
				By("checking if secrets are synced to correct stores")
				err := k8sClient.Get(context.Background(), psKey, updatedPS)
				if err != nil {
					return false
				}

				firstStoreKey := fmt.Sprintf(storePrefixTemplate, PushSecretStore)
				secondStoreKey := "SecretStore/" + secondStoreName

				firstStoreSynced := updatedPS.Status.SyncedPushSecrets[firstStoreKey]
				secondStoreSynced := updatedPS.Status.SyncedPushSecrets[secondStoreKey]

				// First store should have: db-host and app-name (both from storeRef entries)
				_, hasDbHost := firstStoreSynced["db-host"]
				_, hasAppNameFirst := firstStoreSynced["app-name"]
				// First store should NOT have api-key (targeted to second store)
				_, hasApiKeyFirst := firstStoreSynced["api-key"]

				// Second store should have: api-key and app-name (both from storeRef entries)
				_, hasApiKey := secondStoreSynced["api-key"]
				_, hasAppNameSecond := secondStoreSynced["app-name"]
				// Second store should NOT have db-host (targeted to first store)
				_, hasDbHostSecond := secondStoreSynced["db-host"]

				return hasDbHost && hasAppNameFirst && !hasApiKeyFirst &&
					hasApiKey && hasAppNameSecond && !hasDbHostSecond
			}, time.Second*10, time.Second).Should(BeTrue())

			return true
		}
	}

	// Test: Template creates new keys, dataTo matches them
	templateCreatesKeysThenDataToMatches := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		// Source secret has individual components
		tc.secret.Data = map[string][]byte{
			"db_host": []byte("localhost"),
			"db_port": []byte("3306"),
		}
		// Template creates connection string from components
		tc.pushsecret.Spec.Template = &esv1.ExternalSecretTemplate{
			Data: map[string]string{
				"mysql-connection": "mysql://{{ .db_host }}:{{ .db_port }}/mydb",
			},
		}
		// dataTo only matches keys ending in -connection (created by template)
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				Match: &v1alpha1.PushSecretDataToMatch{
					RegExp: ".*-connection$",
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking template key was pushed, not originals")
				setSecretArgs := fakeProvider.GetPushSecretData()
				// Only mysql-connection should be pushed
				_, hasConnection := setSecretArgs["mysql-connection"]
				_, hasDbHost := setSecretArgs["db_host"]
				_, hasDbPort := setSecretArgs["db_port"]
				return hasConnection && !hasDbHost && !hasDbPort
			}, timeout, time.Second).Should(BeTrue())

			cond := GetPushSecretCondition(ps.Status.Conditions, v1alpha1.PushSecretReady)
			return cond != nil && cond.Status == v1.ConditionTrue && cond.Reason == v1alpha1.ReasonSynced
		}
	}

	// Test: Template + dataTo + explicit data combined
	templateWithDataToAndExplicitData := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.secret.Data = map[string][]byte{
			"token":          []byte("abc123"),
			"config-timeout": []byte("30s"),
			"config-retries": []byte("3"),
		}
		// Template creates api-key from token
		tc.pushsecret.Spec.Template = &esv1.ExternalSecretTemplate{
			Data: map[string]string{
				"api-key": "Bearer {{ .token }}",
			},
		}
		// dataTo matches config-* keys
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				Match: &v1alpha1.PushSecretDataToMatch{
					RegExp: "^config-.*",
				},
			},
		}
		// Explicit data for api-key with custom remote path
		tc.pushsecret.Spec.Data = []v1alpha1.PushSecretData{
			{
				Match: v1alpha1.PushSecretMatch{
					SecretKey: "api-key",
					RemoteRef: v1alpha1.PushSecretRemoteRef{
						RemoteKey: "credentials/api-key",
					},
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking all expected keys were pushed")
				setSecretArgs := fakeProvider.GetPushSecretData()
				// api-key should be at credentials/api-key (explicit), config-* from dataTo
				_, hasApiKey := setSecretArgs["credentials/api-key"]
				_, hasTimeout := setSecretArgs["config-timeout"]
				_, hasRetries := setSecretArgs["config-retries"]
				// Original token should NOT be pushed
				_, hasToken := setSecretArgs["token"]
				return hasApiKey && hasTimeout && hasRetries && !hasToken
			}, timeout, time.Second).Should(BeTrue())

			cond := GetPushSecretCondition(ps.Status.Conditions, v1alpha1.PushSecretReady)
			return cond != nil && cond.Status == v1.ConditionTrue && cond.Reason == v1alpha1.ReasonSynced
		}
	}

	failDataToDuplicateAcrossEntries := func(tc *testCase) {
		tc.secret.Data = map[string][]byte{
			"db-host": []byte("localhost"),
			"db-port": []byte("5432"),
		}
		// Create two dataTo entries that both produce the same remote key "app/config"
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				Match: &v1alpha1.PushSecretDataToMatch{
					RegExp: "^db-host$",
				},
				Rewrite: []v1alpha1.PushSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
							Source: ".*",
							Target: "app/config",
						},
					},
				},
			},
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				Match: &v1alpha1.PushSecretDataToMatch{
					RegExp: "^db-port$",
				},
				Rewrite: []v1alpha1.PushSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
							Source: ".*",
							Target: "app/config",
						},
					},
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if PushSecret has error condition for duplicate remote keys")
				cond := GetPushSecretCondition(ps.Status.Conditions, v1alpha1.PushSecretReady)
				if cond == nil {
					return false
				}
				// Should have error status
				return cond.Status == v1.ConditionFalse && cond.Reason == v1alpha1.ReasonErrored
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	failDataToAndDataDuplicateRemoteKey := func(tc *testCase) {
		tc.secret.Data = map[string][]byte{
			"db-host": []byte("localhost"),
			"api-key": []byte("secret123"),
		}
		// Create dataTo entry and explicit data that map to the same remote key
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				Match: &v1alpha1.PushSecretDataToMatch{
					RegExp: "^db-host$",
				},
				Rewrite: []v1alpha1.PushSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
							Source: ".*",
							Target: "myapp/config",
						},
					},
				},
			},
		}
		tc.pushsecret.Spec.Data = []v1alpha1.PushSecretData{
			{
				Match: v1alpha1.PushSecretMatch{
					SecretKey: "api-key",
					RemoteRef: v1alpha1.PushSecretRemoteRef{
						RemoteKey: "myapp/config", // Same remote key as dataTo produces
					},
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if PushSecret has error condition for duplicate remote keys")
				cond := GetPushSecretCondition(ps.Status.Conditions, v1alpha1.PushSecretReady)
				if cond == nil {
					return false
				}
				// Should have error status
				return cond.Status == v1.ConditionFalse && cond.Reason == v1alpha1.ReasonErrored
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	// Note: failDataToMissingStoreRef test removed - missing storeRef is now
	// blocked by CEL validation at admission time

	failDataToStoreRefNotInList := func(tc *testCase) {
		tc.secret.Data = map[string][]byte{
			"key1": []byte("value1"),
		}
		// dataTo with storeRef that doesn't exist in secretStoreRefs
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: "non-existent-store", // Not in secretStoreRefs
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking PushSecret has error for invalid storeRef")
				cond := GetPushSecretCondition(ps.Status.Conditions, v1alpha1.PushSecretReady)
				if cond == nil {
					return false
				}
				return cond.Status == v1.ConditionFalse && cond.Reason == v1alpha1.ReasonErrored
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	syncWithDataToLabelSelector := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		// Add labels to the store
		tc.store = &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretStore,
				Namespace: PushSecretNamespace,
				Labels: map[string]string{
					"env": "test",
				},
			},
			TypeMeta: metav1.TypeMeta{
				Kind: "SecretStore",
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Fake: &esv1.FakeProvider{
						Data: []esv1.FakeProviderData{},
					},
				},
			},
		}
		tc.secret.Data = map[string][]byte{
			"key1": []byte("value1"),
		}
		// Use labelSelector in secretStoreRefs to select stores by labels
		tc.pushsecret.Spec.SecretStoreRefs = []v1alpha1.PushSecretStoreRef{
			{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"env": "test",
					},
				},
			},
		}
		// Use labelSelector in dataTo to target stores with matching labels
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"env": "test",
						},
					},
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking key was pushed via labelSelector")
				setSecretArgs := fakeProvider.GetPushSecretData()
				return len(setSecretArgs) == 1
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	syncWithDataToDuplicateValues := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		// Keys with same value - tests deterministic key mapping
		tc.secret.Data = map[string][]byte{
			"db-host":    []byte("same-value"),
			"cache-host": []byte("same-value"),
		}
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				Rewrite: []v1alpha1.PushSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
							Source: "-host$",
							Target: "/endpoint",
						},
					},
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking both keys are rewritten despite same value")
				setSecretArgs := fakeProvider.GetPushSecretData()
				if len(setSecretArgs) != 2 {
					return false
				}
				_, hasDb := setSecretArgs["db/endpoint"]
				_, hasCache := setSecretArgs["cache/endpoint"]
				return hasDb && hasCache
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	syncWithDataToMultipleRewrites := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.secret.Data = map[string][]byte{
			"db-username": []byte("admin"),
		}
		// Chain multiple rewrites
		tc.pushsecret.Spec.Data = nil
		tc.pushsecret.Spec.DataTo = []v1alpha1.PushSecretDataTo{
			{
				StoreRef: &v1alpha1.PushSecretStoreRef{
					Name: PushSecretStore,
				},
				Rewrite: []v1alpha1.PushSecretRewrite{
					{
						// First rewrite: remove db- prefix
						Regexp: &esv1.ExternalSecretRewriteRegexp{
							Source: "^db-",
							Target: "",
						},
					},
					{
						// Second rewrite: add app/ prefix
						Regexp: &esv1.ExternalSecretRewriteRegexp{
							Source: "^",
							Target: "app/",
						},
					},
				},
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if multiple rewrites were applied")
				setSecretArgs := fakeProvider.GetPushSecretData()
				if len(setSecretArgs) != 1 {
					return false
				}
				// db-username -> username -> app/username
				_, ok := setSecretArgs["app/username"]
				return ok
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	DescribeTable("When reconciling a PushSecret",
		func(tweaks ...testTweaks) {
			tc := makeDefaultTestcase()
			for _, tweak := range tweaks {
				tweak(tc)
			}
			ctx := context.Background()
			By("creating a secret store, secret and pushsecret")
			if tc.store != nil {
				Expect(k8sClient.Create(ctx, tc.store)).To(Succeed())
			}
			if tc.managedStore2 != nil {
				Expect(k8sClient.Create(ctx, tc.managedStore2)).To(Succeed())
			}
			if tc.secret != nil {
				Expect(k8sClient.Create(ctx, tc.secret)).To(Succeed())
			}
			if tc.pushsecret != nil {
				Expect(k8sClient.Create(ctx, tc.pushsecret)).Should(Succeed())
			}
			time.Sleep(2 * time.Second) // prevents race conditions during tests causing failures
			psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
			createdPS := &v1alpha1.PushSecret{}
			By("checking the pushSecret condition")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, psKey, createdPS)
				if err != nil {
					return false
				}
				return tc.assert(createdPS, tc.secret)
			}, timeout, interval).Should(BeTrue())
			// this must be optional so we can test faulty es configuration
		},
		Entry("should sync", syncSuccessfully),
		Entry("should not update existing secret if UpdatePolicy=IfNotExists", updateIfNotExists),
		Entry("should only update parts of secret that don't already exist if UpdatePolicy=IfNotExists", updateIfNotExistsPartialSecrets),
		Entry("should update the PushSecret status correctly if UpdatePolicy=IfNotExists", updateIfNotExistsSyncStatus),
		Entry("should fail if secret existence cannot be verified if UpdatePolicy=IfNotExists", updateIfNotExistsSyncFailed),
		Entry("should sync with template", syncSuccessfullyWithTemplate),
		Entry("should sync with template reusing keys", syncSuccessfullyReusingKeys),
		Entry("should sync with conversion strategy", syncSuccessfullyWithConversionStrategy),
		Entry("should delete if DeletionPolicy=Delete", syncAndDeleteSuccessfully),
		Entry("should delete secrets with properties and update status correctly", syncAndDeleteWithProperties),
		Entry("should delete after DeletionPolicy changed from Delete to None", syncChangePolicyAndDeleteSuccessfully),
		Entry("should cleanup provider secrets when source Secret is deleted", deleteProviderSecretsOnSourceSecretDeleted),
		Entry("should track deletion tasks if Delete fails", failDelete),
		Entry("should track deleted stores if Delete fails", failDeleteStore),
		Entry("should delete all secrets if SecretStore changes", deleteWholeStore),
		Entry("should sync to stores matching labels", syncMatchingLabels),
		Entry("should sync with ClusterStore", syncWithClusterStore),
		Entry("should sync with ClusterStore matching labels", syncWithClusterStoreMatchingLabels),
		Entry("should sync with Generator", syncWithGenerator),
		Entry("should fail if Secret is not created", failNoSecret),
		Entry("should fail if Secret Key does not exist", failNoSecretKey),
		Entry("should fail if SetSecret fails", setSecretFail),
		Entry("should fail if no valid SecretStore", failNoSecretStore),
		Entry("should fail if no valid ClusterSecretStore", failNoClusterStore),
		Entry("should fail if NewClient fails", newClientFail),
		Entry("should not sync to SecretStore in different namespace", secretStoreDifferentNamespace),
		Entry("should not reference secret in different namespace", secretDifferentNamespace),
		Entry("should sync with dataTo matching all keys", syncWithDataToMatchAll),
		Entry("should sync with dataTo using regex pattern", syncWithDataToRegex),
		Entry("should sync with dataTo and regexp rewrite", syncWithDataToRegexpRewrite),
		Entry("should sync with dataTo and transform rewrite", syncWithDataToTransformRewrite),
		Entry("should override dataTo with explicit data", syncDataToWithDataOverride),
		Entry("should sync with dataTo and multiple chained rewrites", syncWithDataToMultipleRewrites),
		Entry("should fail with invalid regex in dataTo", failDataToInvalidRegex),
		Entry("should sync with dataTo and conversion strategy", syncWithDataToConversionStrategy),
		Entry("should sync with dataTo storeRef targeting specific stores", syncWithDataToStoreRef),
		Entry("should match dataTo against template-created keys", templateCreatesKeysThenDataToMatches),
		Entry("should combine template, dataTo and explicit data", templateWithDataToAndExplicitData),
		Entry("should fail with duplicate remote keys across dataTo entries", failDataToDuplicateAcrossEntries),
		Entry("should fail with duplicate remote keys between dataTo and explicit data", failDataToAndDataDuplicateRemoteKey),
		Entry("should fail with dataTo storeRef not in secretStoreRefs", failDataToStoreRefNotInList),
		Entry("should sync with dataTo using labelSelector", syncWithDataToLabelSelector),
		Entry("should sync with dataTo when keys have duplicate values", syncWithDataToDuplicateValues),
	)
})

var _ = Describe("PushSecret Controller Un/Managed Stores", func() {
	const (
		PushSecretName            = "test-ps"
		ManagedPushSecretStore1   = "test-managed-store-1"
		ManagedPushSecretStore2   = "test-managed-store-2"
		UnmanagedPushSecretStore1 = "test-unmanaged-store-1"
		UnmanagedPushSecretStore2 = "test-unmanaged-store-2"
		SecretName                = "test-secret"
	)

	var PushSecretNamespace string
	PushSecretStores := []string{ManagedPushSecretStore1, ManagedPushSecretStore2, UnmanagedPushSecretStore1, UnmanagedPushSecretStore2}

	// if we are in debug and need to increase the timeout for testing, we can do so by using an env var
	if customTimeout := os.Getenv("TEST_CUSTOM_TIMEOUT_SEC"); customTimeout != "" {
		if t, err := strconv.Atoi(customTimeout); err == nil {
			timeout = time.Second * time.Duration(t)
		}
	}

	BeforeEach(func() {
		var err error
		PushSecretNamespace, err = ctest.CreateNamespace("test-ns", k8sClient)
		Expect(err).ToNot(HaveOccurred())
		fakeProvider.Reset()
	})

	AfterEach(func() {
		k8sClient.Delete(context.Background(), &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
		})
		// give a time for reconciler to remove finalizers before removing SecretStores
		time.Sleep(2 * time.Second)
		for _, psstore := range PushSecretStores {
			k8sClient.Delete(context.Background(), &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      psstore,
					Namespace: PushSecretNamespace,
				},
			})
			k8sClient.Delete(context.Background(), &esv1.ClusterSecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: psstore,
				},
			})
		}
		k8sClient.Delete(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretName,
				Namespace: PushSecretNamespace,
			},
		})
		Expect(k8sClient.Delete(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: PushSecretNamespace,
			},
		})).To(Succeed())
	})

	const (
		defaultKey  = "key"
		defaultVal  = "value"
		defaultPath = "path/to/key"
	)

	makeDefaultTestcase := func() *testCase {
		return &testCase{
			pushsecret: &v1alpha1.PushSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PushSecretName,
					Namespace: PushSecretNamespace,
				},
				Spec: v1alpha1.PushSecretSpec{
					SecretStoreRefs: []v1alpha1.PushSecretStoreRef{
						{
							Name: ManagedPushSecretStore1,
							Kind: "SecretStore",
						},
					},
					Selector: v1alpha1.PushSecretSelector{
						Secret: &v1alpha1.PushSecretSecret{
							Name: SecretName,
						},
					},
					Data: []v1alpha1.PushSecretData{
						{
							Match: v1alpha1.PushSecretMatch{
								SecretKey: defaultKey,
								RemoteRef: v1alpha1.PushSecretRemoteRef{
									RemoteKey: defaultPath,
								},
							},
						},
					},
				},
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SecretName,
					Namespace: PushSecretNamespace,
				},
				Data: map[string][]byte{
					defaultKey: []byte(defaultVal),
				},
			},
			managedStore1: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ManagedPushSecretStore1,
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Fake: &esv1.FakeProvider{
							Data: []esv1.FakeProviderData{},
						},
					},
				},
			},
			managedStore2: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ManagedPushSecretStore2,
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Fake: &esv1.FakeProvider{
							Data: []esv1.FakeProviderData{},
						},
					},
				},
			},
			unmanagedStore1: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      UnmanagedPushSecretStore1,
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Fake: &esv1.FakeProvider{
							Data: []esv1.FakeProviderData{},
						},
					},
					Controller: "not-managed",
				},
			},
			unmanagedStore2: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      UnmanagedPushSecretStore2,
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Fake: &esv1.FakeProvider{
							Data: []esv1.FakeProviderData{},
						},
					},
					Controller: "not-managed",
				},
			},
		}
	}

	multipleManagedStoresSyncsSuccessfully := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}

		tc.pushsecret.Spec.SecretStoreRefs = append(tc.pushsecret.Spec.SecretStoreRefs,
			v1alpha1.PushSecretStoreRef{
				Name: ManagedPushSecretStore2,
				Kind: "SecretStore",
			},
		)

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if Provider value got updated")
				secretValue := secret.Data[defaultKey]
				setSecretArgs := fakeProvider.GetPushSecretData()
				providerValue, ok := setSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey]
				if !ok {
					return false
				}
				got := providerValue.Value
				return bytes.Equal(got, secretValue)
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	skipUnmanagedStores := func(tc *testCase) {
		tc.pushsecret.Spec.SecretStoreRefs = []v1alpha1.PushSecretStoreRef{
			{
				Name: UnmanagedPushSecretStore1,
				Kind: "SecretStore",
			},
			{
				Name: UnmanagedPushSecretStore2,
				Kind: "SecretStore",
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, _ *v1.Secret) bool {
			return len(ps.Status.Conditions) == 0
		}
	}

	warnUnmanagedStoresAndSyncManagedStores := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}

		tc.pushsecret.Spec.SecretStoreRefs = []v1alpha1.PushSecretStoreRef{
			{
				Name: ManagedPushSecretStore1,
				Kind: "SecretStore",
			},
			{
				Name: ManagedPushSecretStore2,
				Kind: "SecretStore",
			},
			{
				Name: UnmanagedPushSecretStore1,
				Kind: "SecretStore",
			},
			{
				Name: UnmanagedPushSecretStore2,
				Kind: "SecretStore",
			},
		}

		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			Eventually(func() bool {
				By("checking if Provider value got updated")
				secretValue := secret.Data[defaultKey]
				setSecretArgs := fakeProvider.GetPushSecretData()
				providerValue, ok := setSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey]
				if !ok {
					return false
				}
				got := providerValue.Value
				return bytes.Equal(got, secretValue)
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	DescribeTable("When reconciling a PushSecret with multiple secret stores",
		func(tweaks ...testTweaks) {
			tc := makeDefaultTestcase()
			for _, tweak := range tweaks {
				tweak(tc)
			}
			ctx := context.Background()
			By("creating secret stores, a secret and a pushsecret")
			if tc.managedStore1 != nil {
				Expect(k8sClient.Create(ctx, tc.managedStore1)).To(Succeed())
			}
			if tc.managedStore2 != nil {
				Expect(k8sClient.Create(ctx, tc.managedStore2)).To(Succeed())
			}
			if tc.unmanagedStore1 != nil {
				Expect(k8sClient.Create(ctx, tc.unmanagedStore1)).To(Succeed())
			}
			if tc.unmanagedStore2 != nil {
				Expect(k8sClient.Create(ctx, tc.unmanagedStore2)).To(Succeed())
			}
			if tc.secret != nil {
				Expect(k8sClient.Create(ctx, tc.secret)).To(Succeed())
			}
			if tc.pushsecret != nil {
				Expect(k8sClient.Create(ctx, tc.pushsecret)).Should(Succeed())
			}
			time.Sleep(2 * time.Second) // prevents race conditions during tests causing failures
			psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
			createdPS := &v1alpha1.PushSecret{}
			By("checking the pushSecret condition")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, psKey, createdPS)
				if err != nil {
					return false
				}
				return tc.assert(createdPS, tc.secret)
			}, timeout, interval).Should(BeTrue())
			// this must be optional so we can test faulty es configuration
		},
		Entry("should sync successfully if there are multiple managed stores", multipleManagedStoresSyncsSuccessfully),
		Entry("should skip unmanaged stores", skipUnmanagedStores),
		Entry("should skip unmanaged stores and sync managed stores", warnUnmanagedStoresAndSyncManagedStores),
	)
})

var _ = Describe("mergeDataEntries unit tests", func() {
	Describe("resolveSourceKeyConflicts", func() {
		It("should let explicit data override dataTo for same source key", func() {
			dataTo := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "foo", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "dataTo/foo"}}},
				{Match: v1alpha1.PushSecretMatch{SecretKey: "bar", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "dataTo/bar"}}},
			}
			explicit := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "foo", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "explicit/foo"}}},
			}

			result := resolveSourceKeyConflicts(dataTo, explicit)

			Expect(result).To(HaveLen(2))
			// bar from dataTo, foo from explicit
			keys := make(map[string]string)
			for _, d := range result {
				keys[d.GetSecretKey()] = d.GetRemoteKey()
			}
			Expect(keys["bar"]).To(Equal("dataTo/bar"))
			Expect(keys["foo"]).To(Equal("explicit/foo"))
		})

		It("should keep all entries when no conflicts", func() {
			dataTo := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "a", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "a"}}},
			}
			explicit := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "b", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "b"}}},
			}

			result := resolveSourceKeyConflicts(dataTo, explicit)

			Expect(result).To(HaveLen(2))
		})

		It("should handle empty dataTo", func() {
			explicit := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "x", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "x"}}},
			}

			result := resolveSourceKeyConflicts(nil, explicit)

			Expect(result).To(HaveLen(1))
			Expect(result[0].GetSecretKey()).To(Equal("x"))
		})

		It("should handle empty explicit", func() {
			dataTo := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "y", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "y"}}},
			}

			result := resolveSourceKeyConflicts(dataTo, nil)

			Expect(result).To(HaveLen(1))
			Expect(result[0].GetSecretKey()).To(Equal("y"))
		})
	})

	Describe("validateRemoteKeyUniqueness", func() {
		It("should pass for unique remote keys", func() {
			entries := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "a", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "remote-a"}}},
				{Match: v1alpha1.PushSecretMatch{SecretKey: "b", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "remote-b"}}},
			}

			err := validateRemoteKeyUniqueness(entries)

			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail for duplicate remote keys", func() {
			entries := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "a", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "shared"}}},
				{Match: v1alpha1.PushSecretMatch{SecretKey: "b", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "shared"}}},
			}

			err := validateRemoteKeyUniqueness(entries)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate remote key"))
		})

		It("should pass for same remote key with different properties", func() {
			entries := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "a", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "shared", Property: "field1"}}},
				{Match: v1alpha1.PushSecretMatch{SecretKey: "b", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "shared", Property: "field2"}}},
			}

			err := validateRemoteKeyUniqueness(entries)

			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail for same remote key and same property", func() {
			entries := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "a", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "shared", Property: "field"}}},
				{Match: v1alpha1.PushSecretMatch{SecretKey: "b", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "shared", Property: "field"}}},
			}

			err := validateRemoteKeyUniqueness(entries)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate remote key"))
		})
	})

	Describe("mergeDataEntries", func() {
		It("should merge valid entries", func() {
			dataTo := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "a", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "a"}}},
			}
			explicit := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "b", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "b"}}},
			}

			result, err := mergeDataEntries(dataTo, explicit)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))
		})

		It("should override dataTo with explicit for same source key", func() {
			dataTo := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "key", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "dataTo-path"}}},
			}
			explicit := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "key", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "explicit-path"}}},
			}

			result, err := mergeDataEntries(dataTo, explicit)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].GetRemoteKey()).To(Equal("explicit-path"))
		})

		It("should fail for remote key conflict after merge", func() {
			dataTo := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "a", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "shared"}}},
			}
			explicit := []v1alpha1.PushSecretData{
				{Match: v1alpha1.PushSecretMatch{SecretKey: "b", RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: "shared"}}},
			}

			_, err := mergeDataEntries(dataTo, explicit)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate remote key"))
		})
	})
})
