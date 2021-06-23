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

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
)

type secretData struct {
	item           keyvault.SecretItem
	secretVersions map[string]keyvault.SecretBundle
	lastVersion    string
}

type keyData struct {
	item        keyvault.KeyItem
	keyVersions map[string]keyvault.KeyBundle
	lastVersion string
}

type AzureMock struct {
	mock.Mock
	knownSecrets map[string]map[string]*secretData
	knownKeys    map[string]map[string]*keyData
}

func (m *AzureMock) AddSecret(vaultBaseURL, secretName, secretContent string, enabled bool) string {
	uid := uuid.NewString()
	m.AddSecretWithVersion(vaultBaseURL, secretName, uid, secretContent, enabled)
	return uid
}

func (m *AzureMock) AddSecretWithVersion(vaultBaseURL, secretName, secretVersion, secretContent string, enabled bool) {
	if m.knownSecrets == nil {
		m.knownSecrets = make(map[string]map[string]*secretData)
	}
	if m.knownSecrets[vaultBaseURL] == nil {
		m.knownSecrets[vaultBaseURL] = make(map[string]*secretData)
	}

	secretItemID := vaultBaseURL + secretName
	secretBundleID := secretItemID + "/" + secretVersion

	if m.knownSecrets[vaultBaseURL][secretName] == nil {
		m.knownSecrets[vaultBaseURL][secretName] = &secretData{
			item:           newValidSecretItem(secretItemID, enabled),
			secretVersions: make(map[string]keyvault.SecretBundle),
		}
	} else {
		m.knownSecrets[vaultBaseURL][secretName].item.Attributes.Enabled = &enabled
	}
	m.knownSecrets[vaultBaseURL][secretName].secretVersions[secretVersion] = newValidSecretBundle(secretBundleID, secretContent)
	m.knownSecrets[vaultBaseURL][secretName].lastVersion = secretVersion
}

func newValidSecretBundle(secretBundleID, secretContent string) keyvault.SecretBundle {
	return keyvault.SecretBundle{
		Value: &secretContent,
		ID:    &secretBundleID,
	}
}

func newValidSecretItem(secretItemID string, enabled bool) keyvault.SecretItem {
	return keyvault.SecretItem{
		ID:         &secretItemID,
		Attributes: &keyvault.SecretAttributes{Enabled: &enabled},
	}
}

func (m *AzureMock) ExpectsGetSecret(ctx context.Context, vaultBaseURL, secretName, secretVersion string) {
	data := m.knownSecrets[vaultBaseURL][secretName]
	version := secretVersion
	if version == "" {
		version = data.lastVersion
	}
	returnValue := data.secretVersions[version]
	m.On("GetSecret", ctx, vaultBaseURL, secretName, secretVersion).Return(returnValue, nil)
}

func (m *AzureMock) ExpectsGetSecretsComplete(ctx context.Context, vaultBaseURL string, maxresults *int32) {
	secretMap := m.knownSecrets[vaultBaseURL]
	secretItems := make([]keyvault.SecretItem, len(secretMap))
	i := 0
	for _, value := range secretMap {
		secretItems[i] = value.item
		i++
	}
	firstPage := keyvault.SecretListResult{
		Value:    &secretItems,
		NextLink: nil,
	}
	returnValue := keyvault.NewSecretListResultIterator(keyvault.NewSecretListResultPage(firstPage, func(context.Context, keyvault.SecretListResult) (keyvault.SecretListResult, error) {
		return keyvault.SecretListResult{}, nil
	}))
	m.On("GetSecretsComplete", ctx, vaultBaseURL, maxresults).Return(returnValue, nil)
}

func (m *AzureMock) AddKey(vaultBaseURL, keyName string, key *keyvault.JSONWebKey, enabled bool) string {
	uid := uuid.NewString()
	m.AddKeyWithVersion(vaultBaseURL, keyName, uid, key, enabled)
	return uid
}

