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
	"slices"
	"strings"
	"unicode"
)

// enumMarker introduces the list of permitted values in a kubebuilder validation enum.
const enumMarker = "+kubebuilder:validation:Enum="

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

// generatorKindEnumLine returns the index of the kubebuilder validation enum that
// governs `type GeneratorKind string`, or -1 when there is none.
//
// The annotation is found from the declaration rather than taken as the first one
// in the file. Nothing stops another enum from being added to types_cluster.go, and
// appending a kind to the wrong annotation while silently skipping the right one is
// the failure this tool already has a history of.
func generatorKindEnumLine(lines []string) int {
	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "type GeneratorKind ") {
			continue
		}

		// Walk back over the doc comment attached to the declaration.
		for j := i - 1; j >= 0 && strings.HasPrefix(strings.TrimSpace(lines[j]), "//"); j-- {
			if strings.Contains(lines[j], enumMarker) {
				return j
			}
		}

		return -1
	}

	return -1
}

// enumListsKind reports whether a kubebuilder validation enum annotation already
// carries kind as one of its values. Matching whole values rather than a substring
// keeps a kind from being mistaken for the prefix of a longer one.
func enumListsKind(annotation, kind string) bool {
	_, values, found := strings.Cut(annotation, enumMarker)
	if !found {
		return false
	}

	return slices.Contains(strings.Split(strings.TrimSpace(values), ";"), kind)
}

// lowerCamel converts a PascalCase kind into the lowerCamelCase form used by the
// json tags of GeneratorSpec. A leading acronym is lowered as a whole, so
// ECRAuthorizationToken becomes ecrAuthorizationToken and UUID becomes uuid.
func lowerCamel(name string) string {
	runes := []rune(name)

	// length of the leading run of upper-case letters
	upper := 0
	for upper < len(runes) && unicode.IsUpper(runes[upper]) {
		upper++
	}

	switch {
	case upper == 0:
		return name
	case upper == 1 || upper == len(runes):
		// a single leading capital, or an all-caps name: lower the whole run
		return strings.ToLower(string(runes[:upper])) + string(runes[upper:])
	default:
		// an acronym followed by another word: its last capital starts that word
		return strings.ToLower(string(runes[:upper-1])) + string(runes[upper-1:])
	}
}
