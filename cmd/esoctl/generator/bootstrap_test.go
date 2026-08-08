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

package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const typesClusterFixture = `package v1alpha1

// GeneratorKind represents a kind of generator.
// +kubebuilder:validation:Enum=Password;MFA
type GeneratorKind string

const (
	// GeneratorKindPassword represents a password generator.
	GeneratorKindPassword GeneratorKind = "Password"
	// GeneratorKindMFA represents a Multi-Factor Authentication generator.
	GeneratorKindMFA GeneratorKind = "MFA"
)

// GeneratorSpec defines the configuration for various supported generator types.
type GeneratorSpec struct {
	PasswordSpec *PasswordSpec ` + "`json:\"passwordSpec,omitempty\"`" + `
	MFASpec      *MFASpec      ` + "`json:\"mfaSpec,omitempty\"`" + `
}
`

const registerFixture = `package v1alpha1

var (
	// PasswordKind is the kind name for Password resource.
	PasswordKind = reflect.TypeFor[Password]().Name()
	// MFAKind is the kind name for MFA resource.
	MFAKind = reflect.TypeFor[MFA]().Name()
)

func init() {
	SchemeBuilder.Register(&Password{}, &PasswordList{})
	SchemeBuilder.Register(&MFA{}, &MFAList{})
}
`

func TestLowerCamel(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "Password", want: "password"},
		{name: "ServiceAccountToken", want: "serviceAccountToken"},
		{name: "ECRAuthorizationToken", want: "ecrAuthorizationToken"},
		{name: "STSSessionToken", want: "stsSessionToken"},
		{name: "SSHKey", want: "sshKey"},
		{name: "UUID", want: "uuid"},
		{name: "MFA", want: "mfa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, lowerCamel(tt.name))
		})
	}
}

// writeFixture lays out the minimum tree the update functions expect and
// returns the repository root.
func writeFixture(t *testing.T, relPath, content string) string {
	t.Helper()

	root := t.TempDir()
	full := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o600))

	return root
}

func TestUpdateTypesClusterFile(t *testing.T) {
	relPath := filepath.Join("apis", "generators", "v1alpha1", "types_cluster.go")
	root := writeFixture(t, relPath, typesClusterFixture)

	cfg := Config{
		GeneratorName: "ECRAuthorizationToken",
		PackageName:   "ecr",
		GeneratorKind: "GeneratorKindECRAuthorizationToken",
	}
	require.NoError(t, updateTypesClusterFile(root, cfg))

	data, err := os.ReadFile(filepath.Join(root, relPath))
	require.NoError(t, err)
	got := string(data)

	// The validation enum must list the new kind, otherwise a ClusterGenerator
	// referencing it is rejected at admission time. This is not a compile error,
	// so nothing else catches it.
	assert.Contains(t, got, "+kubebuilder:validation:Enum=Password;MFA;ECRAuthorizationToken")
	assert.Contains(t, got, `GeneratorKindECRAuthorizationToken GeneratorKind = "ECRAuthorizationToken"`)

	// The json tag is user-facing API surface and must be lowerCamelCase.
	assert.Contains(t, got, `json:"ecrAuthorizationTokenSpec,omitempty"`)
	assert.NotContains(t, got, `json:"ecrauthorizationtokenSpec,omitempty"`)
}

func TestUpdateRegisterKindFile(t *testing.T) {
	relPath := filepath.Join("apis", "generators", "v1alpha1", "register.go")
	root := writeFixture(t, relPath, registerFixture)

	cfg := Config{
		GeneratorName: "ECRAuthorizationToken",
		PackageName:   "ecr",
		GeneratorKind: "GeneratorKindECRAuthorizationToken",
	}
	require.NoError(t, updateRegisterKindFile(root, cfg))

	data, err := os.ReadFile(filepath.Join(root, relPath))
	require.NoError(t, err)
	got := string(data)

	assert.Contains(t, got, "ECRAuthorizationTokenKind = reflect.TypeFor[ECRAuthorizationToken]().Name()")
	assert.Contains(t, got, "SchemeBuilder.Register(&ECRAuthorizationToken{}, &ECRAuthorizationTokenList{})")
}
