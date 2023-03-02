package scaleway

import (
	"context"
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: cleanup resources from previous runs

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

		createResources(context.Background(), f, cfg)
	})

	ginkgo.DescribeTable("sync secrets", framework.TableFunc(f, provider),

		//framework.Compose("", f, common.SyncV1Alpha1, useStaticAuth), // not supported
		framework.Compose("", f, common.SimpleDataSync, useStaticAuth),
		framework.Compose("", f, common.SyncWithoutTargetName, useStaticAuth),
		framework.Compose("", f, common.JSONDataWithProperty, useStaticAuth),
		framework.Compose("", f, common.JSONDataWithoutTargetName, useStaticAuth),
		framework.Compose("", f, common.JSONDataWithTemplate, useStaticAuth),
		framework.Compose("", f, common.JSONDataWithTemplateFromLiteral, useStaticAuth),
		framework.Compose("", f, common.TemplateFromConfigmaps, useStaticAuth),
		framework.Compose("", f, common.JSONDataFromSync, useStaticAuth),
		framework.Compose("", f, common.JSONDataFromRewrite, useStaticAuth),
		framework.Compose("", f, common.NestedJSONWithGJSON, useStaticAuth),
		framework.Compose("", f, common.DockerJSONConfig, useStaticAuth),
		framework.Compose("", f, common.DataPropertyDockerconfigJSON, useStaticAuth),
		framework.Compose("", f, common.SSHKeySync, useStaticAuth),
		framework.Compose("", f, common.SSHKeySyncDataProperty, useStaticAuth),
		framework.Compose("", f, common.DeletionPolicyDelete, useStaticAuth),
		//framework.Compose("", f, common.DecodingPolicySync, useStaticAuth), // not supported

		framework.Compose("", f, common.FindByName, useStaticAuth),
		framework.Compose("", f, common.FindByNameAndRewrite, useStaticAuth),
		//framework.Compose("", f, common.FindByNameWithPath, useStaticAuth), // not supported

		framework.Compose("", f, common.FindByTag, useStaticAuth),
		//framework.Compose("", f, common.FindByTagWithPath, useStaticAuth), // not supported
	)
})

func useStaticAuth(tc *framework.TestCase) {

	// TODO

	tc.ExternalSecret.Spec.SecretStoreRef.Name = tc.Framework.Namespace.Name
	if tc.ExternalSecretV1Alpha1 != nil {
		tc.ExternalSecretV1Alpha1.Spec.SecretStoreRef.Name = tc.Framework.Namespace.Name
	}
}

func createResources(ctx context.Context, f *framework.Framework, cfg *config) {

	secretStoreSpec := esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.Namespace.Name,
			Namespace: f.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Scaleway: &esv1beta1.ScalewayProvider{
					Region:    cfg.region,
					ProjectId: cfg.projectId,
					AccessKey: &esv1beta1.ScalewayProviderSecretRef{
						Value: cfg.accessKey, // TODO: test with secretRef as well
					},
					SecretKey: &esv1beta1.ScalewayProviderSecretRef{
						Value: cfg.secretKey, // TODO: test with secretRef as well
					},
				},
			},
		},
	}

	if cfg.apiUrl != nil {
		secretStoreSpec.Spec.Provider.Scaleway.ApiUrl = *cfg.apiUrl
	}

	err := f.CRClient.Create(ctx, &secretStoreSpec)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}
