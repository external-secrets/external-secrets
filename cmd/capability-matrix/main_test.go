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
	"strings"
	"testing"

	runtimeprovider "github.com/external-secrets/external-secrets/runtime/provider"
)

func init() {
	// Register some mock providers to demonstrate the matrix
	runtimeprovider.Register("aws", runtimeprovider.Metadata{
		Stability: runtimeprovider.StabilityStable,
		Capabilities: []runtimeprovider.Capability{
			{Name: runtimeprovider.CapabilityGetSecret, Notes: "Supports Secrets Manager and Parameter Store"},
			{Name: runtimeprovider.CapabilityGetSecretMap, Notes: "Secret map note"},
			{Name: runtimeprovider.CapabilityPushSecret},
			{Name: runtimeprovider.CapabilityDeleteSecret},
		},
		Comment: "AWS Secrets Manager and Parameter Store provider",
	})

	runtimeprovider.Register("vault", runtimeprovider.Metadata{
		Stability: runtimeprovider.StabilityStable,
		Capabilities: []runtimeprovider.Capability{
			{Name: runtimeprovider.CapabilityGetSecret},
			{Name: runtimeprovider.CapabilityGetSecretMap},
			{Name: runtimeprovider.CapabilityPushSecret},
			{Name: runtimeprovider.CapabilityFindByName},
			{Name: runtimeprovider.CapabilityFindByTag},
		},
		Comment: "HashiCorp Vault provider with KV v1 and v2 support",
	})

	runtimeprovider.Register("doppler", runtimeprovider.Metadata{
		Stability: runtimeprovider.StabilityBeta,
		Capabilities: []runtimeprovider.Capability{
			{Name: runtimeprovider.CapabilityGetSecret},
			{Name: runtimeprovider.CapabilityGetSecretMap},
		},
		Comment: "Doppler secrets management provider",
	})
}

func TestGenerateMarkdown(t *testing.T) {
	pL := runtimeprovider.List()
	output := generateMarkdown(pL)

	// 1. Verify there are 3 provider lines in the capability matrix table
	lines := strings.Split(output, "\n")
	providerCount := 0
	inMatrix := false
	matrixEndReached := false

	for _, line := range lines {
		if strings.Contains(line, "## CapabilityName Matrix") {
			inMatrix = true
			continue
		}
		if inMatrix && strings.Contains(line, "## Extra Capabilities") {
			matrixEndReached = true
			break
		}
		// Count table rows that start with "| " and contain provider names
		// Skip header lines (those with "Provider" or dashes)
		if inMatrix && strings.HasPrefix(line, "| ") &&
			!strings.Contains(line, "Provider") &&
			!strings.Contains(line, "-------") {
			providerCount++
		}
	}

	if !matrixEndReached {
		t.Error("Could not find the end of the capability matrix section")
	}

	if providerCount != len(pL) {
		t.Errorf("Expected provider lines in the capability matrix table equal to provider amount, got %d", providerCount)
	}

	// 2. Verify each provider has GetSecret ticked (marked with ✓)
	expectedProviders := []string{"aws", "vault", "doppler"}
	for _, providerName := range expectedProviders {
		// Look for a line that starts with the provider name and has a checkmark in GetSecret column
		// The format is: | providerName | Stability | ✓ |... (GetSecret is the first capability column)
		found := false
		for _, line := range lines {
			if strings.HasPrefix(line, "| "+providerName+" ") && strings.Contains(line, " ✓ ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Provider %s does not have GetSecret capability ticked in the matrix", providerName)
		}
	}

	// 3. Verify each provider has Extra Capabilities section filled in
	for _, providerName := range expectedProviders {
		// Look for the provider section under "## Extra Capabilities per provider"
		sectionHeader := "### " + providerName
		if !strings.Contains(output, sectionHeader) {
			t.Errorf("Provider %s is missing from the Extra Capabilities per provider section", providerName)
			continue
		}

		// Find the section and verify it has capability information
		sectionStart := strings.Index(output, sectionHeader)
		if sectionStart == -1 {
			continue
		}

		// Extract the section content (up to the next ### or end)
		remainingOutput := output[sectionStart:]
		nextSectionStart := strings.Index(remainingOutput[len(sectionHeader):], "###")
		var sectionContent string
		if nextSectionStart > 0 {
			sectionContent = remainingOutput[:len(sectionHeader)+nextSectionStart]
		} else {
			sectionContent = remainingOutput
		}

		// Verify the section is not empty and contains capability information
		// It should NOT contain "No extra capabilities declared" for our test providers
		if strings.Contains(sectionContent, "*No extra capabilities declared*") {
			t.Errorf("Provider %s has no extra capabilities listed, but should have some", providerName)
		}

		// Verify it contains "Supported Extra Capabilities" for providers with extra capabilities
		if !strings.Contains(sectionContent, "**Supported Extra Capabilities:**") {
			t.Errorf("Provider %s extra capabilities section does not contain the expected header", providerName)
		}
	}
}
