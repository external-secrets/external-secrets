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
	"fmt"
	"time"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
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

func namespacedProviderSyncCase(f *framework.Framework) (string, func(*framework.TestCase)) {
	return common.NamespacedProviderSync(f, common.NamespacedProviderSyncConfig{
		Description:        "[fake] should sync a namespaced secret",
		ExternalSecretName: "fake-sync-es",
		TargetSecretName:   "fake-sync-target",
		RemoteKey:          fmt.Sprintf("fake-sync-%s", f.Namespace.Name),
		RemoteSecretValue:  `{"value":"fake-sync-value"}`,
		RemoteProperty:     "value",
		SecretKey:          "value",
		ExpectedValue:      "fake-sync-value",
	})
}

func namespacedProviderRefreshCase(f *framework.Framework) (string, func(*framework.TestCase)) {
	return common.NamespacedProviderRefresh(f, common.NamespacedProviderRefreshConfig{
		Description:         "[fake] should refresh after the provider data changes",
		ExternalSecretName:  "fake-refresh-es",
		TargetSecretName:    "fake-refresh-target",
		RemoteKey:           fmt.Sprintf("fake-refresh-%s", f.Namespace.Name),
		InitialSecretValue:  `{"value":"fake-initial-value"}`,
		UpdatedSecretValue:  `{"value":"fake-updated-value"}`,
		RemoteProperty:      "value",
		SecretKey:           "value",
		InitialExpectedData: "fake-initial-value",
		UpdatedExpectedData: "fake-updated-value",
		RefreshInterval:     defaultV2RefreshInterval,
		WaitTimeout:         30 * time.Second,
	})
}

func namespacedProviderFindCase(f *framework.Framework) (string, func(*framework.TestCase)) {
	return common.NamespacedProviderFind(f, common.NamespacedProviderFindConfig{
		Description:        "[fake] should sync dataFrom.find matches",
		ExternalSecretName: "fake-find-es",
		TargetSecretName:   "fake-find-target",
		MatchRegExp:        fmt.Sprintf("fake-find-%s-(one|two)", f.Namespace.Name),
		MatchingSecrets: map[string]string{
			fmt.Sprintf("fake-find-%s-one", f.Namespace.Name): `{"value":"fake-find-one"}`,
			fmt.Sprintf("fake-find-%s-two", f.Namespace.Name): `{"value":"fake-find-two"}`,
		},
		IgnoredSecrets: map[string]string{
			fmt.Sprintf("fake-find-ignore-%s", f.Namespace.Name): `{"value":"fake-ignore"}`,
		},
	})
}
