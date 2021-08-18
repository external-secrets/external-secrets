package alibaba

import (
	"context"

	//nolint
	. "github.com/onsi/ginkgo"

	//nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/pkg/provider/alibaba"
)

type alibabaProvider struct {
	url       string
	client    *alibaba.KeyManagementService
	framework *framework.Framework
}

const (
	certAuthProviderName    = "cert-auth-provider"
	appRoleAuthProviderName = "app-role-provider"
	kvv1ProviderName        = "kv-v1-provider"
	jwtProviderName         = "jwt-provider"
	kubernetesProviderName  = "kubernetes-provider"
)

func newAlibabaProvider(f *framework.Framework) *alibabaProvider {
	prov := &alibabaProvider{
		framework: f,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

// CreateSecret creates a secret in both kv v1 and v2 provider.
func (s *alibabaProvider) CreateSecret(key, val string) {
}

func (s *alibabaProvider) DeleteSecret(key string) {
	_, err := s.client.DeleteSecret("")
	Expect(err).ToNot(HaveOccurred())
}

func (s *alibabaProvider) BeforeEach() {
	//Creating an Alibaba secret
	alibabaCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider-secret",
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
								Name: "external-secret",
								Key:  "",
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
