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
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"

	//nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/log"
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmetav1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type Provider struct {
	ServiceAccountName      string
	ServiceAccountNamespace string

	region    string
	client    *acm.Client
	framework *framework.Framework
}

func NewProvider(f *framework.Framework, kid, sak, st, region, saName, saNamespace string) *Provider {
	prov := &Provider{
		ServiceAccountName:      saName,
		ServiceAccountNamespace: saNamespace,
		region:                  region,
		framework:               f,
	}

	BeforeAll(func() {
		cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(kid, sak, st)))
		Expect(err).ToNot(HaveOccurred())
		prov.client = acm.NewFromConfig(cfg)
	})

	BeforeEach(func() {
		awscommon.SetupStaticStore(f, awscommon.AccessOpts{KID: kid, SAK: sak, ST: st, Region: region}, esv1.AWSServiceCertificateManager)
		awscommon.CreateReferentStaticStore(f, awscommon.AccessOpts{KID: kid, SAK: sak, ST: st, Region: region}, esv1.AWSServiceCertificateManager)
		prov.SetupReferencedIRSAStore()
		prov.SetupMountedIRSAStore()
	})

	AfterEach(func() {
		prov.TeardownReferencedIRSAStore()
		prov.TeardownMountedIRSAStore()
	})

	return prov
}

func NewFromEnv(f *framework.Framework) *Provider {
	kid := os.Getenv("AWS_ACCESS_KEY_ID")
	sak := os.Getenv("AWS_SECRET_ACCESS_KEY")
	st := os.Getenv("AWS_SESSION_TOKEN")
	region := os.Getenv("AWS_REGION")
	saName := os.Getenv("AWS_SA_NAME")
	saNamespace := os.Getenv("AWS_SA_NAMESPACE")
	prov := NewProvider(f, kid, sak, st, region, saName, saNamespace)
	return prov
}

// CreateSecret imports a self-signed certificate into ACM tagged with the given key.
// The val.Value field is ignored because ACM operates on certificates, not arbitrary strings.
func (s *Provider) CreateSecret(key string, _ framework.SecretEntry) {
	certPEM, keyPEM := generateSelfSignedCert(types.KeyAlgorithmEcPrime256v1)
	_, err := s.client.ImportCertificate(GinkgoT().Context(), &acm.ImportCertificateInput{
		Certificate: certPEM,
		PrivateKey:  keyPEM,
		Tags: []types.Tag{
			{Key: aws.String("managed-by"), Value: aws.String("external-secrets")},
			{Key: aws.String("external-secrets-remote-key"), Value: aws.String(key)},
		},
	})
	Expect(err).ToNot(HaveOccurred())
}

// DeleteSecret removes a certificate from ACM by looking up the remote-key tag.
func (s *Provider) DeleteSecret(key string) {
	arn, err := s.FindCertificateByRemoteKey(key)
	Expect(err).ToNot(HaveOccurred())
	if arn == nil {
		return
	}
	_, err = s.client.DeleteCertificate(GinkgoT().Context(), &acm.DeleteCertificateInput{
		CertificateArn: arn,
	})
	Expect(err).ToNot(HaveOccurred(), "failed to delete certificate %s", aws.ToString(arn))
}

