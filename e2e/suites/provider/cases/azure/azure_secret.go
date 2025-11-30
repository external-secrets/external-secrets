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
package azure

import (

	// nolint
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	withStaticCredentials = "with static credentials"
	withReferentAuth      = "with referent auth"
	withNewSDK            = "with new SDK"
)

// keyvault type=secret should behave just like any other secret store.
var _ = Describe("[azure]", Label("azure", "keyvault", "secret"), func() {
	f := framework.New("eso-azure")
	prov := newFromEnv(f)

	DescribeTable("sync secrets", framework.TableFuncWithExternalSecret(f, prov),
		framework.Compose(withStaticCredentials, f, common.SimpleDataSync, useStaticCredentials),
		framework.Compose(withStaticCredentials, f, common.NestedJSONWithGJSON, useStaticCredentials),
		framework.Compose(withStaticCredentials, f, common.JSONDataFromSync, useStaticCredentials),
		framework.Compose(withStaticCredentials, f, common.JSONDataFromRewrite, useStaticCredentials),
		framework.Compose(withStaticCredentials, f, common.JSONDataWithProperty, useStaticCredentials),
		framework.Compose(withStaticCredentials, f, common.JSONDataWithTemplate, useStaticCredentials),
		framework.Compose(withStaticCredentials, f, common.DockerJSONConfig, useStaticCredentials),
		framework.Compose(withStaticCredentials, f, common.DataPropertyDockerconfigJSON, useStaticCredentials),
		framework.Compose(withStaticCredentials, f, common.SSHKeySync, useStaticCredentials),
		framework.Compose(withStaticCredentials, f, common.SSHKeySyncDataProperty, useStaticCredentials),
		framework.Compose(withStaticCredentials, f, common.SyncWithoutTargetName, useStaticCredentials),
		framework.Compose(withStaticCredentials, f, common.JSONDataWithoutTargetName, useStaticCredentials),

		framework.Compose(withStaticCredentials, f, common.SimpleDataSync, useReferentAuth),

		// New SDK tests
		framework.Compose(withNewSDK, f, common.SimpleDataSync, useNewSDK),
		framework.Compose(withNewSDK, f, common.NestedJSONWithGJSON, useNewSDK),
		framework.Compose(withNewSDK, f, common.JSONDataFromSync, useNewSDK),
		framework.Compose(withNewSDK, f, common.JSONDataFromRewrite, useNewSDK),
		framework.Compose(withNewSDK, f, common.JSONDataWithProperty, useNewSDK),
		framework.Compose(withNewSDK, f, common.JSONDataWithTemplate, useNewSDK),
		framework.Compose(withNewSDK, f, common.DockerJSONConfig, useNewSDK),
		framework.Compose(withNewSDK, f, common.DataPropertyDockerconfigJSON, useNewSDK),
		framework.Compose(withNewSDK, f, common.SSHKeySync, useNewSDK),
		framework.Compose(withNewSDK, f, common.SSHKeySyncDataProperty, useNewSDK),
		framework.Compose(withNewSDK, f, common.SyncWithoutTargetName, useNewSDK),
		framework.Compose(withNewSDK, f, common.JSONDataWithoutTargetName, useNewSDK),

		framework.Compose(withNewSDK, f, common.SimpleDataSync, useReferentAuthNewSDK),
	)
})

func useStaticCredentials(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = tc.Framework.Namespace.Name
}

func useReferentAuth(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = referentAuthName(tc.Framework)
	tc.ExternalSecret.Spec.SecretStoreRef.Kind = esapi.ClusterSecretStoreKind
}

func useNewSDK(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = tc.Framework.Namespace.Name + "-new-sdk"
}

func useReferentAuthNewSDK(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = referentAuthName(tc.Framework) + "-new-sdk"
	tc.ExternalSecret.Spec.SecretStoreRef.Kind = esapi.ClusterSecretStoreKind
}
