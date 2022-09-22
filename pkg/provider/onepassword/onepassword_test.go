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
package onepassword

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/1Password/connect-sdk-go/onepassword"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	fake "github.com/external-secrets/external-secrets/pkg/provider/onepassword/fake"
)

const (
	// vaults and items.
	myVault, myVaultID                       = "my-vault", "my-vault-id"
	myItem, myItemID                         = "my-item", "my-item-id"
	mySharedVault, mySharedVaultID           = "my-shared-vault", "my-shared-vault-id"
	mySharedItem, mySharedItemID             = "my-shared-item", "my-shared-item-id"
	myOtherVault, myOtherVaultID             = "my-other-vault", "my-other-vault-id"
	myOtherItem, myOtherItemID               = "my-other-item", "my-other-item-id"
	myNonMatchingVault, myNonMatchingVaultID = "my-non-matching-vault", "my-non-matching-vault-id"
	myNonMatchingItem, myNonMatchingItemID   = "my-non-matching-item", "my-non-matching-item-id"

	// fields and files.
	key1, key2, key3, key4                   = "key1", "key2", "key3", "key4"
	value1, value2, value3, value4           = "value1", "value2", "value3", "value4"
	sharedKey1, sharedValue1                 = "sharedkey1", "sharedvalue1"
	otherKey1                                = "otherkey1"
	filePNG, filePNGID                       = "file.png", "file-id"
	myFilePNG, myFilePNGID, myContents       = "my-file.png", "my-file-id", "my-contents"
	mySecondFileTXT, mySecondFileTXTID       = "my-second-file.txt", "my-second-file-id"
	mySecondContents                         = "my-second-contents"
	myFile2PNG, myFile2TXT                   = "my-file-2.png", "my-file-2.txt"
	myFile2ID, myContents2                   = "my-file-2-id", "my-contents-2"
	myOtherFilePNG, myOtherFilePNGID         = "my-other-file.png", "my-other-file-id"
	myOtherContents                          = "my-other-contents"
	nonMatchingFilePNG, nonMatchingFilePNGID = "non-matching-file.png", "non-matching-file-id"
	nonMatchingContents                      = "non-matching-contents"

	// other.
	mySecret, token, password = "my-secret", "token", "password"
	one, two, three           = "one", "two", "three"
	connectHost               = "https://example.com"
	setupCheckFormat          = "Setup: '%s', Check: '%s'"
	getSecretMapErrFormat     = "%s: onepassword.GetSecretMap(...): -expected, +got:\n-%#v\n+%#v\n"
	getSecretErrFormat        = "%s: onepassword.GetSecret(...): -expected, +got:\n-%#v\n+%#v\n"
	getAllSecretsErrFormat    = "%s: onepassword.GetAllSecrets(...): -expected, +got:\n-%#v\n+%#v\n"
	validateStoreErrFormat    = "%s: onepassword.validateStore(...): -expected, +got:\n-%#v\n+%#v\n"
	findItemErrFormat         = "%s: onepassword.findItem(...): -expected, +got:\n-%#v\n+%#v\n"
)

