//go:build vaultwarden || all_providers

package vaultwarden

import (
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func newStore(provider *esv1.VaultwardenProvider) *esv1.SecretStore {
	return &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{Vaultwarden: provider},
		},
	}
}

func TestValidateStore_OrgXOR(t *testing.T) {
	p := &Provider{}
	cases := []struct {
		name    string
		orgID   string
		orgName string
		wantErr bool
	}{
		{"both empty - personal scope OK", "", "", false},
		{"id only - OK", "af061424-2700-425a-8800-80e988194e8e", "", false},
		{"name only - OK", "", "Tiberius-Grail", false},
		{"both set - rejected", "af061424-2700-425a-8800-80e988194e8e", "Tiberius-Grail", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &esv1.VaultwardenProvider{
				URL:              "https://vault.example.com",
				OrganizationID:   tc.orgID,
				OrganizationName: tc.orgName,
			}
			_, err := p.ValidateStore(newStore(provider))
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
