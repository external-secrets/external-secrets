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
package vault

import (
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	// nolint
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("[vault] with mTLS", Label("vault-mtls"), func() {
	f := framework.New("eso-vault-mtls")
	prov := newVaultProvider(f, true)

	DescribeTable("sync secrets",
		framework.TableFunc(f, prov),
		// uses token auth
		framework.Compose(withTokenAuth, f, common.FindByName, useTokenAuth),
		// use referent auth
		framework.Compose(withReferentAuth, f, common.JSONDataFromSync, useReferentAuth),
	)
})
