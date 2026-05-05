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

package webhook

import (
	"testing"
	"net/http/httptest"

	"github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/spnego"
)

func TestParseUserString(t *testing.T) {

	tests := map[string]struct {
		input   string
		want    []string
		wantErr string
	}{
		"valid user and realm": {
			input: "user@EXAMPLE.COM",
			want:  []string{"user", "EXAMPLE.COM"},
		},
		"invalid domain backslash format": {
			input:   `EXAMPLE\\user`,
			wantErr: "Unable to parse user and domain, expecting 'user@domain' format",
		},
		"missing realm": {
			input:   "user",
			wantErr: "Unable to parse user, expecting 'user@domain' format",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			got, err := parseUserString(tc.input)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("unexpected error, want %q got %q", tc.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("unexpected user parts length: want %d got %d", len(tc.want), len(got))
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("unexpected part at idx %d: want %q got %q", i, tc.want[i], got[i])
				}
			}
		})
	}
}

// some go library smoke tests

func TestClientNewWithPassword(t *testing.T) {

	cfg := config.New()
	cl := client.NewWithPassword("dummy-user", "TEST.REALM.COM", "dummy-password", cfg, client.DisablePAFXFAST(true))

	if cl == nil {
		t.Fatal("expected kerberos client to be created")
	}
	if cl.Config == nil {
		t.Fatal("expected kerberos client config to be set")
	}
	if cl.Config != cfg {
		t.Fatal("expected kerberos client to keep provided krb config")
	}
}


func TestKrbConfig(t *testing.T) {
	cfg := config.New()
	if cfg == nil {
		t.Fatal("Failed to create KrbConfig")
	}

	cfg.LibDefaults.DNSLookupKDC = true
    if !cfg.LibDefaults.DNSLookupKDC {
        t.Fatal("expected DNSLookupKDC to be settable")
    }
}

func TestSetSPNEGOHeaderWithoutKDC(t *testing.T) {
    req := httptest.NewRequest("GET", "http://example.com", nil)

    cfg := config.New()
    cfg.LibDefaults.DNSLookupKDC = false // keep test deterministic: no DNS KDC lookup

    cl := client.NewWithPassword(
        "dummy-user",
        "TEST.REALM.COM",
        "dummy-password",
        cfg,
        client.DisablePAFXFAST(true),
    )

    h := req.URL.Hostname()
    err := spnego.SetSPNEGOHeader(cl, req, "HTTP/"+h)
    if err == nil {
        t.Fatal("expected SetSPNEGOHeader to fail without KDC/login, got nil error")
    }

    if got := req.Header.Get("Authorization"); got != "" {
        t.Fatalf("expected no Authorization header on failure, got %q", got)
    }
}