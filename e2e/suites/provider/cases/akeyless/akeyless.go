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

package akeyless

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
)

var _ = Describe("[akeyless]", Label("akeyless"), func() {
	f := framework.New("eso-akeyless")
	prov := newFromEnv(f)

	DescribeTable("sync secrets", framework.TableFuncWithExternalSecret(f, prov),
		Entry(common.SimpleDataSync(f)),
		Entry(common.NestedJSONWithGJSON(f)),
		Entry(common.JSONDataFromSync(f)),
		Entry(common.JSONDataFromRewrite(f)),
		Entry(common.JSONDataWithProperty(f)),
		Entry(common.JSONDataWithTemplate(f)),
		Entry(common.DockerJSONConfig(f)),
		Entry(common.DataPropertyDockerconfigJSON(f)),
		Entry(common.SSHKeySync(f)),
		Entry(common.SSHKeySyncDataProperty(f)),
		Entry(common.SyncWithoutTargetName(f)),
		Entry(common.JSONDataWithoutTargetName(f)),
		Entry("dataFrom with property should extract nested json keys", testDataFromJSONWithProperty(f)),
	)
})

func testDataFromJSONWithProperty(f *framework.Framework) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		secretKey := fmt.Sprintf("%s-%s", f.Namespace.Name, "json-property")
		remoteRefKey := f.MakeRemoteRefKey(secretKey)
		secretValue := `{"db":{"username":"my_user","password":"my_pass","port":5432},"apiKey":"myApiKey"}`
		tc.Secrets = map[string]framework.SecretEntry{
			remoteRefKey: {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				"username": []byte("my_user"),
				"password": []byte("my_pass"),
				"port":     []byte("5432"),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esapi.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esapi.ExternalSecretDataRemoteRef{
					Key:      remoteRefKey,
					Property: "db",
				},
			},
		}
	}
}
