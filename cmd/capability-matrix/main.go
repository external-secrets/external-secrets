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

	provider2 "github.com/external-secrets/external-secrets/runtime/provider"
)

const markdownTemplate = `---
title: Provider CapabilityName Matrix
description: Auto-generated documentation of provider capabilities
---

# Provider CapabilityName Matrix

This page lists all External Secrets Operator providers and their supported capabilities.

## CapabilityName Matrix

| Provider | Stability | GetSecret | FindByName | FindByTag | PushSecret | DeleteSecret | ReferentAuth | MetadataPolicy | ValidateStore |
|----------|-----------|-----------|------------|-----------|------------|--------------|--------------|----------------|---------------|
{{- range $provider := .}}
| {{$provider.Name}} | {{$provider.Stability}} |{{if index $provider.DisplayedCapabilities "GetSecret"}} ✓ |{{else}}|{{end}}{{if index $provider.DisplayedCapabilities "FindByName"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "FindByTag"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "PushSecret"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "DeleteSecret"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "ReferentAuthentication"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "MetadataPolicyFetch"}} ✓ |{{else}}  |{{end}}{{if index $provider.DisplayedCapabilities "ValidateStore"}} ✓ |{{else}}|{{end}}{{- end}}

## Extra Capabilities per provider
{{range .}}
### {{.Name}}

{{if .ExtraCapabilities}}

**Supported Extra Capabilities:**
{{range .ExtraCapabilities}}
- ` + "`{{.Name}}`" + `{{if .Notes}}: {{.Notes}}{{end}}
{{- end}}
{{else}}

*No extra capabilities declared*
{{end}}

---
{{end -}}
`

var (
	outputFormat        = flag.String("format", "md", "Output format: json or md")
	capabilityShortList = []provider2.CapabilityName{
		provider2.CapabilityGetSecret,
		provider2.CapabilityFindByName,
		provider2.CapabilityFindByTag,
		provider2.CapabilityPushSecret,
		provider2.CapabilityDeleteSecret,
		provider2.CapabilityReferentAuthentication,
		provider2.CapabilityMetadataPolicyFetch,
		provider2.CapabilityValidateStore,
	}
)

func main() {
	flag.Parse()

	matrix := provider2.List()

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

func generateJSON(matrix map[provider2.Name]provider2.Metadata) string {
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
	ExtraCapabilities     []provider2.Capability
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

func generateMarkdown(matrix map[provider2.Name]provider2.Metadata) string {
	var unsortedTable providerTable
	for pn, metadata := range matrix {
		line := providerInfo{}
		line.Name = string(pn)
		line.Stability = string(metadata.Stability)
		line.DisplayedCapabilities = make(map[string]bool)
		for _, capability := range capabilityShortList {
			line.DisplayedCapabilities[string(capability)] = provider2.HasCapability(string(pn), capability)
		}

		// Extend the info with data outside the short list
		for _, providerCapability := range metadata.Capabilities {
			if _, ok := line.DisplayedCapabilities[string(providerCapability.Name)]; !ok {
				line.ExtraCapabilities = append(line.ExtraCapabilities, providerCapability)
			}
		}
		unsortedTable = append(unsortedTable, line)
	}
	sort.Sort(unsortedTable)

	// Execute template
	tmpl, err := template.New("markdown").Parse(markdownTemplate)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, unsortedTable); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}

	return buf.String()
}
