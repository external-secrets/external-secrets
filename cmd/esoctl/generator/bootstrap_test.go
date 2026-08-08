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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const typesClusterPath = "apis/generators/v1alpha1/types_cluster.go"
const registerPath = "apis/generators/v1alpha1/register.go"

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

func ecrConfig() Config {
	return Config{
		GeneratorName: "ECRAuthorizationToken",
		PackageName:   "ecr",
		GeneratorKind: "GeneratorKindECRAuthorizationToken",
	}
}

// writeFixture lays out the minimum tree the update functions expect and returns
// the repository root together with the absolute path of the file.
func writeFixture(t *testing.T, relPath, content string) (root, file string) {
	t.Helper()

	root = t.TempDir()
	file = filepath.Join(root, filepath.FromSlash(relPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o750))
	require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

	return root, file
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(data)
}

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

func TestEnumListsKindMatchesWholeValues(t *testing.T) {
	const enum = "// +kubebuilder:validation:Enum=Password;MFA"

	assert.True(t, enumListsKind(enum, "Password"))
	assert.True(t, enumListsKind(enum, "MFA"))
	// A prefix of a listed value is not a listed value.
	assert.False(t, enumListsKind(enum, "Pass"))
	assert.False(t, enumListsKind(enum, "SSHKey"))
}

func TestUpdateTypesClusterFile(t *testing.T) {
	root, file := writeFixture(t, typesClusterPath, typesClusterFixture)
	require.NoError(t, updateTypesClusterFile(root, ecrConfig()))

	got := readFile(t, file)

	// The validation enum must list the new kind, otherwise a ClusterGenerator
	// naming it is rejected at admission time. That is not a compile error, so
	// nothing else catches it.
	assert.Contains(t, got, "+kubebuilder:validation:Enum=Password;MFA;ECRAuthorizationToken")
	assert.Contains(t, got, `GeneratorKindECRAuthorizationToken GeneratorKind = "ECRAuthorizationToken"`)

	// The json tag is user-facing API surface and must be lowerCamelCase.
	assert.Contains(t, got, `json:"ecrAuthorizationTokenSpec,omitempty"`)
	assert.NotContains(t, got, `json:"ecrauthorizationtokenSpec,omitempty"`)
}

func TestUpdateRegisterKindFile(t *testing.T) {
	root, file := writeFixture(t, registerPath, registerFixture)
	require.NoError(t, updateRegisterKindFile(root, ecrConfig()))

	got := readFile(t, file)

	assert.Contains(t, got, "ECRAuthorizationTokenKind = reflect.TypeFor[ECRAuthorizationToken]().Name()")
	assert.Contains(t, got, "SchemeBuilder.Register(&ECRAuthorizationToken{}, &ECRAuthorizationTokenList{})")
}

// A run that placed some fragments and not others leaves the tree half-patched.
// The next run has to finish the job, so presence is judged per fragment instead
// of being inferred from any single one of them.
func TestUpdateTypesClusterFileCompletesAPartialTree(t *testing.T) {
	root, file := writeFixture(t, typesClusterPath, typesClusterFixture)
	require.NoError(t, updateTypesClusterFile(root, ecrConfig()))

	// Drop the enum value back out, leaving the constant and the spec field in
	// place: this is the state the stale enum anchor used to produce.
	partial := strings.Replace(readFile(t, file), ";ECRAuthorizationToken", "", 1)
	require.NotContains(t, partial, ";ECRAuthorizationToken")
	require.NoError(t, os.WriteFile(file, []byte(partial), 0o600))

	require.NoError(t, updateTypesClusterFile(root, ecrConfig()))

	got := readFile(t, file)
	assert.Contains(t, got, "+kubebuilder:validation:Enum=Password;MFA;ECRAuthorizationToken")
	// What was already there must not be duplicated.
	assert.Equal(t, 1, strings.Count(got, `GeneratorKindECRAuthorizationToken GeneratorKind = "ECRAuthorizationToken"`))
	assert.Equal(t, 1, strings.Count(got, `json:"ecrAuthorizationTokenSpec,omitempty"`))
}

func TestUpdateRegisterKindFileCompletesAPartialTree(t *testing.T) {
	root, file := writeFixture(t, registerPath, registerFixture)
	require.NoError(t, updateRegisterKindFile(root, ecrConfig()))

	// Drop the scheme registration, keeping the kind constant.
	partial := strings.Replace(readFile(t, file),
		"\tSchemeBuilder.Register(&ECRAuthorizationToken{}, &ECRAuthorizationTokenList{})\n", "", 1)
	require.NotContains(t, partial, "SchemeBuilder.Register(&ECRAuthorizationToken{}")
	require.NoError(t, os.WriteFile(file, []byte(partial), 0o600))

	require.NoError(t, updateRegisterKindFile(root, ecrConfig()))

	got := readFile(t, file)
	assert.Contains(t, got, "SchemeBuilder.Register(&ECRAuthorizationToken{}, &ECRAuthorizationTokenList{})")
	assert.Equal(t, 1, strings.Count(got, "ECRAuthorizationTokenKind = reflect."))
}

func TestUpdateTypesClusterFileLeavesACompleteTreeAlone(t *testing.T) {
	root, file := writeFixture(t, typesClusterPath, typesClusterFixture)
	require.NoError(t, updateTypesClusterFile(root, ecrConfig()))

	complete := readFile(t, file)
	require.NoError(t, updateTypesClusterFile(root, ecrConfig()))

	assert.Equal(t, complete, readFile(t, file))
}

func TestUpdateRegisterKindFileLeavesACompleteTreeAlone(t *testing.T) {
	root, file := writeFixture(t, registerPath, registerFixture)
	require.NoError(t, updateRegisterKindFile(root, ecrConfig()))

	complete := readFile(t, file)
	require.NoError(t, updateRegisterKindFile(root, ecrConfig()))

	assert.Equal(t, complete, readFile(t, file))
}
