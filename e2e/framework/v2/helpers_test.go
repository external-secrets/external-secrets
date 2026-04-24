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

package v2

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"

	. "github.com/onsi/gomega"
)

func TestNewKubernetesStoreProviderUsesProvidedCABundle(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	provider := NewKubernetesStoreProvider("remote-ns", "eso-auth", nil, []byte("inline-ca"))

	Expect(provider).NotTo(BeNil())
	Expect(provider.Kubernetes).NotTo(BeNil())
	Expect(provider.Kubernetes.Server.CABundle).To(Equal([]byte("inline-ca")))
	Expect(provider.Kubernetes.Server.CAProvider).To(BeNil())
	Expect(provider.Kubernetes.Auth).NotTo(BeNil())
	Expect(provider.Kubernetes.Auth.ServiceAccount).To(Equal(&esmeta.ServiceAccountSelector{
		Name:      "eso-auth",
		Namespace: nil,
	}))
}

func TestGetClusterCABundleWaitsForRootCAConfigMap(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())

	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	f := &framework.Framework{
		CRClient: cl,
	}

	go func() {
		time.Sleep(25 * time.Millisecond)
		Expect(cl.Create(context.Background(), &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-root-ca.crt",
				Namespace: "test",
			},
			Data: map[string]string{
				"ca.crt": "root-ca-data",
			},
		})).To(Succeed())
	}()

	Expect(GetClusterCABundle(f, "test")).To(Equal([]byte("root-ca-data")))
}

func TestCreateRuntimeSecretStoreCreatesStoreAndProviderClass(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	Expect(esv1.AddToScheme(scheme)).To(Succeed())
	Expect(esv1alpha1.AddToScheme(scheme)).To(Succeed())

	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	f := &framework.Framework{CRClient: cl}

	store := CreateRuntimeSecretStore(f, "test", "example", "provider-fake.test.svc:8080", &esv1.StoreProviderRef{
		APIVersion: "provider.external-secrets.io/v2alpha1",
		Kind:       "Fake",
		Name:       "fake-config",
	})

	Expect(store).NotTo(BeNil())
	Expect(store.Spec.RuntimeRef).NotTo(BeNil())
	Expect(store.Spec.RuntimeRef.Name).To(Equal(runtimeClassName("example")))
	Expect(store.Spec.RuntimeRef.Kind).To(BeEmpty())
	Expect(store.Spec.Provider).To(BeNil())
	Expect(store.Spec.ProviderRef).NotTo(BeNil())
	Expect(store.Spec.ProviderRef.Name).To(Equal("fake-config"))

	var persistedStore esv1.SecretStore
	err := cl.Get(context.Background(), client.ObjectKey{Name: "example", Namespace: "test"}, &persistedStore)
	Expect(err).NotTo(HaveOccurred())
	Expect(persistedStore.Spec.Provider).To(BeNil())
	Expect(persistedStore.Spec.ProviderRef).NotTo(BeNil())
	Expect(persistedStore.Spec.ProviderRef.Kind).To(Equal("Fake"))

	var runtimeClass esv1alpha1.ProviderClass
	err = cl.Get(context.Background(), client.ObjectKey{Name: runtimeClassName("example"), Namespace: "test"}, &runtimeClass)
	Expect(err).NotTo(HaveOccurred())
	Expect(runtimeClass.Spec.Address).To(Equal("provider-fake.test.svc:8080"))
}

func TestCreateRuntimeClusterSecretStoreCreatesStoreAndClusterProviderClass(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	Expect(esv1.AddToScheme(scheme)).To(Succeed())
	Expect(esv1alpha1.AddToScheme(scheme)).To(Succeed())

	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	f := &framework.Framework{CRClient: cl}

	store := CreateRuntimeClusterSecretStore(f, "example", "provider-fake.external-secrets-system.svc:8080", &esv1.StoreProviderRef{
		APIVersion: "provider.external-secrets.io/v2alpha1",
		Kind:       "Fake",
		Name:       "fake-config",
	}, []esv1.ClusterSecretStoreCondition{{
		Namespaces: []string{"tenant-a"},
	}})

	Expect(store).NotTo(BeNil())
	Expect(store.Spec.RuntimeRef).NotTo(BeNil())
	Expect(store.Spec.RuntimeRef.Name).To(Equal(runtimeClassName("example")))
	Expect(store.Spec.RuntimeRef.Kind).To(BeEmpty())
	Expect(store.Spec.Conditions).To(HaveLen(1))
	Expect(store.Spec.Provider).To(BeNil())
	Expect(store.Spec.ProviderRef).NotTo(BeNil())
	Expect(store.Spec.ProviderRef.Name).To(Equal("fake-config"))

	var persistedStore esv1.ClusterSecretStore
	err := cl.Get(context.Background(), client.ObjectKey{Name: "example"}, &persistedStore)
	Expect(err).NotTo(HaveOccurred())
	Expect(persistedStore.Spec.Provider).To(BeNil())
	Expect(persistedStore.Spec.ProviderRef).NotTo(BeNil())
	Expect(persistedStore.Spec.ProviderRef.Kind).To(Equal("Fake"))

	var runtimeClass esv1alpha1.ClusterProviderClass
	err = cl.Get(context.Background(), client.ObjectKey{Name: runtimeClassName("example")}, &runtimeClass)
	Expect(err).NotTo(HaveOccurred())
	Expect(runtimeClass.Spec.Address).To(Equal("provider-fake.external-secrets-system.svc:8080"))
}

func TestWaitForSecretStoreConditionMatchesReadyStatus(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	Expect(esv1.AddToScheme(scheme)).To(Succeed())

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "test",
		},
		Status: esv1.SecretStoreStatus{
			Conditions: []esv1.SecretStoreStatusCondition{{
				Type:   esv1.SecretStoreReady,
				Status: corev1.ConditionTrue,
			}},
		},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(store).Build()
	f := &framework.Framework{CRClient: cl}

	got := WaitForSecretStoreCondition(f, "test", "example", metav1.ConditionTrue, 100*time.Millisecond)
	Expect(got).NotTo(BeNil())
	Expect(got.GetName()).To(Equal("example"))
}

func TestWaitForClusterSecretStoreConditionMatchesReadyStatus(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	Expect(esv1.AddToScheme(scheme)).To(Succeed())

	store := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: "example",
		},
		Status: esv1.SecretStoreStatus{
			Conditions: []esv1.SecretStoreStatusCondition{{
				Type:   esv1.SecretStoreReady,
				Status: corev1.ConditionTrue,
			}},
		},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(store).Build()
	f := &framework.Framework{CRClient: cl}

	got := WaitForClusterSecretStoreCondition(f, "example", metav1.ConditionTrue, 100*time.Millisecond)
	Expect(got).NotTo(BeNil())
	Expect(got.GetName()).To(Equal("example"))
}
