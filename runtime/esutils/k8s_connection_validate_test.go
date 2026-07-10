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

package esutils

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// storeOfKind returns a minimal GenericStore whose GetKind() reports the
// requested kind. TypeMeta is set as well so the selector validators, which
// read GetObjectKind().GroupVersionKind().Kind, agree with GetKind().
func storeOfKind(kind string) esv1.GenericStore {
	if kind == esv1.ClusterSecretStoreKind {
		return &esv1.ClusterSecretStore{TypeMeta: metav1.TypeMeta{Kind: kind}}
	}
	return &esv1.SecretStore{TypeMeta: metav1.TypeMeta{Kind: kind}}
}

func TestIsReferentKubernetesAuth(t *testing.T) {
	tests := []struct {
		name string
		auth *esv1.KubernetesAuth
		want bool
	}{
		{name: "nil auth", auth: nil, want: false},
		{name: "empty auth", auth: &esv1.KubernetesAuth{}, want: false},
		{
			name: "cert clientCert without namespace is referent",
			auth: &esv1.KubernetesAuth{Cert: &esv1.CertAuth{
				ClientCert: esmeta.SecretKeySelector{Name: "c", Key: "tls.crt"},
				ClientKey:  esmeta.SecretKeySelector{Name: "c", Key: "tls.key", Namespace: new("ns")},
			}},
			want: true,
		},
		{
			name: "cert clientKey without namespace is referent",
			auth: &esv1.KubernetesAuth{Cert: &esv1.CertAuth{
				ClientCert: esmeta.SecretKeySelector{Name: "c", Key: "tls.crt", Namespace: new("ns")},
				ClientKey:  esmeta.SecretKeySelector{Name: "c", Key: "tls.key"},
			}},
			want: true,
		},
		{
			name: "cert with both namespaces set is not referent",
			auth: &esv1.KubernetesAuth{Cert: &esv1.CertAuth{
				ClientCert: esmeta.SecretKeySelector{Name: "c", Key: "tls.crt", Namespace: new("ns")},
				ClientKey:  esmeta.SecretKeySelector{Name: "c", Key: "tls.key", Namespace: new("ns")},
			}},
			want: false,
		},
		{
			name: "serviceAccount without namespace is referent",
			auth: &esv1.KubernetesAuth{ServiceAccount: &esmeta.ServiceAccountSelector{Name: "sa"}},
			want: true,
		},
		{
			name: "serviceAccount with namespace is not referent",
			auth: &esv1.KubernetesAuth{ServiceAccount: &esmeta.ServiceAccountSelector{Name: "sa", Namespace: new("ns")}},
			want: false,
		},
		{
			name: "token without namespace is referent",
			auth: &esv1.KubernetesAuth{Token: &esv1.TokenAuth{BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "token"}}},
			want: true,
		},
		{
			name: "token with namespace is not referent",
			auth: &esv1.KubernetesAuth{Token: &esv1.TokenAuth{BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "token", Namespace: new("ns")}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsReferentKubernetesAuth(tt.auth); got != tt.want {
				t.Errorf("IsReferentKubernetesAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateKubernetesConnection(t *testing.T) {
	tests := []struct {
		name        string
		kind        string
		server      esv1.KubernetesServer
		auth        *esv1.KubernetesAuth
		authRef     *esmeta.SecretKeySelector
		wantErr     bool
		errContains string
		wantWarning bool
	}{
		{
			name:        "no CA and no authRef warns",
			kind:        esv1.SecretStoreKind,
			wantWarning: true,
		},
		{
			name:    "authRef suppresses the no-CA warning",
			kind:    esv1.SecretStoreKind,
			authRef: &esmeta.SecretKeySelector{Name: "kubeconfig", Key: "config"},
		},
		{
			name:   "CABundle suppresses the no-CA warning",
			kind:   esv1.SecretStoreKind,
			server: esv1.KubernetesServer{CABundle: []byte("ca")},
		},
		{
			name:   "SecretStore with CAProvider (no namespace) is valid",
			kind:   esv1.SecretStoreKind,
			server: esv1.KubernetesServer{CAProvider: &esv1.CAProvider{Name: "ca"}},
		},
		{
			name:        "ClusterSecretStore CAProvider requires namespace",
			kind:        esv1.ClusterSecretStoreKind,
			server:      esv1.KubernetesServer{CAProvider: &esv1.CAProvider{Name: "ca"}},
			wantErr:     true,
			errContains: "CAProvider.namespace must not be empty",
		},
		{
			name:        "SecretStore rejects CAProvider namespace",
			kind:        esv1.SecretStoreKind,
			server:      esv1.KubernetesServer{CAProvider: &esv1.CAProvider{Name: "ca", Namespace: new("ns")}},
			wantErr:     true,
			errContains: "CAProvider.namespace must be empty",
		},
		{
			name:   "ClusterSecretStore CAProvider with namespace is valid",
			kind:   esv1.ClusterSecretStoreKind,
			server: esv1.KubernetesServer{CAProvider: &esv1.CAProvider{Name: "ca", Namespace: new("ns")}},
		},
		{
			name:        "cert missing clientCert name",
			kind:        esv1.SecretStoreKind,
			server:      esv1.KubernetesServer{CABundle: []byte("ca")},
			auth:        &esv1.KubernetesAuth{Cert: &esv1.CertAuth{ClientCert: esmeta.SecretKeySelector{Key: "tls.crt"}, ClientKey: esmeta.SecretKeySelector{Name: "c", Key: "tls.key"}}},
			wantErr:     true,
			errContains: "ClientCert.Name cannot be empty",
		},
		{
			name:        "cert missing clientCert key",
			kind:        esv1.SecretStoreKind,
			server:      esv1.KubernetesServer{CABundle: []byte("ca")},
			auth:        &esv1.KubernetesAuth{Cert: &esv1.CertAuth{ClientCert: esmeta.SecretKeySelector{Name: "c"}, ClientKey: esmeta.SecretKeySelector{Name: "c", Key: "tls.key"}}},
			wantErr:     true,
			errContains: "ClientCert.Key cannot be empty",
		},
		{
			name:   "cert valid",
			kind:   esv1.SecretStoreKind,
			server: esv1.KubernetesServer{CABundle: []byte("ca")},
			auth:   &esv1.KubernetesAuth{Cert: &esv1.CertAuth{ClientCert: esmeta.SecretKeySelector{Name: "c", Key: "tls.crt"}, ClientKey: esmeta.SecretKeySelector{Name: "c", Key: "tls.key"}}},
		},
		{
			name:        "token missing name",
			kind:        esv1.SecretStoreKind,
			server:      esv1.KubernetesServer{CABundle: []byte("ca")},
			auth:        &esv1.KubernetesAuth{Token: &esv1.TokenAuth{BearerToken: esmeta.SecretKeySelector{Key: "token"}}},
			wantErr:     true,
			errContains: "BearerToken.Name cannot be empty",
		},
		{
			name:        "token missing key",
			kind:        esv1.SecretStoreKind,
			server:      esv1.KubernetesServer{CABundle: []byte("ca")},
			auth:        &esv1.KubernetesAuth{Token: &esv1.TokenAuth{BearerToken: esmeta.SecretKeySelector{Name: "t"}}},
			wantErr:     true,
			errContains: "BearerToken.Key cannot be empty",
		},
		{
			name:   "token valid",
			kind:   esv1.SecretStoreKind,
			server: esv1.KubernetesServer{CABundle: []byte("ca")},
			auth:   &esv1.KubernetesAuth{Token: &esv1.TokenAuth{BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "token"}}},
		},
		{
			name:   "referent serviceAccount on ClusterSecretStore is allowed",
			kind:   esv1.ClusterSecretStoreKind,
			server: esv1.KubernetesServer{CABundle: []byte("ca")},
			auth:   &esv1.KubernetesAuth{ServiceAccount: &esmeta.ServiceAccountSelector{Name: "sa"}},
		},
		{
			// ClientCert has a cross-namespace selector on a SecretStore, which
			// the delegated ValidateSecretSelector rejects. Exercises the cert
			// selector error-propagation path.
			name:    "cert selector namespace mismatch is rejected",
			kind:    esv1.SecretStoreKind,
			server:  esv1.KubernetesServer{CABundle: []byte("ca")},
			auth:    &esv1.KubernetesAuth{Cert: &esv1.CertAuth{ClientCert: esmeta.SecretKeySelector{Name: "c", Key: "tls.crt", Namespace: new("other")}, ClientKey: esmeta.SecretKeySelector{Name: "c", Key: "tls.key"}}},
			wantErr: true,
		},
		{
			name:    "token selector namespace mismatch is rejected",
			kind:    esv1.SecretStoreKind,
			server:  esv1.KubernetesServer{CABundle: []byte("ca")},
			auth:    &esv1.KubernetesAuth{Token: &esv1.TokenAuth{BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "token", Namespace: new("other")}}},
			wantErr: true,
		},
		{
			name:    "serviceAccount selector namespace mismatch is rejected",
			kind:    esv1.SecretStoreKind,
			server:  esv1.KubernetesServer{CABundle: []byte("ca")},
			auth:    &esv1.KubernetesAuth{ServiceAccount: &esmeta.ServiceAccountSelector{Name: "sa", Namespace: new("other")}},
			wantErr: true,
		},
		{
			name:        "valid config without CA warns only",
			kind:        esv1.SecretStoreKind,
			auth:        &esv1.KubernetesAuth{Token: &esv1.TokenAuth{BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "token"}}},
			wantWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := storeOfKind(tt.kind)
			warnings, err := ValidateKubernetesConnection(store, tt.server, tt.auth, tt.authRef)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateKubernetesConnection() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errContains != "" && (err == nil || !strings.Contains(err.Error(), tt.errContains)) {
				t.Errorf("ValidateKubernetesConnection() error = %v, want contains %q", err, tt.errContains)
			}
			if tt.wantWarning {
				if len(warnings) != 1 || warnings[0] != WarnNoCAConfigured {
					t.Errorf("ValidateKubernetesConnection() warnings = %v, want [%q]", warnings, WarnNoCAConfigured)
				}
			} else if len(warnings) > 0 {
				t.Errorf("ValidateKubernetesConnection() unexpected warnings: %v", warnings)
			}
		})
	}
}
