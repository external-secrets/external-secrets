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

package controller

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCurvePreferences(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    []tls.CurveID
		wantErr bool
	}{
		{
			name:  "well-known curve names",
			input: []string{"X25519", "CurveP256"},
			want:  []tls.CurveID{tls.X25519, tls.CurveP256},
		},
		{
			name:  "curve name aliases",
			input: []string{"P-256", "P384"},
			want:  []tls.CurveID{tls.CurveP256, tls.CurveP384},
		},
		{
			name:  "whitespace is trimmed",
			input: []string{" X25519 ", "CurveP256"},
			want:  []tls.CurveID{tls.X25519, tls.CurveP256},
		},
		{
			name:  "nil input uses Go defaults",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty input uses Go defaults",
			input: []string{},
			want:  nil,
		},
		{
			name:    "unknown curve name",
			input:   []string{"not-a-curve"},
			wantErr: true,
		},
		{
			name:    "empty curve entry",
			input:   []string{"X25519", ""},
			wantErr: true,
		},
		{
			name:  "decimal curve ID",
			input: []string{"29"},
			want:  []tls.CurveID{tls.X25519},
		},
		{
			name:  "all four standard curves",
			input: []string{"X25519", "CurveP256", "CurveP384", "CurveP521"},
			want:  []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384, tls.CurveP521},
		},
		{
			name:  "P-521 alias",
			input: []string{"P-521"},
			want:  []tls.CurveID{tls.CurveP521},
		},
		{
			name:    "invalid numeric curve ID",
			input:   []string{"9999"},
			wantErr: true,
		},
		{
			name:    "zero is not a valid curve ID",
			input:   []string{"0"},
			wantErr: true,
		},
		{
			name:  "valid post-quantum hybrid curve by number",
			input: []string{"4588"},
			want:  []tls.CurveID{tls.X25519MLKEM768},
		},
		{
			name:  "post-quantum hybrid curve by name",
			input: []string{"X25519MLKEM768"},
			want:  []tls.CurveID{tls.X25519MLKEM768},
		},
		{
			name:  "SecP256r1MLKEM768 by name",
			input: []string{"SecP256r1MLKEM768"},
			want:  []tls.CurveID{tls.SecP256r1MLKEM768},
		},
		{
			name:  "SecP384r1MLKEM1024 by name",
			input: []string{"SecP384r1MLKEM1024"},
			want:  []tls.CurveID{tls.SecP384r1MLKEM1024},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCurvePreferences(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseCipherSuites(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []uint16
		wantErr bool
	}{
		{
			name:  "empty string uses Go defaults",
			input: "",
			want:  nil,
		},
		{
			name:  "single cipher suite",
			input: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			want:  []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		},
		{
			name:  "multiple cipher suites preserve order",
			input: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
			want: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
		},
		{
			name:    "unknown cipher suite returns error",
			input:   "NOT_A_REAL_CIPHER",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCipherSuites(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func applyTLSOpts(opts []func(*tls.Config)) *tls.Config {
	cfg := &tls.Config{}
	for _, fn := range opts {
		fn(cfg)
	}
	return cfg
}

func TestBuildTLSConfigFuncs(t *testing.T) {
	tests := []struct {
		name                 string
		ciphers              string
		minVer               string
		curves               []string
		http2                bool
		wantErr              bool
		wantMinVersion       uint16
		wantCipherSuites     bool
		wantCurvePreferences bool
		wantHTTP2Disabled    bool
	}{
		{
			name:              "all empty uses Go defaults, HTTP/2 disabled",
			wantHTTP2Disabled: true,
		},
		{
			name:              "empty minVersion does not set MinVersion",
			minVer:            "",
			wantMinVersion:    0,
			wantHTTP2Disabled: true,
		},
		{
			name:              "explicit minVersion sets MinVersion",
			minVer:            "1.3",
			wantMinVersion:    tls.VersionTLS13,
			wantHTTP2Disabled: true,
		},
		{
			name:              "explicit minVersion 1.2",
			minVer:            "1.2",
			wantMinVersion:    tls.VersionTLS12,
			wantHTTP2Disabled: true,
		},
		{
			name:              "valid cipher suites are applied",
			ciphers:           "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			wantCipherSuites:  true,
			wantHTTP2Disabled: true,
		},
		{
			name:    "invalid cipher suite returns error",
			ciphers: "NOT_A_REAL_CIPHER",
			wantErr: true,
		},
		{
			name:                 "valid curve preferences are applied",
			curves:               []string{"X25519", "CurveP256"},
			wantCurvePreferences: true,
			wantHTTP2Disabled:    true,
		},
		{
			name:    "invalid curve preference returns error",
			curves:  []string{"not-a-curve"},
			wantErr: true,
		},
		{
			name:    "invalid minVersion returns error",
			minVer:  "1.4",
			wantErr: true,
		},
		{
			name:  "HTTP/2 enabled does not add disableHTTP2",
			http2: true,
		},
		{
			name:              "HTTP/2 enabled with RSA MTI cipher succeeds",
			ciphers:           "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			http2:             true,
			wantCipherSuites:  true,
			wantHTTP2Disabled: false,
		},
		{
			name:              "HTTP/2 enabled with ECDSA MTI cipher succeeds",
			ciphers:           "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
			http2:             true,
			wantCipherSuites:  true,
			wantHTTP2Disabled: false,
		},
		{
			name:    "HTTP/2 enabled without MTI cipher fails",
			ciphers: "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			http2:   true,
			wantErr: true,
		},
		{
			name:              "HTTP/2 enabled with min version 1.3 skips MTI check",
			ciphers:           "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			minVer:            "1.3",
			http2:             true,
			wantMinVersion:    tls.VersionTLS13,
			wantCipherSuites:  true,
			wantHTTP2Disabled: false,
		},
		{
			name:              "HTTP/2 disabled allows cipher without MTI",
			ciphers:           "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			http2:             false,
			wantCipherSuites:  true,
			wantHTTP2Disabled: true,
		},
		{
			name:                 "all settings together",
			ciphers:              "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			minVer:               "1.2",
			curves:               []string{"X25519"},
			http2:                false,
			wantMinVersion:       tls.VersionTLS12,
			wantCipherSuites:     true,
			wantCurvePreferences: true,
			wantHTTP2Disabled:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := buildTLSConfigFuncs(tt.ciphers, tt.minVer, tt.curves, tt.http2)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			cfg := applyTLSOpts(opts)

			require.Equal(t, tt.wantMinVersion, cfg.MinVersion,
				"MinVersion mismatch")

			if tt.wantCipherSuites {
				require.NotEmpty(t, cfg.CipherSuites,
					"expected CipherSuites to be set")
			} else {
				require.Empty(t, cfg.CipherSuites,
					"expected CipherSuites to be empty")
			}

			if tt.wantCurvePreferences {
				require.NotEmpty(t, cfg.CurvePreferences,
					"expected CurvePreferences to be set")
			} else {
				require.Empty(t, cfg.CurvePreferences,
					"expected CurvePreferences to be empty")
			}

			if tt.wantHTTP2Disabled {
				require.Equal(t, []string{"http/1.1"}, cfg.NextProtos,
					"expected HTTP/2 to be disabled")
			} else {
				require.Empty(t, cfg.NextProtos,
					"expected NextProtos to be empty (HTTP/2 enabled)")
			}
		})
	}
}

func TestParseTLSVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint16
		wantErr bool
	}{
		{name: "1.0", input: "1.0", want: tls.VersionTLS10},
		{name: "1.1", input: "1.1", want: tls.VersionTLS11},
		{name: "1.2", input: "1.2", want: tls.VersionTLS12},
		{name: "1.3", input: "1.3", want: tls.VersionTLS13},
		{name: "unsupported version returns error", input: "1.4", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTLSVersion(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
