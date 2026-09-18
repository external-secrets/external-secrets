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

// Command updatecli-gomodules generates Updatecli manifests for the
// repository's canonical Go modules.
package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"text/template"
)

const internalModule = "github.com/external-secrets/external-secrets"
const gomodFilename = "go.mod"

var (
	pseudoVersion = regexp.MustCompile(`^v\d+\.\d+\.\d+-\d{14}-[0-9a-f]+$`)
	semverMajor   = regexp.MustCompile(`^v?(\d+)\.`)

	// These modules are intentionally isolated from the workspace represented by
	// the local replacements in the root go.mod.
	isolatedModuleFiles = []string{
		"e2e/go.mod",
		"hack/tools/gen-crd-api-reference-docs/go.mod",
	}

	//go:embed templates/*.tmpl
	manifestTemplates embed.FS
)

type moduleVersion struct {
	Path    string
	Version string
}

type requirement struct {
	Path     string
	Version  string
	Indirect bool
}

type tool struct {
	Path string
}

type replacement struct {
	Old moduleVersion
	New moduleVersion
}

type goMod struct {
	Require []requirement
	Tool    []tool
	Replace []replacement
}

type manifestTarget struct {
	Index      int
	Path       string
	FileTarget bool
}

type manifestDependency struct {
	Index        int
	Module       string
	Kind         string
	Pattern      string
	MatchPattern string
	Targets      []manifestTarget
}

type manifestData struct {
	Dependencies []manifestDependency
	ModuleFiles  []string
}

type dependencyKey struct {
	Module  string
	Kind    string
	Pattern string
}

type generatedManifest struct {
	Template string
	Output   string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "updatecli-gomodules: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Manifest generation successful")
}

func run() error {
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	data, err := buildManifestData(root)
	if err != nil {
		return err
	}

	manifests := []generatedManifest{
		{Template: "gomodules.yaml.tmpl", Output: ".updatecli.d/gomodules.yaml"},
		{Template: "golang.yaml.tmpl", Output: ".updatecli.d/golang.yaml"},
	}
	for _, manifest := range manifests {
		contents, err := renderManifest(manifest.Template, data)
		if err != nil {
			return err
		}
		if err := writeFileAtomically(filepath.Join(root, manifest.Output), contents); err != nil {
			return fmt.Errorf("write %s: %w", manifest.Output, err)
		}
	}
	return nil
}

func buildManifestData(root string) (manifestData, error) {
	files, err := moduleFiles(root)
	if err != nil {
		return manifestData{}, err
	}

	dependencies := make(map[dependencyKey]map[string]bool)
	for _, file := range files {
		if err := collectModuleDependencies(root, file, dependencies); err != nil {
			return manifestData{}, err
		}
	}

	keys := make([]dependencyKey, 0, len(dependencies))
	for key := range dependencies {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Module != keys[j].Module {
			return keys[i].Module < keys[j].Module
		}
		if keys[i].Kind != keys[j].Kind {
			return keys[i].Kind < keys[j].Kind
		}
		return keys[i].Pattern < keys[j].Pattern
	})

	data := manifestData{ModuleFiles: files}
	for dependencyIndex, key := range keys {
		targetsByPath := dependencies[key]
		paths := make([]string, 0, len(targetsByPath))
		for path := range targetsByPath {
			paths = append(paths, path)
		}
		sort.Strings(paths)

		item := manifestDependency{
			Index:        dependencyIndex,
			Module:       key.Module,
			Kind:         key.Kind,
			Pattern:      key.Pattern,
			MatchPattern: fmt.Sprintf(`(?m)^(\s*)%s\s+\S+(\s+// indirect)$`, regexp.QuoteMeta(key.Module)),
		}
		for targetIndex, path := range paths {
			item.Targets = append(item.Targets, manifestTarget{
				Index:      targetIndex,
				Path:       path,
				FileTarget: targetsByPath[path],
			})
		}
		data.Dependencies = append(data.Dependencies, item)
	}
	return data, nil
}

// collectModuleDependencies groups a module file's external dependencies by version filter.
func collectModuleDependencies(root, file string, dependencies map[dependencyKey]map[string]bool) error {
	mod, err := readGoMod(root, file)
	if err != nil {
		return err
	}
	modules, err := directModules(file, mod)
	if err != nil {
		return err
	}
	for module, item := range modules {
		if module == internalModule || strings.HasPrefix(module, internalModule+"/") {
			continue
		}
		kind, pattern, err := versionFilter(item.Version)
		if err != nil {
			return fmt.Errorf("%s: module %s: %w", file, module, err)
		}
		key := dependencyKey{Module: module, Kind: kind, Pattern: pattern}
		if dependencies[key] == nil {
			dependencies[key] = make(map[string]bool)
		}
		dependencies[key][file] = item.Indirect
	}
	return nil
}

