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

package v1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// ValidationProvider is a simple provider that we can use without cyclic import.
type ValidationProvider struct {
	esv1.Provider
}

func (v *ValidationProvider) ValidateStore(_ esv1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

func TestValidateSecretStore(t *testing.T) {
	tests := []struct {
		name        string
		obj         *esv1.SecretStore
		mock        func()
		assertWarns func(t *testing.T, warns admission.Warnings)
		assertErr   func(t *testing.T, err error)
	}{
		{
			name: "valid regex",
			obj: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Conditions: []esv1.ClusterSecretStoreCondition{
						{
							NamespaceRegexes: []string{`.*`},
						},
					},
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{},
					},
				},
			},
			mock: func() {
				esv1.ForceRegister(&ValidationProvider{}, &esv1.SecretStoreProvider{
					AWS: &esv1.AWSProvider{},
				}, esv1.MaintenanceStatusMaintained)
			},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			assertWarns: func(t *testing.T, warns admission.Warnings) {
				require.Equal(t, 0, len(warns))
			},
		},
		{
			name: "invalid regex",
			obj: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Conditions: []esv1.ClusterSecretStoreCondition{
						{
							NamespaceRegexes: []string{`\1`},
						},
					},
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{},
					},
				},
			},
			mock: func() {
				esv1.ForceRegister(&ValidationProvider{}, &esv1.SecretStoreProvider{
					AWS: &esv1.AWSProvider{},
				}, esv1.MaintenanceStatusMaintained)
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "failed to compile 0th namespace regex in 0th condition: error parsing regexp: invalid escape sequence: `\\1`")
			},
			assertWarns: func(t *testing.T, warns admission.Warnings) {
				require.Equal(t, 0, len(warns))
			},
		},
		{
			name: "multiple errors",
			obj: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Conditions: []esv1.ClusterSecretStoreCondition{
						{
							NamespaceRegexes: []string{`\1`, `\2`},
						},
					},
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{},
					},
				},
			},
			assertWarns: func(t *testing.T, warns admission.Warnings) {
				require.Equal(t, 0, len(warns))
			},

			mock: func() {
				esv1.ForceRegister(&ValidationProvider{}, &esv1.SecretStoreProvider{
					AWS: &esv1.AWSProvider{},
				}, esv1.MaintenanceStatusMaintained)
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
			obj: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS:   &esv1.AWSProvider{},
						GCPSM: &esv1.GCPSMProvider{},
					},
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "store error for : secret stores must only have exactly one backend specified, found 2")
			},
			assertWarns: func(t *testing.T, warns admission.Warnings) {
				require.Equal(t, 0, len(warns))
			},
		},
		{
			name: "no registered store backend",
			obj: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Conditions: []esv1.ClusterSecretStoreCondition{
						{
							Namespaces: []string{"default"},
						},
					},
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "store error for : secret stores must only have exactly one backend specified, found 0")
			},
			assertWarns: func(t *testing.T, warns admission.Warnings) {
				require.Equal(t, 0, len(warns))
			},
		},
		{
			name: "unmaintained warning",
			obj: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Conditions: []esv1.ClusterSecretStoreCondition{
						{
							NamespaceRegexes: []string{`.*`},
						},
					},
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{},
					},
				},
			},
			mock: func() {
				esv1.ForceRegister(&ValidationProvider{}, &esv1.SecretStoreProvider{
					AWS: &esv1.AWSProvider{},
				}, esv1.MaintenanceStatusNotMaintained)
			},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			assertWarns: func(t *testing.T, warns admission.Warnings) {
				require.Equal(t, 1, len(warns))
				assert.Equal(t, warns[0], fmt.Sprintf(warnStoreUnmaintained, ""))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mock != nil {
				tt.mock()
			}

			warns, err := validateStore(tt.obj)
			tt.assertErr(t, err)
			tt.assertWarns(t, warns)
		})
	}
}