func (m *AzureMock) AddKeyWithVersion(vaultBaseURL, keyName, keyVersion string, key *keyvault.JSONWebKey, enabled bool) {
	if m.knownKeys == nil {
		m.knownKeys = make(map[string]map[string]*keyData)
	}
	if m.knownKeys[vaultBaseURL] == nil {
		m.knownKeys[vaultBaseURL] = make(map[string]*keyData)
	}

	keyItemID := vaultBaseURL + keyName

	if m.knownKeys[vaultBaseURL][keyName] == nil {
		m.knownKeys[vaultBaseURL][keyName] = &keyData{
			item:        newValidKeyItem(keyItemID, enabled),
			keyVersions: make(map[string]keyvault.KeyBundle),
		}
	} else {
		m.knownKeys[vaultBaseURL][keyName].item.Attributes.Enabled = &enabled
	}
	m.knownKeys[vaultBaseURL][keyName].keyVersions[keyVersion] = newValidKeyBundle(key)
	m.knownKeys[vaultBaseURL][keyName].lastVersion = keyVersion
}

func newValidKeyBundle(key *keyvault.JSONWebKey) keyvault.KeyBundle {
	return keyvault.KeyBundle{
		Key: key,
	}
}

func newValidKeyItem(keyItemID string, enabled bool) keyvault.KeyItem {
	return keyvault.KeyItem{
		Kid:        &keyItemID,
		Attributes: &keyvault.KeyAttributes{Enabled: &enabled},
	}
}

func (m *AzureMock) ExpectsGetKey(ctx context.Context, vaultBaseURL, keyName, keyVersion string) {
	data := m.knownKeys[vaultBaseURL][keyName]
	version := keyVersion
	if version == "" {
		version = data.lastVersion
	}
	returnValue := data.keyVersions[version]
	m.On("GetKey", ctx, vaultBaseURL, keyName, keyVersion).Return(returnValue, nil)
}

func (m *AzureMock) ExpectsGetKeysComplete(ctx context.Context, vaultBaseURL string, maxresults *int32) {
	keyMap := m.knownKeys[vaultBaseURL]
	keyItems := make([]keyvault.KeyItem, len(keyMap))
	i := 0
	for _, value := range keyMap {
		keyItems[i] = value.item
		i++
	}
	firstPage := keyvault.KeyListResult{
		Value:    &keyItems,
		NextLink: nil,
	}
	returnValue := keyvault.NewKeyListResultIterator(keyvault.NewKeyListResultPage(firstPage, func(context.Context, keyvault.KeyListResult) (keyvault.KeyListResult, error) {
		return keyvault.KeyListResult{}, nil
	}))
	m.On("GetKeysComplete", ctx, vaultBaseURL, maxresults).Return(returnValue, nil)
}

func (m *AzureMock) GetKey(ctx context.Context, vaultBaseURL, keyName, keyVersion string) (result keyvault.KeyBundle, err error) {
	args := m.Called(ctx, vaultBaseURL, keyName, keyVersion)
	return args.Get(0).(keyvault.KeyBundle), args.Error(1)
}

func (m *AzureMock) GetSecret(ctx context.Context, vaultBaseURL, secretName, secretVersion string) (result keyvault.SecretBundle, err error) {
	args := m.Called(ctx, vaultBaseURL, secretName, secretVersion)
	return args.Get(0).(keyvault.SecretBundle), args.Error(1)
}
func (m *AzureMock) GetCertificate(ctx context.Context, vaultBaseURL, certificateName, certificateVersion string) (result keyvault.CertificateBundle, err error) {
	args := m.Called(ctx, vaultBaseURL, certificateName, certificateVersion)
	return args.Get(0).(keyvault.CertificateBundle), args.Error(1)
}

func (m *AzureMock) GetSecretsComplete(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.SecretListResultIterator, err error) {
	args := m.Called(ctx, vaultBaseURL, maxresults)
	return args.Get(0).(keyvault.SecretListResultIterator), args.Error(1)
}

func (m *AzureMock) GetKeysComplete(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.KeyListResultIterator, err error) {
	args := m.Called(ctx, vaultBaseURL, maxresults)
	return args.Get(0).(keyvault.KeyListResultIterator), args.Error(1)
}
