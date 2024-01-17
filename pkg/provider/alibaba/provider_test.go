package alibaba

import (
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"testing"
)

func TestValidateAccessKeyStore(t *testing.T) {
	p := Provider{}

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

	err := p.ValidateStore(store)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestValidateRRSAStore(t *testing.T) {
	ps := Provider{}

	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Alibaba: &esv1beta1.AlibabaProvider{
					RegionID: "region-1",
					Auth: esv1beta1.AlibabaAuth{
						RRSAAuth: &esv1beta1.AlibabaRRSAAuth{
							OIDCProviderARN:   "acs:ram::1234:oidc-provider/ack-rrsa-ce123456",
							OIDCTokenFilePath: "/var/run/secrets/tokens/oidc-token",
							RoleARN:           "acs:ram::1234:role/test-role",
							SessionName:       "secrets",
						},
					},
				},
			},
		},
	}

	err := ps.ValidateStore(store)
	if err != nil {
		t.Errorf(err.Error())
	}
}
