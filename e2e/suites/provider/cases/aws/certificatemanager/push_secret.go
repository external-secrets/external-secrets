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

package aws

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	acmtypes "github.com/aws/aws-sdk-go-v2/service/acm/types"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

// PushSecretImport tests importing a kubernetes.io/tls secret into ACM via PushSecret.
func PushSecretImport(prov *Provider, keyAlgorithm acmtypes.KeyAlgorithm) func(f *framework.Framework) (string, func(*framework.TestCase)) {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return fmt.Sprintf("[acm] should import a TLS certificate (%s) into ACM via PushSecret", keyAlgorithm), func(tc *framework.TestCase) {
			sanitizedKeyAlgorithm := strings.ToLower(strings.ReplaceAll(string(keyAlgorithm), "_", "-"))
			remoteKey := fmt.Sprintf("e2e-acm-push-%s-%s", f.Namespace.Name, sanitizedKeyAlgorithm)
			secretName := fmt.Sprintf("tls-source-%s", sanitizedKeyAlgorithm)
			certPEM, keyPEM := generateSelfSignedCert(keyAlgorithm)

			tc.ExternalSecret = nil
			tc.ExpectedSecret = nil
			tc.PushSecretSource = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: f.Namespace.Name,
				},
				Type: v1.SecretTypeTLS,
				Data: map[string][]byte{
					"tls.crt": certPEM,
					"tls.key": keyPEM,
				},
			}
			tc.PushSecret = &esv1alpha1.PushSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("e2e-ps-acm-%s", sanitizedKeyAlgorithm),
					Namespace: f.Namespace.Name,
				},
				Spec: esv1alpha1.PushSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Minute * 1},
					SecretStoreRefs: []esv1alpha1.PushSecretStoreRef{
						{Name: f.Namespace.Name},
					},
					Selector: esv1alpha1.PushSecretSelector{
						Secret: &esv1alpha1.PushSecretSecret{
							Name: secretName,
						},
					},
					Data: []esv1alpha1.PushSecretData{
						{
							Match: esv1alpha1.PushSecretMatch{
								RemoteRef: esv1alpha1.PushSecretRemoteRef{
									RemoteKey: remoteKey,
								},
							},
						},
					},
				},
			}
			tc.VerifyPushSecretOutcome = func(_ *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				waitForPushSecretReady(tc)

				Eventually(func() bool {
					arn, err := prov.FindCertificateByRemoteKey(remoteKey)
					return err == nil && arn != nil
				}, time.Minute*3, time.Second*10).Should(BeTrue(), "certificate should exist in ACM with remote key %s", remoteKey)

				err := tc.Framework.CRClient.Delete(GinkgoT().Context(), tc.PushSecret)
				Expect(err).ToNot(HaveOccurred())
				prov.DeleteSecret(remoteKey)
			}
		}
	}
}

// PushSecretWithTags tests importing a certificate with custom metadata tags.
func PushSecretWithTags(prov *Provider) func(f *framework.Framework) (string, func(*framework.TestCase)) {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "[acm] should import a TLS certificate with custom tags via PushSecret", func(tc *framework.TestCase) {
			remoteKey := fmt.Sprintf("e2e-acm-tags-%s", f.Namespace.Name)
			certPEM, keyPEM := generateSelfSignedCert(acmtypes.KeyAlgorithmEcPrime256v1)

			metadataSpec := map[string]interface{}{
				"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
				"kind":       "PushSecretMetadata",
				"spec": map[string]interface{}{
					"tags": map[string]string{
						"environment": "e2e-test",
						"team":        "platform",
					},
				},
			}
			metadataBytes, err := json.Marshal(metadataSpec)
			Expect(err).ToNot(HaveOccurred())

			tc.ExternalSecret = nil
			tc.ExpectedSecret = nil
			tc.PushSecretSource = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-source-tags",
					Namespace: f.Namespace.Name,
				},
				Type: v1.SecretTypeTLS,
				Data: map[string][]byte{
					"tls.crt": certPEM,
					"tls.key": keyPEM,
				},
			}
			tc.PushSecret = &esv1alpha1.PushSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2e-ps-acm-tags",
					Namespace: f.Namespace.Name,
				},
				Spec: esv1alpha1.PushSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Minute * 1},
					SecretStoreRefs: []esv1alpha1.PushSecretStoreRef{
						{Name: f.Namespace.Name},
					},
					Selector: esv1alpha1.PushSecretSelector{
						Secret: &esv1alpha1.PushSecretSecret{
							Name: "tls-source-tags",
						},
					},
					Data: []esv1alpha1.PushSecretData{
						{
							Match: esv1alpha1.PushSecretMatch{
								RemoteRef: esv1alpha1.PushSecretRemoteRef{
									RemoteKey: remoteKey,
								},
							},
							Metadata: &apiextensionsv1.JSON{Raw: metadataBytes},
						},
					},
				},
			}
			tc.VerifyPushSecretOutcome = func(_ *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				waitForPushSecretReady(tc)

				var arn *string
				var err error
				Eventually(func() bool {
					arn, err = prov.FindCertificateByRemoteKey(remoteKey)
					return err == nil && arn != nil
				}, time.Minute*3, time.Second*10).Should(BeTrue(), "certificate should exist in ACM with remote key %s", remoteKey)

				tags := prov.GetCertificateTags(aws.ToString(arn))
				Expect(hasTagValue(tags, "environment", "e2e-test")).To(BeTrue(), "should have environment tag")
				Expect(hasTagValue(tags, "team", "platform")).To(BeTrue(), "should have team tag")

				err = tc.Framework.CRClient.Delete(GinkgoT().Context(), tc.PushSecret)
				Expect(err).ToNot(HaveOccurred())
				prov.DeleteSecret(remoteKey)
			}
		}
	}
}

