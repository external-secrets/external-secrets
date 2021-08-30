package alibaba

import (
	"context"

	//nolint
	. "github.com/onsi/ginkgo"

	//nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/kms"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

type alibabaProvider struct {
	accessKeyID     string
	accessKeySecret string
	regionID        string
	client          *kms.Client
	framework       *framework.Framework
}

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

// CreateSecret creates a secret in both kv v1 and v2 provider.
func (s *alibabaProvider) CreateSecret(key, val string) {
	client, err := kms.NewClient()
	Expect(err).ToNot(HaveOccurred())
	kmssecretrequest := kms.CreateSecretRequest{
		SecretName: "test-example",
		SecretData: "value",
	}
	client.CreateSecret(&kmssecretrequest)
}

func (s *alibabaProvider) DeleteSecret(key string) {
	client, err := kms.NewClient()
	Expect(err).ToNot(HaveOccurred())
	kmssecretrequest := kms.DeleteSecretRequest{
		SecretName: "test-example",
	}
	client.DeleteSecret(&kmssecretrequest)
}

func (s *alibabaProvider) BeforeEach() {
	//Creating an Alibaba secret
	alibabaCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-example",
			Namespace: s.framework.Namespace.Name,
		},
		StringData: map[string]string{
			//secret
		},
	}
	err := s.framework.CRClient.Create(context.Background(), alibabaCreds)
	Expect(err).ToNot(HaveOccurred())

	//Creating Alibaba secret store
	secretStore := &esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				Alibaba: &esv1alpha1.AlibabaProvider{
					Auth: &esv1alpha1.AlibabaAuth{
						SecretRef: esv1alpha1.AlibabaAuthSecretRef{
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
