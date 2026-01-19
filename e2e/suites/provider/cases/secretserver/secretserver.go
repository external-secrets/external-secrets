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
	"context"
	_ "fmt"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = PDescribe("[secretserver]", Label("secretserver"), func() {

	f := framework.New("eso-secretserver")

	// Initialization is deferred so that assertions work.
	provider := &secretStoreProvider{}

	BeforeEach(func() {

		cfg, err := loadConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())

		provider.init(cfg, f)
		createResources(GinkgoT().Context(), f, cfg)
	})

	DescribeTable("sync secrets", framework.TableFuncWithExternalSecret(f, provider),
		Entry(common.JSONDataWithTemplate(f)),
		Entry(common.JSONDataWithProperty(f)),
		Entry(common.JSONDataWithoutTargetName(f)),
		Entry(common.JSONDataWithTemplateFromLiteral(f)),
		Entry(common.TemplateFromConfigmaps(f)),
		Entry(common.JSONDataFromSync(f)),    // <--
		Entry(common.JSONDataFromRewrite(f)), // <--
		Entry(common.NestedJSONWithGJSON(f)),
		Entry(common.DockerJSONConfig(f)),
		Entry(common.DataPropertyDockerconfigJSON(f)),
		Entry(common.SSHKeySyncDataProperty(f)),
		Entry(common.DecodingPolicySync(f)), // <--
	)
})

func createResources(ctx context.Context, f *framework.Framework, cfg *config) {

	secretName := "secretserver-credential"
	secretKey := "password"
	// Creating a secret to hold the Delinea client secret.
	secretSpec := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: f.Namespace.Name,
		},
		StringData: map[string]string{
			secretKey: cfg.password,
		},
	}

	err := f.CRClient.Create(ctx, &secretSpec)
	Expect(err).ToNot(HaveOccurred())

	// Creating SecretStore.
	secretStoreSpec := esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.Namespace.Name,
			Namespace: f.Namespace.Name,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				SecretServer: &esv1.SecretServerProvider{
					ServerURL: cfg.serverURL,
					Username: &esv1.SecretServerProviderRef{
						Value: cfg.username,
					},
					Password: &esv1.SecretServerProviderRef{
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
	Expect(err).ToNot(HaveOccurred())
}
