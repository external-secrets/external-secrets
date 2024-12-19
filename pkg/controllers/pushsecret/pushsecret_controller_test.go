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

package pushsecret

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	ctest "github.com/external-secrets/external-secrets/pkg/controllers/commontest"
	"github.com/external-secrets/external-secrets/pkg/controllers/pushsecret/psmetrics"
	"github.com/external-secrets/external-secrets/pkg/provider/testing/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	fakeProvider *fake.Client
	timeout      = time.Second * 10
	interval     = time.Millisecond * 250
)

type testCase struct {
	store           v1beta1.GenericStore
	managedStore1   v1beta1.GenericStore
	managedStore2   v1beta1.GenericStore
	unmanagedStore1 v1beta1.GenericStore
	unmanagedStore2 v1beta1.GenericStore
	pushsecret      *v1alpha1.PushSecret
	secret          *v1.Secret
	assert          func(g Gomega, pushsecret *v1alpha1.PushSecret, secret *v1.Secret) bool
}

func init() {
	fakeProvider = fake.New()
	v1beta1.ForceRegister(fakeProvider, &v1beta1.SecretStoreProvider{
		Fake: &v1beta1.FakeProvider{},
	})
	psmetrics.SetUpMetrics()
}

type testTweaks func(*testCase)

// NOTE: because the tests mutate `fakeProvider`, we can't run these blocks in parallel.
// therefore, we run them using the `Serial` Ginkgo decorator to avoid parallelism.
var _ = Describe("PushSecret controller", Serial, func() {
	Context("Single Store", testPushSecretSingleStore)
	Context("Multiple Un/Managed Stores", testPushSecretMultipleStores)
})