func TestFindItem(t *testing.T) {
	type check struct {
		checkNote    string
		findItemName string
		expectedItem *onepassword.Item
		expectedErr  error
	}

	type testCase struct {
		setupNote string
		provider  *ProviderOnePassword
		checks    []check
	}

	testCases := []testCase{
		{
			setupNote: "valid basic: one vault, one item, one field",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1},
				client: fake.NewMockClient().
					AddPredictableVault(myVault).
					AddPredictableItemWithField(myVault, myItem, key1, value1),
			},
			checks: []check{
				{
					checkNote:    "pass",
					findItemName: myItem,
					expectedErr:  nil,
					expectedItem: &onepassword.Item{
						ID:    myItemID,
						Title: myItem,
						Vault: onepassword.ItemVault{ID: myVaultID},
						Fields: []*onepassword.ItemField{
							{
								Label: key1,
								Value: value1,
							},
						},
					},
				},
			},
		},
		{
			setupNote: "multiple vaults, multiple items",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1, mySharedVault: 2},
				client: fake.NewMockClient().
					AddPredictableVault(myVault).
					AddPredictableItemWithField(myVault, myItem, key1, value1).
					AddPredictableVault(mySharedVault).
					AddPredictableItemWithField(mySharedVault, mySharedItem, sharedKey1, sharedValue1),
			},
			checks: []check{
				{
					checkNote:    "can still get myItem",
					findItemName: myItem,
					expectedErr:  nil,
					expectedItem: &onepassword.Item{
						ID:    myItemID,
						Title: myItem,
						Vault: onepassword.ItemVault{ID: myVaultID},
						Fields: []*onepassword.ItemField{
							{
								Label: key1,
								Value: value1,
							},
						},
					},
				},
				{
					checkNote:    "can also get mySharedItem",
					findItemName: mySharedItem,
					expectedErr:  nil,
					expectedItem: &onepassword.Item{
						ID:    mySharedItemID,
						Title: mySharedItem,
						Vault: onepassword.ItemVault{ID: mySharedVaultID},
						Fields: []*onepassword.ItemField{
							{
								Label: sharedKey1,
								Value: sharedValue1,
							},
						},
					},
				},
			},
		},
		{
			setupNote: "multiple vault matches when should be one",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1, mySharedVault: 2},
				client: fake.NewMockClient().
					AppendVault(myVault, onepassword.Vault{
						ID:   myVaultID,
						Name: myVault,
					}).
					AppendVault(myVault, onepassword.Vault{
						ID:   "my-vault-extra-match-id",
						Name: "my-vault-extra-match",
					}),
			},
			checks: []check{
				{
					checkNote:    "two vaults",
					findItemName: myItem,
					expectedErr:  fmt.Errorf("key not found in 1Password Vaults: my-item in: map[my-shared-vault:2 my-vault:1]"),
				},
			},
		},
		{
			setupNote: "no item matches when should be one",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1},
				client: fake.NewMockClient().
					AddPredictableVault(myVault),
			},
			checks: []check{
				{
					checkNote:    "no exist",
					findItemName: "my-item-no-exist",
					expectedErr:  fmt.Errorf(errKeyNotFound, fmt.Errorf("my-item-no-exist in: map[my-vault:1]")),
				},
			},
		},
		{
			setupNote: "multiple item matches when should be one",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1},
				client: fake.NewMockClient().
					AddPredictableVault(myVault).
					AddPredictableItemWithField(myVault, myItem, key1, value1).
					AppendItem(myVaultID, onepassword.Item{
						ID:    "asdf",
						Title: myItem,
						Vault: onepassword.ItemVault{ID: myVaultID},
					}),
			},
			checks: []check{
				{
					checkNote:    "multiple match",
					findItemName: myItem,
					expectedErr:  fmt.Errorf(errExpectedOneItem, fmt.Errorf("'my-item', got 2")),
				},
			},
		},
		{
			setupNote: "ordered vaults",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1, mySharedVault: 2, myOtherVault: 3},
				client: fake.NewMockClient().
					AddPredictableVault(myVault).
					AddPredictableVault(mySharedVault).
					AddPredictableVault(myOtherVault).

					// // my-item
					// returned: my-item in my-vault
					AddPredictableItemWithField(myVault, myItem, key1, value1).
					// preempted: my-item in my-shared-vault
					AppendItem(mySharedVaultID, onepassword.Item{
						ID:    myItemID,
						Title: myItem,
						Vault: onepassword.ItemVault{ID: mySharedVaultID},
					}).
					AppendItemField(mySharedVaultID, myItemID, onepassword.ItemField{
						Label: key1,
						Value: "value1-from-my-shared-vault",
					}).
					// preempted: my-item in my-other-vault
					AppendItem(myOtherVaultID, onepassword.Item{
						ID:    myItemID,
						Title: myItem,
						Vault: onepassword.ItemVault{ID: myOtherVaultID},
					}).
					AppendItemField(myOtherVaultID, myItemID, onepassword.ItemField{
						Label: key1,
						Value: "value1-from-my-other-vault",
					}).

					// // my-shared-item
					// returned: my-shared-item in my-shared-vault
					AddPredictableItemWithField(mySharedVault, mySharedItem, sharedKey1, "sharedvalue1-from-my-shared-vault").
					// preempted: my-shared-item in my-other-vault
					AppendItem(myOtherVaultID, onepassword.Item{
						ID:    mySharedItemID,
						Title: mySharedItem,
						Vault: onepassword.ItemVault{ID: myOtherVaultID},
					}).
					AppendItemField(myOtherVaultID, mySharedItemID, onepassword.ItemField{
						Label: sharedKey1,
						Value: "sharedvalue1-from-my-other-vault",
					}).

					// // my-other-item
					// returned: my-other-item in my-other-vault
					AddPredictableItemWithField(myOtherVault, myOtherItem, otherKey1, "othervalue1-from-my-other-vault"),
			},
			checks: []check{
				{
					// my-item in all three vaults, gets the one from my-vault
					checkNote:    "gets item from my-vault",
					findItemName: myItem,
					expectedErr:  nil,
					expectedItem: &onepassword.Item{
						ID:    myItemID,
						Title: myItem,
						Vault: onepassword.ItemVault{ID: myVaultID},
						Fields: []*onepassword.ItemField{
							{
								Label: key1,
								Value: value1,
							},
						},
					},
				},
				{
					// my-shared-item in my-shared-vault and my-other-vault, gets the one from my-shared-vault
					checkNote:    "gets item from my-shared-vault",
					findItemName: mySharedItem,
					expectedErr:  nil,
					expectedItem: &onepassword.Item{
						ID:    mySharedItemID,
						Title: mySharedItem,
						Vault: onepassword.ItemVault{ID: mySharedVaultID},
						Fields: []*onepassword.ItemField{
							{
								Label: sharedKey1,
								Value: "sharedvalue1-from-my-shared-vault",
							},
						},
					},
				},
				{
					// my-other-item in my-other-vault
					checkNote:    "gets item from my-other-vault",
					findItemName: myOtherItem,
					expectedErr:  nil,
					expectedItem: &onepassword.Item{
						ID:    myOtherItemID,
						Title: myOtherItem,
						Vault: onepassword.ItemVault{ID: myOtherVaultID},
						Fields: []*onepassword.ItemField{
							{
								Label: otherKey1,
								Value: "othervalue1-from-my-other-vault",
							},
						},
					},
				},
			},
		},
	}

	// run the tests
	for _, tc := range testCases {
		for _, check := range tc.checks {
			got, err := tc.provider.findItem(check.findItemName)
			notes := fmt.Sprintf(setupCheckFormat, tc.setupNote, check.checkNote)
			if check.expectedErr == nil && err != nil {
				// expected no error, got one
				t.Errorf(findItemErrFormat, notes, nil, err)
			}
			if check.expectedErr != nil && err == nil {
				// expected an error, didn't get one
				t.Errorf(findItemErrFormat, notes, check.expectedErr.Error(), nil)
			}
			if check.expectedErr != nil && err != nil && err.Error() != check.expectedErr.Error() {
				// expected an error, got the wrong one
				t.Errorf(findItemErrFormat, notes, check.expectedErr.Error(), err.Error())
			}
			if check.expectedItem != nil {
				if !reflect.DeepEqual(check.expectedItem, got) {
					// expected a predefined item, got something else
					t.Errorf(findItemErrFormat, notes, check.expectedItem, got)
				}
			}
		}
	}
}

