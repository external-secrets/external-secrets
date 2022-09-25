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
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	v1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	ctest "github.com/external-secrets/external-secrets/pkg/controllers/commontest"
	"github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
)

var (
	fakeProvider *fake.Client
	timeout      = time.Second * 10
	interval     = time.Millisecond * 250
)

type testCase struct {
	store      v1beta1.GenericStore
	pushsecret *v1alpha1.PushSecret
	secret     *v1.Secret
	assert     func(pushsecret *v1alpha1.PushSecret, secret *v1.Secret) bool
}

func init() {
	fakeProvider = fake.New()
	v1beta1.ForceRegister(fakeProvider, &v1beta1.SecretStoreProvider{
		Fake: &v1beta1.FakeProvider{},
	})
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

var _ = Describe("ExternalSecret controller", func() {
	const (
		PushSecretName             = "test-es"
		PushSecretFQDN             = "externalsecrets.external-secrets.io/test-es"
		PushSecretStore            = "test-store"
		SecretName                 = "test-secret"
		PushSecretTargetSecretName = "test-secret"
		FakeManager                = "fake.manager"
		expectedSecretVal          = "SOMEVALUE was templated"
		targetPropObj              = "{{ .targetProperty | toString | upper }} was templated"
		FooValue                   = "map-foo-value"
		BarValue                   = "map-bar-value"
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
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: PushSecretNamespace,
			},
		})).To(Succeed())
		k8sClient.Delete(context.Background(), &v1beta1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PushSecretStore,
				Namespace: PushSecretNamespace,
			},
		})
		k8sClient.Delete(context.Background(), &v1beta1.ClusterSecretStore{
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
	})

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
						Secret: v1alpha1.PushSecretSecret{
							Name: SecretName,
						},
					},
					Data: []v1alpha1.PushSecretData{
						{
							Match: v1alpha1.PushSecretMatch{
								SecretKey: "key",
								RemoteRef: v1alpha1.PushSecretRemoteRef{
									RemoteKey: "path/to/key",
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
					"key": []byte("value"),
				},
			},
			store: &v1beta1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PushSecretStore,
					Namespace: PushSecretNamespace,
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

	// if target Secret name is not specified it should use the ExternalSecret name.
	syncSuccessfully := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return nil
		}
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data["key"]
			providerValue := fakeProvider.SetSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey].Value
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
	syncMatchingLabels := func(tc *testCase) {
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
						Kind: "SecretStore",
					},
				},
				Selector: v1alpha1.PushSecretSelector{
					Secret: v1alpha1.PushSecretSecret{
						Name: SecretName,
					},
				},
				Data: []v1alpha1.PushSecretData{
					{
						Match: v1alpha1.PushSecretMatch{
							SecretKey: "key",
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: "path/to/key",
							},
						},
					},
				},
			},
		}
		tc.store = &v1beta1.SecretStore{
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
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data["key"]
			providerValue := fakeProvider.SetSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey].Value
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
		tc.store = &v1beta1.ClusterSecretStore{
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
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data["key"]
			providerValue := fakeProvider.SetSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey].Value
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
					},
				},
				Selector: v1alpha1.PushSecretSelector{
					Secret: v1alpha1.PushSecretSecret{
						Name: SecretName,
					},
				},
				Data: []v1alpha1.PushSecretData{
					{
						Match: v1alpha1.PushSecretMatch{
							SecretKey: "key",
							RemoteRef: v1alpha1.PushSecretRemoteRef{
								RemoteKey: "path/to/key",
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
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			secretValue := secret.Data["key"]
			providerValue := fakeProvider.SetSecretArgs[ps.Spec.Data[0].Match.RemoteRef.RemoteKey].Value
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
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
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
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
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
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
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
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			expected := v1alpha1.PushSecretStatusCondition{
				Type:    v1alpha1.PushSecretReady,
				Status:  v1.ConditionFalse,
				Reason:  v1alpha1.ReasonErrored,
				Message: "could not get ClusterSecretStore \"unexisting\", clustersecretstores.external-secrets.io \"unexisting\" not found",
			}
			return checkCondition(ps.Status, expected)
		}
	} // if target Secret name is not specified it should use the ExternalSecret name.
	setSecretFail := func(tc *testCase) {
		fakeProvider.SetSecretFn = func() error {
			return fmt.Errorf("boom")
		}
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
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
		fakeProvider.NewFn = func(context.Context, v1beta1.GenericStore, client.Client, string) (v1beta1.SecretsClient, error) {
			return nil, fmt.Errorf("boom")
		}
		tc.assert = func(ps *v1alpha1.PushSecret, secret *v1.Secret) bool {
			expected := v1alpha1.PushSecretStatusCondition{
				Type:    v1alpha1.PushSecretReady,
				Status:  v1.ConditionFalse,
				Reason:  v1alpha1.ReasonErrored,
				Message: "set secret failed: could not start secrets client",
			}
			return checkCondition(ps.Status, expected)
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
			if tc.secret != nil {
				Expect(k8sClient.Create(ctx, tc.secret)).To(Succeed())
			}
			if tc.pushsecret != nil {
				Expect(k8sClient.Create(ctx, tc.pushsecret)).Should(Succeed())
			}
			time.Sleep(2 * time.Second)
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
		Entry("should sync to stores matching labels", syncMatchingLabels),
		Entry("should sync with ClusterStore", syncWithClusterStore),
		Entry("should sync with ClusterStore matching labels", syncWithClusterStoreMatchingLabels),
		Entry("should fail if Secret is not created", failNoSecret),
		Entry("should fail if Secret Key does not exist", failNoSecretKey),
		Entry("should fail if SetSecret fails", setSecretFail),
		Entry("should fail if no valid SecretStore", failNoSecretStore),
		Entry("should fail if no valid ClusterSecretStore", failNoClusterStore),
		Entry("should fail if NewClient fails", newClientFail),
	)
})
