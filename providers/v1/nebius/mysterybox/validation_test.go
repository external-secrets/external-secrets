// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package mysterybox

import (
	"strings"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	pointer "k8s.io/utils/ptr"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	utilsErrNamespaceNotAllowed = "namespace should either be empty or match the namespace of the SecretStore for a namespaced SecretStore"
	utilsErrRequireNamespace    = "cluster scope requires namespace"
	otherNs                     = "otherns"
)

func TestValidateStore(t *testing.T) {
	p := &Provider{}

	mkStore := func(cfg func(*esv1.SecretStore)) esv1.GenericStore {
		st := &esv1.SecretStore{}
		st.Namespace = "test-ns"
		st.Spec.Provider = &esv1.SecretStoreProvider{NebiusMysterybox: &esv1.NebiusMysteryboxProvider{APIDomain: "api.public"}}
		if cfg != nil {
			cfg(st)
		}
		return st
	}

	tests := []struct {
		name    string
		store   esv1.GenericStore
		wantErr string
	}{
		{
			name:    "nil store",
			store:   nil,
			wantErr: errNilStore,
		},
		{
			name:    "missing provider",
			store:   mkStore(func(s *esv1.SecretStore) { s.Spec.Provider = nil }),
			wantErr: errMissingProvider,
		},
		{
			name:    "missing nebius provider",
			store:   &esv1.SecretStore{Spec: esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{}}},
			wantErr: "invalid provider spec.",
		},
		{
			name:    "invalid auth: none provided",
			store:   mkStore(func(s *esv1.SecretStore) {}),
			wantErr: errMissingAuthOptions,
		},
		{
			name: "invalid auth: both provided",
			store: mkStore(func(s *esv1.SecretStore) {
				nm := s.Spec.Provider.NebiusMysterybox
				nm.Auth.Token = esmeta.SecretKeySelector{Name: "a", Key: "k"}
				nm.Auth.ServiceAccountCreds = esmeta.SecretKeySelector{Name: "b", Key: "k"}
			}),
			wantErr: errInvalidAuthConfig,
		},
		{
			name: "invalid token auth: missing key",
			store: mkStore(func(s *esv1.SecretStore) {
				nm := s.Spec.Provider.NebiusMysterybox
				nm.Auth.Token = esmeta.SecretKeySelector{Name: "tok"}
			}),
			wantErr: errInvalidTokenAuthConfig,
		},
		{
			name: "invalid token auth: missing name",
			store: mkStore(func(s *esv1.SecretStore) {
				nm := s.Spec.Provider.NebiusMysterybox
				nm.Auth.Token = esmeta.SecretKeySelector{Key: "key"}
			}),
			wantErr: errMissingAuthOptions,
		},
		{
			name: "invalid sa creds auth: missing key",
			store: mkStore(func(s *esv1.SecretStore) {
				nm := s.Spec.Provider.NebiusMysterybox
				nm.Auth.ServiceAccountCreds = esmeta.SecretKeySelector{Name: "creds"}
			}),
			wantErr: errInvalidSACredsAuthConfig,
		},
		{
			name: "invalid sa creds auth: missing name",
			store: mkStore(func(s *esv1.SecretStore) {
				nm := s.Spec.Provider.NebiusMysterybox
				nm.Auth.ServiceAccountCreds = esmeta.SecretKeySelector{Key: "key"}
			}),
			wantErr: errMissingAuthOptions,
		},
		{
			name: "valid: token auth",
			store: mkStore(func(s *esv1.SecretStore) {
				nm := s.Spec.Provider.NebiusMysterybox
				nm.APIDomain = apiDomain
				nm.Auth.Token = esmeta.SecretKeySelector{Name: "tok", Key: "k"}
			}),
		},
		{
			name: "valid: service account creds",
			store: mkStore(func(s *esv1.SecretStore) {
				nm := s.Spec.Provider.NebiusMysterybox
				nm.APIDomain = apiDomain
				nm.Auth.ServiceAccountCreds = esmeta.SecretKeySelector{Name: "creds", Key: "k"}
			}),
		},
		{
			name:    "missing apiDomain",
			store:   mkStore(func(s *esv1.SecretStore) { s.Spec.Provider.NebiusMysterybox.APIDomain = "" }),
			wantErr: errMissingAPIDomain,
		},
		{
			name: "token selector different namespace (namespaced store)",
			store: mkStore(func(s *esv1.SecretStore) {
				ns := otherNs
				nm := s.Spec.Provider.NebiusMysterybox
				nm.Auth.Token = esmeta.SecretKeySelector{Name: "tok", Key: "k", Namespace: &ns}
			}),
			wantErr: utilsErrNamespaceNotAllowed,
		},
		{
			name: "sa creds selector different namespace (namespaced store)",
			store: mkStore(func(s *esv1.SecretStore) {
				ns := otherNs
				nm := s.Spec.Provider.NebiusMysterybox
				nm.Auth.Token = esmeta.SecretKeySelector{}
				nm.Auth.ServiceAccountCreds = esmeta.SecretKeySelector{Name: "creds", Key: "k", Namespace: &ns}
			}),
			wantErr: utilsErrNamespaceNotAllowed,
		},
		{
			name: "ca cert specified without secret name",
			store: mkStore(func(s *esv1.SecretStore) {
				ns := otherNs
				nm := s.Spec.Provider.NebiusMysterybox
				nm.Auth.Token = esmeta.SecretKeySelector{Name: "tok", Key: "k"}
				nm.CAProvider = &esv1.NebiusCAProvider{Certificate: esmeta.SecretKeySelector{Namespace: &ns}}
			}),
			wantErr: errInvalidCertificateConfigNoNameSpecified,
		},
		{
			name: "ca cert specified without secret key",
			store: mkStore(func(s *esv1.SecretStore) {
				ns := otherNs
				nm := s.Spec.Provider.NebiusMysterybox
				nm.Auth.Token = esmeta.SecretKeySelector{Name: "tok", Key: "k"}
				nm.CAProvider = &esv1.NebiusCAProvider{Certificate: esmeta.SecretKeySelector{Name: "cacert", Namespace: &ns}}
			}),
			wantErr: errInvalidCertificateConfigNoKeySpecified,
		},
		{
			name: "ca cert selector different namespace (namespaced store)",
			store: mkStore(func(s *esv1.SecretStore) {
				ns := otherNs
				nm := s.Spec.Provider.NebiusMysterybox
				nm.Auth.Token = esmeta.SecretKeySelector{Name: "tok", Key: "k"}
				nm.CAProvider = &esv1.NebiusCAProvider{Certificate: esmeta.SecretKeySelector{Name: "ca", Key: "tls.crt", Namespace: &ns}}
			}),
			wantErr: utilsErrNamespaceNotAllowed,
		},
		{
			name: "matching selector namespace passes",
			store: mkStore(func(s *esv1.SecretStore) {
				ns := s.Namespace
				nm := s.Spec.Provider.NebiusMysterybox
				nm.Auth.Token = esmeta.SecretKeySelector{Name: "tok", Key: "k", Namespace: &ns}
			}),
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.ValidateStore(tt.store)
			if tt.wantErr == "" {
				tassert.NoError(t, err, "%s: unexpected error", tt.name)
				return
			}
			tassert.NotNil(t, err, "%s: expected error containing %q, got nil", tt.name, tt.wantErr)
			if err != nil {
				tassert.Contains(t, err.Error(), tt.wantErr, "%s: error %q does not contain %q", tt.name, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateStoreClusterScope(t *testing.T) {
	p := &Provider{}

	makeStore := func(cfg func(*esv1.NebiusMysteryboxProvider)) esv1.GenericStore {
		css := &esv1.ClusterSecretStore{}
		css.TypeMeta.Kind = esv1.ClusterSecretStoreKind
		nm := &esv1.NebiusMysteryboxProvider{APIDomain: "api.public"}
		if cfg != nil {
			cfg(nm)
		}
		css.Spec.Provider = &esv1.SecretStoreProvider{NebiusMysterybox: nm}
		return css
	}

	tests := []struct {
		name    string
		store   esv1.GenericStore
		wantErr string
	}{
		{
			name: "cluster: token selector requires namespace",
			store: makeStore(func(nm *esv1.NebiusMysteryboxProvider) {
				nm.Auth.Token = esmeta.SecretKeySelector{Name: "tok", Key: "k"}
			}),
			wantErr: utilsErrRequireNamespace,
		},
		{
			name: "cluster: namespaced token passes",
			store: makeStore(func(nm *esv1.NebiusMysteryboxProvider) {
				nm.Auth.Token = esmeta.SecretKeySelector{Name: "tok", Key: "k", Namespace: pointer.To("ns1")}
			}),
			wantErr: "",
		},
		{
			name: "cluster: sa creds selector requires namespace",
			store: makeStore(func(nm *esv1.NebiusMysteryboxProvider) {
				nm.Auth.ServiceAccountCreds = esmeta.SecretKeySelector{Name: "tok", Key: "k"}
			}),
			wantErr: utilsErrRequireNamespace,
		},
		{
			name: "cluster: namespaced sa creds passes",
			store: makeStore(func(nm *esv1.NebiusMysteryboxProvider) {
				nm.Auth.ServiceAccountCreds = esmeta.SecretKeySelector{Name: "tok", Key: "k", Namespace: pointer.To("ns1")}
			}),
			wantErr: "",
		},
		{
			name: "cluster: ca cert requires namespace",
			store: makeStore(func(nm *esv1.NebiusMysteryboxProvider) {
				nm.Auth.Token = esmeta.SecretKeySelector{Name: "tok", Key: "k", Namespace: pointer.To("ns1")}
				nm.CAProvider = &esv1.NebiusCAProvider{Certificate: esmeta.SecretKeySelector{Name: "ca", Key: "tls.crt"}}
			}),
			wantErr: utilsErrRequireNamespace,
		},
		{
			name: "cluster: namespaced ca cert passes",
			store: makeStore(func(nm *esv1.NebiusMysteryboxProvider) {
				nm.Auth.Token = esmeta.SecretKeySelector{Name: "tok", Key: "k", Namespace: pointer.To("ns1")}
				nm.CAProvider = &esv1.NebiusCAProvider{Certificate: esmeta.SecretKeySelector{Name: "ca", Key: "tls.crt", Namespace: pointer.To("ns1")}}
			}),
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.ValidateStore(tt.store)
			if tt.wantErr == "" {
				tassert.NoError(t, err, "%s: unexpected error", tt.name)
				return
			}
			if err == nil {
				tassert.Failf(t, "%s: expected error containing %q, got nil", tt.name, tt.wantErr)
			} else {
				tassert.Contains(t, err.Error(), tt.wantErr, "%s: expected error to contain substring", tt.name)
			}
		})
	}
}

func TestValidateStore_APIDomainCases(t *testing.T) {
	p := &Provider{}
	mkStore := func(domain string) esv1.GenericStore {
		st := &esv1.SecretStore{}
		st.Namespace = "test-ns"
		st.Spec.Provider = &esv1.SecretStoreProvider{NebiusMysterybox: &esv1.NebiusMysteryboxProvider{APIDomain: domain}}
		nm := st.Spec.Provider.NebiusMysterybox
		nm.Auth.Token = esmeta.SecretKeySelector{Name: "tok", Key: "k"}
		return st
	}
	cases := []struct {
		name   string
		domain string
		valid  bool
	}{
		{name: "simple domain with port", domain: "example.com:443", valid: true},
		{name: "simple domain without port", domain: "example.com", valid: true},
		{name: "subdomain", domain: "sub.example.com", valid: true},
		{name: "hyphen in middle", domain: "a-b.com", valid: true},
		{name: "uppercase allowed", domain: "EXAMPLE.COM", valid: true},

		{name: "single label not allowed", domain: "com", valid: false},
		{name: "empty label (double dot)", domain: "a..com", valid: false},
		{name: "leading dot", domain: ".example.com", valid: false},
		{name: "trailing dot", domain: "example.com.", valid: false},
		{name: "label starts with hyphen", domain: "-abc.com", valid: false},
		{name: "label ends with hyphen", domain: "abc-.com", valid: false},
		{name: "invalid char underscore", domain: "ab_c.com", valid: false},
		{name: "invalid char space", domain: "exa mple.com", valid: false},
		{name: "numeric TLD not allowed", domain: "example.123", valid: false},
		{name: "ip address not a domain", domain: "127.0.0.1", valid: false},
	}

	longLabel := strings.Repeat("a", 64) + ".com"
	cases = append(cases, struct {
		name   string
		domain string
		valid  bool
	}{name: "label too long", domain: longLabel, valid: false})

	manyLabels := strings.Repeat("a.", 127) + "a"
	cases = append(cases, struct {
		name   string
		domain string
		valid  bool
	}{name: "domain too long", domain: manyLabels, valid: false})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := mkStore(tc.domain)
			_, err := p.ValidateStore(store)
			if tc.valid {
				tassert.NoError(t, err, "%s: expected valid, got error", tc.name)
			} else {
				tassert.Error(t, err, "%s: expected error for domain %q", tc.name, tc.domain)
			}
			if err != nil {
				tassert.Contains(t, err.Error(), errInvalidAPIDomain, "%s: error should contain invalid api domain", tc.name)
			}
		})
	}
}
