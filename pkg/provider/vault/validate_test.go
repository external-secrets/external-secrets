/*
Copyright © 2025 ESO Maintainer Team

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
package vault

import (
	"testing"

	pointer "k8s.io/utils/ptr"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const fakeValidationValue = "fake-value"

func TestValidateStore(t *testing.T) {
	type args struct {
		auth        esv1.VaultAuth
		clientTLS   esv1.VaultClientTLS
		version     esv1.VaultKVStoreVersion
		checkAndSet *esv1.VaultCheckAndSet
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
				auth: esv1.VaultAuth{
					AppRole: &esv1.VaultAppRole{
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
				auth: esv1.VaultAuth{
					AppRole: &esv1.VaultAppRole{
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
				auth: esv1.VaultAuth{
					AppRole: &esv1.VaultAppRole{
						RoleID: fakeValidationValue,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid approle with roleId and no roleRef",
			args: args{
				auth: esv1.VaultAuth{
					AppRole: &esv1.VaultAppRole{
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
				auth: esv1.VaultAuth{
					Cert: &esv1.VaultCertAuth{
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
				auth: esv1.VaultAuth{
					Cert: &esv1.VaultCertAuth{
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
				auth: esv1.VaultAuth{
					Jwt: &esv1.VaultJwtAuth{
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
				auth: esv1.VaultAuth{
					Kubernetes: &esv1.VaultKubernetesAuth{
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
				auth: esv1.VaultAuth{
					Kubernetes: &esv1.VaultKubernetesAuth{
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
				auth: esv1.VaultAuth{
					Ldap: &esv1.VaultLdapAuth{
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
				auth: esv1.VaultAuth{
					UserPass: &esv1.VaultUserPassAuth{
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
				auth: esv1.VaultAuth{
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
				auth: esv1.VaultAuth{
					AppRole: &esv1.VaultAppRole{
						RoleRef: &esmeta.SecretKeySelector{
							Name: fakeValidationValue,
						},
					},
				},
				clientTLS: esv1.VaultClientTLS{
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
				auth: esv1.VaultAuth{
					AppRole: &esv1.VaultAppRole{
						RoleRef: &esmeta.SecretKeySelector{
							Name: fakeValidationValue,
						},
					},
				},
				clientTLS: esv1.VaultClientTLS{
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
				auth: esv1.VaultAuth{
					AppRole: &esv1.VaultAppRole{
						RoleRef: &esmeta.SecretKeySelector{
							Name: fakeValidationValue,
						},
					},
				},
				clientTLS: esv1.VaultClientTLS{
					KeySecretRef: &esmeta.SecretKeySelector{
						Name: "tls-auth-certs",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid CAS config with KV v2",
			args: args{
				auth: esv1.VaultAuth{
					AppRole: &esv1.VaultAppRole{
						RoleRef: &esmeta.SecretKeySelector{
							Name: fakeValidationValue,
						},
					},
				},
				version: esv1.VaultKVStoreV2,
				checkAndSet: &esv1.VaultCheckAndSet{
					Required: true,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid CAS config with KV v1",
			args: args{
				auth: esv1.VaultAuth{
					AppRole: &esv1.VaultAppRole{
						RoleRef: &esmeta.SecretKeySelector{
							Name: fakeValidationValue,
						},
					},
				},
				version: esv1.VaultKVStoreV1,
				checkAndSet: &esv1.VaultCheckAndSet{
					Required: true,
				},
			},
			wantErr: true,
		},
		{
			name: "CAS config not required is valid with any version",
			args: args{
				auth: esv1.VaultAuth{
					AppRole: &esv1.VaultAppRole{
						RoleRef: &esmeta.SecretKeySelector{
							Name: fakeValidationValue,
						},
					},
				},
				version: esv1.VaultKVStoreV1,
				checkAndSet: &esv1.VaultCheckAndSet{
					Required: false,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Provider{
				NewVaultClient: nil,
			}
			auth := tt.args.auth
			store := &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Vault: &esv1.VaultProvider{
							Auth:        &auth,
							ClientTLS:   tt.args.clientTLS,
							Version:     tt.args.version,
							CheckAndSet: tt.args.checkAndSet,
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