func TestValidateStore(t *testing.T) {
	type testCase struct {
		checkNote    string
		store        *esv1beta1.SecretStore
		clusterStore *esv1beta1.ClusterSecretStore
		expectedErr  error
	}

	testCases := []testCase{
		{
			checkNote: "invalid: nil provider",
			store: &esv1beta1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: nil,
				},
			},
			expectedErr: fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreNilSpecProvider)),
		},
		{
			checkNote: "invalid: nil OnePassword provider spec",
			store: &esv1beta1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						OnePassword: nil,
					},
				},
			},
			expectedErr: fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreNilSpecProviderOnePassword)),
		},
		{
			checkNote: "valid secretStore",
			store: &esv1beta1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						OnePassword: &esv1beta1.OnePasswordProvider{
							Auth: &esv1beta1.OnePasswordAuth{
								SecretRef: &esv1beta1.OnePasswordAuthSecretRef{
									ConnectToken: esmeta.SecretKeySelector{
										Name: mySecret,
										Key:  token,
									},
								},
							},
							ConnectHost: connectHost,
							Vaults: map[string]int{
								myVault: 1,
							},
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			checkNote: "invalid: illegal namespace on SecretStore",
			store: &esv1beta1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						OnePassword: &esv1beta1.OnePasswordProvider{
							Auth: &esv1beta1.OnePasswordAuth{
								SecretRef: &esv1beta1.OnePasswordAuthSecretRef{
									ConnectToken: esmeta.SecretKeySelector{
										Name:      mySecret,
										Namespace: pointer.StringPtr("my-namespace"),
										Key:       token,
									},
								},
							},
							ConnectHost: connectHost,
							Vaults: map[string]int{
								myVault:      1,
								myOtherVault: 2,
							},
						},
					},
				},
			},
			expectedErr: fmt.Errorf(errOnePasswordStore, fmt.Errorf("namespace not allowed with namespaced SecretStore")),
		},
		{
			checkNote: "invalid: more than one vault with the same number",
			store: &esv1beta1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						OnePassword: &esv1beta1.OnePasswordProvider{
							Auth: &esv1beta1.OnePasswordAuth{
								SecretRef: &esv1beta1.OnePasswordAuthSecretRef{
									ConnectToken: esmeta.SecretKeySelector{
										Name: mySecret,
										Key:  token,
									},
								},
							},
							ConnectHost: connectHost,
							Vaults: map[string]int{
								myVault:      1,
								myOtherVault: 1,
							},
						},
					},
				},
			},
			expectedErr: fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreNonUniqueVaultNumbers)),
		},
		{
			checkNote: "valid: clusterSecretStore",
			clusterStore: &esv1beta1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "ClusterSecretStore",
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						OnePassword: &esv1beta1.OnePasswordProvider{
							Auth: &esv1beta1.OnePasswordAuth{
								SecretRef: &esv1beta1.OnePasswordAuthSecretRef{
									ConnectToken: esmeta.SecretKeySelector{
										Name:      mySecret,
										Namespace: pointer.StringPtr("my-namespace"),
										Key:       token,
									},
								},
							},
							ConnectHost: connectHost,
							Vaults: map[string]int{
								myVault: 1,
							},
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			checkNote: "invalid: clusterSecretStore without namespace",
			clusterStore: &esv1beta1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "ClusterSecretStore",
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						OnePassword: &esv1beta1.OnePasswordProvider{
							Auth: &esv1beta1.OnePasswordAuth{
								SecretRef: &esv1beta1.OnePasswordAuthSecretRef{
									ConnectToken: esmeta.SecretKeySelector{
										Name: mySecret,
										Key:  token,
									},
								},
							},
							ConnectHost: connectHost,
							Vaults: map[string]int{
								myVault:      1,
								myOtherVault: 2,
							},
						},
					},
				},
			},
			expectedErr: fmt.Errorf(errOnePasswordStore, fmt.Errorf("cluster scope requires namespace")),
		},
		{
			checkNote: "invalid: missing connectTokenSecretRef.name",
			store: &esv1beta1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						OnePassword: &esv1beta1.OnePasswordProvider{
							Auth: &esv1beta1.OnePasswordAuth{
								SecretRef: &esv1beta1.OnePasswordAuthSecretRef{
									ConnectToken: esmeta.SecretKeySelector{
										Key: token,
									},
								},
							},
							ConnectHost: connectHost,
							Vaults: map[string]int{
								myVault:      1,
								myOtherVault: 2,
							},
						},
					},
				},
			},
			expectedErr: fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreMissingRefName)),
		},
		{
			checkNote: "invalid: missing connectTokenSecretRef.key",
			store: &esv1beta1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						OnePassword: &esv1beta1.OnePasswordProvider{
							Auth: &esv1beta1.OnePasswordAuth{
								SecretRef: &esv1beta1.OnePasswordAuthSecretRef{
									ConnectToken: esmeta.SecretKeySelector{
										Name: mySecret,
									},
								},
							},
							ConnectHost: connectHost,
							Vaults: map[string]int{
								myVault:      1,
								myOtherVault: 2,
							},
						},
					},
				},
			},
			expectedErr: fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreMissingRefKey)),
		},
		{
			checkNote: "invalid: at least one vault",
			store: &esv1beta1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						OnePassword: &esv1beta1.OnePasswordProvider{
							Auth: &esv1beta1.OnePasswordAuth{
								SecretRef: &esv1beta1.OnePasswordAuthSecretRef{
									ConnectToken: esmeta.SecretKeySelector{
										Name: mySecret,
										Key:  token,
									},
								},
							},
							ConnectHost: connectHost,
							Vaults:      map[string]int{},
						},
					},
				},
			},
			expectedErr: fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreAtLeastOneVault)),
		},
		{
			checkNote: "invalid: url",
			store: &esv1beta1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						OnePassword: &esv1beta1.OnePasswordProvider{
							Auth: &esv1beta1.OnePasswordAuth{
								SecretRef: &esv1beta1.OnePasswordAuthSecretRef{
									ConnectToken: esmeta.SecretKeySelector{
										Name: mySecret,
										Key:  token,
									},
								},
							},
							ConnectHost: ":/invalid.invalid",
							Vaults: map[string]int{
								myVault: 1,
							},
						},
					},
				},
			},
			expectedErr: fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreInvalidConnectHost, fmt.Errorf("parse \":/invalid.invalid\": missing protocol scheme"))),
		},
	}

	// run the tests
	for _, tc := range testCases {
		var err error
		if tc.store == nil {
			err = validateStore(tc.clusterStore)
		} else {
			err = validateStore(tc.store)
		}
		notes := fmt.Sprintf("Check: '%s'", tc.checkNote)
		if tc.expectedErr == nil && err != nil {
			// expected no error, got one
			t.Errorf(validateStoreErrFormat, notes, nil, err)
		}
		if tc.expectedErr != nil && err == nil {
			// expected an error, didn't get one
			t.Errorf(validateStoreErrFormat, notes, tc.expectedErr.Error(), nil)
		}
		if tc.expectedErr != nil && err != nil && err.Error() != tc.expectedErr.Error() {
			// expected an error, got the wrong one
			t.Errorf(validateStoreErrFormat, notes, tc.expectedErr.Error(), err.Error())
		}
	}
}