func testPushSecretSingleStore() {
	const (
		PushSecretName  = "test-ps"
		PushSecretStore = "test-store"
		SecretName      = "test-secret"
	)

	var PushSecretNamespace string

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
		_ = k8sClient.Delete(context.Background(), &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
		})

		// give a time for reconciler to remove finalizers before removing SecretStores
		time.Sleep(2 * time.Second)

		_ = k8sClient.Delete(context.Background(), &v1beta1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretStore,
				Namespace: PushSecretNamespace,
			},
		})

		_ = k8sClient.Delete(context.Background(), &v1beta1.ClusterSecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name: PushSecretStore,
			},
		})

		_ = k8sClient.Delete(context.Background(), &v1.Secret{
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
			store: &v1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PushSecretStore,
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: v1beta1.SecretStoreSpec{
					Provider: &v1beta1.SecretStoreProvider{
						Fake: &v1beta1.FakeProvider{
							Data: []v1beta1.FakeProviderData{},
						},
					},
				},
			},
		}
	}

	syncSuccessfully := func(tc *testCase) {
		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data[defaultKey]

			By("ensuring the update was pushed to the provider")
			g.Eventually(func(g Gomega) bool {
				providerValue, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[0].Match.RemoteRef.RemoteKey)
				g.Expect(ok).To(BeTrue())
				g.Expect(providerValue.Value).To(Equal(secretValue))
				return true
			}, time.Second*10, time.Second).Should(BeTrue())

			return true
		}
	}

	updateIfNotExists := func(tc *testCase) {
		fakeProvider.SecretExistsFn = func(ctx context.Context, ref v1beta1.PushSecretRemoteRef) (bool, error) {
			// we are faking that the secret exists, to test the UpdatePolicy=IfNotExists logic
			return true, nil
		}
		tc.pushsecret.Spec.UpdatePolicy = v1alpha1.PushSecretUpdatePolicyIfNotExists
		tc.secret.Data[defaultKey] = []byte(newVal)

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("updating the source secret")
			g.Expect(k8sClient.Update(context.Background(), secret, &client.UpdateOptions{})).Should(Succeed())

			By("ensuring the update was not pushed to the provider")
			g.Consistently(func() bool {
				_, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[0].Match.RemoteRef.RemoteKey)
				return ok // the pushed secrets should not include this key
			}, time.Second*10, time.Second).Should(BeFalse())

			return true
		}
	}

	updateIfNotExistsPartialSecrets := func(tc *testCase) {
		fakeProvider.SecretExistsFn = func(ctx context.Context, ref v1beta1.PushSecretRemoteRef) (bool, error) {
			// we are faking that the secret exists, to test the UpdatePolicy=IfNotExists logic
			if ref.GetRemoteKey() == defaultPath {
				return true, nil
			}

			// for the other keys, we mark it as existing after our first PushSecret referencing that key
			_, ok := fakeProvider.LoadPushedSecret(ref.GetRemoteKey())
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

		tc.secret.Data[defaultKey] = []byte(newVal) // change initial value in secret
		tc.secret.Data[otherKey] = []byte(otherVal)

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("updating the source secret")
			g.Expect(k8sClient.Update(context.Background(), secret, &client.UpdateOptions{})).Should(Succeed())

			By("ensuring the update was not pushed to the provider (for the key that already exists)")
			g.Consistently(func() bool {
				_, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[0].Match.RemoteRef.RemoteKey)
				return ok
			}, time.Second*10, time.Second).Should(BeFalse())

			By("ensuring the update was pushed to the provider (for the key that does not exist)")
			g.Eventually(func(g Gomega) bool {
				providerValue, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[1].Match.RemoteRef.RemoteKey)
				g.Expect(ok).To(BeTrue())
				g.Expect(providerValue.Value).To(Equal([]byte(otherVal)))
				return true
			}, time.Second*10, time.Second).Should(BeTrue())

			return true
		}
	}

	updateIfNotExistsSyncStatus := func(tc *testCase) {
		fakeProvider.SecretExistsFn = func(ctx context.Context, ref v1beta1.PushSecretRemoteRef) (bool, error) {
			_, ok := fakeProvider.LoadPushedSecret(ref.GetRemoteKey())
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

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("updating the source secret")
			g.Expect(k8sClient.Update(context.Background(), secret, &client.UpdateOptions{})).Should(Succeed())

			By("ensuring the PushSecret has the correct syncedPushSecrets status")
			storeKey := fmt.Sprintf(storePrefixTemplate, PushSecretStore)
			g.Expect(ps.Status.SyncedPushSecrets).To(HaveKey(storeKey))
			g.Expect(ps.Status.SyncedPushSecrets[storeKey]).To(HaveKey(defaultPath))
			g.Expect(ps.Status.SyncedPushSecrets[storeKey]).To(HaveKey(otherPath))

			By("ensuring the PushSecret has the correct conditions")
			expected := []v1alpha1.PushSecretStatusCondition{
				{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionTrue,
					Reason:  v1alpha1.ReasonSynced,
					Message: "PushSecret synced successfully. Existing secrets in providers unchanged.",
				},
			}
			opts := cmpopts.IgnoreFields(v1alpha1.PushSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(ps.Status.Conditions).To(BeComparableTo(expected, opts))

			return true
		}
	}

	updateIfNotExistsSyncFailed := func(tc *testCase) {
		fakeProvider.SecretExistsFn = func(ctx context.Context, ref v1beta1.PushSecretRemoteRef) (bool, error) {
			return false, errors.New("don't know")
		}
		tc.pushsecret.Spec.UpdatePolicy = v1alpha1.PushSecretUpdatePolicyIfNotExists
		tc.secret.Data[defaultKey] = []byte(newVal)

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("updating the source secret")
			g.Expect(k8sClient.Update(context.Background(), secret, &client.UpdateOptions{})).Should(Succeed())

			By("ensuring the PushSecret has the correct conditions")
			expected := []v1alpha1.PushSecretStatusCondition{
				{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  v1alpha1.ReasonErrored,
					Message: fmt.Errorf(errProviderPush, fmt.Errorf(errProviderSecretExists, errors.New("don't know"))).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(v1alpha1.PushSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(ps.Status.Conditions).To(BeComparableTo(expected, opts))

			By("ensuring the update was not pushed to the provider")
			g.Consistently(func() bool {
				_, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[0].Match.RemoteRef.RemoteKey)
				return ok
			}, time.Second*10, time.Second).Should(BeFalse())

			return true
		}
	}

	syncSuccessfullyWithTemplate := func(tc *testCase) {
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
				Template: &v1beta1.ExternalSecretTemplate{
					Metadata: v1beta1.ExternalSecretTemplateMetadata{
						Labels: map[string]string{
							"foos": "ball",
						},
						Annotations: map[string]string{
							"hihi": "ga",
						},
					},
					Type:          v1.SecretTypeOpaque,
					EngineVersion: v1beta1.TemplateEngineV2,
					Data: map[string]string{
						defaultKey: "{{ .key | toString | upper }} was templated",
					},
				},
			},
		}

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("ensuring the update was pushed to the provider")
			g.Eventually(func(g Gomega) bool {
				providerValue, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[0].Match.RemoteRef.RemoteKey)
				g.Expect(ok).To(BeTrue())
				g.Expect(providerValue.Value).To(Equal([]byte("VALUE was templated")))
				return true
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	// if conversion strategy is defined, revert the keys based on the strategy.
	syncSuccessfullyWithConversionStrategy := func(tc *testCase) {
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

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data["some-array_U005b_0_U005d_.entity"]

			By("ensuring the update was pushed to the provider")
			g.Eventually(func(g Gomega) bool {
				providerValue, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[0].Match.RemoteRef.RemoteKey)
				g.Expect(ok).To(BeTrue())
				g.Expect(providerValue.Value).To(Equal(secretValue))
				return true
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	syncAndDeleteSuccessfully := func(tc *testCase) {
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

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("updating the PushSecret")
			ps.Spec.Data[0].Match.RemoteRef.RemoteKey = newKey
			g.Expect(k8sClient.Update(context.Background(), ps, &client.UpdateOptions{})).Should(Succeed())

			g.Eventually(func(g Gomega) bool {
				By("getting the updated PushSecret")
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				updatedPS := &v1alpha1.PushSecret{}
				g.Expect(k8sClient.Get(context.Background(), psKey, updatedPS)).Should(Succeed())

				By("ensuring the PushSecret has the correct syncedPushSecrets status")
				storeKey := fmt.Sprintf(storePrefixTemplate, PushSecretStore)
				g.Expect(updatedPS.Status.SyncedPushSecrets).To(HaveKey(storeKey))
				g.Expect(updatedPS.Status.SyncedPushSecrets[storeKey]).To(HaveKey(newKey))
				g.Expect(updatedPS.Status.SyncedPushSecrets[storeKey][newKey].Match.SecretKey).To(Equal(defaultKey))
				return true
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	// if PushSecret's DeletionPolicy is cleared, it should delete successfully (not be stuck on finalizer)
	syncChangePolicyAndDeleteSuccessfully := func(tc *testCase) {
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

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("updating the PushSecret DeletionPolicy to None")
			ps.Spec.DeletionPolicy = v1alpha1.PushSecretDeletionPolicyNone
			g.Expect(k8sClient.Update(context.Background(), ps, &client.UpdateOptions{})).Should(Succeed())

			By("deleting the PushSecret")
			g.Expect(k8sClient.Delete(context.Background(), ps, &client.DeleteOptions{})).Should(Succeed())

			By("ensuring the PushSecret is deleted")
			g.Eventually(func() bool {
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				updatedPS := &v1alpha1.PushSecret{}
				err := k8sClient.Get(context.Background(), psKey, updatedPS)
				return apierrors.IsNotFound(err)
			}, time.Second*10, time.Second).Should(BeTrue())

			return true
		}
	}

	failDelete := func(tc *testCase) {
		fakeProvider.DeleteSecretFn = func(ctx context.Context, ref v1beta1.PushSecretRemoteRef) error {
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

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("updating the PushSecret remote key")
			ps.Spec.Data[0].Match.RemoteRef.RemoteKey = newKey
			g.Expect(k8sClient.Update(context.Background(), ps, &client.UpdateOptions{})).Should(Succeed())

			g.Eventually(func(g Gomega) bool {
				By("getting the updated PushSecret")
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				updatedPS := &v1alpha1.PushSecret{}
				g.Expect(k8sClient.Get(context.Background(), psKey, updatedPS)).Should(Succeed())

				By("ensuring the PushSecret has the correct syncedPushSecrets status")
				storeKey := fmt.Sprintf(storePrefixTemplate, PushSecretStore)
				g.Expect(updatedPS.Status.SyncedPushSecrets).To(HaveKey(storeKey))
				g.Expect(updatedPS.Status.SyncedPushSecrets[storeKey]).To(HaveKey(newKey), "new key should be present")
				g.Expect(updatedPS.Status.SyncedPushSecrets[storeKey]).To(HaveKey(defaultPath), "old key should still be present because delete failed")
				return true
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	failDeleteStore := func(tc *testCase) {
		fakeProvider.DeleteSecretFn = func(ctx context.Context, ref v1beta1.PushSecretRemoteRef) error {
			return errors.New("boom")
		}
		tc.pushsecret.Spec.DeletionPolicy = v1alpha1.PushSecretDeletionPolicyDelete

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("creating a second SecretStore")
			secondStore := &v1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-store",
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: v1beta1.SecretStoreSpec{
					Provider: &v1beta1.SecretStoreProvider{
						Fake: &v1beta1.FakeProvider{
							Data: []v1beta1.FakeProviderData{},
						},
					},
				},
			}
			g.Expect(k8sClient.Create(context.Background(), secondStore, &client.CreateOptions{})).Should(Succeed())

			By("updating the PushSecret to use the new SecretStore")
			ps.Spec.SecretStoreRefs[0].Name = "new-store"
			g.Expect(k8sClient.Update(context.Background(), ps, &client.UpdateOptions{})).Should(Succeed())

			g.Eventually(func(g Gomega) bool {
				By("getting the updated PushSecret")
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				updatedPS := &v1alpha1.PushSecret{}
				g.Expect(k8sClient.Get(context.Background(), psKey, updatedPS)).Should(Succeed())

				By("ensuring the PushSecret has the correct syncedPushSecrets status")
				oldStoreKey := fmt.Sprintf(storePrefixTemplate, PushSecretStore)
				newStoreKey := fmt.Sprintf(storePrefixTemplate, "new-store")
				g.Expect(updatedPS.Status.SyncedPushSecrets).To(HaveKey(oldStoreKey), "old store should still be present because delete failed")
				g.Expect(updatedPS.Status.SyncedPushSecrets).To(HaveKey(newStoreKey), "new store should be present")
				return true
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	deleteWholeStore := func(tc *testCase) {
		tc.pushsecret.Spec.DeletionPolicy = v1alpha1.PushSecretDeletionPolicyDelete

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("creating a second SecretStore")
			secondStore := &v1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-store",
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: v1beta1.SecretStoreSpec{
					Provider: &v1beta1.SecretStoreProvider{
						Fake: &v1beta1.FakeProvider{
							Data: []v1beta1.FakeProviderData{},
						},
					},
				},
			}
			g.Expect(k8sClient.Create(context.Background(), secondStore, &client.CreateOptions{})).Should(Succeed())

			By("updating the PushSecret to use the new SecretStore")
			ps.Spec.SecretStoreRefs[0].Name = "new-store"
			g.Expect(k8sClient.Update(context.Background(), ps, &client.UpdateOptions{})).Should(Succeed())

			g.Eventually(func(g Gomega) bool {
				By("getting the updated PushSecret")
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				updatedPS := &v1alpha1.PushSecret{}
				g.Expect(k8sClient.Get(context.Background(), psKey, updatedPS)).Should(Succeed())

				By("ensuring the PushSecret has the correct syncedPushSecrets status")
				oldStoreKey := fmt.Sprintf(storePrefixTemplate, PushSecretStore)
				newStoreKey := fmt.Sprintf(storePrefixTemplate, "new-store")
				g.Expect(updatedPS.Status.SyncedPushSecrets).NotTo(HaveKey(oldStoreKey), "old store should not be present")
				g.Expect(updatedPS.Status.SyncedPushSecrets).To(HaveKey(newStoreKey))
				g.Expect(updatedPS.Status.SyncedPushSecrets[newStoreKey]).To(HaveKey(defaultPath))
				g.Expect(updatedPS.Status.SyncedPushSecrets[newStoreKey][defaultPath].Match.SecretKey).To(Equal(defaultKey))
				g.Expect(updatedPS.Status.SyncedPushSecrets).To(HaveLen(1))
				return true
			}, time.Second*10, time.Second).Should(BeTrue())
			return true
		}
	}

	syncMatchingLabels := func(tc *testCase) {
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
		tc.store = &v1beta1.SecretStore{
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
			Spec: v1beta1.SecretStoreSpec{
				Provider: &v1beta1.SecretStoreProvider{
					Fake: &v1beta1.FakeProvider{
						Data: []v1beta1.FakeProviderData{},
					},
				},
			},
		}

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data[defaultKey]

			By("ensuring the update was pushed to the provider")
			providerValue, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[0].Match.RemoteRef.RemoteKey)
			g.Expect(ok).To(BeTrue())
			g.Expect(providerValue.Value).To(Equal(secretValue))

			By("ensuring the PushSecret has the correct conditions")
			expected := []v1alpha1.PushSecretStatusCondition{
				{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionTrue,
					Reason:  v1alpha1.ReasonSynced,
					Message: "PushSecret synced successfully",
				},
			}
			opts := cmpopts.IgnoreFields(v1alpha1.PushSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(ps.Status.Conditions).To(BeComparableTo(expected, opts))

			return true
		}
	}

	syncWithClusterStore := func(tc *testCase) {
		tc.store = &v1beta1.ClusterSecretStore{
			TypeMeta: metav1.TypeMeta{
				Kind: "ClusterSecretStore",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: PushSecretStore,
			},
			Spec: v1beta1.SecretStoreSpec{
				Provider: &v1beta1.SecretStoreProvider{
					Fake: &v1beta1.FakeProvider{
						Data: []v1beta1.FakeProviderData{},
					},
				},
			},
		}
		tc.pushsecret.Spec.SecretStoreRefs[0].Kind = "ClusterSecretStore"

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data[defaultKey]

			By("ensuring the update was pushed to the provider")
			providerValue, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[0].Match.RemoteRef.RemoteKey)
			g.Expect(ok).To(BeTrue())
			g.Expect(providerValue.Value).To(Equal(secretValue))

			By("ensuring the PushSecret has the correct conditions")
			expected := []v1alpha1.PushSecretStatusCondition{
				{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionTrue,
					Reason:  v1alpha1.ReasonSynced,
					Message: "PushSecret synced successfully",
				},
			}
			opts := cmpopts.IgnoreFields(v1alpha1.PushSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(ps.Status.Conditions).To(BeComparableTo(expected, opts))

			return true
		}
	}

	syncWithGenerator := func(tc *testCase) {
		tc.pushsecret.Spec.Selector.Secret = nil
		tc.pushsecret.Spec.Selector.GeneratorRef = &v1beta1.GeneratorRef{
			APIVersion: "generators.external-secrets.io/v1alpha1",
			Kind:       "Fake",
			Name:       "test",
		}

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("ensuring the update was pushed to the provider")
			providerValue, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[0].Match.RemoteRef.RemoteKey)
			g.Expect(ok).To(BeTrue())
			g.Expect(providerValue.Value).To(Equal([]byte("foo-bar-from-generator")))

			By("ensuring the PushSecret has the correct conditions")
			expected := []v1alpha1.PushSecretStatusCondition{
				{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionTrue,
					Reason:  v1alpha1.ReasonSynced,
					Message: "PushSecret synced successfully",
				},
			}
			opts := cmpopts.IgnoreFields(v1alpha1.PushSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(ps.Status.Conditions).To(BeComparableTo(expected, opts))

			return true
		}
	}

	syncWithClusterStoreMatchingLabels := func(tc *testCase) {
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
		tc.store = &v1beta1.ClusterSecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name: PushSecretStore,
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Spec: v1beta1.SecretStoreSpec{
				Provider: &v1beta1.SecretStoreProvider{
					Fake: &v1beta1.FakeProvider{
						Data: []v1beta1.FakeProviderData{},
					},
				},
			},
		}

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data[defaultKey]

			By("ensuring the update was pushed to the provider")
			providerValue, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[0].Match.RemoteRef.RemoteKey)
			g.Expect(ok).To(BeTrue())
			g.Expect(providerValue.Value).To(Equal(secretValue))

			By("ensuring the PushSecret has the correct conditions")
			expected := []v1alpha1.PushSecretStatusCondition{
				{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionTrue,
					Reason:  v1alpha1.ReasonSynced,
					Message: "PushSecret synced successfully",
				},
			}
			opts := cmpopts.IgnoreFields(v1alpha1.PushSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(ps.Status.Conditions).To(BeComparableTo(expected, opts))

			return true
		}
	}

	failNoSecret := func(tc *testCase) {
		tc.secret = nil

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("ensuring the PushSecret has the correct conditions")
			expected := []v1alpha1.PushSecretStatusCondition{
				{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  v1alpha1.ReasonErrored,
					Message: fmt.Errorf(errResolveSelector, fmt.Errorf(errGetSecret, "test-secret", errors.New(`secrets "test-secret" not found`))).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(v1alpha1.PushSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(ps.Status.Conditions).To(BeComparableTo(expected, opts))

			return true
		}
	}

	failNoSecretKey := func(tc *testCase) {
		tc.pushsecret.Spec.Data[0].Match.SecretKey = "unexisting"

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("ensuring the PushSecret has the correct conditions")
			expected := []v1alpha1.PushSecretStatusCondition{
				{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  v1alpha1.ReasonErrored,
					Message: fmt.Errorf(errProviderPush, fmt.Errorf(errSecretKeyNotExists, "unexisting")).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(v1alpha1.PushSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(ps.Status.Conditions).To(BeComparableTo(expected, opts))

			return true
		}
	}

	failNoSecretStore := func(tc *testCase) {
		tc.store = nil
		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("ensuring the PushSecret has the correct conditions")
			expected := []v1alpha1.PushSecretStatusCondition{
				{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  v1alpha1.ReasonErrored,
					Message: fmt.Errorf(errExtractStores, fmt.Errorf(errGetSecretStore, "test-store", errors.New(`secretstores.external-secrets.io "test-store" not found`))).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(v1alpha1.PushSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(ps.Status.Conditions).To(BeComparableTo(expected, opts))

			return true
		}
	}

	setSecretFail := func(tc *testCase) {
		fakeProvider.SetSecretFn = func(ctx context.Context, secret *v1.Secret, data v1beta1.PushSecretData) error {
			return errors.New("boom")
		}
		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("ensuring the PushSecret has the correct conditions")
			expected := []v1alpha1.PushSecretStatusCondition{
				{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  v1alpha1.ReasonErrored,
					Message: fmt.Errorf(errProviderPush, fmt.Errorf(errProviderPushKey, "key", "test-store", errors.New("boom"))).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(v1alpha1.PushSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(ps.Status.Conditions).To(BeComparableTo(expected, opts))

			return true
		}
	}

	failNoClusterStore := func(tc *testCase) {
		tc.store = nil
		tc.pushsecret.Spec.SecretStoreRefs[0].Kind = "ClusterSecretStore"
		tc.pushsecret.Spec.SecretStoreRefs[0].Name = "unexisting"

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("ensuring the PushSecret has the correct conditions")
			expected := []v1alpha1.PushSecretStatusCondition{
				{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  v1alpha1.ReasonErrored,
					Message: fmt.Errorf(errExtractStores, fmt.Errorf(errGetClusterSecretStore, "unexisting", errors.New(`clustersecretstores.external-secrets.io "unexisting" not found`))).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(v1alpha1.PushSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(ps.Status.Conditions).To(BeComparableTo(expected, opts))

			return true
		}
	}

	newClientFail := func(tc *testCase) {
		fakeProvider.NewFn = func(context.Context, v1beta1.GenericStore, client.Client, string) (v1beta1.SecretsClient, error) {
			return nil, errors.New("boom")
		}

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("ensuring the PushSecret has the correct conditions")
			expected := []v1alpha1.PushSecretStatusCondition{
				{
					Type:    v1alpha1.PushSecretReady,
					Status:  v1.ConditionFalse,
					Reason:  v1alpha1.ReasonErrored,
					Message: fmt.Errorf(errProviderPush, fmt.Errorf(errGetSecretClient, "test-store", errors.New("boom"))).Error(),
				},
			}
			opts := cmpopts.IgnoreFields(v1alpha1.PushSecretStatusCondition{}, "LastTransitionTime")
			g.Expect(ps.Status.Conditions).To(BeComparableTo(expected, opts))

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

			if tc.store != nil {
				By("creating the SecretStore")
				Expect(k8sClient.Create(ctx, tc.store)).To(Succeed())
			}
			if tc.secret != nil {
				By("creating the Secret")
				Expect(k8sClient.Create(ctx, tc.secret)).To(Succeed())
			}
			if tc.pushsecret != nil {
				By("creating the PushSecret")
				Expect(k8sClient.Create(ctx, tc.pushsecret)).Should(Succeed())
			}

			By("waiting 2 seconds for the PushSecret to sync")
			time.Sleep(2 * time.Second)

			Eventually(func(g Gomega) bool {
				By("getting the PushSecret")
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				createdPS := &v1alpha1.PushSecret{}
				g.Expect(k8sClient.Get(ctx, psKey, createdPS)).To(Succeed())

				By("running the assertion")
				return tc.assert(g, createdPS, tc.secret)
			}, timeout, interval).Should(BeTrue())
		},
		Entry("should sync", syncSuccessfully),
		Entry("should not update existing secret if UpdatePolicy=IfNotExists", updateIfNotExists),
		Entry("should only update parts of secret that don't already exist if UpdatePolicy=IfNotExists", updateIfNotExistsPartialSecrets),
		Entry("should update the PushSecret status correctly if UpdatePolicy=IfNotExists", updateIfNotExistsSyncStatus),
		Entry("should fail if secret existence cannot be verified if UpdatePolicy=IfNotExists", updateIfNotExistsSyncFailed),
		Entry("should sync with template", syncSuccessfullyWithTemplate),
		Entry("should sync with conversion strategy", syncSuccessfullyWithConversionStrategy),
		Entry("should delete if DeletionPolicy=Delete", syncAndDeleteSuccessfully),
		Entry("should delete after DeletionPolicy changed from Delete to None", syncChangePolicyAndDeleteSuccessfully),
		Entry("should track deletion tasks if Delete fails", failDelete),
		Entry("should track deleted stores if Delete fails", failDeleteStore),
		Entry("should delete all secrets if SecretStore changes", deleteWholeStore),
		Entry("should sync to stores matching labels", syncMatchingLabels),
		Entry("should sync with ClusterStore", syncWithClusterStore),
		Entry("should sync with Generator", syncWithGenerator),
		Entry("should sync with ClusterStore matching labels", syncWithClusterStoreMatchingLabels),
		Entry("should fail if Secret is not created", failNoSecret),
		Entry("should fail if Secret Key does not exist", failNoSecretKey),
		Entry("should fail if no valid SecretStore", failNoSecretStore),
		Entry("should fail if SetSecret fails", setSecretFail),
		Entry("should fail if no valid ClusterSecretStore", failNoClusterStore),
		Entry("should fail if NewClient fails", newClientFail),
	)
}

func testPushSecretMultipleStores() {
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
		_ = k8sClient.Delete(context.Background(), &v1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretName,
				Namespace: PushSecretNamespace,
			},
		})

		// give a time for reconciler to remove finalizers before removing SecretStores
		// TODO: Secret Stores should have finalizers bound to PushSecrets if DeletionPolicy == Delete
		time.Sleep(2 * time.Second)

		for _, psstore := range PushSecretStores {
			_ = k8sClient.Delete(context.Background(), &v1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      psstore,
					Namespace: PushSecretNamespace,
				},
			})
			_ = k8sClient.Delete(context.Background(), &v1beta1.ClusterSecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name: psstore,
				},
			})
		}

		_ = k8sClient.Delete(context.Background(), &v1.Secret{
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
			managedStore1: &v1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ManagedPushSecretStore1,
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: v1beta1.SecretStoreSpec{
					Provider: &v1beta1.SecretStoreProvider{
						Fake: &v1beta1.FakeProvider{
							Data: []v1beta1.FakeProviderData{},
						},
					},
				},
			},
			managedStore2: &v1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ManagedPushSecretStore2,
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: v1beta1.SecretStoreSpec{
					Provider: &v1beta1.SecretStoreProvider{
						Fake: &v1beta1.FakeProvider{
							Data: []v1beta1.FakeProviderData{},
						},
					},
				},
			},
			unmanagedStore1: &v1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      UnmanagedPushSecretStore1,
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: v1beta1.SecretStoreSpec{
					Provider: &v1beta1.SecretStoreProvider{
						Fake: &v1beta1.FakeProvider{
							Data: []v1beta1.FakeProviderData{},
						},
					},
					Controller: "not-managed",
				},
			},
			unmanagedStore2: &v1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      UnmanagedPushSecretStore2,
					Namespace: PushSecretNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: v1beta1.SecretStoreSpec{
					Provider: &v1beta1.SecretStoreProvider{
						Fake: &v1beta1.FakeProvider{
							Data: []v1beta1.FakeProviderData{},
						},
					},
					Controller: "not-managed",
				},
			},
		}
	}

	multipleManagedStoresSyncsSuccessfully := func(tc *testCase) {
		tc.pushsecret.Spec.SecretStoreRefs = append(tc.pushsecret.Spec.SecretStoreRefs,
			v1alpha1.PushSecretStoreRef{
				Name: ManagedPushSecretStore2,
				Kind: "SecretStore",
			},
		)

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data[defaultKey]

			By("ensuring the update was pushed to the provider")
			g.Eventually(func(g Gomega) bool {
				providerValue, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[0].Match.RemoteRef.RemoteKey)
				g.Expect(ok).To(BeTrue())
				g.Expect(providerValue.Value).To(Equal(secretValue))
				return true
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

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			By("ensuring the PushSecret has the correct conditions")
			g.Expect(ps.Status.Conditions).To(HaveLen(0))
			return true
		}
	}

	warnUnmanagedStoresAndSyncManagedStores := func(tc *testCase) {
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

		tc.assert = func(g Gomega, ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data[defaultKey]

			By("ensuring the update was pushed to the provider")
			g.Eventually(func(g Gomega) bool {
				providerValue, ok := fakeProvider.LoadPushedSecret(ps.Spec.Data[0].Match.RemoteRef.RemoteKey)
				g.Expect(ok).To(BeTrue())
				g.Expect(providerValue.Value).To(Equal(secretValue))
				return true
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

			if tc.managedStore1 != nil {
				By("creating the first managed SecretStore 1")
				Expect(k8sClient.Create(ctx, tc.managedStore1)).To(Succeed())
			}
			if tc.managedStore2 != nil {
				By("creating the second managed SecretStore 2")
				Expect(k8sClient.Create(ctx, tc.managedStore2)).To(Succeed())
			}
			if tc.unmanagedStore1 != nil {
				By("creating the first unmanaged SecretStore 1")
				Expect(k8sClient.Create(ctx, tc.unmanagedStore1)).To(Succeed())
			}
			if tc.unmanagedStore2 != nil {
				By("creating the second unmanaged SecretStore 2")
				Expect(k8sClient.Create(ctx, tc.unmanagedStore2)).To(Succeed())
			}
			if tc.secret != nil {
				By("creating the Secret")
				Expect(k8sClient.Create(ctx, tc.secret)).To(Succeed())
			}
			if tc.pushsecret != nil {
				By("creating the PushSecret")
				Expect(k8sClient.Create(ctx, tc.pushsecret)).Should(Succeed())
			}

			By("waiting 2 seconds for the PushSecret to sync")
			time.Sleep(2 * time.Second)

			Eventually(func(g Gomega) bool {
				By("getting the PushSecret")
				psKey := types.NamespacedName{Name: PushSecretName, Namespace: PushSecretNamespace}
				createdPS := &v1alpha1.PushSecret{}
				g.Expect(k8sClient.Get(ctx, psKey, createdPS)).To(Succeed())

				By("running the assertion")
				return tc.assert(g, createdPS, tc.secret)
			}, timeout, interval).Should(BeTrue())
		},
		Entry("should sync successfully if there are multiple managed stores", multipleManagedStoresSyncsSuccessfully),
		Entry("should skip unmanaged stores", skipUnmanagedStores),
		Entry("should skip unmanaged stores and sync managed stores", warnUnmanagedStoresAndSyncManagedStores),
	)
}
