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
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

// azure keyvault type=key should retrieve a jwk from the api.
var _ = Describe("[azure]", Label("azure", "keyvault", "key"), func() {
	f := framework.New("eso-azure-keytype")
	prov := newFromEnv(f)
	var jwk *keyvault.JSONWebKey
	var keyName string

	BeforeEach(func() {
		keyName = fmt.Sprintf("%s-%s", f.Namespace.Name, "keytest")
		jwk = prov.CreateKey(keyName)
	})

	AfterEach(func() {
		prov.DeleteKey(keyName)
	})

	ff := framework.TableFunc(f, prov)

	It("should sync keyvault objects with type=key", func() {
		ff(func(tc *framework.TestCase) {
			secretKey := "azkv-key"
			keyBytes, _ := json.Marshal(jwk)

			tc.ExpectedSecret = &v1.Secret{
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{
					secretKey: keyBytes,
				},
			}
			tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
				{
					SecretKey: secretKey,
					RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
						Key: "key/" + keyName,
					},
				},
			}
		})
	})

})
