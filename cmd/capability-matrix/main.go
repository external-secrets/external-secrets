/*
Copyright © 2026 ESO Maintainer Team

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
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"sort"
	"text/template"

	_ "github.com/external-secrets/external-secrets/pkg/register"
	runtimeprovider "github.com/external-secrets/external-secrets/runtime/provider"
)

const markdownTemplate = `---
title: Provider Capabilities Matrix
description: Auto-generated documentation of provider capabilities
---

# Provider Capabilities Matrix

This page lists all External Secrets Operator providers and their supported capabilities.

## Capabilities Matrix

| Provider | Stability | GetSecret | FindByName | FindByTag | PushSecret | DeleteSecret | ReferentAuth | MetadataPolicy | ValidateStore |
|----------|-----------|-----------|------------|-----------|------------|--------------|--------------|----------------|---------------|
{{- range $provider := .}}
| {{$provider.Name}} | {{$provider.Stability}} |{{if index $provider.DisplayedCapabilities "GetSecret"}} ✓ |{{else}}|{{end}}{{if index $provider.DisplayedCapabilities "FindByName"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "FindByTag"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "PushSecret"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "DeleteSecret"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "ReferentAuthentication"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "MetadataPolicyFetch"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "ValidateStore"}} ✓ |{{else}}|{{end}}{{- end}}

## Extra Capabilities per provider
{{range .}}

{{if .ExtraCapabilities}}
### {{.Name}} provider capabilities

{{range .ExtraCapabilities}}
- ` + "`{{.Name}}`" + `{{if .Notes}}: {{.Notes}}{{end}}
{{- end}}

{{end}}
{{end }}
`

var (
	outputFormat        = flag.String("format", "md", "Output format: json or md")
	capabilityShortList = []runtimeprovider.CapabilityName{
		runtimeprovider.CapabilityGetSecret,
		runtimeprovider.CapabilityFindByName,
		runtimeprovider.CapabilityFindByTag,
		runtimeprovider.CapabilityPushSecret,
		runtimeprovider.CapabilityDeleteSecret,
		runtimeprovider.CapabilityReferentAuthentication,
		runtimeprovider.CapabilityMetadataPolicyFetch,
		runtimeprovider.CapabilityValidateStore,
	}
)

func main() {
	flag.Parse()

	providers := runtimeprovider.List()
	matrix := make(map[runtimeprovider.Name]runtimeprovider.Metadata, len(providers))

	for providerName, providerData := range providers {
		matrix[providerName] = providerData.Metadata
	}

	var output string
	switch *outputFormat {
	case "json":
		output = generateJSON(matrix)
	case "md":
		output = generateMarkdown(matrix)
	default:
		log.Fatalf("Unknown format: %s", *outputFormat)
	}

	fmt.Print(output)
}

func generateJSON(matrix map[runtimeprovider.Name]runtimeprovider.Metadata) string {
	data, err := json.MarshalIndent(matrix, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	return string(data)
}

type providerInfo struct {
	Name                  string
	Stability             string
	DisplayedCapabilities map[string]bool
	ExtraCapabilities     []runtimeprovider.Capability
}

type providerTable []providerInfo

func (p providerTable) Len() int {
	return len(p)
}
func (p providerTable) Less(i, j int) bool {
	return p[i].Name < p[j].Name
}
func (p providerTable) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func newProviderTable(matrix map[runtimeprovider.Name]runtimeprovider.Metadata) providerTable {
	var table providerTable
	for pn, metadata := range matrix {
		line := providerInfo{}
		line.Name = string(pn)
		line.Stability = string(metadata.Stability)
		line.DisplayedCapabilities = make(map[string]bool)

		// Build a map of provider capabilities for quick lookup
		providerCapabilities := make(map[runtimeprovider.CapabilityName]bool)
		for _, capability := range metadata.Capabilities {
			providerCapabilities[capability.Name] = true
		}

		// Check which short-list capabilities this provider has
		for _, capability := range capabilityShortList {
			line.DisplayedCapabilities[string(capability)] = providerCapabilities[capability]
		}

		// Extend the info with metadata outside the short list
		for _, providerCapability := range metadata.Capabilities {
			if _, ok := line.DisplayedCapabilities[string(providerCapability.Name)]; !ok {
				line.ExtraCapabilities = append(line.ExtraCapabilities, providerCapability)
			}
		}
		table = append(table, line)
	}
	sort.Sort(table)
	return table
}

func generateMarkdown(matrix map[runtimeprovider.Name]runtimeprovider.Metadata) string {
	sortedTable := newProviderTable(matrix)

	// Execute template
	tmpl, err := template.New("markdown").Parse(markdownTemplate)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, sortedTable); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}

	return buf.String()
}
