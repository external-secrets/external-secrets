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