// PushSecretDelete tests that deleting a PushSecret with deletionPolicy=Delete removes the cert from ACM.
func PushSecretDelete(prov *Provider) func(f *framework.Framework) (string, func(*framework.TestCase)) {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "[acm] should delete certificate from ACM when PushSecret is deleted", func(tc *framework.TestCase) {
			remoteKey := fmt.Sprintf("e2e-acm-del-%s", f.Namespace.Name)
			certPEM, keyPEM := generateSelfSignedCert(acmtypes.KeyAlgorithmEcPrime256v1)

			tc.ExternalSecret = nil
			tc.ExpectedSecret = nil
			tc.PushSecretSource = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-source-del",
					Namespace: f.Namespace.Name,
				},
				Type: v1.SecretTypeTLS,
				Data: map[string][]byte{
					"tls.crt": certPEM,
					"tls.key": keyPEM,
				},
			}
			tc.PushSecret = &esv1alpha1.PushSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2e-ps-acm-del",
					Namespace: f.Namespace.Name,
				},
				Spec: esv1alpha1.PushSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Minute * 1},
					DeletionPolicy:  esv1alpha1.PushSecretDeletionPolicyDelete,
					SecretStoreRefs: []esv1alpha1.PushSecretStoreRef{
						{Name: f.Namespace.Name},
					},
					Selector: esv1alpha1.PushSecretSelector{
						Secret: &esv1alpha1.PushSecretSecret{
							Name: "tls-source-del",
						},
					},
					Data: []esv1alpha1.PushSecretData{
						{
							Match: esv1alpha1.PushSecretMatch{
								RemoteRef: esv1alpha1.PushSecretRemoteRef{
									RemoteKey: remoteKey,
								},
							},
						},
					},
				},
			}
			tc.VerifyPushSecretOutcome = func(_ *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				waitForPushSecretReady(tc)

				Eventually(func() bool {
					arn, err := prov.FindCertificateByRemoteKey(remoteKey)
					return err == nil && arn != nil
				}, time.Minute*3, time.Second*10).Should(BeTrue(), "certificate should exist before deletion")

				err := tc.Framework.CRClient.Delete(GinkgoT().Context(), tc.PushSecret)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					arn, err := prov.FindCertificateByRemoteKey(remoteKey)
					return err == nil && arn == nil
				}, time.Minute*3, time.Second*10).Should(BeTrue(), "certificate should be deleted from ACM")
			}
		}
	}
}

func waitForPushSecretReady(tc *framework.TestCase) {
	Eventually(func() bool {
		s := &esv1alpha1.PushSecret{}
		err := tc.Framework.CRClient.Get(
			GinkgoT().Context(),
			types.NamespacedName{Name: tc.PushSecret.Name, Namespace: tc.PushSecret.Namespace},
			s,
		)
		if err != nil {
			return false
		}
		for _, c := range s.Status.Conditions {
			if c.Type == esv1alpha1.PushSecretReady && c.Status == v1.ConditionTrue {
				return true
			}
		}
		return false
	}, time.Minute*3, time.Second*5).Should(BeTrue(), "PushSecret should become Ready")
}
