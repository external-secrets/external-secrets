/*
Copyright © The ESO Authors

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

package fake

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	fakev2alpha1 "github.com/external-secrets/external-secrets/apis/provider/fake/v2alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	fakeProviderAPIVersion   = "provider.external-secrets.io/v2alpha1"
	fakeProviderKind         = "Fake"
	defaultV2WaitTimeout     = 3 * time.Minute
	defaultV2PollInterval    = 2 * time.Second
	defaultV2RefreshInterval = 10 * time.Second
)

var _ = Describe("[fake] v2 namespaced provider", Label("fake", "v2", "namespaced-provider"), func() {
	f := framework.New("eso-fake-v2-provider")
	prov := NewProviderV2(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
	})

	DescribeTable("namespaced provider",
		framework.TableFuncWithExternalSecret(f, prov),
		Entry(common.FakeProviderSync(f)),
		Entry(common.FakeProviderRefresh(f)),
		Entry(common.FakeProviderFind(f)),
		Entry(common.StatusNotUpdatedAfterSuccessfulSync(f)),
	)
})

var _ = Describe("[fake] v2 cluster provider", Label("fake", "v2", "cluster-provider"), func() {
	f := framework.New("eso-fake-v2-clusterprovider")
	prov := NewProviderV2(f)
	harness := newFakeClusterProviderExternalSecretHarness(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
	})

	DescribeTable("cluster provider external secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		Entry(common.ClusterProviderManifestNamespace(f, harness)),
		Entry(common.ClusterProviderProviderNamespace(f, harness)),
		Entry(common.ClusterProviderDeniedByConditions(f, harness)),
	)
})

var _ = Describe("[fake] v2 push secret", Label("fake", "v2", "push-secret"), func() {
	f := framework.New("eso-fake-v2-push")
	prov := NewProviderV2(f)
	harness := newFakeClusterProviderPushHarness(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
	})

	DescribeTable("push secret",
		framework.TableFuncWithPushSecret(f, prov, nil),
		// The fake backend stores raw pushed values in provider memory, so provider-
		// specific metadata validation such as namespaced remoteNamespace rejection
		// is not meaningful here.
		Entry(fakePushSecretImplicitProviderKind(f)),
		Entry(common.ClusterProviderPushManifestNamespace(f, harness)),
		Entry(common.ClusterProviderPushProviderNamespace(f, harness)),
		Entry(common.ClusterProviderPushDeniedByConditions(f, harness)),
	)
})

type ProviderV2 struct {
	framework *framework.Framework
}

func NewProviderV2(f *framework.Framework) *ProviderV2 {
	prov := &ProviderV2{
		framework: f,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func (s *ProviderV2) BeforeEach() {
	if !framework.IsV2ProviderMode() {
		return
	}

	frameworkv2.ScaleDeploymentBySelectorAndWait(s.framework, fakeBackendTarget(), 1, defaultV2WaitTimeout)
	s.createStore()
	frameworkv2.WaitForSecretStoreReady(s.framework, s.framework.Namespace.Name, s.framework.Namespace.Name, defaultV2WaitTimeout)
}

func (s *ProviderV2) CreateSecret(key string, val framework.SecretEntry) {
	s.updateStore(func(fake *esv1.FakeProvider) {
		fake.Data = upsertFakeProviderData(fake.Data, esv1.FakeProviderData{
			Key:   key,
			Value: val.Value,
		})
	})
}

func (s *ProviderV2) DeleteSecret(key string) {
	s.updateStore(func(fake *esv1.FakeProvider) {
		fake.Data = removeFakeProviderData(fake.Data, key, "")
	})
}

func (s *ProviderV2) createStore() {
	createNamespacedFakeSecretStore(s.framework, s.framework.Namespace.Name, s.framework.Namespace.Name)
}

func (s *ProviderV2) updateStore(mutate func(*esv1.FakeProvider)) {
	newLegacyRuntimeRefProvider(s.framework, esv1.SecretStoreRef{
		Name: s.framework.Namespace.Name,
		Kind: esv1.SecretStoreKind,
	}, s.framework.Namespace.Name).mutateStore(mutate)
}

func fakeBackendTarget() frameworkv2.BackendTarget {
	return frameworkv2.BackendTarget{
		Namespace:        frameworkv2.ProviderNamespace,
		PodLabelSelector: "app.kubernetes.io/name=external-secrets-provider-fake",
	}
}

func (s *ProviderV2) prepareNamespacedOperationalRuntime() *common.OperationalRuntime {
	return &common.OperationalRuntime{
		Provider: s,
		ProviderRef: esv1.SecretStoreRef{
			Name: s.framework.Namespace.Name,
			Kind: esv1.SecretStoreKind,
		},
		DefaultRemoteNamespace: s.framework.Namespace.Name,
		WaitForRemoteSecret: func(_, name, _ string, expectedValue string) {
			waitForPushedValueViaExternalSecret(s.framework, esv1.SecretStoreRef{
				Name: s.framework.Namespace.Name,
				Kind: esv1.SecretStoreKind,
			}, name, expectedValue)
		},
		MakeUnavailable: func() {
			frameworkv2.ScaleDeploymentBySelector(s.framework, fakeBackendTarget(), 0)
		},
		RestoreAvailability: func() {
			frameworkv2.ScaleDeploymentBySelector(s.framework, fakeBackendTarget(), 1)
		},
		RestartBackend: func() {
			frameworkv2.DeleteOneProviderPodBySelector(s.framework, fakeBackendTarget())
		},
	}
}

type fakeClusterProviderScenario struct {
	f                      *framework.Framework
	namePrefix             string
	authScope              esv1.AuthenticationScope
	defaultRemoteNamespace string
}

func newFakeClusterProviderScenario(f *framework.Framework, prefix string, authScope esv1.AuthenticationScope) *fakeClusterProviderScenario {
	providerNamespace := f.Namespace.Name
	if authScope == esv1.AuthenticationScopeProviderNamespace {
		providerNamespace = common.CreateProviderCaseNamespace(f, prefix+"-provider", defaultV2PollInterval)
	}

	s := &fakeClusterProviderScenario{
		f:                      f,
		namePrefix:             fmt.Sprintf("%s-%s", f.Namespace.Name, prefix),
		authScope:              authScope,
		defaultRemoteNamespace: fakeConfigNamespaceForAuthScope(authScope, f.Namespace.Name, providerNamespace),
	}
	return s
}

func (s *fakeClusterProviderScenario) createClusterProvider(conditions []esv1.ClusterSecretStoreCondition) string {
	clusterProviderName := fmt.Sprintf("%s-cluster-provider", s.namePrefix)
	runtimeName := fakeRuntimeClassName(clusterProviderName)
	providerName := fmt.Sprintf("%s-config", clusterProviderName)
	Expect(s.f.CreateObjectWithRetry(&esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: runtimeName,
		},
		Spec: esv1alpha1.ClusterProviderClassSpec{
			Address: frameworkv2.ProviderAddress("fake"),
		},
	})).To(Succeed())
	createFakeProviderConfig(s.f, s.defaultRemoteNamespace, providerName)
	Expect(s.f.CreateObjectWithRetry(newRuntimeRefClusterSecretStore(
		clusterProviderName,
		runtimeName,
		"",
		fakeStoreProviderRef(providerName, providerReferenceNamespace(s.authScope, s.defaultRemoteNamespace)),
	))).To(Succeed())

	var store esv1.ClusterSecretStore
	Expect(s.f.CRClient.Get(context.Background(), types.NamespacedName{Name: clusterProviderName}, &store)).To(Succeed())
	base := store.DeepCopy()
	store.Spec.Conditions = conditions
	Expect(s.f.CRClient.Patch(context.Background(), &store, client.MergeFrom(base))).To(Succeed())
	return clusterProviderName
}

func (s *fakeClusterProviderScenario) CreateSecret(key string, val framework.SecretEntry) {
	newLegacyRuntimeRefProvider(s.f, esv1.SecretStoreRef{
		Name: s.clusterProviderName(),
		Kind: esv1.ClusterSecretStoreKind,
	}, s.f.Namespace.Name).mutateStore(func(fake *esv1.FakeProvider) {
		fake.Data = upsertFakeProviderData(fake.Data, esv1.FakeProviderData{
			Key:   key,
			Value: val.Value,
		})
	})
}

func (s *fakeClusterProviderScenario) DeleteSecret(key string) {
	newLegacyRuntimeRefProvider(s.f, esv1.SecretStoreRef{
		Name: s.clusterProviderName(),
		Kind: esv1.ClusterSecretStoreKind,
	}, s.f.Namespace.Name).mutateStore(func(fake *esv1.FakeProvider) {
		fake.Data = removeFakeProviderData(fake.Data, key, "")
	})
}

func newFakeClusterProviderExternalSecretHarness(f *framework.Framework) common.ClusterProviderExternalSecretHarness {
	return common.ClusterProviderExternalSecretHarness{
		Prepare: func(tc *framework.TestCase, cfg common.ClusterProviderConfig) *common.ClusterProviderExternalSecretRuntime {
			s := newFakeClusterProviderScenario(f, cfg.Name, cfg.AuthScope)
			clusterProviderName := s.createClusterProvider(cfg.Conditions)
			frameworkv2.WaitForClusterSecretStoreReady(f, clusterProviderName, defaultV2WaitTimeout)

			return &common.ClusterProviderExternalSecretRuntime{
				ClusterProviderName: clusterProviderName,
				StoreRef: esv1.SecretStoreRef{
					Name: clusterProviderName,
					Kind: esv1.ClusterSecretStoreKind,
				},
				Provider: s,
			}
		},
	}
}

func newFakeClusterProviderPushHarness(f *framework.Framework) common.ClusterProviderPushHarness {
	return common.ClusterProviderPushHarness{
		Prepare: func(tc *framework.TestCase, cfg common.ClusterProviderConfig) *common.ClusterProviderPushRuntime {
			s := newFakeClusterProviderScenario(f, cfg.Name, cfg.AuthScope)
			clusterProviderName := s.createClusterProvider(cfg.Conditions)
			frameworkv2.WaitForClusterSecretStoreReady(f, clusterProviderName, defaultV2WaitTimeout)

			return &common.ClusterProviderPushRuntime{
				ClusterProviderName: clusterProviderName,
				StoreRef: esv1.SecretStoreRef{
					Name: clusterProviderName,
					Kind: esv1.ClusterSecretStoreKind,
				},
				DefaultRemoteNamespace: s.defaultRemoteNamespace,
				WaitForRemoteSecretValue: func(_, name, _ string, expectedValue string) {
					waitForPushedValueViaExternalSecret(f, esv1.SecretStoreRef{
						Name: clusterProviderName,
						Kind: esv1.ClusterSecretStoreKind,
					}, name, expectedValue)
				},
			}
		},
	}
}

func newFakeOperationalExternalSecretHarness(f *framework.Framework, prov *ProviderV2) common.OperationalExternalSecretHarness {
	return common.OperationalExternalSecretHarness{
		PrepareNamespaced: func(_ *framework.TestCase) *common.OperationalRuntime {
			return prov.prepareNamespacedOperationalRuntime()
		},
		PrepareCluster: func(_ *framework.TestCase, cfg common.ClusterProviderConfig) *common.OperationalRuntime {
			s := newFakeClusterProviderScenario(f, cfg.Name, cfg.AuthScope)
			clusterProviderName := s.createClusterProvider(cfg.Conditions)
			frameworkv2.WaitForClusterSecretStoreReady(f, clusterProviderName, defaultV2WaitTimeout)

			return &common.OperationalRuntime{
				Provider: s,
				ProviderRef: esv1.SecretStoreRef{
					Name: clusterProviderName,
					Kind: esv1.ClusterSecretStoreKind,
				},
				DefaultRemoteNamespace: s.defaultRemoteNamespace,
				WaitForRemoteSecret: func(_, name, _ string, expectedValue string) {
					waitForPushedValueViaExternalSecret(f, esv1.SecretStoreRef{
						Name: clusterProviderName,
						Kind: esv1.ClusterSecretStoreKind,
					}, name, expectedValue)
				},
				MakeUnavailable: func() {
					frameworkv2.ScaleDeploymentBySelectorAndWait(f, fakeBackendTarget(), 0, defaultV2WaitTimeout)
				},
				RestoreAvailability: func() {
					frameworkv2.ScaleDeploymentBySelectorAndWait(f, fakeBackendTarget(), 1, defaultV2WaitTimeout)
				},
				RestartBackend: func() {
					frameworkv2.DeleteOneProviderPodBySelector(f, fakeBackendTarget())
				},
			}
		},
	}
}

func newFakeOperationalPushHarness(f *framework.Framework, prov *ProviderV2) common.OperationalPushSecretHarness {
	return common.OperationalPushSecretHarness{
		PrepareNamespaced: func(_ *framework.TestCase) *common.OperationalRuntime {
			return prov.prepareNamespacedOperationalRuntime()
		},
		PrepareCluster: func(_ *framework.TestCase, cfg common.ClusterProviderConfig) *common.OperationalRuntime {
			s := newFakeClusterProviderScenario(f, cfg.Name, cfg.AuthScope)
			clusterProviderName := s.createClusterProvider(cfg.Conditions)
			frameworkv2.WaitForClusterSecretStoreReady(f, clusterProviderName, defaultV2WaitTimeout)

			return &common.OperationalRuntime{
				Provider: s,
				ProviderRef: esv1.SecretStoreRef{
					Name: clusterProviderName,
					Kind: esv1.ClusterSecretStoreKind,
				},
				DefaultRemoteNamespace: s.defaultRemoteNamespace,
				WaitForRemoteSecret: func(_, name, _ string, expectedValue string) {
					waitForPushedValueViaExternalSecret(f, esv1.SecretStoreRef{
						Name: clusterProviderName,
						Kind: esv1.ClusterSecretStoreKind,
					}, name, expectedValue)
				},
				MakeUnavailable: func() {
					frameworkv2.ScaleDeploymentBySelectorAndWait(f, fakeBackendTarget(), 0, defaultV2WaitTimeout)
				},
				RestoreAvailability: func() {
					frameworkv2.ScaleDeploymentBySelectorAndWait(f, fakeBackendTarget(), 1, defaultV2WaitTimeout)
				},
				RestartBackend: func() {
					frameworkv2.DeleteOneProviderPodBySelector(f, fakeBackendTarget())
				},
			}
		},
	}
}

func fakePushSecretImplicitProviderKind(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[fake] should support namespaced Provider refs when push kind is omitted", func(tc *framework.TestCase) {
		tc.PushSecretSource = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-push-implicit-kind-source",
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"value": []byte("implicit-kind-value"),
			},
		}
		tc.PushSecret.ObjectMeta.Name = "fake-push-implicit-kind"
		tc.PushSecret.Spec.SecretStoreRefs[0].Kind = ""
		tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
			Secret: &esv1alpha1.PushSecretSecret{
				Name: tc.PushSecretSource.Name,
			},
		}
		tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{{
			Match: esv1alpha1.PushSecretMatch{
				SecretKey: "value",
				RemoteRef: esv1alpha1.PushSecretRemoteRef{
					RemoteKey: "fake-push-implicit-kind-remote",
					Property:  "value",
				},
			},
		}}
		tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
			commonWaitForPushSecretReady(tc.Framework, ps.Namespace, ps.Name, corev1.ConditionTrue)
			waitForPushedValueViaExternalSecret(tc.Framework, esv1.SecretStoreRef{
				Name: f.Namespace.Name,
				Kind: esv1.SecretStoreKind,
			}, "fake-push-implicit-kind-remote", "implicit-kind-value")
		}
	}
}

func createNamespacedFakeSecretStore(f *framework.Framework, namespace, name string) {
	runtimeName := fakeRuntimeClassName(name)
	providerName := fmt.Sprintf("%s-config", name)
	Expect(f.CreateObjectWithRetry(&esv1alpha1.ProviderClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runtimeName,
			Namespace: namespace,
		},
		Spec: esv1alpha1.ProviderClassSpec{
			Address: frameworkv2.ProviderAddress("fake"),
		},
	})).To(Succeed())
	createFakeProviderConfig(f, namespace, providerName)
	Expect(f.CreateObjectWithRetry(newRuntimeRefSecretStore(namespace, name, runtimeName, "", fakeStoreProviderRef(providerName, "")))).To(Succeed())
}

func (s *fakeClusterProviderScenario) clusterProviderName() string {
	return fmt.Sprintf("%s-cluster-provider", s.namePrefix)
}

func fakeRuntimeClassName(name string) string {
	return fmt.Sprintf("%s-runtime", name)
}

func commonWaitForPushSecretReady(f *framework.Framework, namespace, name string, status corev1.ConditionStatus) {
	Eventually(func(g Gomega) {
		var ps esv1alpha1.PushSecret
		g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, &ps)).To(Succeed())

		for _, condition := range ps.Status.Conditions {
			if condition.Type == esv1alpha1.PushSecretReady && condition.Status == status {
				return
			}
		}
		g.Expect(false).To(BeTrue())
	}, time.Minute, 5*time.Second).Should(Succeed())
}

func waitForPushedValueViaExternalSecret(f *framework.Framework, storeRef esv1.SecretStoreRef, remoteKey, expectedValue string) {
	externalSecretName := fmt.Sprintf("fake-readback-%s", remoteKey)
	targetName := fmt.Sprintf("%s-target", externalSecretName)

	externalSecret := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalSecretName,
			Namespace: f.Namespace.Name,
		},
		Spec: esv1.ExternalSecretSpec{
			RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
			SecretStoreRef:  storeRef,
			Target: esv1.ExternalSecretTarget{
				Name: targetName,
			},
			Data: []esv1.ExternalSecretData{{
				SecretKey: "value",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{
					Key: remoteKey,
				},
			}},
		},
	}
	Expect(createOrUpdateReadbackExternalSecret(context.Background(), f, externalSecret)).To(Succeed())

	DeferCleanup(func() {
		err := f.CRClient.Delete(context.Background(), externalSecret)
		if err != nil && !apierrors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	})

	_, err := f.WaitForSecretValue(f.Namespace.Name, targetName, &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"value": []byte(expectedValue),
		},
	})
	Expect(err).NotTo(HaveOccurred())
}

func createOrUpdateReadbackExternalSecret(ctx context.Context, f *framework.Framework, externalSecret *esv1.ExternalSecret) error {
	if err := f.CreateObjectWithRetryContext(ctx, externalSecret); err != nil {
		return err
	}

	var existing esv1.ExternalSecret
	if err := f.CRClient.Get(ctx, client.ObjectKeyFromObject(externalSecret), &existing); err != nil {
		return err
	}
	if reflect.DeepEqual(existing.Spec, externalSecret.Spec) {
		return nil
	}

	externalSecret.SetResourceVersion(existing.GetResourceVersion())
	externalSecret.SetUID(existing.GetUID())
	return f.CRClient.Update(ctx, externalSecret)
}

func createFakeProviderConfig(f *framework.Framework, namespace, name string) {
	Expect(f.CRClient.Create(context.Background(), &fakev2alpha1.Fake{
		TypeMeta: metav1.TypeMeta{
			Kind:       fakeProviderKind,
			APIVersion: fakeProviderAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: esv1.FakeProvider{
			Data: []esv1.FakeProviderData{},
		},
	})).To(Succeed())
}

func updateFakeProviderConfig(f *framework.Framework, namespace, name string, mutate func(*fakev2alpha1.Fake)) {
	var fake fakev2alpha1.Fake
	Expect(f.CRClient.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &fake)).To(Succeed())
	base := fake.DeepCopy()
	mutate(&fake)
	Expect(f.CRClient.Patch(context.Background(), &fake, client.MergeFrom(base))).To(Succeed())
}

func fakeStoreProviderRef(name, namespace string) *esv1.StoreProviderRef {
	return &esv1.StoreProviderRef{
		APIVersion: fakeProviderAPIVersion,
		Kind:       fakeProviderKind,
		Name:       name,
		Namespace:  namespace,
	}
}

func providerReferenceNamespace(authScope esv1.AuthenticationScope, providerNamespace string) string {
	if authScope == esv1.AuthenticationScopeProviderNamespace {
		return providerNamespace
	}
	return ""
}

func fakeConfigNamespaceForAuthScope(authScope esv1.AuthenticationScope, manifestNamespace, providerNamespace string) string {
	if authScope == esv1.AuthenticationScopeProviderNamespace {
		return providerNamespace
	}
	return manifestNamespace
}
