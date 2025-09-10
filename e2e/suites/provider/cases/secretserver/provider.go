/*
Copyright © 2025 ESO Maintainer Team

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

package secretserver

import (
	"encoding/json"

	"github.com/DelineaXPM/tss-sdk-go/v3/server"
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/onsi/gomega"
)

type secretStoreProvider struct {
	api       *server.Server
	cfg       *config
	framework *framework.Framework
	secretID  map[string]int
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
		ServerURL: cfg.serverURL,
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
		SiteID:           1,
		FolderID:         10,
		Name:             key,
		Fields:           fields,
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	p.secretID[key] = s.ID
}

func (p *secretStoreProvider) DeleteSecret(key string) {
	err := p.api.DeleteSecret(p.secretID[key])
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}