func renderManifest(name string, data manifestData) ([]byte, error) {
	manifest, err := template.New(name).Delims("<%", "%>").ParseFS(manifestTemplates, "templates/"+name)
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}

	var output bytes.Buffer
	if err := manifest.Execute(&output, data); err != nil {
		return nil, fmt.Errorf("render template %s: %w", name, err)
	}
	return output.Bytes(), nil
}

func writeFileAtomically(path string, contents []byte) error {
	directory := filepath.Dir(path)
	file, err := os.CreateTemp(directory, ".updatecli-*")
	if err != nil {
		return err
	}
	temporaryPath := file.Name()
	defer func() { _ = os.Remove(temporaryPath) }()

	if _, err := file.Write(contents); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Chmod(0o644); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func moduleFiles(root string) ([]string, error) {
	rootMod, err := readGoMod(root, gomodFilename)
	if err != nil {
		return nil, err
	}

	files := map[string]struct{}{gomodFilename: {}}
	for _, replace := range rootMod.Replace {
		if replace.Old.Path != internalModule && !strings.HasPrefix(replace.Old.Path, internalModule+"/") {
			continue
		}
		if replace.New.Version != "" || filepath.IsAbs(replace.New.Path) {
			continue
		}
		path := filepath.Clean(filepath.Join(replace.New.Path, gomodFilename))
		if path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("root go.mod replacement escapes repository: %s", replace.New.Path)
		}
		files[filepath.ToSlash(path)] = struct{}{}
	}
	for _, file := range isolatedModuleFiles {
		files[file] = struct{}{}
	}

	result := make([]string, 0, len(files))
	for file := range files {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(file))); err != nil {
			return nil, fmt.Errorf("canonical module %s: %w", file, err)
		}
		result = append(result, file)
	}
	sort.Strings(result)
	return result, nil
}

func readGoMod(root, file string) (goMod, error) {
	directory := filepath.Dir(filepath.Join(root, filepath.FromSlash(file)))
	// Use the running program's Go toolchain rather than resolving Go through PATH.
	goBinary := filepath.Join(runtime.GOROOT(), "bin", "go")
	//nolint:gosec // The executable is from GOROOT; the directory is from the canonical module inventory.
	command := exec.Command(goBinary, "-C", directory, "mod", "edit", "-json")
	command.Env = environmentWithoutGoWork()
	var stderr bytes.Buffer
	command.Stderr = &stderr
	contents, err := command.Output()
	if err != nil {
		return goMod{}, fmt.Errorf("read %s: %w: %s", file, err, strings.TrimSpace(stderr.String()))
	}

	var mod goMod
	if err := json.Unmarshal(contents, &mod); err != nil {
		return goMod{}, fmt.Errorf("decode %s: %w", file, err)
	}
	return mod, nil
}

func environmentWithoutGoWork() []string {
	environment := []string{"GOWORK=off"}
	for _, variable := range os.Environ() {
		if !strings.HasPrefix(variable, "GOWORK=") {
			environment = append(environment, variable)
		}
	}
	return environment
}

func directModules(file string, mod goMod) (map[string]requirement, error) {
	requirements := make(map[string]requirement, len(mod.Require))
	result := make(map[string]requirement)
	for _, item := range mod.Require {
		requirements[item.Path] = item
		if !item.Indirect {
			result[item.Path] = item
		}
	}

	var missingTools []string
	for _, tool := range mod.Tool {
		var selected requirement
		for module, item := range requirements {
			if (tool.Path == module || strings.HasPrefix(tool.Path, module+"/")) && len(module) > len(selected.Path) {
				selected = item
			}
		}
		if selected.Path == "" {
			missingTools = append(missingTools, tool.Path)
			continue
		}
		result[selected.Path] = selected
	}
	if len(missingTools) > 0 {
		sort.Strings(missingTools)
		return nil, fmt.Errorf("%s: tool packages missing from require directives: %s", file, strings.Join(missingTools, ", "))
	}
	return result, nil
}

func versionFilter(version string) (string, string, error) {
	if pseudoVersion.MatchString(version) {
		return "latest", "", nil
	}
	match := semverMajor.FindStringSubmatch(version)
	if match == nil {
		return "", "", fmt.Errorf("unsupported Go module version %q", version)
	}
	if strings.Contains(version, "-") {
		return "semver", match[1] + ".x.x-0", nil
	}
	return "semver", match[1] + ".x", nil
}
