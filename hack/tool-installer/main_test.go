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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExpand(t *testing.T) {
	t.Parallel()

	got := expand("tool-{{version}}-{{version-no-v}}", "v1.2.3")
	if got != "tool-v1.2.3-1.2.3" {
		t.Fatalf("expand() = %q", got)
	}
}

func TestFormatJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "lock.json")
	if err := os.WriteFile(path, []byte(`{"z":1,"a":{"z":"last","a":"first"}}`), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := formatJSON(path); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"a\": {\n    \"a\": \"first\",\n    \"z\": \"last\"\n  },\n  \"z\": 1\n}\n"
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("formatJSON() = %q, want %q", got, want)
	}
	if err := formatJSON(path); err != nil {
		t.Fatalf("second formatJSON() call: %v", err)
	}
	gotAgain, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotAgain) != want {
		t.Fatalf("second formatJSON() call = %q, want %q", gotAgain, want)
	}
}

func TestInstalledVersions(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".installed-versions")
	want := map[string]installedVersion{
		"z-tool": {Version: "v2.0.0", SHA256: "bbbb"},
		"a-tool": {Version: "v1.0.0", SHA256: "aaaa"},
	}
	if err := writeInstalledVersions(path, want); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Index(data, []byte(`"a-tool"`)) > bytes.Index(data, []byte(`"z-tool"`)) {
		t.Fatalf("installed versions are not canonically ordered: %s", data)
	}
	got, err := readInstalledVersions(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, version := range want {
		if got[name] != version {
			t.Errorf("installed version %q = %#v, want %#v", name, got[name], version)
		}
	}
}

func TestHelmPluginVersion(t *testing.T) {
	t.Parallel()

	output := "NAME\tVERSION\tSTATUS\n schema\t2.2.1\tinstalled\nunittest v1.0.0 installed\n"
	for name, want := range map[string]string{
		"schema":   "2.2.1",
		"unittest": "1.0.0",
		"missing":  "",
	} {
		if got := helmPluginVersion(output, name); got != want {
			t.Errorf("helmPluginVersion(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestExtractFile(t *testing.T) {
	t.Parallel()

	var archive bytes.Buffer
	compressed := gzip.NewWriter(&archive)
	writer := tar.NewWriter(compressed)
	content := []byte("binary")
	if err := writer.WriteHeader(&tar.Header{Name: "directory/tool", Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}

	var got bytes.Buffer
	if err := extractFile(&got, &archive, "directory/tool"); err != nil {
		t.Fatal(err)
	}
	if got.String() != string(content) {
		t.Fatalf("extractFile() = %q", got.String())
	}
}
