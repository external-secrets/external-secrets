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
package template

import (
	"context"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

type templateProvider struct {
	framework *framework.Framework
}

func newProvider(f *framework.Framework) *templateProvider {
	prov := &templateProvider{
		framework: f,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func (s *templateProvider) CreateSecret(key string, val framework.SecretEntry) {
	// noop: this provider implements static key/value pairs
}

func (s *templateProvider) DeleteSecret(key string) {
	// noop: this provider implements static key/value pairs
}

func (s *templateProvider) BeforeEach() {
	// Create a secret store - change these values to match YAML
	By("creating a secret store for credentials")
	secretStore := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Fake: &esv1beta1.FakeProvider{
					Data: []esv1beta1.FakeProviderData{
						{
							Key:   "foo",
							Value: "bar",
						},
						{
							Key:   "baz",
							Value: "bang",
						},
						{
							Key: "map",
							ValueMap: map[string]string{
								"foo": "barmap",
								"bar": "bangmap",
							},
						},
						{
							Key:   "json",
							Value: `{"foo":{"bar":"baz"}}`,
						},
					},
				},
			},
		},
	}

	err := s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
