/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/
package conjur

import (
	// nolint
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
)

const (
	withTokenAuth    = "with apikey auth"
	withJWTK8s       = "with jwt k8s provider"
	withJWTK8sHostID = "with jwt k8s hostid provider"
)

var _ = Describe("[conjur]", Label("conjur"), func() {
	f := framework.New("eso-conjur")
	prov := newConjurProvider(f)

	DescribeTable("sync secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		// use api key auth
		framework.Compose(withTokenAuth, f, common.FindByName, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.FindByNameAndRewrite, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.FindByTag, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.SimpleDataSync, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.SyncWithoutTargetName, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataFromSync, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataFromRewrite, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataWithProperty, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataWithTemplate, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.DataPropertyDockerconfigJSON, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataWithoutTargetName, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.DecodingPolicySync, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataWithTemplateFromLiteral, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.TemplateFromConfigmaps, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.SSHKeySync, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.SSHKeySyncDataProperty, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.DockerJSONConfig, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.NestedJSONWithGJSON, useApiKeyAuth),
		framework.Compose(withTokenAuth, f, common.SyncV1Alpha1, useApiKeyAuth),

		// use jwt k8s provider
		framework.Compose(withJWTK8s, f, common.FindByName, useJWTK8sProvider),
		framework.Compose(withJWTK8s, f, common.FindByNameAndRewrite, useJWTK8sProvider),
		framework.Compose(withJWTK8s, f, common.FindByTag, useJWTK8sProvider),
		framework.Compose(withJWTK8s, f, common.SimpleDataSync, useJWTK8sProvider),
		framework.Compose(withJWTK8s, f, common.SyncWithoutTargetName, useJWTK8sProvider),
		framework.Compose(withJWTK8s, f, common.JSONDataFromSync, useJWTK8sProvider),
		framework.Compose(withJWTK8s, f, common.JSONDataFromRewrite, useJWTK8sProvider),

		// use jwt k8s hostid provider
		framework.Compose(withJWTK8sHostID, f, common.FindByName, useJWTK8sHostIDProvider),
		framework.Compose(withJWTK8sHostID, f, common.FindByNameAndRewrite, useJWTK8sHostIDProvider),
		framework.Compose(withJWTK8sHostID, f, common.FindByTag, useJWTK8sHostIDProvider),
		framework.Compose(withJWTK8sHostID, f, common.SimpleDataSync, useJWTK8sHostIDProvider),
		framework.Compose(withJWTK8sHostID, f, common.SyncWithoutTargetName, useJWTK8sHostIDProvider),
		framework.Compose(withJWTK8sHostID, f, common.JSONDataFromSync, useJWTK8sHostIDProvider),
		framework.Compose(withJWTK8sHostID, f, common.JSONDataFromRewrite, useJWTK8sHostIDProvider),
	)
})

func useApiKeyAuth(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = tc.Framework.Namespace.Name
}

func useJWTK8sProvider(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = jwtK8sProviderName
}

func useJWTK8sHostIDProvider(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = jwtK8sHostIDProviderName
}
