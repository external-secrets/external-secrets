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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	fakev2alpha1 "github.com/external-secrets/external-secrets/apis/provider/fake/v2alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const invalidRuntimeAddress = "provider-does-not-exist.external-secrets-system.svc:8080"

var _ = Describe("[fake] v2 runtime ref resolution", Label("fake", "v2", "runtime-resolution"), func() {
	f := framework.New("eso-fake-v2-runtime-ref")

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
	})

	DescribeTable("runtime ref scope",
		framework.TableFuncWithExternalSecret(f, newLegacyRuntimeRefProvider(f, esv1.SecretStoreRef{}, "")),
		Entry(namespacedProviderClassDefaultCase(f)),
		Entry(explicitClusterProviderClassCase(f)),
		Entry(clusterSecretStoreDefaultCase(f)),
	)
})

func namespacedProviderClassDefaultCase(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "uses ProviderClass by default for SecretStore", func(tc *framework.TestCase) {
		storeName := "runtime-ref-store"
		runtimeName := fmt.Sprintf("%s-providerclass-default-runtime", f.Namespace.Name)
		providerName := fmt.Sprintf("%s-providerclass-default-config", storeName)

		tc.ExternalSecret.ObjectMeta.Name = "providerclass-default"
		tc.ExternalSecret.Spec.SecretStoreRef = esv1.SecretStoreRef{
			Name: storeName,
			Kind: esv1.SecretStoreKind,
		}
		tc.ExternalSecret.Spec.Target.Name = "providerclass-default-target"
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: "value",
			RemoteRef: esv1.ExternalSecretDataRemoteRef{
				Key:      "default-source",
				Property: "value",
			},
		}}
		tc.Secrets = map[string]framework.SecretEntry{
			"default-source": {Value: `{"value":"from-providerclass"}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"value": []byte("from-providerclass"),
			},
		}

		shadowRuntime := &esv1alpha1.ClusterProviderClass{
			ObjectMeta: metav1.ObjectMeta{Name: runtimeName},
			Spec:       esv1alpha1.ClusterProviderClassSpec{Address: invalidRuntimeAddress},
		}
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			Expect(f.CreateObjectWithRetry(&esv1alpha1.ProviderClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      runtimeName,
					Namespace: f.Namespace.Name,
				},
				Spec: esv1alpha1.ProviderClassSpec{
					Address: frameworkv2.ProviderAddress("fake"),
				},
			})).To(Succeed())
			createFakeProviderConfig(f, f.Namespace.Name, providerName)
			Expect(f.CreateObjectWithRetry(shadowRuntime)).To(Succeed())
			Expect(f.CreateObjectWithRetry(newRuntimeRefSecretStore(f.Namespace.Name, storeName, runtimeName, "", fakeStoreProviderRef(providerName, "")))).To(Succeed())

			tc.ProviderOverride = newLegacyRuntimeRefProvider(f, tc.ExternalSecret.Spec.SecretStoreRef, f.Namespace.Name)
			appendCleanup(tc, func() {
				deleteObject(f, shadowRuntime)
			})
		}
	}
}

func explicitClusterProviderClassCase(f *framework.Framework) (string, func(*framework.TestCase)) {
	return fmt.Sprintf("uses ClusterProviderClass when SecretStore runtimeRef.kind=%s", esv1.StoreRuntimeRefKindClusterProviderClass), func(tc *framework.TestCase) {
		storeName := "runtime-ref-store"
		runtimeName := fmt.Sprintf("%s-clusterproviderclass-explicit-runtime", f.Namespace.Name)
		providerName := fmt.Sprintf("%s-clusterproviderclass-explicit-config", storeName)

		tc.ExternalSecret.ObjectMeta.Name = "clusterproviderclass-explicit"
		tc.ExternalSecret.Spec.SecretStoreRef = esv1.SecretStoreRef{
			Name: storeName,
			Kind: esv1.SecretStoreKind,
		}
		tc.ExternalSecret.Spec.Target.Name = "clusterproviderclass-explicit-target"
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: "value",
			RemoteRef: esv1.ExternalSecretDataRemoteRef{
				Key:      "explicit-source",
				Property: "value",
			},
		}}
		tc.Secrets = map[string]framework.SecretEntry{
			"explicit-source": {Value: `{"value":"from-clusterproviderclass"}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"value": []byte("from-clusterproviderclass"),
			},
		}

		shadowRuntime := &esv1alpha1.ProviderClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:      runtimeName,
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.ProviderClassSpec{Address: invalidRuntimeAddress},
		}
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			validRuntime := &esv1alpha1.ClusterProviderClass{
				ObjectMeta: metav1.ObjectMeta{Name: runtimeName},
				Spec:       esv1alpha1.ClusterProviderClassSpec{Address: frameworkv2.ProviderAddress("fake")},
			}
			Expect(f.CreateObjectWithRetry(validRuntime)).To(Succeed())
			createFakeProviderConfig(f, f.Namespace.Name, providerName)
			Expect(f.CreateObjectWithRetry(shadowRuntime)).To(Succeed())
			Expect(f.CreateObjectWithRetry(newRuntimeRefSecretStore(f.Namespace.Name, storeName, runtimeName, esv1.StoreRuntimeRefKindClusterProviderClass, fakeStoreProviderRef(providerName, "")))).To(Succeed())

			tc.ProviderOverride = newLegacyRuntimeRefProvider(f, tc.ExternalSecret.Spec.SecretStoreRef, f.Namespace.Name)
			appendCleanup(tc, func() {
				deleteObject(f, validRuntime)
			})
		}
	}
}

