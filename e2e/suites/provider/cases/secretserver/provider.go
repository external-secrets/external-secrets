package secretserver

import (
	"encoding/json"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/onsi/gomega"
)


type secretStoreProvider struct {
	api *server.Server
	cfg *config
	framework *framework.Framework
	secretID map[string]int
}

func (p *secretStoreProvider) init(cfg *config, f *framework.Framework) {
	p.cfg = cfg
	p.secretID = make(map[string]int)
	p.framework = f
	secretserverClient, err := server.New(server.Configuration{
		Credentials: server.UserCredential{
			Username: cfg.username,
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

	fields := make([]server.SecretField, 1)
	fields[0].FieldID = 329 // Data
	fields[0].ItemValue = val.Value

	s, err := p.api.CreateSecret(server.Secret{
		SecretTemplateID: 6051, // custom template
		SiteID: 1,
		FolderID: 10,
		Name: key,
		Fields: fields,
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	p.secretID[key] = s.ID
}

func (p *secretStoreProvider) DeleteSecret(key string) {
	err := p.api.DeleteSecret(p.secretID[key])
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}
