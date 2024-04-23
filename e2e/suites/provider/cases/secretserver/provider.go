package secretserver

import (
	"encoding/json"
	_"fmt"
	_"strconv"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
/*	"github.com/DelineaXPM/dsv-sdk-go/v2/vault"*/
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/onsi/gomega"
)


type secretStoreProvider struct {
	api *server.Server
	cfg *config
	secretID map[string]int
}


func (p *secretStoreProvider) init(cfg *config) {

	p.cfg = cfg
	p.secretID = make(map[string]int)
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

/*
Make sure and look this up
https://rasteamdev.qa.devsecretservercloud.com/Documents/restapi/TokenAuth/#tag/Secrets/operation/SecretsService_SearchV2
*/

func (p *secretStoreProvider) CreateSecret(key string, val framework.SecretEntry) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(val.Value), &data)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	fields := make([]server.SecretField, 1)
/*
		fields[0].FieldID = 108 // machine
		fields[0].ItemValue = "Secret Server TEST MACHINE"
		fields[1].FieldID = 111 // username
		fields[1].ItemValue = "secretserver_username"
		fields[2].FieldID = 110 // password
		fields[2].ItemValue = "secretserver_password"
*/

		fields[0].FieldID = 439 // Data
		fields[0].ItemValue = "{\"key\":\"foo\"}"


	s, err := p.api.CreateSecret(server.Secret{
		SecretTemplateID: 6098,
		SiteID: 1,
		FolderID: 73,
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
