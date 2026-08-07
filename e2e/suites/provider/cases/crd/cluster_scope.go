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

package crd

import (
	"time"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// A cluster-scoped target kind takes different branches all the way down: the
// RESTMapper reports a cluster scope, getObject drops the namespace (and
// rejects a '/' in the key), and GetAllSecrets lists without a namespace
// selector and keys results by bare object name even for a ClusterSecretStore.
// None of that is reachable with the namespaced kind, so it gets its own suite.
var clusterCRDGVK = schema.GroupVersionKind{Group: crdGroup, Version: crdVersion, Kind: clusterCRDKind}

var _ = Describe("[crd] cluster-scoped kind ", Label("crd"), func() {
	f := framework.New("eso-crd-cluster")
	prov := NewClusterScopedProvider(f)

	DescribeTable("sync secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		Entry(syncClusterScopedViaSecretStore(f, prov)),
		Entry(syncClusterScopedViaClusterStore(f, prov)),
		Entry(findClusterScoped(f, prov)),
	)
})

// ClusterScopedProvider drives the cluster-scoped test kind. Object names are
// derived from the test namespace because the objects share one cluster-wide
// name space with every parallel spec.
type ClusterScopedProvider struct {
	framework *framework.Framework
}

func NewClusterScopedProvider(f *framework.Framework) *ClusterScopedProvider {
	prov := &ClusterScopedProvider{framework: f}
	BeforeEach(prov.BeforeEach)
	AfterEach(prov.AfterEach)
	return prov
}

func (s *ClusterScopedProvider) BeforeEach() {
	ensureCRD(s.framework, clusterScopedTestCRD())
	grantClusterRead(s.framework, crdGroup, clusterCRDPlural, "default", []string{"get", "list", "watch"})
	s.createStores()
}

func (s *ClusterScopedProvider) AfterEach() {
	ctx := GinkgoT().Context()
	ns := s.framework.Namespace.Name
	_ = s.framework.CRClient.Delete(ctx, &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: referentStoreName(s.framework)},
	})
	_ = s.framework.CRClient.Delete(ctx, &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName(ns)},
	})
	_ = s.framework.CRClient.Delete(ctx, &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRoleName(ns)},
	})
}

// objectName qualifies a base name with the test namespace. Cluster-scoped
// objects have no namespace to isolate them, so without this two parallel specs
// would fight over the same object.
func (s *ClusterScopedProvider) objectName(base string) string {
	return s.framework.Namespace.Name + "-" + base
}

// CreateSecret seeds a cluster-scoped CR. The framework passes the key through
// verbatim, so specs build it with objectName.
func (s *ClusterScopedProvider) CreateSecret(key string, val framework.SecretEntry) {
	createTestResource(s.framework, clusterCRDGVK, "", key, val)
}

func (s *ClusterScopedProvider) DeleteSecret(key string) {
	deleteTestResource(s.framework, clusterCRDGVK, "", key)
}

func (s *ClusterScopedProvider) createStores() {
	ctx := GinkgoT().Context()
	ns := s.framework.Namespace.Name
	res := esv1.CRDProviderResource{Group: crdGroup, Version: crdVersion, Kind: clusterCRDKind}

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: ns, Namespace: ns},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{CRD: inClusterProviderSpec("default", res)},
		},
	}
	Expect(s.framework.CRClient.Create(ctx, store)).To(Succeed())

	prov := inClusterProviderSpec("default", res)
	prov.Server.CAProvider.Namespace = &ns
	css := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: referentStoreName(s.framework)},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{CRD: prov},
		},
	}
	Expect(s.framework.CRClient.Create(ctx, css)).To(Succeed())
}

// syncClusterScopedViaSecretStore reads a cluster-scoped CR through a
// SecretStore. The store's own namespace is irrelevant here: the RESTMapper
// reports a cluster scope, so the read must not be namespaced.
func syncClusterScopedViaSecretStore(_ *framework.Framework, prov *ClusterScopedProvider) (string, func(*framework.TestCase)) {
	return "[crd] should sync a property from a cluster-scoped CR", func(tc *framework.TestCase) {
		key := prov.objectName("e2e-crd-cluster-a")
		tc.Secrets = map[string]framework.SecretEntry{
			key: {Value: `{"password":"cluster-pass"}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"pw": []byte("cluster-pass")},
		}
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "pw",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: key, Property: "spec.password"},
			},
		}
	}
}

// syncClusterScopedViaClusterStore covers the ClusterSecretStore key form for a
// cluster-scoped kind: a bare object name with no '/' separator, which is only
// legal because the resource has no namespace.
func syncClusterScopedViaClusterStore(f *framework.Framework, prov *ClusterScopedProvider) (string, func(*framework.TestCase)) {
	return "[crd] should sync a cluster-scoped CR via a ClusterSecretStore with a bare key", func(tc *framework.TestCase) {
		key := prov.objectName("e2e-crd-cluster-b")
		tc.Secrets = map[string]framework.SecretEntry{
			key: {Value: `{"token":"cluster-token"}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"token": []byte("cluster-token")},
		}
		tc.ExternalSecret.Spec.SecretStoreRef.Name = referentStoreName(f)
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1.ClusterSecretStoreKind
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "token",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: key, Property: "spec.token"},
			},
		}
	}
}

// findClusterScoped lists cluster-scoped CRs. Keys stay bare object names even
// through a ClusterSecretStore, because there is no namespace to prefix.
func findClusterScoped(f *framework.Framework, prov *ClusterScopedProvider) (string, func(*framework.TestCase)) {
	return "[crd] should find cluster-scoped CRs via dataFrom.find", func(tc *framework.TestCase) {
		hit := prov.objectName("e2e-crd-cluster-find")
		miss := prov.objectName("e2e-crd-cluster-skip")
		tc.Secrets = map[string]framework.SecretEntry{
			hit:  {Value: `{"marker":"hit"}`},
			miss: {Value: `{"marker":"miss"}`},
		}
		tc.ExternalSecret.Spec.SecretStoreRef.Name = referentStoreName(f)
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1.ClusterSecretStoreKind
		tc.ExternalSecret.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{
			{Find: &esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "^" + hit + "$"}}},
		}
		tc.ExpectedSecret = nil
		tc.AfterSync = func(_ framework.SecretStoreProvider, _ *corev1.Secret) {
			Eventually(func(g Gomega) {
				sec := targetSecret(g, f)
				g.Expect(sec.Data).To(HaveKey(hit))
				g.Expect(sec.Data).ToNot(HaveKey(miss))
				g.Expect(string(sec.Data[hit])).To(ContainSubstring(`"marker":"hit"`))
			}, time.Minute, time.Second).Should(Succeed())
		}
	}
}