// most functionality is tested in TestFindItem
//
//	here we just check that an empty Property defaults to "password",
//	files are loaded, and
//	the data or errors are properly returned
func TestGetSecret(t *testing.T) {
	type check struct {
		checkNote     string
		ref           esv1beta1.ExternalSecretDataRemoteRef
		expectedValue string
		expectedErr   error
	}

	type testCase struct {
		setupNote string
		provider  *ProviderOnePassword
		checks    []check
	}

	testCases := []testCase{
		{
			setupNote: "one vault, one item, two fields",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1},
				client: fake.NewMockClient().
					AddPredictableVault(myVault).
					AddPredictableItemWithField(myVault, myItem, key1, value1).
					AppendItemField(myVaultID, myItemID, onepassword.ItemField{
						Label: password,
						Value: value2,
					}),
			},
			checks: []check{
				{
					checkNote: key1,
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key:      myItem,
						Property: key1,
					},
					expectedValue: value1,
					expectedErr:   nil,
				},
				{
					checkNote: "'password' (defaulted property)",
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key: myItem,
					},
					expectedValue: value2,
					expectedErr:   nil,
				},
				{
					checkNote: "'ref.version' not implemented",
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key:      myItem,
						Property: key1,
						Version:  "123",
					},
					expectedErr: fmt.Errorf(errVersionNotImplemented),
				},
			},
		},
		{
			setupNote: "files are loaded",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1},
				client: fake.NewMockClient().
					AddPredictableVault(myVault).
					AppendItem(myVaultID, onepassword.Item{
						ID:       myItemID,
						Title:    myItem,
						Vault:    onepassword.ItemVault{ID: myVaultID},
						Category: documentCategory,
						Files: []*onepassword.File{
							{
								ID:   myFilePNGID,
								Name: myFilePNG,
							},
						},
					}).
					SetFileContents(myFilePNG, []byte(myContents)),
			},
			checks: []check{
				{
					checkNote: "file named my-file.png",
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key:      myItem,
						Property: myFilePNG,
					},
					expectedValue: myContents,
					expectedErr:   nil,
				},
				{
					checkNote: "empty ref.Property",
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key: myItem,
					},
					expectedValue: myContents,
					expectedErr:   nil,
				},
				{
					checkNote: "file non existent",
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key:      myItem,
						Property: "you-cant-find-me.png",
					},
					expectedErr: fmt.Errorf(errDocumentNotFound, fmt.Errorf("'my-item', 'you-cant-find-me.png'")),
				},
			},
		},
		{
			setupNote: "one vault, one item, two fields w/ same Label",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1},
				client: fake.NewMockClient().
					AddPredictableVault(myVault).
					AddPredictableItemWithField(myVault, myItem, key1, value1).
					AppendItemField(myVaultID, myItemID, onepassword.ItemField{
						Label: key1,
						Value: value2,
					}),
			},
			checks: []check{
				{
					checkNote: key1,
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key:      myItem,
						Property: key1,
					},
					expectedErr: fmt.Errorf(errExpectedOneField, fmt.Errorf("'key1' in 'my-item', got 2")),
				},
			},
		},
	}

	// run the tests
	for _, tc := range testCases {
		for _, check := range tc.checks {
			got, err := tc.provider.GetSecret(context.Background(), check.ref)
			notes := fmt.Sprintf(setupCheckFormat, tc.setupNote, check.checkNote)
			if check.expectedErr == nil && err != nil {
				// expected no error, got one
				t.Errorf(getSecretErrFormat, notes, nil, err)
			}
			if check.expectedErr != nil && err == nil {
				// expected an error, didn't get one
				t.Errorf(getSecretErrFormat, notes, check.expectedErr.Error(), nil)
			}
			if check.expectedErr != nil && err != nil && err.Error() != check.expectedErr.Error() {
				// expected an error, got the wrong one
				t.Errorf(getSecretErrFormat, notes, check.expectedErr.Error(), err.Error())
			}
			if check.expectedValue != "" {
				if check.expectedValue != string(got) {
					// expected a predefined value, got something else
					t.Errorf(getSecretErrFormat, notes, check.expectedValue, string(got))
				}
			}
		}
	}
}

