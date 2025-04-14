package delinea

import (
	"context"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("[delinea]", ginkgo.Label("delinea"), func() {

	f := framework.New("eso-delinea")

	// Initialization is deferred so that assertions work.
	provider := &secretStoreProvider{}

	ginkgo.BeforeEach(func() {

		cfg, err := loadConfigFromEnv()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		provider.init(cfg)

		createResources(context.Background(), f, cfg)
	})

	ginkgo.DescribeTable("sync secrets", framework.TableFuncWithExternalSecret(f, provider),

		ginkgo.Entry(common.JSONDataWithProperty(f)),
		ginkgo.Entry(common.JSONDataWithoutTargetName(f)),
		ginkgo.Entry(common.JSONDataWithTemplate(f)),
		ginkgo.Entry(common.JSONDataWithTemplateFromLiteral(f)),
		ginkgo.Entry(common.TemplateFromConfigmaps(f)),
		ginkgo.Entry(common.JSONDataFromSync(f)),
		ginkgo.Entry(common.JSONDataFromRewrite(f)),
		ginkgo.Entry(common.NestedJSONWithGJSON(f)),
		ginkgo.Entry(common.DockerJSONConfig(f)),
		ginkgo.Entry(common.DataPropertyDockerconfigJSON(f)),
		ginkgo.Entry(common.SSHKeySyncDataProperty(f)),
		ginkgo.Entry(common.DecodingPolicySync(f)),

		// V1Alpha1 is not supported.
		// ginkgo.Entry(common.SyncV1Alpha1(f)),

		// Non-JSON values are not supported by DSV.
		// ginkgo.Entry(common.SimpleDataSync(f)),
		// ginkgo.Entry(common.SyncWithoutTargetName(f)),
		// ginkgo.Entry(common.SSHKeySync(f)),
		// ginkgo.Entry(common.DeletionPolicyDelete(f)),

		// FindByName is not supported.
		// ginkgo.Entry(common.FindByName(f)),
		// ginkgo.Entry(common.FindByNameAndRewrite(f)),
		// ginkgo.Entry(common.FindByNameWithPath(f)),

		// FindByTag is not supported.
		// ginkgo.Entry(common.FindByTag(f)),
		// ginkgo.Entry(common.FindByTagWithPath(f)),
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
	secretStoreSpec := esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.Namespace.Name,
			Namespace: f.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Delinea: &esv1beta1.DelineaProvider{
					Tenant:      cfg.tenant,
					TLD:         cfg.tld,
					URLTemplate: cfg.urlTemplate,
					ClientID: &esv1beta1.DelineaProviderSecretRef{
						Value: cfg.clientID,
					},
					ClientSecret: &esv1beta1.DelineaProviderSecretRef{
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
