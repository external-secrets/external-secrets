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

// Package generator provides functionality for bootstrapping new generators.
package generator

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templates embed.FS

const generatorImportPath = "github.com/external-secrets/external-secrets/generators/v1/"

// Config holds the configuration for bootstrapping a generator.
type Config struct {
	GeneratorName string
	PackageName   string
	Description   string
	GeneratorKind string
}

// Bootstrap creates a new generator with all necessary files.
func Bootstrap(rootDir string, cfg Config) error {
	// Create generator CRD
	if err := createGeneratorCRD(rootDir, cfg); err != nil {
		return fmt.Errorf("failed to create CRD: %w", err)
	}

	// Create generator implementation
	if err := createGeneratorImplementation(rootDir, cfg); err != nil {
		return fmt.Errorf("failed to create implementation: %w", err)
	}

	// Update register file
	if err := updateRegisterFile(rootDir, cfg); err != nil {
		return fmt.Errorf("failed to update register file: %w", err)
	}

	// Update types_cluster.go
	if err := updateTypesClusterFile(rootDir, cfg); err != nil {
		return fmt.Errorf("failed to update types_cluster.go: %w", err)
	}

	// Update main go.mod
	if err := updateMainGoMod(rootDir, cfg); err != nil {
		return fmt.Errorf("failed to update main go.mod: %w", err)
	}

	// Update resolver file
	if err := updateResolverFile(rootDir, cfg); err != nil {
		return fmt.Errorf("failed to update resolver file: %w", err)
	}

	// Update register kind file
	if err := updateRegisterKindFile(rootDir, cfg); err != nil {
		return fmt.Errorf("failed to update register kind file: %w", err)
	}

	// Update ExternalSecret GeneratorRef validation
	if err := updateExternalSecretGeneratorRef(rootDir, cfg); err != nil {
		return fmt.Errorf("failed to update ExternalSecret GeneratorRef: %w", err)
	}

	return nil
}

func createGeneratorCRD(rootDir string, cfg Config) error {
	crdDir := filepath.Join(rootDir, "apis", "generators", "v1alpha1")
	crdFile := filepath.Join(crdDir, fmt.Sprintf("types_%s.go", cfg.PackageName))

	// Check if file already exists
	if _, err := os.Stat(crdFile); err == nil {
		return fmt.Errorf("CRD file already exists: %s", crdFile)
	}

	tmplContent, err := templates.ReadFile("templates/crd.go.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl := template.Must(template.New("crd").Parse(string(tmplContent)))
	f, err := os.Create(filepath.Clean(crdFile))
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := tmpl.Execute(f, cfg); err != nil {
		return err
	}

	fmt.Printf("✓ Created CRD: %s\n", crdFile)
	return nil
}

func createGeneratorImplementation(rootDir string, cfg Config) error {
	genDir := filepath.Join(rootDir, "generators", "v1", cfg.PackageName)
	if err := os.MkdirAll(genDir, 0o750); err != nil {
		return err
	}

	// Create main generator file
	genFile := filepath.Join(genDir, fmt.Sprintf("%s.go", cfg.PackageName))
	if _, err := os.Stat(genFile); err == nil {
		return fmt.Errorf("implementation file already exists: %s", genFile)
	}

	if err := createFromTemplate("templates/implementation.go.tmpl", genFile, cfg); err != nil {
		return err
	}
	fmt.Printf("✓ Created implementation: %s\n", genFile)

	// Create test file
	testFile := filepath.Join(genDir, fmt.Sprintf("%s_test.go", cfg.PackageName))
	if err := createFromTemplate("templates/test.go.tmpl", testFile, cfg); err != nil {
		return err
	}
	fmt.Printf("✓ Created test file: %s\n", testFile)

	// Create go.mod
	goModFile := filepath.Join(genDir, "go.mod")
	if err := createFromTemplate("templates/go.mod.tmpl", goModFile, cfg); err != nil {
		return err
	}
	fmt.Printf("✓ Created go.mod: %s\n", goModFile)

	// Create empty go.sum
	goSumFile := filepath.Join(genDir, "go.sum")
	if err := os.WriteFile(goSumFile, []byte(""), 0o600); err != nil {
		return err
	}
	fmt.Printf("✓ Created go.sum: %s\n", goSumFile)

	return nil
}

func createFromTemplate(tmplPath, outputFile string, cfg Config) error {
	tmplContent, err := templates.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", tmplPath, err)
	}

	tmpl := template.Must(template.New(filepath.Base(tmplPath)).Parse(string(tmplContent)))
	f, err := os.Create(filepath.Clean(outputFile))
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return tmpl.Execute(f, cfg)
}