// most functionality is tested in TestFindItem. here we just check:
//
//	all keys are fetched and the map is compiled correctly,
//	files are loaded, and the data or errors are properly returned.
func TestGetSecretMap(t *testing.T) {
	type check struct {
		checkNote   string
		ref         esv1beta1.ExternalSecretDataRemoteRef
		expectedMap map[string][]byte
		expectedErr error
	}

	type testCase struct {
		setupNote string
		provider  *ProviderOnePassword
		checks    []check
	}

	testCases := []testCase{
		{
			setupNote: "one vault, one item, two fields",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1},
				client: fake.NewMockClient().
					AddPredictableVault(myVault).
					AddPredictableItemWithField(myVault, myItem, key1, value1).
					AppendItemField(myVaultID, myItemID, onepassword.ItemField{
						Label: password,
						Value: value2,
					}),
			},
			checks: []check{
				{
					checkNote: "all Properties",
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key: myItem,
					},
					expectedMap: map[string][]byte{
						key1:     []byte(value1),
						password: []byte(value2),
					},
					expectedErr: nil,
				},
				{
					checkNote: "limit by Property",
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key:      myItem,
						Property: password,
					},
					expectedMap: map[string][]byte{
						password: []byte(value2),
					},
					expectedErr: nil,
				},
				{
					checkNote: "'ref.version' not implemented",
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key:      myItem,
						Property: key1,
						Version:  "123",
					},
					expectedErr: fmt.Errorf(errVersionNotImplemented),
				},
			},
		},
		{
			setupNote: "files",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1},
				client: fake.NewMockClient().
					AddPredictableVault(myVault).
					AppendItem(myVaultID, onepassword.Item{
						ID:       myItemID,
						Title:    myItem,
						Vault:    onepassword.ItemVault{ID: myVaultID},
						Category: documentCategory,
						Files: []*onepassword.File{
							{
								ID:   myFilePNGID,
								Name: myFilePNG,
							},
							{
								ID:   myFile2ID,
								Name: myFile2PNG,
							},
						},
					}).
					SetFileContents(myFilePNG, []byte(myContents)).
					SetFileContents(myFile2PNG, []byte(myContents2)),
			},
			checks: []check{
				{
					checkNote: "all Properties",
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key: myItem,
					},
					expectedMap: map[string][]byte{
						myFilePNG:  []byte(myContents),
						myFile2PNG: []byte(myContents2),
					},
					expectedErr: nil,
				},
				{
					checkNote: "limit by Property",
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key:      myItem,
						Property: myFilePNG,
					},
					expectedMap: map[string][]byte{
						myFilePNG: []byte(myContents),
					},
					expectedErr: nil,
				},
			},
		},
		{
			setupNote: "one vault, one item, two fields w/ same Label",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1},
				client: fake.NewMockClient().
					AddPredictableVault(myVault).
					AddPredictableItemWithField(myVault, myItem, key1, value1).
					AppendItemField(myVaultID, myItemID, onepassword.ItemField{
						Label: key1,
						Value: value2,
					}),
			},
			checks: []check{
				{
					checkNote: key1,
					ref: esv1beta1.ExternalSecretDataRemoteRef{
						Key: myItem,
					},
					expectedMap: nil,
					expectedErr: fmt.Errorf(errExpectedOneField, fmt.Errorf("'key1' in 'my-item', got 2")),
				},
			},
		},
	}

	// run the tests
	for _, tc := range testCases {
		for _, check := range tc.checks {
			gotMap, err := tc.provider.GetSecretMap(context.Background(), check.ref)
			notes := fmt.Sprintf(setupCheckFormat, tc.setupNote, check.checkNote)
			if check.expectedErr == nil && err != nil {
				// expected no error, got one
				t.Errorf(getSecretMapErrFormat, notes, nil, err)
			}
			if check.expectedErr != nil && err == nil {
				// expected an error, didn't get one
				t.Errorf(getSecretMapErrFormat, notes, check.expectedErr.Error(), nil)
			}
			if check.expectedErr != nil && err != nil && err.Error() != check.expectedErr.Error() {
				// expected an error, got the wrong one
				t.Errorf(getSecretMapErrFormat, notes, check.expectedErr.Error(), err.Error())
			}
			if !reflect.DeepEqual(check.expectedMap, gotMap) {
				// expected a predefined map, got something else
				t.Errorf(getSecretMapErrFormat, notes, check.expectedMap, gotMap)
			}
		}
	}
}

