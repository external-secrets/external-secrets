package scaleway

import (
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/onsi/gomega"
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const remoteRefPrefix = "name:"
const cleanupTag = "eso-e2e" // tag for easy cleanup

type secretStoreProvider struct {
	api *smapi.API
	cfg *config
}

func (p *secretStoreProvider) init(cfg *config) {

	p.cfg = cfg

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

// cleanup prevents accumulation of secrets after aborted runs.
func (p *secretStoreProvider) cleanup() {

	for {
		listResp, err := p.api.ListSecrets(&smapi.ListSecretsRequest{
			ProjectID: &p.cfg.projectId,
			Tags:      []string{cleanupTag},
		})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		for _, secret := range listResp.Secrets {
			err := p.api.DeleteSecret(&smapi.DeleteSecretRequest{
				SecretID: secret.ID,
			})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		if uint32(len(listResp.Secrets)) == listResp.TotalCount {
			break
		}
	}
}

func (p *secretStoreProvider) CreateSecret(key string, val framework.SecretEntry) {

	gomega.Expect(key).To(gomega.HavePrefix(remoteRefPrefix))
	secretName := key[len(remoteRefPrefix):]

	var tags []string
	for tag := range val.Tags {
		tags = append(tags, tag)
	}

	tags = append(tags, cleanupTag)

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

	gomega.Expect(key).To(gomega.HavePrefix(remoteRefPrefix))
	secretName := key[len(remoteRefPrefix):]

	p.api.GetSecret(&smapi.GetSecretRequest{
		Region:   "",
		SecretID: "",
	})
	res, err := p.api.ListSecrets(&smapi.ListSecretsRequest{
		Name: &secretName,
	})
	if _, isErrNotFound := err.(*scw.ResourceNotFoundError); isErrNotFound {
		return
	}
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	for _, secret := range res.Secrets {
		err = p.api.DeleteSecret(&smapi.DeleteSecretRequest{
			SecretID: secret.ID,
		})
		if _, isErrNotFound := err.(*scw.ResourceNotFoundError); isErrNotFound {
			return
		}
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}