func updateRegisterFile(rootDir string, cfg Config) error {
	registerFile := filepath.Join(rootDir, "pkg", "register", "generators.go")

	data, err := os.ReadFile(filepath.Clean(registerFile))
	if err != nil {
		return err
	}

	content := string(data)

	// Check if already registered
	if strings.Contains(content, fmt.Sprintf("%q", cfg.PackageName)) {
		fmt.Printf("⚠ Generator already registered in %s\n", registerFile)
		return nil
	}

	// Add import
	importLine := fmt.Sprintf("\t%s \"%s%s\"",
		cfg.PackageName, generatorImportPath, cfg.PackageName)

	// Find the last import before the closing parenthesis
	lines := strings.Split(content, "\n")
	newLines := make([]string, 0, len(lines)+2)
	importAdded := false
	registerAdded := false

	for i, line := range lines {
		newLines = append(newLines, line)

		// Add import after the last generator import
		if !importAdded && strings.Contains(line, "\""+generatorImportPath) {
			// Look ahead to see if next line is still an import or closing paren
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == ")" {
				newLines = append(newLines, importLine)
				importAdded = true
			}
		}

		// Add register call after the last Register call
		if !registerAdded && strings.Contains(line, "genv1alpha1.Register(") {
			// Look ahead to see if next line is closing brace
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "}" {
				registerLine := fmt.Sprintf("\tgenv1alpha1.Register(%s.Kind(), %s.NewGenerator())",
					cfg.PackageName, cfg.PackageName)
				newLines = append(newLines, registerLine)
				registerAdded = true
			}
		}
	}

	if !importAdded || !registerAdded {
		return fmt.Errorf("failed to add import or register call to %s", registerFile)
	}

	if err := os.WriteFile(filepath.Clean(registerFile), []byte(strings.Join(newLines, "\n")), 0o600); err != nil {
		return err
	}

	fmt.Printf("✓ Updated register file: %s\n", registerFile)
	return nil
}

func updateTypesClusterFile(rootDir string, cfg Config) error {
	typesClusterFile := filepath.Join(rootDir, "apis", "generators", "v1alpha1", "types_cluster.go")

	data, err := os.ReadFile(filepath.Clean(typesClusterFile))
	if err != nil {
		return err
	}

	content := string(data)

	// Check if already exists
	if strings.Contains(content, cfg.GeneratorKind) {
		fmt.Printf("⚠ Generator kind already exists in types_cluster.go\n")
		return nil
	}

	lines := strings.Split(content, "\n")
	newLines := make([]string, 0, len(lines)+2)
	enumAdded := false
	constAdded := false
	specAdded := false

	for i, line := range lines {
		// Update the enum validation annotation
		if !enumAdded && strings.Contains(line, "+kubebuilder:validation:Enum=") {
			// Add the new generator to the enum list
			line = strings.TrimRight(line, "\n")
			if strings.HasSuffix(line, "Grafana") {
				line = line + ";" + cfg.GeneratorName
			}
			enumAdded = true
		}

		newLines = append(newLines, line)

		// Add const after the last GeneratorKind const
		if !constAdded && strings.Contains(line, "GeneratorKind") && strings.Contains(line, "GeneratorKind = \"") {
			// Look ahead to check if next line is closing paren
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == ")" {
				constLine := fmt.Sprintf("\t// %s represents a %s generator.",
					cfg.GeneratorKind, strings.ToLower(cfg.GeneratorName))
				newLines = append(newLines, constLine)
				constValueLine := fmt.Sprintf("\t%s GeneratorKind = %q",
					cfg.GeneratorKind, cfg.GeneratorName)
				newLines = append(newLines, constValueLine)
				constAdded = true
			}
		}

		// Add spec field to GeneratorSpec struct
		if !specAdded && strings.Contains(line, "Spec") && strings.Contains(line, "`json:") && strings.Contains(line, "omitempty") {
			// Look ahead to check if next line is closing brace of the struct
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "}" {
				// Add the new spec field
				jsonTag := strings.ToLower(cfg.GeneratorName) + "Spec"
				specLine := fmt.Sprintf("\t%sSpec             *%sSpec             `json:\"%s,omitempty\"`",
					cfg.GeneratorName, cfg.GeneratorName, jsonTag)
				newLines = append(newLines, specLine)
				specAdded = true
			}
		}
	}

	if !enumAdded || !constAdded || !specAdded {
		fmt.Printf("⚠ Warning: Could not fully update types_cluster.go. Please manually add:\n")
		if !enumAdded {
			fmt.Printf("   1. Add '%s' to the kubebuilder:validation:Enum annotation\n", cfg.GeneratorName)
		}
		if !constAdded {
			fmt.Printf("   2. Add the const: %s GeneratorKind = \"%s\"\n", cfg.GeneratorKind, cfg.GeneratorName)
		}
		if !specAdded {
			fmt.Printf("   3. Add to GeneratorSpec struct: %sSpec *%sSpec `json:\"%sSpec,omitempty\"`\n",
				cfg.GeneratorName, cfg.GeneratorName, strings.ToLower(cfg.GeneratorName))
		}
	} else {
		if err := os.WriteFile(filepath.Clean(typesClusterFile), []byte(strings.Join(newLines, "\n")), 0o600); err != nil {
			return err
		}
		fmt.Printf("✓ Updated types_cluster.go\n")
	}

	return nil
}

