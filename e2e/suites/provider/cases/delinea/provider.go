package delinea

import (
	"encoding/json"

	"github.com/DelineaXPM/dsv-sdk-go/v2/vault"
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/onsi/gomega"
)

type secretStoreProvider struct {
	api *vault.Vault
	cfg *config
}

func (p *secretStoreProvider) init(cfg *config) {

	p.cfg = cfg

	dsvClient, err := vault.New(vault.Configuration{
		Credentials: vault.ClientCredential{
			ClientID:     cfg.clientID,
			ClientSecret: cfg.clientSecret,
		},
		Tenant:      cfg.tenant,
		URLTemplate: cfg.urlTemplate,
		TLD:         cfg.tld,
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	p.api = dsvClient
}

func (p *secretStoreProvider) CreateSecret(key string, val framework.SecretEntry) {
	var data map[string]any
	err := json.Unmarshal([]byte(val.Value), &data)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	_, err = p.api.CreateSecret(key, &vault.SecretCreateRequest{
		Data: data,
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

func (p *secretStoreProvider) DeleteSecret(key string) {
	err := p.api.DeleteSecret(key)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}
