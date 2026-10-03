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
)

func TestToLowerCamel(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"ServiceAccountToken": "serviceAccountToken",
		"MFA":                 "mFA",
		"UUID":                "uUID",
		"Fake":                "fake",
		"":                    "",
		// Invalid UTF-8 lead byte must be preserved (not rewritten as U+FFFD).
		"\xff":     "\xff",
		"\xffName": "\xffName",
	}
	for in, want := range cases {
		if got := toLowerCamel(in); got != want {
			t.Fatalf("toLowerCamel(%q)=%q want %q", in, got, want)
		}
	}
}

func TestUpdateTypesClusterFileAppendsEnumAndCamelJSON(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "apis", "generators", "v1alpha1")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	// Minimal fixture: ends with MFA (not Grafana) so the old suffix check would fail.
	src := `package v1alpha1

// +kubebuilder:validation:Enum=Fake;Grafana;MFA
type GeneratorKind string

const (
	GeneratorKindFake GeneratorKind = "Fake"
	GeneratorKindMFA  GeneratorKind = "MFA"
)

type GeneratorSpec struct {
	FakeSpec *FakeSpec ` + "`json:\"fakeSpec,omitempty\"`" + `
	MFASpec  *MFASpec  ` + "`json:\"mfaSpec,omitempty\"`" + `
}
`
	path := filepath.Join(dir, "types_cluster.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		GeneratorName: "ServiceAccountToken",
		PackageName:   "serviceaccount",
		GeneratorKind: "GeneratorKindServiceAccountToken",
	}
	if err := updateTypesClusterFile(root, cfg); err != nil {
		t.Fatalf("updateTypesClusterFile: %v", err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, ";ServiceAccountToken") {
		t.Fatalf("enum not updated:\n%s", text)
	}
	if !strings.Contains(text, `json:"serviceAccountTokenSpec,omitempty"`) {
		t.Fatalf("json tag not lowerCamelCase:\n%s", text)
	}
	if !strings.Contains(text, "GeneratorKindServiceAccountToken") {
		t.Fatalf("const missing:\n%s", text)
	}
}

