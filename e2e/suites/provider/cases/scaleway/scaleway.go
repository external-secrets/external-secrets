package scaleway

import (
	"context"
	"sync"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var cleanupOnce sync.Once

var _ = ginkgo.Describe("[scaleway]", ginkgo.Label("scaleway"), func() {

	f := framework.New("eso-scaleway")
	f.MakeRemoteRefKey = func(base string) string {
		return "name:" + base
	}

	// Initialization is deferred so that assertions work.
	provider := &secretStoreProvider{}

	ginkgo.BeforeEach(func() {

		cfg, err := loadConfigFromEnv()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		provider.init(cfg)

		cleanupOnce.Do(provider.cleanup)

		createResources(context.Background(), f, cfg)
	})

	ginkgo.DescribeTable("sync secrets", framework.TableFuncWithExternalSecret(f, provider),

		//ginkgo.Entry(common.SyncV1Alpha1(f)), // not supported
		ginkgo.Entry(common.SimpleDataSync(f)),
		ginkgo.Entry(common.SyncWithoutTargetName(f)),
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
		ginkgo.Entry(common.SSHKeySync(f)),
		ginkgo.Entry(common.SSHKeySyncDataProperty(f)),
		ginkgo.Entry(common.DeletionPolicyDelete(f)),
		//ginkgo.Entry(common.DecodingPolicySync(f)), // not supported

		ginkgo.Entry(common.FindByName(f)),
		ginkgo.Entry(common.FindByNameAndRewrite(f)),
		//ginkgo.Entry(common.FindByNameWithPath(f)), // not supported

		ginkgo.Entry(common.FindByTag(f)),
		//ginkgo.Entry(common.FindByTagWithPath(f)), // not supported
	)
})

func createResources(ctx context.Context, f *framework.Framework, cfg *config) {

	apiKeySecretName := "scw-api-key"
	apiKeySecretKey := "secret-key"

	// Creating a secret to hold the API key.

	secretSpec := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiKeySecretName,
			Namespace: f.Namespace.Name,
		},
		StringData: map[string]string{
			"secret-key": cfg.secretKey,
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
				Scaleway: &esv1beta1.ScalewayProvider{
					Region:    cfg.region,
					ProjectID: cfg.projectId,
					AccessKey: &esv1beta1.ScalewayProviderSecretRef{
						Value: cfg.accessKey, // TODO: test with secretRef as well
					},
					SecretKey: &esv1beta1.ScalewayProviderSecretRef{
						SecretRef: &esmeta.SecretKeySelector{
							Name: apiKeySecretName,
							Key:  apiKeySecretKey,
						},
					},
				},
			},
		},
	}

	if cfg.apiUrl != nil {
		secretStoreSpec.Spec.Provider.Scaleway.APIURL = *cfg.apiUrl
	}

	err = f.CRClient.Create(ctx, &secretStoreSpec)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}
