//Copyright External Secrets Inc. All Rights Reserved

package alibaba

import (
	"context"
	"os"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/kms"

	//nolint
	. "github.com/onsi/ginkgo/v2"

	//nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type alibabaProvider struct {
	accessKeyID     string
	accessKeySecret string
	regionID        string
	framework       *framework.Framework
}

const (
	secretName = "secretName"
)

func newAlibabaProvider(f *framework.Framework, accessKeyID, accessKeySecret, regionID string) *alibabaProvider {
	prov := &alibabaProvider{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		regionID:        regionID,
		framework:       f,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func newFromEnv(f *framework.Framework) *alibabaProvider {
	accessKeyID := os.Getenv("ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ACCESS_KEY_SECRET")
	regionID := os.Getenv("REGION_ID")
	return newAlibabaProvider(f, accessKeyID, accessKeySecret, regionID)
}

// CreateSecret creates a secret in both kv v1 and v2 provider.
func (s *alibabaProvider) CreateSecret(key string, val framework.SecretEntry) {
	client, err := kms.NewClientWithAccessKey(s.regionID, s.accessKeyID, s.accessKeySecret)
	Expect(err).ToNot(HaveOccurred())
	kmssecretrequest := kms.CreateCreateSecretRequest()
	kmssecretrequest.SecretName = secretName
	kmssecretrequest.SecretData = "value"
	_, err = client.CreateSecret(kmssecretrequest)
	Expect(err).ToNot(HaveOccurred())
}

func (s *alibabaProvider) DeleteSecret(key string) {
	client, err := kms.NewClientWithAccessKey(s.regionID, s.accessKeyID, s.accessKeySecret)
	Expect(err).ToNot(HaveOccurred())
	kmssecretrequest := kms.CreateDeleteSecretRequest()
	kmssecretrequest.SecretName = secretName
	_, err = client.DeleteSecret(kmssecretrequest)
	Expect(err).ToNot(HaveOccurred())
}

func (s *alibabaProvider) BeforeEach() {
	// Creating an Alibaba secret
	alibabaCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: s.framework.Namespace.Name,
		},
		StringData: map[string]string{
			secretName: "value",
		},
	}
	err := s.framework.CRClient.Create(context.Background(), alibabaCreds)
	Expect(err).ToNot(HaveOccurred())

	// Creating Alibaba secret store
	secretStore := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Alibaba: &esv1beta1.AlibabaProvider{
					Auth: &esv1beta1.AlibabaAuth{
						SecretRef: esv1beta1.AlibabaAuthSecretRef{
							AccessKeyID: esmeta.SecretKeySelector{
								Name: "kms-secret",
								Key:  "keyid",
							},
							AccessKeySecret: esmeta.SecretKeySelector{
								Name: "kms-secret",
								Key:  "accesskey",
							},
						},
					},
				},
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
