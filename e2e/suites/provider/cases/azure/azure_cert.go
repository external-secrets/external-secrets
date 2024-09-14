// Copyright External Secrets Inc. All Rights Reserved
package azure

import (
	"fmt"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"

	// nolint
	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// azure keyvault type=cert should get a certificate from the api.
var _ = Describe("[azure]", Label("azure", "keyvault", "cert"), func() {
	f := framework.New("eso-azure-certtype")
	prov := newFromEnv(f)
	var certBytes []byte
	var certName string

	BeforeEach(func() {
		certName = fmt.Sprintf("%s-%s", f.Namespace.Name, "certtest")
		prov.CreateCertificate(certName)
		certBytes = prov.GetCertificate(certName)
	})

	AfterEach(func() {
		prov.DeleteCertificate(certName)
	})

	ff := framework.TableFuncWithExternalSecret(f, prov)
	It("should sync keyvault objects with type=cert", func() {
		ff(func(tc *framework.TestCase) {
			secretKey := "azkv-cert"

			tc.ExpectedSecret = &v1.Secret{
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{
					secretKey: certBytes,
				},
			}
			tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
				{
					SecretKey: secretKey,
					RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
						Key: "cert/" + certName,
					},
				},
			}
		})
	})

})
