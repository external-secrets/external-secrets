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
	accountName string
	systemName  string
	apiKey      string
	password    string
}

type fakeA2A struct {
	passwords  map[string]string
	accounts   []accountEntry
	apiKeys    map[string][]sg.APIKey
	lastFilter string
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

func (f *fakeA2A) RetrieveAPIKey(_ context.Context, apiKey sg.Secret) ([]sg.APIKey, error) {
	if f.apiKeys != nil {
		keys, ok := f.apiKeys[apiKey.ExposeString()]
		if ok {
			return keys, nil
		}
	}
	return nil, nil
}

func (f *fakeA2A) SetPassword(_ context.Context, apiKey, newPassword sg.Secret) error {
	if f.passwords == nil {
		f.passwords = map[string]string{}
	}
	f.passwords[apiKey.ExposeString()] = newPassword.ExposeString()
	return nil
}

func (f *fakeA2A) GetRetrievableAccounts(_ context.Context, filter string) ([]sg.A2ARetrievableAccount, error) {
	f.lastFilter = filter
	out := make([]sg.A2ARetrievableAccount, 0, len(f.accounts))
	for _, account := range f.accounts {
		if filter != "" && filter != buildAccountSystemFilter(account.accountName, account.systemName) {
			continue
		}
		out = append(out, sg.A2ARetrievableAccount{
			AccountName: account.accountName,
			AssetName:   account.systemName,
			APIKey:      sg.NewSecretString(account.apiKey),
		})
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
