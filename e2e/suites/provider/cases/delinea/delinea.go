package delinea

import (
	"context"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("[delinea]", Label("delinea"), func() {

	f := framework.New("eso-delinea")

	// Initialization is deferred so that assertions work.
	provider := &secretStoreProvider{}

	BeforeEach(func() {

		cfg, err := loadConfigFromEnv()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		provider.init(cfg)

		createResources(GinkgoT().Context(), f, cfg)
	})

	DescribeTable("sync secrets", framework.TableFuncWithExternalSecret(f, provider),

		Entry(common.JSONDataWithProperty(f)),
		Entry(common.JSONDataWithoutTargetName(f)),
		Entry(common.JSONDataWithTemplate(f)),
		Entry(common.JSONDataWithTemplateFromLiteral(f)),
		Entry(common.TemplateFromConfigmaps(f)),
		Entry(common.JSONDataFromSync(f)),
		Entry(common.JSONDataFromRewrite(f)),
		Entry(common.NestedJSONWithGJSON(f)),
		Entry(common.DockerJSONConfig(f)),
		Entry(common.DataPropertyDockerconfigJSON(f)),
		Entry(common.SSHKeySyncDataProperty(f)),
		Entry(common.DecodingPolicySync(f)),
	)
})

func createResources(ctx context.Context, f *framework.Framework, cfg *config) {

	secretName := "delinea-credential"
	secretKey := "client-secret"

	// Creating a secret to hold the Delinea client secret.
	secretSpec := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: f.Namespace.Name,
		},
		StringData: map[string]string{
			secretKey: cfg.clientSecret,
		},
	}

	err := f.CRClient.Create(ctx, &secretSpec)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// Creating SecretStore.
	secretStoreSpec := esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.Namespace.Name,
			Namespace: f.Namespace.Name,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Delinea: &esv1.DelineaProvider{
					Tenant:      cfg.tenant,
					TLD:         cfg.tld,
					URLTemplate: cfg.urlTemplate,
					ClientID: &esv1.DelineaProviderSecretRef{
						Value: cfg.clientID,
					},
					ClientSecret: &esv1.DelineaProviderSecretRef{
						SecretRef: &esmeta.SecretKeySelector{
							Name: secretName,
							Key:  secretKey,
						},
					},
				},
			},
		},
	}

	err = f.CRClient.Create(ctx, &secretStoreSpec)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}
