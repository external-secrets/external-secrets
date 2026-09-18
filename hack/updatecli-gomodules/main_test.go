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

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestVersionFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		version     string
		wantKind    string
		wantPattern string
		wantError   bool
	}{
		{name: "stable", version: "v1.2.3", wantKind: "semver", wantPattern: "1.x"},
		{name: "major zero", version: "v0.9.0", wantKind: "semver", wantPattern: "0.x"},
		{name: "prerelease", version: "v2.0.0-beta.1", wantKind: "semver", wantPattern: "2.x.x-0"},
		{name: "pseudo-version", version: "v0.0.0-20260911184034-7970a1e230da", wantKind: "latest"},
		{name: "invalid", version: "main", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			kind, pattern, err := versionFilter(test.version)
			if test.wantError {
				if err == nil {
					t.Fatal("versionFilter() expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("versionFilter() error = %v", err)
			}
			if kind != test.wantKind || pattern != test.wantPattern {
				t.Errorf("versionFilter() = (%q, %q), want (%q, %q)", kind, pattern, test.wantKind, test.wantPattern)
			}
		})
	}
}

func TestModuleFilesUsesCanonicalModules(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod": `module github.com/external-secrets/external-secrets

go 1.26.0

replace (
	github.com/external-secrets/external-secrets/apis => ./apis
	example.com/external => ./ignored
)
`,
		"apis/go.mod": "module github.com/external-secrets/external-secrets/apis\n\ngo 1.26.0\n",
		"e2e/go.mod":  "module github.com/external-secrets/external-secrets-e2e\n\ngo 1.26.0\n",
		"hack/tools/gen-crd-api-reference-docs/go.mod": "module github.com/external-secrets/gen-crd-api-reference-docs\n\ngo 1.26.0\n",
		"ignored/go.mod": "module example.com/external\n\ngo 1.26.0\n",
		"untracked-module-that-must-not-be-read/go.mod": "module example.com/untracked\n\ngo 1.26.0\n",
	}
	for file, contents := range files {
		path := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create directory for %s: %v", file, err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
	}

	got, err := moduleFiles(root)
	if err != nil {
		t.Fatalf("moduleFiles() error = %v", err)
	}
	want := []string{
		"apis/go.mod",
		"e2e/go.mod",
		"go.mod",
		"hack/tools/gen-crd-api-reference-docs/go.mod",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("moduleFiles() = %#v, want %#v", got, want)
	}
}

func TestRenderManifests(t *testing.T) {
	t.Parallel()

	data := manifestData{
		ModuleFiles: []string{"go.mod", "apis/go.mod"},
		Dependencies: []manifestDependency{
			{
				Index:        0,
				Module:       "example.com/module",
				Kind:         "semver",
				Pattern:      "1.x",
				MatchPattern: `(?m)^(\\s*)example\\.com/module\\s+\\S+(\\s+// indirect)$`,
				Targets: []manifestTarget{
					{Index: 0, Path: "go.mod"},
					{Index: 1, Path: "apis/go.mod", FileTarget: true},
				},
			},
		},
	}

	gomodules, err := renderManifest("gomodules.yaml.tmpl", data)
	if err != nil {
		t.Fatalf("renderManifest(gomodules) error = %v", err)
	}
	golang, err := renderManifest("golang.yaml.tmpl", data)
	if err != nil {
		t.Fatalf("renderManifest(golang) error = %v", err)
	}
	for _, expected := range []string{
		"kind: golang/module",
		"module: 'example.com/module'",
		"file: 'apis/go.mod'",
		`{{ source "dependency_0" }}`,
	} {
		if !strings.Contains(string(gomodules), expected) {
			t.Errorf("gomodules manifest does not contain %q", expected)
		}
	}
	for _, expected := range []string{"go-version_0:", "file: 'apis/go.mod'"} {
		if !strings.Contains(string(golang), expected) {
			t.Errorf("golang manifest does not contain %q", expected)
		}
	}
}

func TestDirectModules(t *testing.T) {
	t.Parallel()

	mod := goMod{
		Require: []requirement{
			{Path: "example.com/direct", Version: "v1.0.0"},
			{Path: "example.com/transitive", Version: "v2.0.0", Indirect: true},
			{Path: "example.com/tool", Version: "v3.0.0", Indirect: true},
		},
		Tool: []tool{{Path: "example.com/tool/cmd/generate"}},
	}

	got, err := directModules("go.mod", mod)
	if err != nil {
		t.Fatalf("directModules() error = %v", err)
	}
	want := map[string]requirement{
		"example.com/direct": {Path: "example.com/direct", Version: "v1.0.0"},
		"example.com/tool":   {Path: "example.com/tool", Version: "v3.0.0", Indirect: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("directModules() = %#v, want %#v", got, want)
	}
}

func TestDirectModulesRejectsToolWithoutRequirement(t *testing.T) {
	t.Parallel()

	_, err := directModules("go.mod", goMod{Tool: []tool{{Path: "example.com/tool/cmd/generate"}}})
	if err == nil {
		t.Fatal("directModules() expected an error")
	}
}
