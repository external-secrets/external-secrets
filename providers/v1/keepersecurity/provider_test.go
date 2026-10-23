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

package keepersecurity

import (
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func makeKeeperSecurityStore(folderID string) *esv1.SecretStore {
	return &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				KeeperSecurity: &esv1.KeeperSecurityProvider{
					Auth:     smmeta.SecretKeySelector{Name: "keeper-creds"},
					FolderID: folderID,
				},
			},
		},
	}
}

func TestValidateStore(t *testing.T) {
	testCases := []struct {
		label string
		store *esv1.SecretStore
	}{
		{
			label: "valid store without folderID",
			store: makeKeeperSecurityStore(""),
		},
		{
			label: "valid store with folderID",
			store: makeKeeperSecurityStore(folderID),
		},
	}

	p := Provider{}

	for _, tc := range testCases {
		t.Run(tc.label, func(t *testing.T) {
			_, err := p.ValidateStore(tc.store)
			if err != nil {
				t.Errorf("ValidateStore() unexpected error: %v", err)
			}
		})
	}
}
