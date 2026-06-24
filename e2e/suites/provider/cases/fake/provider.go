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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	// nolint
	. "github.com/onsi/gomega"
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
	var store esv1.SecretStore
	err := s.framework.CRClient.Get(GinkgoT().Context(), types.NamespacedName{
		Namespace: s.framework.Namespace.Name,
		Name:      s.framework.Namespace.Name,
	}, &store)
	Expect(err).ToNot(HaveOccurred())
	base := store.DeepCopy()

	store.Spec.Provider.Fake.Data = upsertFakeProviderData(store.Spec.Provider.Fake.Data, esv1.FakeProviderData{
		Key:   key,
		Value: val.Value,
	})
	err = s.framework.CRClient.Patch(GinkgoT().Context(), &store, client.MergeFrom(base))
	Expect(err).ToNot(HaveOccurred())
}

func (s *Provider) BeforeEach() {
	s.CreateStore()
}

func (s *Provider) DeleteSecret(key string) {
	var store esv1.SecretStore
	err := s.framework.CRClient.Get(GinkgoT().Context(), types.NamespacedName{
		Namespace: s.framework.Namespace.Name,
		Name:      s.framework.Namespace.Name,
	}, &store)
	Expect(err).ToNot(HaveOccurred())
	base := store.DeepCopy()
	store.Spec.Provider.Fake.Data = removeFakeProviderData(store.Spec.Provider.Fake.Data, key, "")
	err = s.framework.CRClient.Patch(GinkgoT().Context(), &store, client.MergeFrom(base))
	Expect(err).ToNot(HaveOccurred())
}

func (s *Provider) CreateStore() {
	// Create a secret store - change these values to match YAML
	By("creating a secret store for credentials")
	fakeStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{
					Data: []esv1.FakeProviderData{},
				},
			},
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), fakeStore)
	Expect(err).ToNot(HaveOccurred())
}

func upsertFakeProviderData(data []esv1.FakeProviderData, entry esv1.FakeProviderData) []esv1.FakeProviderData {
	for i := range data {
		if data[i].Key == entry.Key && data[i].Version == entry.Version {
			data[i] = entry
			return data
		}
	}
	return append(data, entry)
}

func removeFakeProviderData(data []esv1.FakeProviderData, key, version string) []esv1.FakeProviderData {
	filtered := make([]esv1.FakeProviderData, 0, len(data))
	for _, entry := range data {
		if entry.Key == key && entry.Version == version {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
