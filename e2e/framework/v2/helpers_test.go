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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	k8sv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1"

	. "github.com/onsi/gomega"
)

func TestCreateKubernetesProviderUsesProvidedCABundle(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(k8sv2alpha1.AddToScheme(scheme)).To(Succeed())

	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	f := &framework.Framework{
		CRClient: cl,
	}

	CreateKubernetesProvider(f, "provider-ns", "example", "remote-ns", "eso-auth", nil, []byte("inline-ca"))

	var provider k8sv2alpha1.Kubernetes
	err := cl.Get(context.Background(), client.ObjectKey{
		Namespace: "provider-ns",
		Name:      "example",
	}, &provider)
	Expect(err).NotTo(HaveOccurred())

	Expect(provider.Spec.Server.CABundle).To(Equal([]byte("inline-ca")))
	Expect(provider.Spec.Server.CAProvider).To(BeNil())
	Expect(provider.Spec.Auth).NotTo(BeNil())
	Expect(provider.Spec.Auth.ServiceAccount).To(Equal(&esmeta.ServiceAccountSelector{
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

func TestWaitForProviderConnectionConditionMatchesReadyStatus(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(providerStoreGVK, &unstructured.Unstructured{})

	storeObj := &unstructured.Unstructured{}
	storeObj.SetGroupVersionKind(providerStoreGVK)
	storeObj.SetName("example")
	storeObj.SetNamespace("test")
	storeObj.Object["status"] = map[string]any{
		"conditions": []any{map[string]any{
			"type":   providerStoreReady,
			"status": string(corev1.ConditionTrue),
		}},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(storeObj).Build()
	f := &framework.Framework{CRClient: cl}

	store := WaitForProviderConnectionCondition(f, "test", "example", metav1.ConditionTrue, 100*time.Millisecond)
	Expect(store).NotTo(BeNil())
	Expect(store.GetName()).To(Equal("example"))
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

func TestWaitForClusterProviderConditionMatchesReadyStatus(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(clusterProviderStoreGVK, &unstructured.Unstructured{})

	storeObj := &unstructured.Unstructured{}
	storeObj.SetGroupVersionKind(clusterProviderStoreGVK)
	storeObj.SetName("example")
	storeObj.Object["status"] = map[string]any{
		"conditions": []any{map[string]any{
			"type":   providerStoreReady,
			"status": string(corev1.ConditionTrue),
		}},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(storeObj).Build()
	f := &framework.Framework{CRClient: cl}

	store := WaitForClusterProviderCondition(f, "example", metav1.ConditionTrue, 100*time.Millisecond)
	Expect(store).NotTo(BeNil())
	Expect(store.GetName()).To(Equal("example"))
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
