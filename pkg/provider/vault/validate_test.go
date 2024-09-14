// Copyright External Secrets Inc. All Rights Reserved
package vault

import (
	"testing"

	pointer "k8s.io/utils/ptr"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const fakeValidationValue = "fake-value"

func TestValidateStore(t *testing.T) {
	type args struct {
		auth      esv1beta1.VaultAuth
		clientTLS esv1beta1.VaultClientTLS
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "empty auth",
			args: args{},
		},

		{
			name: "invalid approle with namespace",
			args: args{
				auth: esv1beta1.VaultAuth{
					AppRole: &esv1beta1.VaultAppRole{
						SecretRef: esmeta.SecretKeySelector{
							Namespace: pointer.To("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid approle with roleId and no roleRef",
			args: args{
				auth: esv1beta1.VaultAuth{
					AppRole: &esv1beta1.VaultAppRole{
						RoleID:  "",
						RoleRef: nil,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid approle with roleId and no roleRef",
			args: args{
				auth: esv1beta1.VaultAuth{
					AppRole: &esv1beta1.VaultAppRole{
						RoleID: fakeValidationValue,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid approle with roleId and no roleRef",
			args: args{
				auth: esv1beta1.VaultAuth{
					AppRole: &esv1beta1.VaultAppRole{
						RoleRef: &esmeta.SecretKeySelector{
							Name: fakeValidationValue,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid clientcert",
			args: args{
				auth: esv1beta1.VaultAuth{
					Cert: &esv1beta1.VaultCertAuth{
						ClientCert: esmeta.SecretKeySelector{
							Namespace: pointer.To("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid cert secret",
			args: args{
				auth: esv1beta1.VaultAuth{
					Cert: &esv1beta1.VaultCertAuth{
						SecretRef: esmeta.SecretKeySelector{
							Namespace: pointer.To("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid jwt secret",
			args: args{
				auth: esv1beta1.VaultAuth{
					Jwt: &esv1beta1.VaultJwtAuth{
						SecretRef: &esmeta.SecretKeySelector{
							Namespace: pointer.To("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid kubernetes sa",
			args: args{
				auth: esv1beta1.VaultAuth{
					Kubernetes: &esv1beta1.VaultKubernetesAuth{
						ServiceAccountRef: &esmeta.ServiceAccountSelector{
							Namespace: pointer.To("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid kubernetes secret",
			args: args{
				auth: esv1beta1.VaultAuth{
					Kubernetes: &esv1beta1.VaultKubernetesAuth{
						SecretRef: &esmeta.SecretKeySelector{
							Namespace: pointer.To("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid ldap secret",
			args: args{
				auth: esv1beta1.VaultAuth{
					Ldap: &esv1beta1.VaultLdapAuth{
						SecretRef: esmeta.SecretKeySelector{
							Namespace: pointer.To("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid userpass secret",
			args: args{
				auth: esv1beta1.VaultAuth{
					UserPass: &esv1beta1.VaultUserPassAuth{
						SecretRef: esmeta.SecretKeySelector{
							Namespace: pointer.To("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid token secret",
			args: args{
				auth: esv1beta1.VaultAuth{
					TokenSecretRef: &esmeta.SecretKeySelector{
						Namespace: pointer.To("invalid"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid clientTls config",
			args: args{
				auth: esv1beta1.VaultAuth{
					AppRole: &esv1beta1.VaultAppRole{
						RoleRef: &esmeta.SecretKeySelector{
							Name: fakeValidationValue,
						},
					},
				},
				clientTLS: esv1beta1.VaultClientTLS{
					CertSecretRef: &esmeta.SecretKeySelector{
						Name: "tls-auth-certs",
					},
					KeySecretRef: &esmeta.SecretKeySelector{
						Name: "tls-auth-certs",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid clientTls config, missing SecretRef",
			args: args{
				auth: esv1beta1.VaultAuth{
					AppRole: &esv1beta1.VaultAppRole{
						RoleRef: &esmeta.SecretKeySelector{
							Name: fakeValidationValue,
						},
					},
				},
				clientTLS: esv1beta1.VaultClientTLS{
					CertSecretRef: &esmeta.SecretKeySelector{
						Name: "tls-auth-certs",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid clientTls config, missing ClientCert",
			args: args{
				auth: esv1beta1.VaultAuth{
					AppRole: &esv1beta1.VaultAppRole{
						RoleRef: &esmeta.SecretKeySelector{
							Name: fakeValidationValue,
						},
					},
				},
				clientTLS: esv1beta1.VaultClientTLS{
					KeySecretRef: &esmeta.SecretKeySelector{
						Name: "tls-auth-certs",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Provider{
				NewVaultClient: nil,
			}
			store := &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Vault: &esv1beta1.VaultProvider{
							Auth:      tt.args.auth,
							ClientTLS: tt.args.clientTLS,
						},
					},
				},
			}
			if _, err := c.ValidateStore(store); (err != nil) != tt.wantErr {
				t.Errorf("connector.ValidateStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
