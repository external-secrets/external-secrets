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

package fake

import (
	"context"
	"encoding/json"
	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

type Provider struct {
	framework *framework.Framework
}

func NewProvider(f *framework.Framework) *Provider {
	prov := &Provider{
		framework: f,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func (s *Provider) CreateSecret(key string, val framework.SecretEntry) {
	var store esv1beta1.SecretStore
	err := s.framework.CRClient.Get(context.Background(), types.NamespacedName{
		Namespace: s.framework.Namespace.Name,
		Name:      s.framework.Namespace.Name,
	}, &store)
	Expect(err).ToNot(HaveOccurred())
	base := store.DeepCopy()

	mapData := make(map[string]string)
	_ = json.Unmarshal([]byte(val.Value), &mapData)
	store.Spec.Provider.Fake.Data = append(store.Spec.Provider.Fake.Data, esv1beta1.FakeProviderData{
		Key:      key,
		Value:    val.Value,
		ValueMap: mapData,
	})
	err = s.framework.CRClient.Patch(context.Background(), &store, client.MergeFrom(base))
	Expect(err).ToNot(HaveOccurred())
}

func (s *Provider) BeforeEach() {
	s.CreateStore()
}

func (s *Provider) DeleteSecret(key string) {
	var store esv1beta1.SecretStore
	err := s.framework.CRClient.Get(context.Background(), types.NamespacedName{
		Namespace: s.framework.Namespace.Name,
		Name:      s.framework.Namespace.Name,
	}, &store)
	Expect(err).ToNot(HaveOccurred())
	base := store.DeepCopy()
	data := make([]esv1beta1.FakeProviderData, 0)
	for _, v := range store.Spec.Provider.Fake.Data {
		if v.Key != key {
			data = append(data, v)
		}
	}
	store.Spec.Provider.Fake.Data = data
	err = s.framework.CRClient.Patch(context.Background(), &store, client.MergeFrom(base))
	Expect(err).ToNot(HaveOccurred())
}

func (s *Provider) CreateStore() {
	// Create a secret store - change these values to match YAML
	By("creating a secret store for credentials")
	fakeStore := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Fake: &esv1beta1.FakeProvider{
					Data: []esv1beta1.FakeProviderData{},
				},
			},
		},
	}
	err := s.framework.CRClient.Create(context.Background(), fakeStore)
	Expect(err).ToNot(HaveOccurred())
}
