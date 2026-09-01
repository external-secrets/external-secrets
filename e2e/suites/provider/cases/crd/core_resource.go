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
	"encoding/json"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// The provider is not limited to custom resources: resource.group: "" addresses
// the core API group, which is a distinct discovery path (no CRD, empty group
// string in both the RESTMapper lookup and the SelfSubjectAccessReview). A
// ConfigMap is the natural stand-in; the core v1 Secret is deliberately blocked
// and is covered in the admission suite.
var _ = Describe("[crd] core api resource ", Label("crd"), func() {
	f := framework.New("eso-crd-core")
	prov := NewCoreResourceProvider(f)

	DescribeTable("sync secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		Entry(syncConfigMapProperty(f)),
		Entry(syncConfigMapDataAsMap(f)),
	)
})

// CoreResourceProvider drives a store whose target resource is a core
// ConfigMap.
type CoreResourceProvider struct {
	framework *framework.Framework
}

func NewCoreResourceProvider(f *framework.Framework) *CoreResourceProvider {
	prov := &CoreResourceProvider{framework: f}
	BeforeEach(prov.BeforeEach)
	return prov
}

func (s *CoreResourceProvider) BeforeEach() {
	s.createStore()
}

// CreateSecret seeds a ConfigMap whose data is the parsed JSON value. Unlike
// the CRD kinds there is no spec envelope: a ConfigMap carries its payload
// under the top-level "data" field, so property paths read "data.<key>".
func (s *CoreResourceProvider) CreateSecret(key string, val framework.SecretEntry) {
	body := map[string]string{}
	Expect(json.Unmarshal([]byte(val.Value), &body)).To(Succeed())

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key,
			Namespace: s.framework.Namespace.Name,
			Labels:    val.Tags,
		},
		Data: body,
	}
	Expect(s.framework.CRClient.Create(GinkgoT().Context(), cm)).To(Succeed())
}

func (s *CoreResourceProvider) DeleteSecret(key string) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: key, Namespace: s.framework.Namespace.Name},
	}
	Expect(s.framework.CRClient.Delete(GinkgoT().Context(), cm)).To(Succeed())
}

func (s *CoreResourceProvider) createStore() {
	ctx := GinkgoT().Context()
	ns := s.framework.Namespace.Name

	role := readRole("eso-crd-core-read", ns, "", "configmaps", []string{"get", "list", "watch"})
	rb := bindRole("eso-crd-core-rb", ns, role.Name, "default")
	Expect(s.framework.CRClient.Create(ctx, role)).To(Succeed())
	Expect(s.framework.CRClient.Create(ctx, rb)).To(Succeed())

	res := esv1.CRDProviderResource{Group: "", Version: "v1", Kind: "ConfigMap"}
	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: ns, Namespace: ns},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{CRD: inClusterProviderSpec("default", res)},
		},
	}
	Expect(s.framework.CRClient.Create(ctx, store)).To(Succeed())
}

// syncConfigMapProperty reads one key out of a ConfigMap addressed through the
// core API group.
func syncConfigMapProperty(_ *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should sync a property from a core ConfigMap", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-cm": {Value: `{"username":"cm-user","password":"cm-pass"}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"pw": []byte("cm-pass")},
		}
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "pw",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-cm", Property: "data.password"},
			},
		}
	}
}

// syncConfigMapDataAsMap projects the whole ConfigMap data block into the
// target Secret via dataFrom.extract.
func syncConfigMapDataAsMap(_ *framework.Framework) (string, func(*framework.TestCase)) {
	return "[crd] should extract a core ConfigMap data block into a map", func(tc *framework.TestCase) {
		tc.Secrets = map[string]framework.SecretEntry{
			"e2e-crd-cm-map": {Value: `{"host":"db.example","port":"5432"}`},
		}
		tc.ExpectedSecret = &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"host": []byte("db.example"),
				"port": []byte("5432"),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{
			{Extract: &esv1.ExternalSecretDataRemoteRef{Key: "e2e-crd-cm-map", Property: "data"}},
		}
	}
}
