package delinea

import (
	"encoding/json"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
/*	"github.com/DelineaXPM/dsv-sdk-go/v2/vault"*/
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/onsi/gomega"
)

type secretStoreProvider struct {
	api *server.Server
	cfg *config
}

func (p *secretStoreProvider) init(cfg *config) {

	p.cfg = cfg

	secretserverClient, err := server.New(server.Configuration{
		Credentials: vault.ClientCredential{
			Username:     cfg.username,
			Password: cfg.password,
		},
		ServerURL:      cfg.serverURL,
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	p.api = secretserverClient
}

func (p *secretStoreProvider) CreateSecret(key string, val framework.SecretEntry) {
	var data map[string]interface{}
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