// FindCertificateByRemoteKey searches for a certificate in ACM with the matching remote-key tag.
func (s *Provider) FindCertificateByRemoteKey(remoteKey string) (*string, error) {
	paginator := acm.NewListCertificatesPaginator(s.client, &acm.ListCertificatesInput{
		Includes: &types.Filters{
			KeyTypes: []types.KeyAlgorithm{
				types.KeyAlgorithmRsa1024,
				types.KeyAlgorithmRsa2048,
				types.KeyAlgorithmRsa3072,
				types.KeyAlgorithmRsa4096,
				types.KeyAlgorithmEcPrime256v1,
				types.KeyAlgorithmEcSecp384r1,
				types.KeyAlgorithmEcSecp521r1,
			},
		},
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(GinkgoT().Context())
		if err != nil {
			return nil, err
		}
		for _, cert := range page.CertificateSummaryList {
			if cert.CertificateArn == nil {
				continue
			}
			tags, err := s.client.ListTagsForCertificate(GinkgoT().Context(), &acm.ListTagsForCertificateInput{
				CertificateArn: cert.CertificateArn,
			})
			if err != nil {
				return nil, err
			}
			if hasTagValue(tags.Tags, "external-secrets-remote-key", remoteKey) {
				return cert.CertificateArn, nil
			}
		}
	}
	return nil, nil
}

// GetCertificateTags returns the tags for a certificate given its ARN.
func (s *Provider) GetCertificateTags(arn string) []types.Tag {
	out, err := s.client.ListTagsForCertificate(GinkgoT().Context(), &acm.ListTagsForCertificateInput{
		CertificateArn: aws.String(arn),
	})
	Expect(err).ToNot(HaveOccurred())
	return out.Tags
}

func (s *Provider) SetupMountedIRSAStore() {
	secretStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      awscommon.MountedIRSAStoreName(s.framework),
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AWS: &esv1.AWSProvider{
					Service: esv1.AWSServiceCertificateManager,
					Region:  s.region,
					Auth:    esv1.AWSAuth{},
				},
			},
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s *Provider) TeardownMountedIRSAStore() {
	err := s.framework.CRClient.Delete(GinkgoT().Context(), &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      awscommon.MountedIRSAStoreName(s.framework),
			Namespace: s.framework.Namespace.Name,
		},
	})
	Expect(err).ToNot(HaveOccurred())
}

func (s *Provider) SetupReferencedIRSAStore() {
	log.Logf("creating IRSA ClusterSecretStore %s", s.framework.Namespace.Name)
	secretStore := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: awscommon.ReferencedIRSAStoreName(s.framework),
		},
	}
	_, err := controllerutil.CreateOrUpdate(GinkgoT().Context(), s.framework.CRClient, secretStore, func() error {
		secretStore.Spec.Provider = &esv1.SecretStoreProvider{
			AWS: &esv1.AWSProvider{
				Service: esv1.AWSServiceCertificateManager,
				Region:  s.region,
				Auth: esv1.AWSAuth{
					JWTAuth: &esv1.AWSJWTAuth{
						ServiceAccountRef: &esmetav1.ServiceAccountSelector{
							Name:      s.ServiceAccountName,
							Namespace: &s.ServiceAccountNamespace,
						},
					},
				},
			},
		}
		return nil
	})
	Expect(err).ToNot(HaveOccurred())
}

func (s *Provider) TeardownReferencedIRSAStore() {
	err := s.framework.CRClient.Delete(GinkgoT().Context(), &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: awscommon.ReferencedIRSAStoreName(s.framework),
		},
	})
	Expect(err).ToNot(HaveOccurred())
}

func hasTagValue(tags []types.Tag, key, value string) bool {
	for _, t := range tags {
		if aws.ToString(t.Key) == key && aws.ToString(t.Value) == value {
			return true
		}
	}
	return false
}

func generateSelfSignedCert(keyAlgorithm types.KeyAlgorithm) (certPEM, keyPEM []byte) {
	var rsaKey *rsa.PrivateKey
	var ecdsaKey *ecdsa.PrivateKey
	var certDER []byte
	var keyDER []byte
	var err error

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "e2e.test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	switch keyAlgorithm {
	case types.KeyAlgorithmRsa2048:
		rsaKey, err = rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).ToNot(HaveOccurred())

		certDER, err = x509.CreateCertificate(rand.Reader, tmpl, tmpl, &rsaKey.PublicKey, rsaKey)
		Expect(err).ToNot(HaveOccurred())

		keyDER, err = x509.MarshalPKCS8PrivateKey(rsaKey)
		Expect(err).ToNot(HaveOccurred())
	case types.KeyAlgorithmEcPrime256v1:
		ecdsaKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		Expect(err).ToNot(HaveOccurred())

		certDER, err = x509.CreateCertificate(rand.Reader, tmpl, tmpl, &ecdsaKey.PublicKey, ecdsaKey)
		Expect(err).ToNot(HaveOccurred())

		keyDER, err = x509.MarshalPKCS8PrivateKey(ecdsaKey)
		Expect(err).ToNot(HaveOccurred())
	default:
		err = fmt.Errorf("unsupported key algorithm: %s", keyAlgorithm)
		Expect(err).ToNot(HaveOccurred())
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM
}
