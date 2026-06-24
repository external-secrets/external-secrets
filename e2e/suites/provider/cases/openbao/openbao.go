/*
Copyright © The ESO Authors

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

package openbao

import (
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	. "github.com/onsi/ginkgo/v2"
)

const (
	withTokenAuth = "with token auth"
)

var _ = Describe("[OpenBao]", Label("openbao"), Ordered, func() {
	f := framework.New("openbao")
	openbao := addon.NewOpenBao()
	prov := newOpenBaoProvider(f, openbao)

	BeforeAll(func() {
		addon.InstallGlobalAddon(openbao)
	})

	DescribeTable("sync secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		// uses token auth
		framework.Compose(withTokenAuth, f, common.FindByName, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.FindByNameAndRewrite, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataFromSync, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataFromRewrite, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataWithProperty, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataWithTemplate, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.DataPropertyDockerconfigJSON, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataWithoutTargetName, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.DecodingPolicySync, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataWithTemplateFromLiteral, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.TemplateFromConfigmaps, useTokenAuth(prov)),
	)
})

func useTokenAuth(prov *openBaoProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateTokenStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = tc.Framework.Namespace.Name
	}
}
