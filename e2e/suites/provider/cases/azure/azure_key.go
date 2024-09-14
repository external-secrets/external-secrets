// Copyright External Secrets Inc. All Rights Reserved
package azure

import (
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
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
