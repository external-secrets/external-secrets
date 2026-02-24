/*
Copyright © 2025 ESO Maintainer Team

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

package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// ValidationProvider is a simple provider that we can use without cyclic import.
type ValidationProvider struct {
	esv1beta1.Provider
}

func (v *ValidationProvider) ValidateStore(_ esv1beta1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

func TestValidateSecretStore(t *testing.T) {
	tests := []struct {
		name      string
		obj       *esv1beta1.SecretStore
		mock      func()
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "valid regex",
			obj: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Conditions: []esv1beta1.ClusterSecretStoreCondition{
						{
							NamespaceRegexes: []string{`.*`},
						},
					},
					Provider: &esv1beta1.SecretStoreProvider{
						AWS: &esv1beta1.AWSProvider{},
					},
				},
			},
			mock: func() {
				esv1beta1.ForceRegister(&ValidationProvider{}, &esv1beta1.SecretStoreProvider{
					AWS: &esv1beta1.AWSProvider{},
				})
			},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "invalid regex",
			obj: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Conditions: []esv1beta1.ClusterSecretStoreCondition{
						{
							NamespaceRegexes: []string{`\1`},
						},
					},
					Provider: &esv1beta1.SecretStoreProvider{
						AWS: &esv1beta1.AWSProvider{},
					},
				},
			},
			mock: func() {
				esv1beta1.ForceRegister(&ValidationProvider{}, &esv1beta1.SecretStoreProvider{
					AWS: &esv1beta1.AWSProvider{},
				})
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "failed to compile 0th namespace regex in 0th condition: error parsing regexp: invalid escape sequence: `\\1`")
			},
		},
		{
			name: "multiple errors",
			obj: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Conditions: []esv1beta1.ClusterSecretStoreCondition{
						{
							NamespaceRegexes: []string{`\1`, `\2`},
						},
					},
					Provider: &esv1beta1.SecretStoreProvider{
						AWS: &esv1beta1.AWSProvider{},
					},
				},
			},
			mock: func() {
				esv1beta1.ForceRegister(&ValidationProvider{}, &esv1beta1.SecretStoreProvider{
					AWS: &esv1beta1.AWSProvider{},
				})
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(
					t,
					err,
					"failed to compile 0th namespace regex in 0th condition: error parsing regexp: invalid escape sequence: `\\1`\nfailed to compile 1th namespace regex in 0th condition: error parsing regexp: invalid escape sequence: `\\2`",
				)
			},
		},
		{
			name: "secret store must have only a single backend",
			obj: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						AWS:   &esv1beta1.AWSProvider{},
						GCPSM: &esv1beta1.GCPSMProvider{},
					},
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "store error for : secret stores must only have exactly one backend specified, found 2")
			},
		},
		{
			name: "no registered store backend",
			obj: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Conditions: []esv1beta1.ClusterSecretStoreCondition{
						{
							Namespaces: []string{"default"},
						},
					},
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "store error for : secret stores must only have exactly one backend specified, found 0")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mock != nil {
				tt.mock()
			}

			_, err := validateStore(tt.obj)
			tt.assertErr(t, err)
		})
	}
}
