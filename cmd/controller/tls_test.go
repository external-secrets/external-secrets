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
			name:  "HTTP/2 enabled does not add disableHTTP2",
			http2: true,
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