func clusterSecretStoreDefaultCase(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "uses ClusterProviderClass by default for ClusterSecretStore", func(tc *framework.TestCase) {
		storeName := fmt.Sprintf("%s-runtime-ref-cluster-store", f.Namespace.Name)
		runtimeName := fmt.Sprintf("%s-clustersecretstore-default-runtime", f.Namespace.Name)
		providerName := fmt.Sprintf("%s-clustersecretstore-default-config", storeName)

		tc.ExternalSecret.ObjectMeta.Name = "clustersecretstore-default"
		tc.ExternalSecret.Spec.SecretStoreRef = esv1.SecretStoreRef{
			Name: storeName,
			Kind: esv1.ClusterSecretStoreKind,
		}
		tc.ExternalSecret.Spec.Target.Name = "clustersecretstore-default-target"
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{{
			SecretKey: "value",
			RemoteRef: esv1.ExternalSecretDataRemoteRef{
				Key:      "cluster-default-source",
				Property: "value",
			},
		}}
		tc.Secrets = map[string]framework.SecretEntry{
			"cluster-default-source": {Value: `{"value":"from-cluster-default"}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"value": []byte("from-cluster-default"),
			},
		}

		clusterStore := newRuntimeRefClusterSecretStore(storeName, runtimeName, "", fakeStoreProviderRef(providerName, ""))
		validRuntime := &esv1alpha1.ClusterProviderClass{
			ObjectMeta: metav1.ObjectMeta{Name: runtimeName},
			Spec:       esv1alpha1.ClusterProviderClassSpec{Address: frameworkv2.ProviderAddress("fake")},
		}
		tc.Prepare = func(tc *framework.TestCase, _ framework.SecretStoreProvider) {
			Expect(f.CreateObjectWithRetry(validRuntime)).To(Succeed())
			createFakeProviderConfig(f, f.Namespace.Name, providerName)
			Expect(f.CreateObjectWithRetry(&esv1alpha1.ProviderClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      runtimeName,
					Namespace: f.Namespace.Name,
				},
				Spec: esv1alpha1.ProviderClassSpec{Address: invalidRuntimeAddress},
			})).To(Succeed())
			Expect(f.CreateObjectWithRetry(clusterStore)).To(Succeed())

			tc.ProviderOverride = newLegacyRuntimeRefProvider(f, tc.ExternalSecret.Spec.SecretStoreRef, f.Namespace.Name)
			appendCleanup(tc, func() {
				deleteObject(f, validRuntime)
				deleteObject(f, clusterStore)
			})
		}
	}
}

type legacyRuntimeRefProvider struct {
	framework      *framework.Framework
	storeRef       esv1.SecretStoreRef
	storeNamespace string
}

func newLegacyRuntimeRefProvider(f *framework.Framework, storeRef esv1.SecretStoreRef, storeNamespace string) *legacyRuntimeRefProvider {
	return &legacyRuntimeRefProvider{
		framework:      f,
		storeRef:       storeRef,
		storeNamespace: storeNamespace,
	}
}

func (p *legacyRuntimeRefProvider) CreateSecret(key string, val framework.SecretEntry) {
	p.mutateStore(func(fake *esv1.FakeProvider) {
		fake.Data = upsertFakeProviderData(fake.Data, esv1.FakeProviderData{
			Key:   key,
			Value: val.Value,
		})
	})
}

func (p *legacyRuntimeRefProvider) DeleteSecret(key string) {
	p.mutateStore(func(fake *esv1.FakeProvider) {
		fake.Data = removeFakeProviderData(fake.Data, key, "")
	})
}

func (p *legacyRuntimeRefProvider) mutateStore(mutate func(*esv1.FakeProvider)) {
	ctx, cancel := context.WithTimeout(GinkgoT().Context(), 30*time.Second)
	defer cancel()

	switch p.storeRef.Kind {
	case esv1.SecretStoreKind:
		var store esv1.SecretStore
		Expect(p.framework.CRClient.Get(ctx, types.NamespacedName{
			Name:      p.storeRef.Name,
			Namespace: p.storeNamespace,
		}, &store)).To(Succeed())
		Expect(store.Spec.ProviderRef).NotTo(BeNil())
		updateFakeProviderConfig(p.framework, storeProviderRefNamespace(store.Spec.ProviderRef.Namespace, store.Namespace, p.storeNamespace), store.Spec.ProviderRef.Name, func(fake *fakev2alpha1.Fake) {
			mutate(&fake.Spec)
		})
	case esv1.ClusterSecretStoreKind:
		var store esv1.ClusterSecretStore
		Expect(p.framework.CRClient.Get(ctx, types.NamespacedName{Name: p.storeRef.Name}, &store)).To(Succeed())
		Expect(store.Spec.ProviderRef).NotTo(BeNil())
		updateFakeProviderConfig(p.framework, storeProviderRefNamespace(store.Spec.ProviderRef.Namespace, "", p.storeNamespace), store.Spec.ProviderRef.Name, func(fake *fakev2alpha1.Fake) {
			mutate(&fake.Spec)
		})
	default:
		Fail(fmt.Sprintf("unsupported runtime-ref store kind %q", p.storeRef.Kind))
	}
}

func newRuntimeRefSecretStore(namespace, name, runtimeName, runtimeKind string, providerRef *esv1.StoreProviderRef) *esv1.SecretStore {
	runtimeRef := &esv1.StoreRuntimeRef{Name: runtimeName}
	if runtimeKind != "" {
		runtimeRef.Kind = runtimeKind
	}
	providerRefCopy := *providerRef

	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: runtimeRef,
			ProviderRef: &providerRefCopy,
		},
	}
}

func newRuntimeRefClusterSecretStore(name, runtimeName, runtimeKind string, providerRef *esv1.StoreProviderRef) *esv1.ClusterSecretStore {
	runtimeRef := &esv1.StoreRuntimeRef{Name: runtimeName}
	if runtimeKind != "" {
		runtimeRef.Kind = runtimeKind
	}
	providerRefCopy := *providerRef

	return &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: esv1.SecretStoreSpec{
			RuntimeRef: runtimeRef,
			ProviderRef: &providerRefCopy,
		},
	}
}

func storeProviderRefNamespace(explicitNamespace, storeNamespace, sourceNamespace string) string {
	if explicitNamespace != "" {
		return explicitNamespace
	}
	if storeNamespace != "" {
		return storeNamespace
	}
	return sourceNamespace
}

func appendCleanup(tc *framework.TestCase, cleanup func()) {
	prev := tc.Cleanup
	tc.Cleanup = func() {
		cleanup()
		if prev != nil {
			prev()
		}
	}
}

func deleteObject(f *framework.Framework, obj client.Object) {
	err := f.CRClient.Delete(context.Background(), obj)
	if client.IgnoreNotFound(err) != nil {
		Expect(err).NotTo(HaveOccurred())
	}
}
