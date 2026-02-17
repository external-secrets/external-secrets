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
	"testing"

	runtimeprovider "github.com/external-secrets/external-secrets/runtime/provider"
)

//func TestGenerateMarkdown(t *testing.T) {
//	awsMetadata := runtimeprovider.Metadata{
//		Stability: runtimeprovider.StabilityStable,
//		Capabilities: []runtimeprovider.Capability{
//			{Name: runtimeprovider.CapabilityGetSecret, Notes: "Supports Secrets Manager and Parameter Store"},
//			{Name: runtimeprovider.CapabilityGetSecretMap, Notes: "Secret map note"},
//			{Name: runtimeprovider.CapabilityPushSecret},
//			{Name: runtimeprovider.CapabilityDeleteSecret},
//		},
//		Comment: "AWS Secrets Manager and Parameter Store provider",
//	}
//	vaultMetadata := runtimeprovider.Metadata{
//		Stability: runtimeprovider.StabilityStable,
//		Capabilities: []runtimeprovider.Capability{
//			{Name: runtimeprovider.CapabilityGetSecret},
//			{Name: runtimeprovider.CapabilityGetSecretMap},
//			{Name: runtimeprovider.CapabilityPushSecret},
//			{Name: runtimeprovider.CapabilityFindByName},
//			{Name: runtimeprovider.CapabilityFindByTag},
//		},
//		Comment: "HashiCorp Vault provider with KV v1 and v2 support",
//	}
//	doplerMetadata := runtimeprovider.Metadata{
//		Stability: runtimeprovider.StabilityBeta,
//		Capabilities: []runtimeprovider.Capability{
//			{Name: runtimeprovider.CapabilityGetSecret},
//		},
//		Comment: "Doppler secrets management provider",
//	}
//
//	// Register some mock providers to demonstrate the matrix
//	runtimeprovider.Register("aws", awsMetadata)
//
//	runtimeprovider.Register("vault", vaultMetadata)
//
//	runtimeprovider.Register("doppler", doplerMetadata)
//	pL := runtimeprovider.List()
//	output := generateMarkdown(pL)
//
//	// 1. Verify there are 3 provider lines in the capability matrix table
//	lines := strings.Split(output, "\n")
//	providerCount := 0
//	inMatrix := false
//	matrixEndReached := false
//
//	for _, line := range lines {
//		if strings.Contains(line, "## Capabilities Matrix") {
//			inMatrix = true
//			continue
//		}
//		if inMatrix && strings.Contains(line, "## Extra Capabilities") {
//			matrixEndReached = true
//			break
//		}
//
//		// Count table rows that start with "| " and contain provider names
//		// Skip header lines (those with "Provider" or dashes)
//		if inMatrix && strings.HasPrefix(line, "| ") &&
//			!strings.Contains(line, "Provider") &&
//			!strings.Contains(line, "-------") {
//			providerCount++
//		}
//	}
//	if !matrixEndReached {
//		t.Error("Could not find the end of the capability matrix section")
//	}
//
//	if providerCount != len(pL) {
//		t.Errorf("Expected provider lines in the capability matrix table equal to provider amount, got %d", providerCount)
//	}
//
//	// 2. Verify each provider has GetSecret ticked (marked with ✓)
//	expectedProviders := []string{"aws", "vault", "doppler"}
//	for _, providerName := range expectedProviders {
//		// Look for a line that starts with the provider name and has a checkmark in GetSecret column
//		// The format is: | providerName | Stability | ✓ |... (GetSecret is the first capability column)
//		found := false
//		for _, line := range lines {
//			if strings.HasPrefix(line, "| "+providerName+" ") && strings.Contains(line, " ✓ ") {
//				found = true
//				break
//			}
//		}
//		if !found {
//			t.Errorf("Provider %s does not have GetSecret capability ticked in the matrix", providerName)
//		}
//
//		// 3. Verify each provider has Extra Capabilities section filled in
//		for _, providerName := range expectedProviders {
//			// Look for the provider section under "## Extra Capabilities per provider"
//			sectionHeader := "### " + providerName
//			if !strings.Contains(output, sectionHeader) {
//				t.Errorf("Provider %s is missing from the Extra Capabilities per provider section", providerName)
//				continue
//			}
//
//			// Find the section and verify it has capability information
//			sectionStart := strings.Index(output, sectionHeader)
//			if sectionStart == -1 {
//				continue
//			}
//
//			// Extract the section content (up to the next ### or end)
//			remainingOutput := output[sectionStart:]
//			nextSectionStart := strings.Index(remainingOutput[len(sectionHeader):], "###")
//			var sectionContent string
//			if nextSectionStart > 0 {
//				sectionContent = remainingOutput[:len(sectionHeader)+nextSectionStart]
//			} else {
//				sectionContent = remainingOutput
//			}
//
//			// Verify the section is not empty and contains capability information
//			// It should NOT contain "No extra capabilities declared" for our test providers
//			if strings.Contains(sectionContent, "*No extra capabilities declared*") {
//				t.Errorf("Provider %s has no extra capabilities listed, but should have some", providerName)
//			}
//
//			// Verify it contains "Supported Extra Capabilities" for providers with extra capabilities
//			if !strings.Contains(sectionContent, "**Supported Extra Capabilities:**") {
//				t.Errorf("Provider %s extra capabilities section does not contain the expected header", providerName)
//			}
//		}
//	}
//}
//
//func Test_generateMarkdown_table_has_two_lines(t *testing.T) {
//	awsMetadata := runtimeprovider.Metadata{
//		Stability: runtimeprovider.StabilityStable,
//		Capabilities: []runtimeprovider.Capability{
//			{Name: runtimeprovider.CapabilityGetSecret, Notes: "Supports Secrets Manager and Parameter Store"},
//			{Name: runtimeprovider.CapabilityGetSecretMap, Notes: "Secret map note"},
//			{Name: runtimeprovider.CapabilityPushSecret},
//			{Name: runtimeprovider.CapabilityDeleteSecret},
//		},
//		Comment: "AWS Secrets Manager and Parameter Store provider",
//	}
//	vaultMetadata := runtimeprovider.Metadata{
//		Stability: runtimeprovider.StabilityStable,
//		Capabilities: []runtimeprovider.Capability{
//			{Name: runtimeprovider.CapabilityGetSecret},
//			{Name: runtimeprovider.CapabilityGetSecretMap},
//			{Name: runtimeprovider.CapabilityPushSecret},
//		},
//		Comment: "HashiCorp Vault provider with KV v1 and v2 support",
//	}
//
//	var tests = []struct {
//		name string
//		args []runtimeprovider.Metadata
//		want int
//	}{
//		{
//			name: "Two lines in the matrix for two providers",
//			args: []runtimeprovider.Metadata{
//				awsMetadata,
//				vaultMetadata,
//			},
//			want: 2,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			for i, provider := range tt.args {
//				runtimeprovider.Register(fmt.Sprintf("provider%d", i), provider)
//			}
//
//			if got := generateMarkdown(tt.args.matrix); got != tt.want {
//				t.Errorf("generateMarkdown() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_generateMarkdown_has_detailed_output(t *testing.T) {
//	awsMetadata := runtimeprovider.Metadata{
//		Stability: runtimeprovider.StabilityStable,
//		Capabilities: []runtimeprovider.Capability{
//			{Name: runtimeprovider.CapabilityGetSecret, Notes: "Supports Secrets Manager and Parameter Store"},
//			{Name: runtimeprovider.CapabilityGetSecretMap, Notes: "Secret map note"},
//			{Name: runtimeprovider.CapabilityPushSecret},
//			{Name: runtimeprovider.CapabilityDeleteSecret},
//		},
//		Comment: "AWS Secrets Manager and Parameter Store provider",
//	}
//	vaultMetadata := runtimeprovider.Metadata{
//		Stability: runtimeprovider.StabilityStable,
//		Capabilities: []runtimeprovider.Capability{
//			{Name: runtimeprovider.CapabilityGetSecret},
//			{Name: runtimeprovider.CapabilityGetSecretMap},
//			{Name: runtimeprovider.CapabilityPushSecret},
//			{Name: runtimeprovider.CapabilityFindByName},
//			{Name: runtimeprovider.CapabilityFindByTag},
//		},
//		Comment: "HashiCorp Vault provider with KV v1 and v2 support",
//	}
//	dopplerMetadata := runtimeprovider.Metadata{
//		Stability: runtimeprovider.StabilityBeta,
//		Capabilities: []runtimeprovider.Capability{
//			{Name: runtimeprovider.CapabilityGetSecret},
//		},
//		Comment: "Doppler secrets management provider",
//	}
//
//	var tests = []struct {
//		name string
//		args []runtimeprovider.Metadata
//		want string
//	}{
//		{
//			name: "Two lines in the matrix for two providers",
//			args: []runtimeprovider.Metadata{
//				awsMetadata,
//				vaultMetadata,
//			},
//			want: 2,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			runtimeprovider.Register("aws", awsMetadata)
//			if got := generateMarkdown(tt.args.matrix); got != tt.want {
//				t.Errorf("generateMarkdown() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

