package auth

import (
	"context"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestValidateAccessKeyStore(t *testing.T) {

	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Alibaba: &esv1beta1.AlibabaProvider{
					Service:  "ParameterStore",
					RegionID: "region-1",
					Auth: esv1beta1.AlibabaAuth{
						SecretRef: &esv1beta1.AlibabaAuthSecretRef{
							AccessKeyID: esmeta.SecretKeySelector{
								Name: "accessKeyID",
								Key:  "key-1",
							},
							AccessKeySecret: esmeta.SecretKeySelector{
								Name: "accessKeySecret",
								Key:  "key-1",
							},
						},
					},
				},
			},
		},
	}

	k8sClient := clientfake.NewClientBuilder().Build()
	_, err := NewAuth(context.Background(), k8sClient, store, "namespace")
	if err != nil && err.Error() != "failed to create Alibaba AccessKey credentials: could not fetch AccessKeyID secret: secrets \"accessKeyID\" not found" {
		t.Errorf(err.Error())
	}
	return
}
