package scaleway

import (
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/onsi/gomega"
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type secretStoreProvider struct {
	api *smapi.API
}

func (p *secretStoreProvider) init(cfg *config) {

	options := []scw.ClientOption{
		scw.WithDefaultRegion(scw.Region(cfg.region)),
		scw.WithDefaultProjectID(cfg.projectId),
		scw.WithAuth(cfg.accessKey, cfg.secretKey),
	}

	if cfg.apiUrl != nil {
		options = append(options, scw.WithAPIURL(*cfg.apiUrl))
	}

	scwClient, err := scw.NewClient(options...)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	p.api = smapi.NewAPI(scwClient)
}

func (p *secretStoreProvider) CreateSecret(key string, val framework.SecretEntry) {

	gomega.Expect(key).To(gomega.HavePrefix("name:"))
	secretName := key[len("name:"):]

	var tags []string
	for tag := range val.Tags {
		tags = append(tags, tag)
	}

	secret, err := p.api.CreateSecret(&smapi.CreateSecretRequest{
		Name: secretName,
		Tags: tags,
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	_, err = p.api.CreateSecretVersion(&smapi.CreateSecretVersionRequest{
		SecretID: secret.ID,
		Data:     []byte(val.Value),
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

func (p *secretStoreProvider) DeleteSecret(key string) {

	gomega.Expect(key).To(gomega.HavePrefix("name:"))
	secretName := key[len("name:"):]

	secret, err := p.api.GetSecretByName(&smapi.GetSecretByNameRequest{
		SecretName: secretName,
	})
	if _, isErrNotFound := err.(*scw.ResourceNotFoundError); isErrNotFound {
		return
	}
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	err = p.api.DeleteSecret(&smapi.DeleteSecretRequest{
		SecretID: secret.ID,
	})
	if _, isErrNotFound := err.(*scw.ResourceNotFoundError); isErrNotFound {
		return
	}
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}
