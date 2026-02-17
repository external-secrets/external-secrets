//go:build all_providers

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

package provider_test

import (
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	_ "github.com/external-secrets/external-secrets/pkg/register" // registers all built providers
	"github.com/external-secrets/external-secrets/runtime/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectiveBuild(t *testing.T) {
	providers := provider.List()
	t.Logf("Registered providers: %d", len(providers))
	assert.NotEmpty(t, providers, "No providers registered - check build tags")
	for name := range providers {
		t.Logf("  - %s", name)
	}
}

// TestSingleRegistrySync verifies the single registry backs both provider.List()
// and esv1.GetProviderByName() / esv1.List() via hooks.
func TestSingleRegistrySync(t *testing.T) {
	providers := provider.List()
	require.NotEmpty(t, providers, "No providers registered")

	for providerName, entry := range providers {
		pName := string(providerName)

		// The esv1 stubs must resolve to the same provider instance via hooks.
		p, ok := esv1.GetProviderByName(pName)
		assert.True(t, ok, "Provider %s is in runtime registry but esv1.GetProviderByName() returned false", pName)
		assert.Equal(t, entry.Provider, p, "Provider %s: esv1.GetProviderByName() returned different instance than registry", pName)
	}

	// esv1.List() must also return the same set.
	apiProviders := esv1.List()
	assert.Equal(t, len(providers), len(apiProviders),
		"provider.List() and esv1.List() have different lengths — single registry out of sync")
}

func TestAllProvidersHaveStability(t *testing.T) {
	providers := provider.List()
	require.NotEmpty(t, providers, "No providers registered")

	for pName, entry := range providers {
		assert.NotEmpty(t, entry.Metadata.Stability,
			"Provider %s has no stability set", string(pName))
	}
}

func TestAllProvidersHaveProviderInstance(t *testing.T) {
	providers := provider.List()
	require.NotEmpty(t, providers, "No providers registered")

	for pName, entry := range providers {
		assert.NotNil(t, entry.Provider,
			"Provider %s has a registry entry but nil Provider instance", string(pName))
	}
}