func TestUpdateRegisterKindFileUsesTypeFor(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "apis", "generators", "v1alpha1")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	src := `package v1alpha1

import "reflect"

var (
	MFAKind = reflect.TypeFor[MFA]().Name()
)

func init() {
	SchemeBuilder.Register(&MFA{}, &MFAList{})
}
`
	path := filepath.Join(dir, "register.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := Config{GeneratorName: "ServiceAccountToken", PackageName: "serviceaccount"}
	if err := updateRegisterKindFile(root, cfg); err != nil {
		t.Fatalf("updateRegisterKindFile: %v", err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "ServiceAccountTokenKind = reflect.TypeFor[ServiceAccountToken]().Name()") {
		t.Fatalf("TypeFor kind line missing:\n%s", text)
	}
	if strings.Contains(text, "reflect.TypeOf(ServiceAccountToken{})") {
		t.Fatalf("still emitted TypeOf:\n%s", text)
	}
	if !strings.Contains(text, "SchemeBuilder.Register(&ServiceAccountToken{}, &ServiceAccountTokenList{})") {
		t.Fatalf("SchemeBuilder register missing:\n%s", text)
	}
}

func TestUpdateTypesClusterFileErrorsWhenIncomplete(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "apis", "generators", "v1alpha1")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	// No enum / const / spec anchors.
	path := filepath.Join(dir, "types_cluster.go")
	if err := os.WriteFile(path, []byte("package v1alpha1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{GeneratorName: "X", PackageName: "x", GeneratorKind: "GeneratorKindX"}
	err := updateTypesClusterFile(root, cfg)
	if err == nil {
		t.Fatal("expected error on incomplete update")
	}
}

func TestEnumAnnotationHasValueExactMatch(t *testing.T) {
	t.Parallel()
	line := "// +kubebuilder:validation:Enum=Fake;ServiceAccountToken;MFA"
	if !enumAnnotationHasValue(line, "ServiceAccountToken") {
		t.Fatal("expected exact match for ServiceAccountToken")
	}
	if !enumAnnotationHasValue(line, "Fake") {
		t.Fatal("expected exact match for Fake")
	}
	// Substring / suffix / prefix collisions must not count as present.
	if enumAnnotationHasValue(line, "Token") {
		t.Fatal("suffix Token must not match ServiceAccountToken")
	}
	if enumAnnotationHasValue(line, "Service") {
		t.Fatal("prefix Service must not match ServiceAccountToken")
	}
	if enumAnnotationHasValue(line, "Account") {
		t.Fatal("substring Account must not match ServiceAccountToken")
	}
}

func TestUpdateTypesClusterFileRejectsEnumSubstringCollision(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "apis", "generators", "v1alpha1")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	// Existing enum ends with ServiceAccountToken; adding Token must still append.
	src := `package v1alpha1

// +kubebuilder:validation:Enum=Fake;ServiceAccountToken
type GeneratorKind string

const (
	GeneratorKindFake                 GeneratorKind = "Fake"
	GeneratorKindServiceAccountToken  GeneratorKind = "ServiceAccountToken"
)

type GeneratorSpec struct {
	FakeSpec                *FakeSpec                ` + "`json:\"fakeSpec,omitempty\"`" + `
	ServiceAccountTokenSpec *ServiceAccountTokenSpec ` + "`json:\"serviceAccountTokenSpec,omitempty\"`" + `
}
`
	path := filepath.Join(dir, "types_cluster.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		GeneratorName: "Token",
		PackageName:   "token",
		GeneratorKind: "GeneratorKindToken",
	}
	if err := updateTypesClusterFile(root, cfg); err != nil {
		t.Fatalf("updateTypesClusterFile: %v", err)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, ";Token") {
		t.Fatalf("Token not appended to enum:\n%s", text)
	}
	if !strings.Contains(text, `GeneratorKindToken GeneratorKind = "Token"`) {
		t.Fatalf("const missing:\n%s", text)
	}
	if !strings.Contains(text, `json:"tokenSpec,omitempty"`) {
		t.Fatalf("spec missing:\n%s", text)
	}
}

func TestUpdateTypesClusterFileCompletesPartialExisting(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "apis", "generators", "v1alpha1")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	// Const present but enum and spec missing — must not return success early.
	src := `package v1alpha1

// +kubebuilder:validation:Enum=Fake;MFA
type GeneratorKind string

const (
	GeneratorKindFake GeneratorKind = "Fake"
	GeneratorKindMFA  GeneratorKind = "MFA"
	GeneratorKindToken GeneratorKind = "Token"
)

type GeneratorSpec struct {
	FakeSpec *FakeSpec ` + "`json:\"fakeSpec,omitempty\"`" + `
	MFASpec  *MFASpec  ` + "`json:\"mfaSpec,omitempty\"`" + `
}
`
	path := filepath.Join(dir, "types_cluster.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		GeneratorName: "Token",
		PackageName:   "token",
		GeneratorKind: "GeneratorKindToken",
	}
	if err := updateTypesClusterFile(root, cfg); err != nil {
		t.Fatalf("updateTypesClusterFile: %v", err)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !contentHasExactEnumValue(text, "Token") {
		t.Fatalf("enum not completed:\n%s", text)
	}
	if !strings.Contains(text, `json:"tokenSpec,omitempty"`) {
		t.Fatalf("spec not completed:\n%s", text)
	}
	// Const must not be duplicated.
	if strings.Count(text, `GeneratorKindToken GeneratorKind = "Token"`) != 1 {
		t.Fatalf("const duplicated:\n%s", text)
	}
}

func TestUpdateRegisterKindFileCompletesPartialExisting(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "apis", "generators", "v1alpha1")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	// Kind const present but SchemeBuilder.Register missing.
	src := `package v1alpha1

import "reflect"

var (
	MFAKind   = reflect.TypeFor[MFA]().Name()
	TokenKind = reflect.TypeFor[Token]().Name()
)

func init() {
	SchemeBuilder.Register(&MFA{}, &MFAList{})
}
`
	path := filepath.Join(dir, "register.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{GeneratorName: "Token", PackageName: "token"}
	if err := updateRegisterKindFile(root, cfg); err != nil {
		t.Fatalf("updateRegisterKindFile: %v", err)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "SchemeBuilder.Register(&Token{}, &TokenList{})") {
		t.Fatalf("scheme register not completed:\n%s", text)
	}
	if strings.Count(text, "TokenKind = reflect.TypeFor[Token]().Name()") != 1 {
		t.Fatalf("kind const duplicated:\n%s", text)
	}
}