func TestGetAllSecrets(t *testing.T) {
	type check struct {
		checkNote   string
		ref         esv1beta1.ExternalSecretFind
		expectedMap map[string][]byte
		expectedErr error
	}

	type testCase struct {
		setupNote string
		provider  *ProviderOnePassword
		checks    []check
	}

	testCases := []testCase{
		{
			setupNote: "three vaults, three items, all different field Labels",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1, myOtherVault: 2, myNonMatchingVault: 3},
				client: fake.NewMockClient().
					AddPredictableVault(myVault).
					AddPredictableItemWithField(myVault, myItem, key1, value1).
					AppendItemField(myVaultID, myItemID, onepassword.ItemField{
						Label: key2,
						Value: value2,
					}).
					AddPredictableVault(myOtherVault).
					AddPredictableItemWithField(myOtherVault, myOtherItem, key3, value3).
					AppendItemField(myOtherVaultID, myOtherItemID, onepassword.ItemField{
						Label: key4,
						Value: value4,
					}).
					AddPredictableVault(myNonMatchingVault).
					AddPredictableItemWithField(myNonMatchingVault, myNonMatchingItem, "non-matching5", "value5").
					AppendItemField(myNonMatchingVaultID, myNonMatchingItemID, onepassword.ItemField{
						Label: "non-matching6",
						Value: "value6",
					}),
			},
			checks: []check{
				{
					checkNote: "find some with path only",
					ref: esv1beta1.ExternalSecretFind{
						Path: pointer.StringPtr(myItem),
					},
					expectedMap: map[string][]byte{
						key1: []byte(value1),
						key2: []byte(value2),
					},
					expectedErr: nil,
				},
				{
					checkNote: "find most with regex 'key*'",
					ref: esv1beta1.ExternalSecretFind{
						Name: &esv1beta1.FindName{
							RegExp: "key*",
						},
					},
					expectedMap: map[string][]byte{
						key1: []byte(value1),
						key2: []byte(value2),
						key3: []byte(value3),
						key4: []byte(value4),
					},
					expectedErr: nil,
				},
				{
					checkNote: "find some with regex 'key*' and path 'my-other-item'",
					ref: esv1beta1.ExternalSecretFind{
						Name: &esv1beta1.FindName{
							RegExp: "key*",
						},
						Path: pointer.StringPtr(myOtherItem),
					},
					expectedMap: map[string][]byte{
						key3: []byte(value3),
						key4: []byte(value4),
					},
					expectedErr: nil,
				},
				{
					checkNote: "find none with regex 'asdf*'",
					ref: esv1beta1.ExternalSecretFind{
						Name: &esv1beta1.FindName{
							RegExp: "asdf*",
						},
					},
					expectedMap: map[string][]byte{},
					expectedErr: nil,
				},
				{
					checkNote: "find none with path 'no-exist'",
					ref: esv1beta1.ExternalSecretFind{
						Name: &esv1beta1.FindName{
							RegExp: "key*",
						},
						Path: pointer.StringPtr("no-exist"),
					},
					expectedMap: map[string][]byte{},
					expectedErr: nil,
				},
				{
					checkNote: "error when find.tags",
					ref: esv1beta1.ExternalSecretFind{
						Name: &esv1beta1.FindName{
							RegExp: "key*",
						},
						Tags: map[string]string{
							"asdf": "fdas",
						},
					},
					expectedErr: fmt.Errorf(errTagsNotImplemented),
				},
			},
		},
		{
			setupNote: "3 vaults, 4 items, 5 files",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1, myOtherVault: 2, myNonMatchingVault: 3},
				client: fake.NewMockClient().
					// my-vault
					AddPredictableVault(myVault).
					AppendItem(myVaultID, onepassword.Item{
						ID:       myItemID,
						Title:    myItem,
						Vault:    onepassword.ItemVault{ID: myVaultID},
						Category: documentCategory,
						Files: []*onepassword.File{
							{
								ID:   myFilePNGID,
								Name: myFilePNG,
							},
							{
								ID:   mySecondFileTXTID,
								Name: mySecondFileTXT,
							},
						},
					}).
					SetFileContents(myFilePNG, []byte(myContents)).
					SetFileContents(mySecondFileTXT, []byte(mySecondContents)).
					AppendItem(myVaultID, onepassword.Item{
						ID:       "my-item-2-id",
						Title:    "my-item-2",
						Vault:    onepassword.ItemVault{ID: myVaultID},
						Category: documentCategory,
						Files: []*onepassword.File{
							{
								ID:   myFile2ID,
								Name: myFile2TXT,
							},
						},
					}).
					SetFileContents(myFile2TXT, []byte(myContents2)).
					// my-other-vault
					AddPredictableVault(myOtherVault).
					AppendItem(myOtherVaultID, onepassword.Item{
						ID:       myOtherItemID,
						Title:    myOtherItem,
						Vault:    onepassword.ItemVault{ID: myOtherVaultID},
						Category: documentCategory,
						Files: []*onepassword.File{
							{
								ID:   myOtherFilePNGID,
								Name: myOtherFilePNG,
							},
						},
					}).
					SetFileContents(myOtherFilePNG, []byte(myOtherContents)).
					// my-non-matching-vault
					AddPredictableVault(myNonMatchingVault).
					AppendItem(myNonMatchingVaultID, onepassword.Item{
						ID:       myNonMatchingItemID,
						Title:    myNonMatchingItem,
						Vault:    onepassword.ItemVault{ID: myNonMatchingVaultID},
						Category: documentCategory,
						Files: []*onepassword.File{
							{
								ID:   nonMatchingFilePNGID,
								Name: nonMatchingFilePNG,
							},
						},
					}).
					SetFileContents(nonMatchingFilePNG, []byte(nonMatchingContents)),
			},
			checks: []check{
				{
					checkNote: "find most with regex '^my-*'",
					ref: esv1beta1.ExternalSecretFind{
						Name: &esv1beta1.FindName{
							RegExp: "^my-*",
						},
					},
					expectedMap: map[string][]byte{
						myFilePNG:       []byte(myContents),
						mySecondFileTXT: []byte(mySecondContents),
						myFile2TXT:      []byte(myContents2),
						myOtherFilePNG:  []byte(myOtherContents),
					},
					expectedErr: nil,
				},
				{
					checkNote: "find some with regex '^my-*' and path 'my-other-item'",
					ref: esv1beta1.ExternalSecretFind{
						Name: &esv1beta1.FindName{
							RegExp: "^my-*",
						},
						Path: pointer.StringPtr(myOtherItem),
					},
					expectedMap: map[string][]byte{
						myOtherFilePNG: []byte(myOtherContents),
					},
					expectedErr: nil,
				},
				{
					checkNote: "find none with regex '^asdf*'",
					ref: esv1beta1.ExternalSecretFind{
						Name: &esv1beta1.FindName{
							RegExp: "^asdf*",
						},
					},
					expectedMap: map[string][]byte{},
					expectedErr: nil,
				},
				{
					checkNote: "find none with path 'no-exist'",
					ref: esv1beta1.ExternalSecretFind{
						Name: &esv1beta1.FindName{
							RegExp: "^my-*",
						},
						Path: pointer.StringPtr("no-exist"),
					},
					expectedMap: map[string][]byte{},
					expectedErr: nil,
				},
			},
		},
		{
			setupNote: "two fields/files with same name, first one wins",
			provider: &ProviderOnePassword{
				vaults: map[string]int{myVault: 1, myOtherVault: 2},
				client: fake.NewMockClient().
					// my-vault
					AddPredictableVault(myVault).
					AddPredictableItemWithField(myVault, myItem, key1, value1).
					AddPredictableItemWithField(myVault, "my-second-item", key1, "value-second").
					AppendItem(myVaultID, onepassword.Item{
						ID:       "file-item-id",
						Title:    "file-item",
						Vault:    onepassword.ItemVault{ID: myVaultID},
						Category: documentCategory,
						Files: []*onepassword.File{
							{
								ID:   filePNGID,
								Name: filePNG,
							},
						},
					}).
					SetFileContents(filePNG, []byte(myContents)).
					AppendItem(myVaultID, onepassword.Item{
						ID:       "file-item-2-id",
						Title:    "file-item-2",
						Vault:    onepassword.ItemVault{ID: myVaultID},
						Category: documentCategory,
						Files: []*onepassword.File{
							{
								ID:   "file-2-id",
								Name: filePNG,
							},
						},
					}).
					// my-other-vault
					AddPredictableVault(myOtherVault).
					AddPredictableItemWithField(myOtherVault, myOtherItem, key1, "value-other").
					AppendItem(myOtherVaultID, onepassword.Item{
						ID:       "file-item-other-id",
						Title:    "file-item-other",
						Vault:    onepassword.ItemVault{ID: myOtherVaultID},
						Category: documentCategory,
						Files: []*onepassword.File{
							{
								ID:   "other-file-id",
								Name: filePNG,
							},
						},
					}),
			},
			checks: []check{
				{
					checkNote: "find fields with regex '^key*'",
					ref: esv1beta1.ExternalSecretFind{
						Name: &esv1beta1.FindName{
							RegExp: "^key*",
						},
					},
					expectedMap: map[string][]byte{
						key1: []byte(value1),
					},
					expectedErr: nil,
				},
				{
					checkNote: "find files with regex '^file*item*'",
					ref: esv1beta1.ExternalSecretFind{
						Name: &esv1beta1.FindName{
							RegExp: "^file*",
						},
					},
					expectedMap: map[string][]byte{
						filePNG: []byte(myContents),
					},
					expectedErr: nil,
				},
			},
		},
	}

	// run the tests
	for _, tc := range testCases {
		for _, check := range tc.checks {
			gotMap, err := tc.provider.GetAllSecrets(context.Background(), check.ref)
			notes := fmt.Sprintf(setupCheckFormat, tc.setupNote, check.checkNote)
			if check.expectedErr == nil && err != nil {
				// expected no error, got one
				t.Fatalf(getAllSecretsErrFormat, notes, nil, err)
			}
			if check.expectedErr != nil && err == nil {
				// expected an error, didn't get one
				t.Errorf(getAllSecretsErrFormat, notes, check.expectedErr.Error(), nil)
			}
			if check.expectedErr != nil && err != nil && err.Error() != check.expectedErr.Error() {
				// expected an error, got the wrong one
				t.Errorf(getAllSecretsErrFormat, notes, check.expectedErr.Error(), err.Error())
			}
			if !reflect.DeepEqual(check.expectedMap, gotMap) {
				// expected a predefined map, got something else
				t.Errorf(getAllSecretsErrFormat, notes, check.expectedMap, gotMap)
			}
		}
	}
}

