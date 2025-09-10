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
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
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

	ff := framework.TableFuncWithExternalSecret(f, prov)

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
			tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
				{
					SecretKey: secretKey,
					RemoteRef: esv1.ExternalSecretDataRemoteRef{
						Key: "key/" + keyName,
					},
				},
			}
		})
	})

	It("should sync keyvault objects with type=key using new SDK", func() {
		ff(func(tc *framework.TestCase) {
			secretKey := "azkv-key-new-sdk"

			// Convert old SDK key to new SDK key format
			// First marshal the old SDK key
			oldKeyBytes, _ := json.Marshal(jwk)

			// Unmarshal into the new SDK type
			var newSDKKey azkeys.JSONWebKey
			json.Unmarshal(oldKeyBytes, &newSDKKey)

			// Marshal the new SDK key - this will have the new SDK's field ordering
			keyBytes, _ := json.Marshal(newSDKKey)

			tc.ExpectedSecret = &v1.Secret{
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{
					secretKey: keyBytes,
				},
			}
			tc.ExternalSecret.Spec.SecretStoreRef.Name = tc.Framework.Namespace.Name + "-new-sdk"
			tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
				{
					SecretKey: secretKey,
					RemoteRef: esv1.ExternalSecretDataRemoteRef{
						Key: "key/" + keyName,
					},
				},
			}
		})
	})

})
