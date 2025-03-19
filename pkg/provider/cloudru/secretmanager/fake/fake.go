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

	smsV2 "github.com/cloudru-tech/secret-manager-sdk/api/v2"

	"github.com/external-secrets/external-secrets/pkg/provider/cloudru/secretmanager/adapter"
)

type MockSecretProvider struct {
	ListSecretsFns  []func() ([]*smsV2.Secret, error)
	AccessSecretFns []func() ([]byte, error)
}

func (m *MockSecretProvider) ListSecrets(_ context.Context, _ *adapter.ListSecretsRequest) ([]*smsV2.Secret, error) {
	fn := m.ListSecretsFns[0]
	if len(m.ListSecretsFns) > 1 {
		m.ListSecretsFns = m.ListSecretsFns[1:]
	} else {
		m.ListSecretsFns = nil
	}

	return fn()
}

func (m *MockSecretProvider) AccessSecretVersionByPath(_ context.Context, _, _ string, _ *int32) ([]byte, error) {
	fn := m.AccessSecretFns[0]
	if len(m.AccessSecretFns) > 1 {
		m.AccessSecretFns = m.AccessSecretFns[1:]
	} else {
		m.AccessSecretFns = nil
	}
	return fn()
}

func (m *MockSecretProvider) AccessSecretVersion(_ context.Context, _, _ string) ([]byte, error) {
	fn := m.AccessSecretFns[0]
	if len(m.AccessSecretFns) > 1 {
		m.AccessSecretFns = m.AccessSecretFns[1:]
	} else {
		m.AccessSecretFns = nil
	}
	return fn()
}

func (m *MockSecretProvider) MockListSecrets(list []*smsV2.Secret, err error) {
	m.ListSecretsFns = append(m.ListSecretsFns, func() ([]*smsV2.Secret, error) { return list, err })
}

func (m *MockSecretProvider) MockAccessSecretVersion(data []byte, err error) {
	m.AccessSecretFns = append(m.AccessSecretFns, func() ([]byte, error) { return data, err })
}

func (m *MockSecretProvider) MockAccessSecretVersionPath(data []byte, err error) {
	m.AccessSecretFns = append(m.AccessSecretFns, func() ([]byte, error) { return data, err })
}

func (m *MockSecretProvider) Close() error { return nil }
