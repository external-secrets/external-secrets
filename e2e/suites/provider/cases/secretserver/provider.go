package secretserver

import (
	"encoding/json"
	"fmt"
	_"strconv"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	"github.com/external-secrets/external-secrets-e2e/framework"
	_"github.com/tidwall/gjson"
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

/*
Make sure and look this up
https://rasteamdev.qa.devsecretservercloud.com/Documents/restapi/TokenAuth/#tag/Secrets/operation/SecretsService_SearchV2
*/

func (p *secretStoreProvider) CreateSecret(key string, val framework.SecretEntry) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(val.Value), &data)
	fmt.Printf("\n\n CREATE SECRET VALUE = %+v", string(val.Value))
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
		fields[0].ItemValue = val.Value


	s, err := p.api.CreateSecret(server.Secret{
		SecretTemplateID: 6098, // custom template
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
/*	err := p.api.DeleteSecret(1111)*/
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}
