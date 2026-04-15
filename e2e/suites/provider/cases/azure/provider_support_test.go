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

package azure

import (
	"reflect"
	"testing"
)

func TestAzureStaticEnvConfigMissingEnv(t *testing.T) {
	t.Parallel()

	cfg := azureStaticEnvConfig{}
	want := []string{
		"TFC_VAULT_URL",
		"TFC_AZURE_TENANT_ID",
		"TFC_AZURE_CLIENT_ID",
		"TFC_AZURE_CLIENT_SECRET",
	}

	if got := cfg.missingStaticEnv(); !reflect.DeepEqual(got, want) {
		t.Fatalf("missingStaticEnv() = %v, want %v", got, want)
	}
}

func TestAzureStaticEnvConfigMissingEnvComplete(t *testing.T) {
	t.Parallel()

	cfg := azureStaticEnvConfig{
		VaultURL:     "https://example.vault.azure.net/",
		TenantID:     "tenant",
		ClientID:     "client",
		ClientSecret: "secret",
	}

	if got := cfg.missingStaticEnv(); len(got) != 0 {
		t.Fatalf("missingStaticEnv() = %v, want none", got)
	}
}
