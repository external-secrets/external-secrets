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
package conjur

import (
	// nolint
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
)

const (
	withTokenAuth    = "with apikey auth"
	withJWTK8s       = "with jwt k8s provider"
	withJWTK8sHostID = "with jwt k8s hostid provider"
)

var _ = Describe("[conjur]", Label("conjur"), Ordered, func() {
	f := framework.New("eso-conjur")
	conjur := addon.NewConjur()
	prov := newConjurProvider(f, conjur)

	BeforeAll(func() {
		addon.InstallGlobalAddon(conjur)
	})

	DescribeTable("sync secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		// use api key auth
		framework.Compose(withTokenAuth, f, common.FindByName, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.FindByNameAndRewrite, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.FindByTag, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.SimpleDataSync, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.SyncWithoutTargetName, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataFromSync, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataFromRewrite, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataWithProperty, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataWithTemplate, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.DataPropertyDockerconfigJSON, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataWithoutTargetName, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.DecodingPolicySync, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataWithTemplateFromLiteral, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.TemplateFromConfigmaps, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.SSHKeySync, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.SSHKeySyncDataProperty, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.DockerJSONConfig, useApiKeyAuth(prov)),
		framework.Compose(withTokenAuth, f, common.NestedJSONWithGJSON, useApiKeyAuth(prov)),

		// use jwt k8s provider
		framework.Compose(withJWTK8s, f, common.FindByName, useJWTK8sProvider(prov)),
		framework.Compose(withJWTK8s, f, common.FindByNameAndRewrite, useJWTK8sProvider(prov)),
		framework.Compose(withJWTK8s, f, common.FindByTag, useJWTK8sProvider(prov)),
		framework.Compose(withJWTK8s, f, common.SimpleDataSync, useJWTK8sProvider(prov)),
		framework.Compose(withJWTK8s, f, common.SyncWithoutTargetName, useJWTK8sProvider(prov)),
		framework.Compose(withJWTK8s, f, common.JSONDataFromSync, useJWTK8sProvider(prov)),
		framework.Compose(withJWTK8s, f, common.JSONDataFromRewrite, useJWTK8sProvider(prov)),

		// use jwt k8s hostid provider
		framework.Compose(withJWTK8sHostID, f, common.FindByName, useJWTK8sHostIDProvider(prov)),
		framework.Compose(withJWTK8sHostID, f, common.FindByNameAndRewrite, useJWTK8sHostIDProvider(prov)),
		framework.Compose(withJWTK8sHostID, f, common.FindByTag, useJWTK8sHostIDProvider(prov)),
		framework.Compose(withJWTK8sHostID, f, common.SimpleDataSync, useJWTK8sHostIDProvider(prov)),
		framework.Compose(withJWTK8sHostID, f, common.SyncWithoutTargetName, useJWTK8sHostIDProvider(prov)),
		framework.Compose(withJWTK8sHostID, f, common.JSONDataFromSync, useJWTK8sHostIDProvider(prov)),
		framework.Compose(withJWTK8sHostID, f, common.JSONDataFromRewrite, useJWTK8sHostIDProvider(prov)),
	)
})

func useApiKeyAuth(prov *conjurProvider) func(tc *framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateApiKeyStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = defaultStoreName
	}
}

func useJWTK8sProvider(prov *conjurProvider) func(tc *framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateJWTK8sStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = jwtK8sProviderName
	}
}

func useJWTK8sHostIDProvider(prov *conjurProvider) func(tc *framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateJWTK8sHostIDStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = jwtK8sHostIDProviderName
	}
}
