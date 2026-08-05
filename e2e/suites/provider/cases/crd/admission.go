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
	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// ValidateStore runs behind the SecretStore validating webhook, so the rules it
// enforces are only really proven once a store round-trips through a live API
// server: the webhook has to be reachable, the provider has to be registered in
// the running controller, and the CRD schema has to carry the CEL rules. These
// specs create deliberately invalid stores and expect the API to refuse them.
// No RBAC or ServiceAccount is needed, because a rejected store never connects.
var _ = Describe("[crd] store admission ", Label("crd"), func() {
	f := framework.New("eso-crd-admission")

	It("[crd] should reject a store that targets the core v1 Secret", func() {
		res := esv1.CRDProviderResource{Group: "", Version: "v1", Kind: "Secret"}
		err := f.CRClient.Create(GinkgoT().Context(), crdSecretStore(f, "reject-secret-kind", inClusterProviderSpec("default", res)))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("use the Kubernetes provider"))
	})

	It("[crd] should reject a SecretStore whitelist rule that constrains the namespace", func() {
		prov := inClusterProviderSpec("default", namespacedResource())
		// A SecretStore only ever reads its own namespace, so a namespace rule
		// can never match and would silently deny every read.
		prov.Whitelist = &esv1.CRDProviderWhitelist{
			Rules: []esv1.CRDProviderWhitelistRule{{Namespace: "^prod$"}},
		}
		err := f.CRClient.Create(GinkgoT().Context(), crdSecretStore(f, "reject-ns-rule", prov))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not supported for a SecretStore"))
	})

	It("[crd] should reject an empty whitelist rule", func() {
		prov := inClusterProviderSpec("default", namespacedResource())
		// A rule with no name, namespace, or properties matches everything, which
		// looks like a restriction but silently widens access.
		prov.Whitelist = &esv1.CRDProviderWhitelist{
			Rules: []esv1.CRDProviderWhitelistRule{{}},
		}
		err := f.CRClient.Create(GinkgoT().Context(), crdSecretStore(f, "reject-empty-rule", prov))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("whitelist rule must define name, namespace, or properties"))
	})

	It("[crd] should reject an invalid whitelist regex", func() {
		prov := inClusterProviderSpec("default", namespacedResource())
		prov.Whitelist = &esv1.CRDProviderWhitelist{
			Rules: []esv1.CRDProviderWhitelistRule{{Name: "^[unterminated"}},
		}
		err := f.CRClient.Create(GinkgoT().Context(), crdSecretStore(f, "reject-bad-regex", prov))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid whitelist.rules[0].name regex"))
	})

	It("[crd] should reject server.url without auth or authRef", func() {
		// Enforced by the CEL rule on the CRD schema rather than by the webhook,
		// so this also proves the generated schema shipped with the rule intact.
		prov := &esv1.CRDProvider{
			Server:   esv1.KubernetesServer{URL: "https://remote-api.example.com"},
			Resource: namespacedResource(),
		}
		err := f.CRClient.Create(GinkgoT().Context(), crdSecretStore(f, "reject-url-no-auth", prov))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("one of auth or authRef is required"))
	})
})

func crdSecretStore(f *framework.Framework, name string, prov *esv1.CRDProvider) *esv1.SecretStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: f.Namespace.Name},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{CRD: prov},
		},
	}
}
