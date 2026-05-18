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
package vault

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/vault/fake"
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
							Namespace: new("invalid"),
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
							Namespace: new("invalid"),
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
							Namespace: new("invalid"),
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
							Namespace: new("invalid"),
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
							Namespace: new("invalid"),
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
							Namespace: new("invalid"),
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
							Namespace: new("invalid"),
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
							Namespace: new("invalid"),
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
						Namespace: new("invalid"),
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

func TestValidateKVMount(t *testing.T) {
	path := "secret"
	emptyPath := ""
	readErr := errors.New("permission denied")

	tests := []struct {
		name             string
		storePath        *string
		storeVersion     esv1.VaultKVStoreVersion
		mountData        map[string]any
		readErr          error
		wantRead         bool
		wantReadPaths    []string
		wantResult       esv1.ValidationResult
		wantErrSubstring string
	}{
		{
			name:         "token valid and kv v2 matching",
			storePath:    &path,
			storeVersion: esv1.VaultKVStoreV2,
			mountData: map[string]any{
				"type": "kv",
				"options": map[string]any{
					"version": "2",
				},
			},
			wantRead:   true,
			wantResult: esv1.ValidationResultReady,
		},
		{
			name:         "token valid and kv v2 matching with string options map",
			storePath:    &path,
			storeVersion: esv1.VaultKVStoreV2,
			mountData: map[string]any{
				"type": "kv",
				"options": map[string]string{
					"version": "2",
				},
			},
			wantRead:   true,
			wantResult: esv1.ValidationResultReady,
		},
		{
			name:         "token valid and kv v1 matching",
			storePath:    &path,
			storeVersion: esv1.VaultKVStoreV1,
			mountData: map[string]any{
				"type": "kv",
			},
			wantRead:   true,
			wantResult: esv1.ValidationResultReady,
		},
		{
			name:             "token valid and mount missing",
			storePath:        &path,
			storeVersion:     esv1.VaultKVStoreV2,
			wantRead:         true,
			wantResult:       esv1.ValidationResultError,
			wantErrSubstring: "was not found",
		},
		{
			name:         "token valid and mount is not kv",
			storePath:    &path,
			storeVersion: esv1.VaultKVStoreV2,
			mountData: map[string]any{
				"type": "database",
			},
			wantRead:         true,
			wantResult:       esv1.ValidationResultError,
			wantErrSubstring: "is not a kv secret engine",
		},
		{
			name:         "token valid and version mismatch",
			storePath:    &path,
			storeVersion: esv1.VaultKVStoreV1,
			mountData: map[string]any{
				"type": "kv",
				"options": map[string]any{
					"version": "2",
				},
			},
			wantRead:         true,
			wantResult:       esv1.ValidationResultError,
			wantErrSubstring: "configured for kv version v1",
		},
		{
			name:         "nil path skips mount validation",
			storePath:    nil,
			storeVersion: esv1.VaultKVStoreV2,
			wantResult:   esv1.ValidationResultReady,
		},
		{
			name:         "empty path skips mount validation",
			storePath:    &emptyPath,
			storeVersion: esv1.VaultKVStoreV2,
			wantResult:   esv1.ValidationResultReady,
		},
		{
			name:         "internal ui mount read error preserves ready",
			storePath:    &path,
			storeVersion: esv1.VaultKVStoreV2,
			readErr:      readErr,
			wantReadPaths: []string{
				"sys/internal/ui/mounts/secret",
			},
			wantResult: esv1.ValidationResultReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var readPaths []string
			var readIssue string
			c := &client{
				store: &esv1.VaultProvider{
					Path:    tt.storePath,
					Version: tt.storeVersion,
				},
				storeKind: esv1.SecretStoreKind,
				token:     validValidationToken(),
				logical: fake.Logical{
					ReadWithDataWithContextFn: func(_ context.Context, readPath string, data map[string][]string) (*vaultapi.Secret, error) {
						readPaths = append(readPaths, readPath)
						if data != nil {
							readIssue = "unexpected mount read data"
						}
						switch readPath {
						case "sys/internal/ui/mounts/" + path:
							if tt.readErr != nil {
								return nil, tt.readErr
							}
							if tt.mountData == nil {
								return nil, nil
							}
							return &vaultapi.Secret{Data: tt.mountData}, nil
						default:
							readIssue = "unexpected mount path: " + readPath
							return nil, nil
						}
					},
				},
			}

			got, err := c.Validate()
			if got != tt.wantResult {
				t.Fatalf("client.Validate() result = %v, want %v", got, tt.wantResult)
			}
			wantReadPaths := tt.wantReadPaths
			if wantReadPaths == nil && tt.wantRead {
				wantReadPaths = []string{"sys/internal/ui/mounts/" + path}
			}
			if !equalStringSlices(readPaths, wantReadPaths) {
				t.Fatalf("mount read paths = %v, want %v", readPaths, wantReadPaths)
			}
			if readIssue != "" {
				t.Fatal(readIssue)
			}
			if tt.wantErrSubstring == "" {
				if err != nil {
					t.Fatalf("client.Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("client.Validate() error = nil, want substring %q", tt.wantErrSubstring)
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstring) {
				t.Fatalf("client.Validate() error = %v, want substring %q", err, tt.wantErrSubstring)
			}
		})
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func validValidationToken() fake.Token {
	return fake.Token{
		LookupSelfWithContextFn: func(context.Context) (*vaultapi.Secret, error) {
			return &vaultapi.Secret{
				Data: map[string]any{
					"expire_time": nil,
					"ttl":         json.Number("0"),
					"type":        "service",
				},
			}, nil
		},
	}
}
