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
	_ "github.com/external-secrets/external-secrets/pkg/register" // to register all built providers
	"github.com/external-secrets/external-secrets/runtime/provider"
	"github.com/stretchr/testify/assert"
)

func TestSelectiveBuild(t *testing.T) {
	// This test verifies that build tags work correctly
	// When run with specific tags, only those providers should be registered

	providers := provider.List()
	t.Logf("Registered providers: %d", len(providers))

	// Should have at least one provider
	assert.NotEmpty(t, providers, "No providers registered - check build tags")

	// Log what we got
	for name := range providers {
		t.Logf("  - %s", name)
	}
}

func TestAllProvidersRegisteredInAPIRegistry(t *testing.T) {
	// Test that metadata and provider registration are in sync
	// This test is future-proof: it works with any set of providers
	// based on the build tags used.

	providers := provider.List()
	assert.NotEmpty(t, providers, "No providers registered metadata")

	for providerName := range providers {
		// Verify provider is also registered in API
		pName := string(providerName)
		_, ok := esv1.GetProviderByName(pName)
		assert.True(t, ok,
			"Provider %s has metadata but is not registered in API", providerName)
	}
}

func TestAllProvidersRegisteredInProviderRegistry(t *testing.T) {
	ProvidersRegisteredInAPI := esv1.List()
	assert.NotEmpty(t, ProvidersRegisteredInAPI, "No providers registered in API")
	for providerName := range ProvidersRegisteredInAPI {
		_, ok := provider.Get(providerName)
		assert.True(t, ok, "Provider %s is in API but has no registered metadata", providerName)
	}
}

func TestProvidersHaveStability(t *testing.T) {
	providers := provider.List()
	assert.NotEmpty(t, providers, "No providers registered in API")

	for pName := range providers {
		meta, _ := provider.Get(string(pName))
		assert.NotEmpty(t, meta.Stability, "Provider %s has no stability", string(pName))
	}
}