func TestSortVaults(t *testing.T) {
	type testCase struct {
		vaults   map[string]int
		expected []string
	}

	testCases := []testCase{
		{
			vaults: map[string]int{
				one:   1,
				three: 3,
				two:   2,
			},
			expected: []string{
				one,
				two,
				three,
			},
		},
		{
			vaults: map[string]int{
				"four": 100,
				one:    1,
				three:  3,
				two:    2,
			},
			expected: []string{
				one,
				two,
				three,
				"four",
			},
		},
	}

	// run the tests
	for _, tc := range testCases {
		got := sortVaults(tc.vaults)
		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("onepassword.sortVaults(...): -expected, +got:\n-%#v\n+%#v\n", tc.expected, got)
		}
	}
}

func TestHasUniqueVaultNumbers(t *testing.T) {
	type testCase struct {
		vaults   map[string]int
		expected bool
	}

	testCases := []testCase{
		{
			vaults: map[string]int{
				one:   1,
				three: 3,
				two:   2,
			},
			expected: true,
		},
		{
			vaults: map[string]int{
				"four":  100,
				one:     1,
				three:   3,
				two:     2,
				"eight": 100,
			},
			expected: false,
		},
		{
			vaults: map[string]int{
				one:   1,
				"1":   1,
				three: 3,
				two:   2,
			},
			expected: false,
		},
	}

	// run the tests
	for _, tc := range testCases {
		got := hasUniqueVaultNumbers(tc.vaults)
		if got != tc.expected {
			t.Errorf("onepassword.hasUniqueVaultNumbers(...): -expected, +got:\n-%#v\n+%#v\n", tc.expected, got)
		}
	}
}
