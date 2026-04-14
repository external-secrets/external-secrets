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

package common

import (
	"fmt"
	"time"

	"github.com/external-secrets/external-secrets-e2e/framework"
)

func FakeProviderSync(f *framework.Framework) (string, func(*framework.TestCase)) {
	remoteKey := fmt.Sprintf("fake-sync-%s", f.Namespace.Name)
	return NamespacedProviderSync(f, NamespacedProviderSyncConfig{
		Description:        "[fake] should sync a namespaced secret",
		ExternalSecretName: "fake-sync-es",
		TargetSecretName:   "fake-sync-target",
		RemoteKey:          remoteKey,
		RemoteSecretValue:  `{"value":"fake-sync-value"}`,
		RemoteProperty:     "value",
		SecretKey:          "value",
		ExpectedValue:      "fake-sync-value",
	})
}

func FakeProviderRefresh(f *framework.Framework) (string, func(*framework.TestCase)) {
	remoteKey := fmt.Sprintf("fake-refresh-%s", f.Namespace.Name)
	return NamespacedProviderRefresh(f, NamespacedProviderRefreshConfig{
		Description:         "[fake] should refresh after the provider data changes",
		ExternalSecretName:  "fake-refresh-es",
		TargetSecretName:    "fake-refresh-target",
		RemoteKey:           remoteKey,
		InitialSecretValue:  `{"value":"fake-initial-value"}`,
		UpdatedSecretValue:  `{"value":"fake-updated-value"}`,
		RemoteProperty:      "value",
		SecretKey:           "value",
		InitialExpectedData: "fake-initial-value",
		UpdatedExpectedData: "fake-updated-value",
		RefreshInterval:     10 * time.Second,
		WaitTimeout:         30 * time.Second,
	})
}

func FakeProviderFind(f *framework.Framework) (string, func(*framework.TestCase)) {
	remoteKeyOne := fmt.Sprintf("fake-find-%s-one", f.Namespace.Name)
	remoteKeyTwo := fmt.Sprintf("fake-find-%s-two", f.Namespace.Name)
	remoteKeyThree := fmt.Sprintf("fake-find-ignore-%s", f.Namespace.Name)
	return NamespacedProviderFind(f, NamespacedProviderFindConfig{
		Description:        "[fake] should sync dataFrom.find matches",
		ExternalSecretName: "fake-find-es",
		TargetSecretName:   "fake-find-target",
		MatchRegExp:        fmt.Sprintf("fake-find-%s-(one|two)", f.Namespace.Name),
		MatchingSecrets: map[string]string{
			remoteKeyOne: `{"value":"fake-find-one"}`,
			remoteKeyTwo: `{"value":"fake-find-two"}`,
		},
		IgnoredSecrets: map[string]string{
			remoteKeyThree: `{"value":"fake-ignore"}`,
		},
	})
}
