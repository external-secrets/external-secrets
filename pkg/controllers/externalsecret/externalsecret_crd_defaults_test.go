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

package externalsecret

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	ctest "github.com/external-secrets/external-secrets/pkg/controllers/commontest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Guards issue #6797: the API server must not inject values into these optional
// strategy fields, because an injected value is a permanent diff for any
// apply-based reconciler comparing against the source manifest.
var _ = Describe("ExternalSecret CRD defaulting", func() {
	var namespace string

	BeforeEach(func() {
		var err error
		namespace, err = ctest.CreateNamespace("test-crd-defaults", k8sClient)
		Expect(err).ToNot(HaveOccurred())
	})

	It("leaves omitted strategy fields absent", func() {
		es := &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-defaults",
				Namespace: namespace,
			},
			Spec: esv1.ExternalSecretSpec{
				SecretStoreRef: esv1.SecretStoreRef{Name: "unused"},
				Target: esv1.ExternalSecretTarget{
					Template: &esv1.ExternalSecretTemplate{
						TemplateFrom: []esv1.TemplateFrom{{
							Secret: &esv1.TemplateRef{
								Name:  "tpl",
								Items: []esv1.TemplateRefItem{{Key: "key"}},
							},
						}},
					},
				},
				Data: []esv1.ExternalSecretData{{
					SecretKey: "key",
					RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "remote-key"},
				}},
				DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
					{Extract: &esv1.ExternalSecretDataRemoteRef{Key: "remote-key"}},
					{Find: &esv1.ExternalSecretFind{Path: new("some/path")}},
				},
			},
		}
		Expect(k8sClient.Create(context.Background(), es)).To(Succeed())

		stored := &esv1.ExternalSecret{}
		Expect(k8sClient.Get(context.Background(), types.NamespacedName{
			Name:      es.Name,
			Namespace: es.Namespace,
		}, stored)).To(Succeed())

		By("checking data[].remoteRef")
		remoteRef := stored.Spec.Data[0].RemoteRef
		Expect(remoteRef.MetadataPolicy).To(BeEmpty())
		Expect(remoteRef.ConversionStrategy).To(BeEmpty())
		Expect(remoteRef.DecodingStrategy).To(BeEmpty())

		By("checking dataFrom[].extract")
		extract := stored.Spec.DataFrom[0].Extract
		Expect(extract.MetadataPolicy).To(BeEmpty())
		Expect(extract.ConversionStrategy).To(BeEmpty())
		Expect(extract.DecodingStrategy).To(BeEmpty())

		By("checking dataFrom[].find")
		find := stored.Spec.DataFrom[1].Find
		Expect(find.ConversionStrategy).To(BeEmpty())
		Expect(find.DecodingStrategy).To(BeEmpty())

		By("checking target.template.templateFrom[]")
		Expect(stored.Spec.Target.Template.TemplateFrom[0].ValuesDecodingStrategy).To(BeEmpty())
	})
})
