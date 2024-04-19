package secretserver

import (
	"encoding/json"
	"fmt"
	"strconv"

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

	fields := make([]server.SecretField, 3)
		fields[0].FieldID = 108 // machine
		fields[0].ItemValue = "Secret Server TEST MACHINE"
		fields[1].FieldID = 111 // username
		fields[1].ItemValue = "secretserver_username"
		fields[2].FieldID = 110 // password
		fields[2].ItemValue = "secretserver_password"

	_, err = p.api.CreateSecret(server.Secret{
		SecretTemplateID: 6007,
		SiteID: 1,
		FolderID: 73,
		Name: key,
		Fields: fields,
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

func (p *secretStoreProvider) DeleteSecret(key string) {
	fmt.Println("DELETE SECRET KEY = ", key)
	id, _ := strconv.Atoi(key)
/*
	if err != nil {
		return nil, errors.New("incorrect string to integer conversion")
	}
*/
	err := p.api.DeleteSecret(id)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}