func Test_providerTable_Len(t *testing.T) {
	alpha := providerInfo{
		Name:                  "a",
		Stability:             string(runtimeprovider.StabilityAlpha),
		DisplayedCapabilities: map[string]bool{},
		ExtraCapabilities:     []runtimeprovider.Capability{},
	}
	beta := providerInfo{
		Name:                  "b",
		Stability:             string(runtimeprovider.StabilityBeta),
		DisplayedCapabilities: map[string]bool{},
		ExtraCapabilities:     []runtimeprovider.Capability{},
	}
	tests := []struct {
		name string
		p    providerTable
		want int
	}{
		{
			name: "empty table",
			p:    providerTable{},
			want: 0,
		},
		{
			name: "single table",
			p:    providerTable{alpha},
			want: 1,
		},
		{
			name: "dual table",
			p:    providerTable{alpha, beta},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Len(); got != tt.want {
				t.Errorf("Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_providerTable_Less(t *testing.T) {
	type args struct {
		i int
		j int
	}

	alpha := providerInfo{
		Name:                  "a",
		Stability:             string(runtimeprovider.StabilityAlpha),
		DisplayedCapabilities: map[string]bool{},
		ExtraCapabilities:     []runtimeprovider.Capability{},
	}
	beta := providerInfo{
		Name:                  "b",
		Stability:             string(runtimeprovider.StabilityBeta),
		DisplayedCapabilities: map[string]bool{},
		ExtraCapabilities:     []runtimeprovider.Capability{},
	}

	tests := []struct {
		name string
		p    providerTable
		args args
		want bool
	}{
		{
			name: "alpha less than beta",
			p:    providerTable{alpha, beta},
			args: args{0, 1},
			want: true,
		},
		{
			name: "beta less than alpha",
			p:    providerTable{alpha, beta},
			args: args{1, 0},
			want: false,
		},
		{
			name: "beta less than alpha (order of provider table)",
			p:    providerTable{beta, alpha},
			args: args{0, 1},
			want: false,
		},
		{
			name: "equal index means no swap",
			p:    providerTable{alpha, beta},
			args: args{0, 0},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Less(tt.args.i, tt.args.j); got != tt.want {
				t.Errorf("Less() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_providerTable_Swap(t *testing.T) {
	alpha := providerInfo{
		Name:                  "a",
		Stability:             string(runtimeprovider.StabilityAlpha),
		DisplayedCapabilities: map[string]bool{},
		ExtraCapabilities:     []runtimeprovider.Capability{},
	}
	beta := providerInfo{
		Name:                  "b",
		Stability:             string(runtimeprovider.StabilityBeta),
		DisplayedCapabilities: map[string]bool{},
		ExtraCapabilities:     []runtimeprovider.Capability{},
	}
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		p    providerTable
		args args
		want providerTable
	}{
		{
			name: "swap alpha with beta",
			p:    providerTable{beta, alpha},
			args: args{0, 1},
			want: providerTable{alpha, beta},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.p.Swap(tt.args.i, tt.args.j)
			for k := range tt.p {
				if tt.p[k].Name != tt.want[k].Name {
					t.Errorf("Swap() index %d = %v, want %v", k, tt.p[k].Name, tt.want[k].Name)
				}
			}
		})
	}
}

func Test_newProviderTable(t *testing.T) {
	firstProviderMetadata := runtimeprovider.Metadata{
		Stability: runtimeprovider.StabilityStable,
		Capabilities: []runtimeprovider.Capability{
			{Name: runtimeprovider.CapabilityGetSecret, Notes: "Supports Secrets Manager and Parameter Store"},
			{Name: runtimeprovider.CapabilityGetSecretMap, Notes: "Secret map note"},
			{Name: runtimeprovider.CapabilityPushSecret},
			{Name: runtimeprovider.CapabilityDeleteSecret},
		},
		Comment: "Provider A",
	}
	anotherProviderMetadata := runtimeprovider.Metadata{
		Stability: runtimeprovider.StabilityAlpha,
		Capabilities: []runtimeprovider.Capability{
			{Name: runtimeprovider.CapabilityGetSecretMap},
		},
		Comment: "Provider B",
	}
	thirdProviderMetadata := runtimeprovider.Metadata{
		Stability: runtimeprovider.StabilityAlpha,
		Capabilities: []runtimeprovider.Capability{
			{Name: runtimeprovider.CapabilityGetSecret},
		},
		Comment: "Provider C",
	}
	providersEmpty := map[runtimeprovider.Name]runtimeprovider.Metadata{}
	providersAB := map[runtimeprovider.Name]runtimeprovider.Metadata{
		runtimeprovider.Name("providerA"): firstProviderMetadata,
		runtimeprovider.Name("providerB"): anotherProviderMetadata,
	}
	providersAA := map[runtimeprovider.Name]runtimeprovider.Metadata{
		runtimeprovider.Name("providerA"): firstProviderMetadata,
		runtimeprovider.Name("providerB"): firstProviderMetadata,
	}
	providersABC := map[runtimeprovider.Name]runtimeprovider.Metadata{
		runtimeprovider.Name("providerA"): firstProviderMetadata,
		runtimeprovider.Name("providerB"): anotherProviderMetadata,
		runtimeprovider.Name("providerC"): thirdProviderMetadata,
	}
	tests := []struct {
		name                    string
		args                    map[runtimeprovider.Name]runtimeprovider.Metadata
		wantProviders           int
		wantMatrixLinesWithTick int
		wantExtraCapabilities   int
	}{
		{
			name:                    "empty table",
			args:                    providersEmpty,
			wantProviders:           0,
			wantMatrixLinesWithTick: 0,
			wantExtraCapabilities:   0,
		},
		{
			name:                    "two providers but only one in matrix",
			args:                    providersAB,
			wantProviders:           2,
			wantMatrixLinesWithTick: 1, //Only A has tick with CapabilityGetSecret
			wantExtraCapabilities:   2, //CapabilityGetSecretMap should appear here
		},
		{
			name:                    "two providers in matrix",
			args:                    providersAA,
			wantProviders:           2,
			wantMatrixLinesWithTick: 2,
			wantExtraCapabilities:   2,
		},
		{
			name:                    "three providers, mixed",
			args:                    providersABC,
			wantProviders:           3,
			wantMatrixLinesWithTick: 2, //A and C
			wantExtraCapabilities:   2, //A and B
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a local registry for this test to avoid global state pollution
			table := newProviderTable(tt.args)
			gotProviders := len(table)
			if gotProviders != tt.wantProviders {
				t.Errorf("newProviderTable() gotProviders = %v, want %v", gotProviders, tt.wantProviders)
			}
			gotProviderTicks := 0
			gotProviderExtraCapabilities := 0
			for _, provider := range table {
				for _, tick := range provider.DisplayedCapabilities {
					if tick {
						gotProviderTicks++
						// provider has at least one tick in its displayed capabilities, moving on to next provider
						break
					}
				}
				if len(provider.ExtraCapabilities) > 0 {
					gotProviderExtraCapabilities++
				}
			}

			if gotProviderTicks != tt.wantMatrixLinesWithTick {
				t.Errorf("Provider table only had %v lines with ticks, wanted %v", gotProviderTicks, tt.wantMatrixLinesWithTick)
			}

			if gotProviderExtraCapabilities != tt.wantExtraCapabilities {
				t.Errorf("%v providers were listed with extra capabilities, expected %v", gotProviderExtraCapabilities, tt.wantExtraCapabilities)
			}
		})
	}
}
