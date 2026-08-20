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

package safeguard

import (
	"context"

	sg "github.com/OneIdentity/safeguard-go"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type accountEntry struct {
	accountID   int
	accountName string
	assetName   string
	apiKey      string
	password    string
}

type fakeA2A struct {
	passwords map[string]string
	accounts  []accountEntry
}

func (f *fakeA2A) RetrievePassword(_ context.Context, apiKey sg.Secret) (sg.Secret, error) {
	value, ok := f.passwords[apiKey.ExposeString()]
	if !ok {
		return sg.Secret{}, esv1.NoSecretError{}
	}
	return sg.NewSecretString(value), nil
}

func (f *fakeA2A) RetrievePrivateKey(_ context.Context, apiKey sg.Secret, _ sg.KeyFormat) (sg.Secret, error) {
	return sg.NewSecretString("private-key-for-" + apiKey.ExposeString()), nil
}

func (f *fakeA2A) RetrieveAPIKey(_ context.Context, _ sg.Secret) ([]sg.APIKey, error) {
	return nil, nil
}

func (f *fakeA2A) SetPassword(_ context.Context, apiKey sg.Secret, newPassword sg.Secret) error {
	if f.passwords == nil {
		f.passwords = map[string]string{}
	}
	f.passwords[apiKey.ExposeString()] = newPassword.ExposeString()
	return nil
}

func (f *fakeA2A) GetRetrievableAccounts(_ context.Context, _ string) ([]sg.A2ARetrievableAccount, error) {
	out := make([]sg.A2ARetrievableAccount, len(f.accounts))
	for i, account := range f.accounts {
		out[i] = sg.A2ARetrievableAccount{
			AccountID:   account.accountID,
			AccountName: account.accountName,
			AssetName:   account.assetName,
			APIKey:      sg.NewSecretString(account.apiKey),
		}
		if account.password != "" {
			if f.passwords == nil {
				f.passwords = map[string]string{}
			}
			f.passwords[account.apiKey] = account.password
		}
	}
	return out, nil
}

func (f *fakeA2A) Close() error { return nil }
