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
	"unicode"
	"unicode/utf8"
)

// toLowerCamel converts an exported type name to lowerCamelCase for JSON tags
// (ServiceAccountToken -> serviceAccountToken).
func toLowerCamel(name string) string {
	if name == "" {
		return name
	}
	r, size := utf8.DecodeRuneInString(name)
	// size==0: empty (already handled). size==1 with RuneError: invalid UTF-8 byte.
	// Leave malformed input unchanged rather than replacing it with U+FFFD.
	if r == utf8.RuneError && size <= 1 {
		return name
	}
	return string(unicode.ToLower(r)) + name[size:]
}

// enumAnnotationHasValue reports whether a kubebuilder Enum annotation line
// already lists value as a complete ;-separated entry (not a substring/suffix).
func enumAnnotationHasValue(line, value string) bool {
	const marker = "+kubebuilder:validation:Enum="
	idx := strings.Index(line, marker)
	if idx < 0 {
		return false
	}
	for _, part := range strings.Split(line[idx+len(marker):], ";") {
		if strings.TrimSpace(part) == value {
			return true
		}
	}
	return false
}

// contentHasExactEnumValue is true when any Enum annotation in content lists value exactly.
func contentHasExactEnumValue(content, value string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "+kubebuilder:validation:Enum=") && enumAnnotationHasValue(line, value) {
			return true
		}
	}
	return false
}

// FindRootDir finds the root directory of the external-secrets repository.
func FindRootDir(startDir string) string {
	dir := startDir
	for {
		// Check if go.mod exists and contains external-secrets
		goModPath := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(filepath.Clean(goModPath)); err == nil {
			if strings.Contains(string(data), "module github.com/external-secrets/external-secrets") {
				return dir
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