func updateMainGoMod(rootDir string, cfg Config) error {
	goModFile := filepath.Join(rootDir, "go.mod")

	data, err := os.ReadFile(filepath.Clean(goModFile))
	if err != nil {
		return err
	}

	content := string(data)
	replaceLine := fmt.Sprintf("\t%s%s => ./generators/v1/%s",
		generatorImportPath,
		cfg.PackageName, cfg.PackageName)

	// Check if already exists
	if strings.Contains(content, replaceLine) {
		fmt.Printf("⚠ Replace directive already exists in go.mod\n")
		return nil
	}

	lines := strings.Split(content, "\n")
	newLines := make([]string, 0, len(lines)+1)
	added := false
	lastGeneratorIdx := -1

	// First pass: find where to insert
	for i, line := range lines {
		if strings.Contains(line, generatorImportPath) {
			lastGeneratorIdx = i
			// Extract the package name from the current line
			currentPkg := extractGeneratorPackage(line)
			if currentPkg != "" && cfg.PackageName < currentPkg && !added {
				// Insert before this line (alphabetically)
				newLines = append(newLines, replaceLine)
				added = true
			}
		}

		newLines = append(newLines, line)

		// If this was the last generator and we haven't added yet, add after it
		if i == lastGeneratorIdx && !added && lastGeneratorIdx != -1 {
			// Check if next line is NOT a generator (meaning this is the last one)
			if i+1 >= len(lines) || !strings.Contains(lines[i+1], generatorImportPath) {
				newLines = append(newLines, replaceLine)
				added = true
			}
		}
	}

	// This shouldn't happen in practice but handle it
	if !added {
		return fmt.Errorf("could not find appropriate position to insert replace directive")
	}

	if err := os.WriteFile(filepath.Clean(goModFile), []byte(strings.Join(newLines, "\n")), 0o600); err != nil {
		return err
	}

	fmt.Printf("✓ Updated main go.mod\n")
	return nil
}

func extractGeneratorPackage(line string) string {
	if !strings.Contains(line, generatorImportPath) {
		return ""
	}
	// Extract package name from line like:
	// "\tgithub.com/external-secrets/external-secrets/generators/v1/uuid => ./generators/v1/uuid"
	parts := strings.Split(line, "/")
	if len(parts) == 0 {
		return ""
	}
	lastPart := parts[len(parts)-1]
	// Remove everything after space (the => part)
	if idx := strings.Index(lastPart, " "); idx != -1 {
		lastPart = lastPart[:idx]
	}
	return strings.TrimSpace(lastPart)
}

func updateResolverFile(rootDir string, cfg Config) error {
	resolverFile := filepath.Join(rootDir, "runtime", "esutils", "resolvers", "generator.go")

	data, err := os.ReadFile(filepath.Clean(resolverFile))
	if err != nil {
		return err
	}

	content := string(data)

	// Check if already exists
	if strings.Contains(content, fmt.Sprintf("GeneratorKind%s", cfg.GeneratorName)) {
		fmt.Printf("⚠ Generator already exists in resolver file\n")
		return nil
	}

	// Create the case statement to add
	caseBlock := fmt.Sprintf(`	case genv1alpha1.GeneratorKind%s:
		if gen.Spec.Generator.%sSpec == nil {
			return nil, fmt.Errorf("when kind is %%s, %sSpec must be set", gen.Spec.Kind)
		}
		return &genv1alpha1.%s{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.SchemeGroupVersion.String(),
				Kind:       genv1alpha1.%sKind,
			},
			Spec: *gen.Spec.Generator.%sSpec,
		}, nil`,
		cfg.GeneratorName, cfg.GeneratorName, cfg.GeneratorName,
		cfg.GeneratorName, cfg.GeneratorName, cfg.GeneratorName)

	lines := strings.Split(content, "\n")
	newLines := make([]string, 0, len(lines)+10)
	added := false

	for _, line := range lines {
		// Find the default case and add before it
		if !added && strings.TrimSpace(line) == "default:" {
			newLines = append(newLines, caseBlock)
			added = true
		}

		newLines = append(newLines, line)
	}

	if !added {
		return fmt.Errorf("could not find default case in resolver file")
	}

	if err := os.WriteFile(filepath.Clean(resolverFile), []byte(strings.Join(newLines, "\n")), 0o600); err != nil {
		return err
	}

	fmt.Printf("✓ Updated resolver file: %s\n", resolverFile)
	return nil
}

