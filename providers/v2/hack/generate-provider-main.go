/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package main provides a tool to generate provider main.go files from provider configuration.
package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed schema/provider-config.schema.json templates/*.tmpl
var embeddedFS embed.FS

// ProviderConfig represents the structure of provider.yaml.
type ProviderConfig struct {
	Provider struct {
		Name        string `yaml:"name" json:"name"`
		DisplayName string `yaml:"displayName" json:"displayName"`
		V2Package   string `yaml:"v2Package" json:"v2Package"`
	} `yaml:"provider" json:"provider"`
	Stores []struct {
		GVK struct {
			Group   string `yaml:"group" json:"group"`
			Version string `yaml:"version" json:"version"`
			Kind    string `yaml:"kind" json:"kind"`
		} `yaml:"gvk" json:"gvk"`
		V1Provider     string `yaml:"v1Provider" json:"v1Provider"`
		V1ProviderFunc string `yaml:"v1ProviderFunc" json:"v1ProviderFunc"`
	} `yaml:"stores" json:"stores"`
	Generators []struct {
		GVK struct {
			Group   string `yaml:"group" json:"group"`
			Version string `yaml:"version" json:"version"`
			Kind    string `yaml:"kind" json:"kind"`
		} `yaml:"gvk" json:"gvk"`
		V1Generator     string `yaml:"v1Generator" json:"v1Generator"`
		V1GeneratorFunc string `yaml:"v1GeneratorFunc" json:"v1GeneratorFunc"`
	} `yaml:"generators" json:"generators"`
	ConfigPackage string `yaml:"configPackage" json:"configPackage"`
}

// ImportInfo tracks package imports with aliases.
type ImportInfo struct {
	Path  string
	Alias string
}

// TemplateData contains all data needed for template rendering.
type TemplateData struct {
	Provider               ProviderConfig
	HasStores              bool
	HasGenerators          bool
	UniqueStoreImports     []ImportInfo
	UniqueGeneratorImports []ImportInfo
	Stores                 []StoreInfo
	Generators             []GeneratorInfo
}

type StoreInfo struct {
	GVK struct {
		Group   string
		Version string
		Kind    string
	}
	V1Provider     string
	V1ProviderFunc string
	ImportAlias    string
}

type GeneratorInfo struct {
	GVK struct {
		Group   string
		Version string
		Kind    string
	}
	V1Generator     string
	V1GeneratorFunc string
	ImportAlias     string
}

var (
	providersDir = flag.String("providers-dir", "providers/v2", "Base directory for v2 providers")
	dryRun       = flag.Bool("dry-run", false, "Print what would be generated without writing files")
	verbose      = flag.Bool("verbose", false, "Enable verbose output")
)

func main() {
	flag.Parse()

	log.SetFlags(0)

	// Load the JSON schema
	schemaBytes, err := embeddedFS.ReadFile("schema/provider-config.schema.json")
	if err != nil {
		log.Fatalf("Failed to read schema: %v", err)
	}

	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)

	// Find all provider.yaml files
	providerConfigs, err := findProviderConfigs(*providersDir)
	if err != nil {
		log.Fatalf("Failed to find provider configs: %v", err)
	}

	if len(providerConfigs) == 0 {
		log.Printf("No provider.yaml files found in %s", *providersDir)
		return
	}

	log.Printf("Found %d provider configuration(s)", len(providerConfigs))

	// Load templates
	mainTemplate, err := loadTemplate("templates/main.go.tmpl")
	if err != nil {
		log.Fatalf("Failed to load main.go template: %v", err)
	}

	dockerfileTemplate, err := loadTemplate("templates/Dockerfile.tmpl")
	if err != nil {
		log.Fatalf("Failed to load Dockerfile template: %v", err)
	}

	// Process each provider
	var generatedCount int
	for _, configPath := range providerConfigs {
		providerDir := filepath.Dir(configPath)
		log.Printf("\nProcessing: %s", configPath)

		// Load and validate config
		config, err := loadAndValidateConfig(configPath, schemaLoader)
		if err != nil {
			log.Fatalf("Failed to load/validate config %s: %v", configPath, err)
		}

		if *verbose {
			log.Printf("  Provider: %s (%s)", config.Provider.Name, config.Provider.DisplayName)
			log.Printf("  Stores: %d, Generators: %d", len(config.Stores), len(config.Generators))
		}

		// Prepare template data
		templateData := prepareTemplateData(config)

		// Generate main.go
		mainContent, err := executeTemplate(mainTemplate, templateData)
		if err != nil {
			log.Fatalf("Failed to generate main.go for %s: %v", config.Provider.Name, err)
		}

		// Format with goimports/gofmt
		formattedMain, err := formatGoCode(mainContent)
		if err != nil {
			log.Printf("Warning: Failed to format main.go for %s: %v", config.Provider.Name, err)
			formattedMain = mainContent // Use unformatted if formatting fails
		}

		mainPath := filepath.Join(providerDir, "main.go")
		if *dryRun {
			log.Printf("  Would write: %s (%d bytes)", mainPath, len(formattedMain))
		} else {
			if err := os.WriteFile(mainPath, formattedMain, 0600); err != nil {
				log.Fatalf("Failed to write main.go: %v", err)
			}
			log.Printf("  ✓ Generated: main.go")
		}

		// Generate Dockerfile
		dockerContent, err := executeTemplate(dockerfileTemplate, templateData)
		if err != nil {
			log.Fatalf("Failed to generate Dockerfile for %s: %v", config.Provider.Name, err)
		}

		dockerPath := filepath.Join(providerDir, "Dockerfile")
		if *dryRun {
			log.Printf("  Would write: %s (%d bytes)", dockerPath, len(dockerContent))
		} else {
			if err := os.WriteFile(dockerPath, dockerContent, 0600); err != nil {
				log.Fatalf("Failed to write Dockerfile: %v", err)
			}
			log.Printf("  ✓ Generated: Dockerfile")
		}

		generatedCount++
	}

	log.Printf("\n✓ Successfully generated files for %d provider(s)", generatedCount)
}

func findProviderConfigs(baseDir string) ([]string, error) {
	var configs []string

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "provider.yaml" {
			configs = append(configs, path)
		}
		return nil
	})

	return configs, err
}

func loadAndValidateConfig(configPath string, schemaLoader gojsonschema.JSONLoader) (*ProviderConfig, error) {
	// Read YAML file
	yamlBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Parse YAML into config struct
	var config ProviderConfig
	if err := yaml.Unmarshal(yamlBytes, &config); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	// Convert to JSON for schema validation
	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("convert to JSON: %w", err)
	}

	// Validate against schema
	documentLoader := gojsonschema.NewBytesLoader(jsonBytes)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("validate schema: %w", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, desc := range result.Errors() {
			errMsgs = append(errMsgs, fmt.Sprintf("  - %s", desc))
		}
		return nil, fmt.Errorf("schema validation failed:\n%s", strings.Join(errMsgs, "\n"))
	}

	return &config, nil
}

func loadTemplate(name string) (*template.Template, error) {
	content, err := embeddedFS.ReadFile(name)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func prepareTemplateData(config *ProviderConfig) *TemplateData {
	data := &TemplateData{
		Provider:      *config,
		HasStores:     len(config.Stores) > 0,
		HasGenerators: len(config.Generators) > 0,
	}

	// Collect unique imports for stores
	storeImports := make(map[string]string) // path -> alias
	seenStoreImports := make(map[string]int)
	for _, store := range config.Stores {
		alias := generateImportAlias(store.V1Provider, seenStoreImports)
		storeImports[store.V1Provider] = alias

		storeInfo := StoreInfo{
			V1Provider:     store.V1Provider,
			V1ProviderFunc: store.V1ProviderFunc,
			ImportAlias:    alias,
		}
		storeInfo.GVK.Group = store.GVK.Group
		storeInfo.GVK.Version = store.GVK.Version
		storeInfo.GVK.Kind = store.GVK.Kind

		data.Stores = append(data.Stores, storeInfo)
	}

	for path, alias := range storeImports {
		data.UniqueStoreImports = append(data.UniqueStoreImports, ImportInfo{
			Path:  path,
			Alias: alias,
		})
	}

	// Collect unique imports for generators
	generatorImports := make(map[string]string) // path -> alias
	seenGenImports := make(map[string]int)
	for _, gen := range config.Generators {
		alias := generateImportAlias(gen.V1Generator, seenGenImports)
		generatorImports[gen.V1Generator] = alias

		genInfo := GeneratorInfo{
			V1Generator:     gen.V1Generator,
			V1GeneratorFunc: gen.V1GeneratorFunc,
			ImportAlias:     alias,
		}
		genInfo.GVK.Group = gen.GVK.Group
		genInfo.GVK.Version = gen.GVK.Version
		genInfo.GVK.Kind = gen.GVK.Kind

		data.Generators = append(data.Generators, genInfo)
	}

	for path, alias := range generatorImports {
		data.UniqueGeneratorImports = append(data.UniqueGeneratorImports, ImportInfo{
			Path:  path,
			Alias: alias,
		})
	}

	return data
}

func generateImportAlias(importPath string, seenImports map[string]int) string {
	// Extract the last segment of the import path
	parts := strings.Split(importPath, "/")
	lastPart := parts[len(parts)-1]

	// If this is the first time we see this import path, use the package name
	_, exists := seenImports[importPath]
	if !exists {
		seenImports[importPath] = 1
		return lastPart
	}

	// If we've seen it before, return the same alias
	return lastPart
}

func executeTemplate(tmpl *template.Template, data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func formatGoCode(code []byte) ([]byte, error) {
	// Try goimports first (better formatting)
	cmd := exec.Command("goimports")
	cmd.Stdin = bytes.NewReader(code)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return out.Bytes(), nil
	}

	// Fallback to gofmt if goimports is not available
	cmd = exec.Command("gofmt")
	cmd.Stdin = bytes.NewReader(code)
	out.Reset()
	stderr.Reset()
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("gofmt failed: %w, stderr: %s", err, stderr.String())
	}

	return out.Bytes(), nil
}

