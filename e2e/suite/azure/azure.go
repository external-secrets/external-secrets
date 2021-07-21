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
package azure

import (
	"os"

	// nolint
	. "github.com/onsi/ginkgo"
	// nolint
	. "github.com/onsi/ginkgo/extensions/table"

	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/suite/common"
)

var _ = Describe("[azure] ", func() {
	f := framework.New("eso-azure")
	vaultURL := os.Getenv("VAULT_URL")
	tenantID := os.Getenv("TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	prov := newazureProvider(f, clientID, clientSecret, tenantID, vaultURL)

	DescribeTable("sync secrets", framework.TableFunc(f, prov),
		Entry(common.SimpleDataSync(f)),
		Entry(common.NestedJSONWithGJSON(f)),
		// TODO: dataFrom is not working as expected RN
		// see: https://github.com/external-secrets/external-secrets/issues/263
		// Entry(common.JSONDataFromSync(f)),
		Entry(common.JSONDataWithProperty(f)),
		Entry(common.JSONDataWithTemplate(f)),
		Entry(common.DockerJSONConfig(f)),
	)
})