func updateRegisterKindFile(rootDir string, cfg Config) error {
	registerFile := filepath.Join(rootDir, "apis", "generators", "v1alpha1", "register.go")

	data, err := os.ReadFile(filepath.Clean(registerFile))
	if err != nil {
		return err
	}

	content := string(data)

	// Check if already exists
	if strings.Contains(content, fmt.Sprintf("%sKind", cfg.GeneratorName)) {
		fmt.Printf("⚠ Generator kind already exists in register.go\n")
		return nil
	}

	lines := strings.Split(content, "\n")
	newLines := make([]string, 0, len(lines)+4)
	kindAdded := false
	schemeAdded := false

	for i, line := range lines {
		newLines = append(newLines, line)

		// Add Kind constant before closing paren of var block
		if !kindAdded && strings.Contains(line, "Kind = reflect.TypeOf(") && strings.Contains(line, "{}).Name()") {
			// Look ahead to see if next line is closing paren
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == ")" {
				// Add blank line and then the new kind
				newLines = append(newLines, "")
				kindComment := fmt.Sprintf("\t// %sKind is the kind name for %s resource.", cfg.GeneratorName, cfg.GeneratorName)
				newLines = append(newLines, kindComment)
				kindLine := fmt.Sprintf("\t%sKind = reflect.TypeOf(%s{}).Name()", cfg.GeneratorName, cfg.GeneratorName)
				newLines = append(newLines, kindLine)
				kindAdded = true
			}
		}

		// Add SchemeBuilder.Register call before closing brace of init function
		if !schemeAdded && strings.Contains(line, "SchemeBuilder.Register(&") && strings.Contains(line, "List{})") {
			// Look ahead to see if next line is closing brace
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "}" {
				registerLine := fmt.Sprintf("\tSchemeBuilder.Register(&%s{}, &%sList{})", cfg.GeneratorName, cfg.GeneratorName)
				newLines = append(newLines, registerLine)
				schemeAdded = true
			}
		}
	}

	if !kindAdded || !schemeAdded {
		fmt.Printf("⚠ Warning: Could not fully update register.go. Please manually add:\n")
		if !kindAdded {
			fmt.Printf("   1. Add Kind constant: %sKind = reflect.TypeOf(%s{}).Name()\n", cfg.GeneratorName, cfg.GeneratorName)
		}
		if !schemeAdded {
			fmt.Printf("   2. Add SchemeBuilder registration: SchemeBuilder.Register(&%s{}, &%sList{})\n", cfg.GeneratorName, cfg.GeneratorName)
		}
	} else {
		if err := os.WriteFile(filepath.Clean(registerFile), []byte(strings.Join(newLines, "\n")), 0o600); err != nil {
			return err
		}
		fmt.Printf("✓ Updated register.go\n")
	}

	return nil
}

func updateExternalSecretGeneratorRef(rootDir string, cfg Config) error {
	// Update v1 ExternalSecret types
	externalSecretFile := filepath.Join(rootDir, "apis", "externalsecrets", "v1", "externalsecret_types.go")

	data, err := os.ReadFile(filepath.Clean(externalSecretFile))
	if err != nil {
		return fmt.Errorf("failed to read v1 externalsecret_types.go: %w", err)
	}

	content := string(data)

	// Check if already exists
	if strings.Contains(content, ";"+cfg.GeneratorName) {
		fmt.Printf("⚠ Generator %s already exists in v1 GeneratorRef enum\n", cfg.GeneratorName)
		return nil
	}

	lines := strings.Split(content, "\n")
	newLines := make([]string, 0, len(lines))
	updated := false

	for _, line := range lines {
		// Look for the GeneratorRef Kind enum validation line
		// Pattern: // +kubebuilder:validation:Enum=ACRAccessToken;ClusterGenerator;...
		if !updated && strings.Contains(line, "+kubebuilder:validation:Enum=") &&
			strings.Contains(line, "ACRAccessToken") &&
			strings.Contains(line, "ClusterGenerator") {
			// This is the enum line we need to update
			// Add the new generator to the end of the enum list
			line = strings.TrimRight(line, "\n")
			line = line + ";" + cfg.GeneratorName
			updated = true
		}

		newLines = append(newLines, line)
	}

	if !updated {
		fmt.Printf("⚠ Warning: Could not find GeneratorRef Kind enum in v1. Please manually add '%s' to the enum.\n",
			cfg.GeneratorName)
		return nil
	}

	if err := os.WriteFile(filepath.Clean(externalSecretFile), []byte(strings.Join(newLines, "\n")), 0o600); err != nil {
		return fmt.Errorf("failed to write v1 externalsecret_types.go: %w", err)
	}

	fmt.Printf("✓ Updated v1 ExternalSecret GeneratorRef enum\n")

	return nil
}
