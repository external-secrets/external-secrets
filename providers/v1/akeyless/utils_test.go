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

package akeyless

import (
	"testing"

	"github.com/stretchr/testify/require"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestIgnoreCacheEnabled(t *testing.T) {
	t.Parallel()

	trueVal := true
	falseVal := false

	tests := []struct {
		name string
		prov *esv1.AkeylessProvider
		want bool
	}{
		{
			name: "nil provider",
			prov: nil,
			want: false,
		},
		{
			name: "unset",
			prov: &esv1.AkeylessProvider{},
			want: false,
		},
		{
			name: "explicit false",
			prov: &esv1.AkeylessProvider{IgnoreCache: &falseVal},
			want: false,
		},
		{
			name: "explicit true",
			prov: &esv1.AkeylessProvider{IgnoreCache: &trueVal},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, ignoreCacheEnabled(tt.prov))
		})
	}
}

func TestResolveAkeylessAPIURL(t *testing.T) {
	prevGetenv := getenv
	prevProbe := probeAkeylessURL
	t.Cleanup(func() {
		getenv = prevGetenv
		probeAkeylessURL = prevProbe
	})
	probeAkeylessURL = func(string) string { return "" }

	tests := []struct {
		name       string
		configured string
		env        string
		want       string
	}{
		{
			name: "default global api",
			want: defaultAPIUrl,
		},
		{
			name: "AKEYLESS_URL vault eu maps to api eu",
			env:  "https://vault.eu.akeyless.io",
			want: "https://api.eu.akeyless.io",
		},
		{
			name: "AKEYLESS_URL bare host",
			env:  "vault.us.akeyless.io",
			want: "https://api.us.akeyless.io",
		},
		{
			name:       "store api url wins over env",
			configured: "https://api.eu.akeyless.io",
			env:        "https://vault.us.akeyless.io",
			want:       "https://api.eu.akeyless.io",
		},
		{
			name:       "store vault url maps to api",
			configured: "https://vault.eu.akeyless.io",
			want:       "https://api.eu.akeyless.io",
		},
		{
			name:       "fedramp vault-akeyless host",
			configured: "https://vault-akeyless.staging.secondfront.com",
			want:       "https://api-akeyless.staging.secondfront.com",
		},
		{
			name:       "gateway without v2 gets v2 path",
			configured: "https://gw.example.com:8080",
			want:       "https://gw.example.com:8080/v2",
		},
		{
			name:       "gateway already v2 unchanged",
			configured: "https://gw.example.com:8080/v2",
			want:       "https://gw.example.com:8080/v2",
		},
		{
			name:       "saas api with trailing slash trimmed",
			configured: "https://api.akeyless.io/",
			want:       "https://api.akeyless.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv = func(string) string { return tt.env }
			require.Equal(t, tt.want, resolveAkeylessAPIURL(tt.configured))
		})
	}
}

func TestGetV2UrlLeavesSaaSProbeAlone(t *testing.T) {
	prevProbe := probeAkeylessURL
	t.Cleanup(func() { probeAkeylessURL = prevProbe })
	probeAkeylessURL = func(string) string { return `{"error":"unknown command ''"}` }

	require.Equal(t, "https://api.custom.example", getV2Url("https://api.custom.example"))
}

func TestIsAkeylessSaaSAPIHost(t *testing.T) {
	t.Parallel()
	require.True(t, isAkeylessSaaSAPIHost("api.akeyless.io"))
	require.True(t, isAkeylessSaaSAPIHost("api.eu.akeyless.io"))
	require.True(t, isAkeylessSaaSAPIHost("api-akeyless.staging.secondfront.com"))
	require.False(t, isAkeylessSaaSAPIHost("vault.eu.akeyless.io"))
	require.False(t, isAkeylessSaaSAPIHost("gw.example.com"))
}
