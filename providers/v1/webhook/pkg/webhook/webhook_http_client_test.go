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
    "bytes"
    "context"
    "crypto/x509"
    "encoding/pem"
    "net/http"
    "testing"

    "github.com/Azure/go-ntlmssp"
    esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const testCACertPEM = `-----BEGIN CERTIFICATE-----
MIIDFTCCAf2gAwIBAgIUCwAK6sXkh7g/NjkpX3R+QkJ5eaowDQYJKoZIhvcNAQEL
BQAwGjEYMBYGA1UEAwwPd2ViaG9vay10ZXN0LWNhMB4XDTI2MDYyMzA1NTk0M1oX
DTM2MDYyMDA1NTk0M1owGjEYMBYGA1UEAwwPd2ViaG9vay10ZXN0LWNhMIIBIjAN
BgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAhhUlShZKYuUjYNwbhCog4n57M6ox
RT6l44ItSUx8hHHL+N6t5fMeG/poUyb0A46D0AMwEHEV9jge0yHS6AFipys9Y8Lb
hPMHs6IBCl63SkxNvAeG0mk09W46H8jEh4ghz+I2T4W1W85WOb2uHSO3Fxy5WDWb
rJFbCZsoD4mYFJ9vzeuykvD8UE5u7UDt2z+waD8Wq2IdsDjEwQwdhvrQYYFM9mtZ
1wqVlfcdJqhMWpXoTThipUkxZVV8z+8iHCAe+19n0dCg5raafqduXkFRAZebT10O
U2K3o2c90oaY0zHRfhWshuxjmHJ6psx9HTWtha6ar6G5045oJDEumJBxoQIDAQAB
o1MwUTAdBgNVHQ4EFgQUI+J6jxoMKnCtvMBWfbEdGHx46MQwHwYDVR0jBBgwFoAU
I+J6jxoMKnCtvMBWfbEdGHx46MQwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0B
AQsFAAOCAQEAG3H6DXD+eoODwRWJNa5P2H97ET2BYqAW/xrOAUp2If7z1b59wQH4
7OJ8T4PBl9X8TdQX8Su9CNE/MWLhyykj7VbjFai8r0XPWvwIMHwslJQnCJ3E8lmw
+5A25COfmgUhzVWZuwH+mQ1ipyw/uhv/XaB5F0Dq0euVo56hIHMUpIayWTGEn81a
5Uzq+2FroHcVJ8A3AZTXGc5TkFeUJioNZQUmJekJ0GEYr8BvHqv5D1E1qoPLKOrD
3w0vsWsVcsvLvLNlKPda+T6BgEmDxbuEusPHVvtpJ10ZEzq4KqlMcOi4GhLI9Gj/
d0OvwbJ2Ff3GvJm++4wRnz1enL4pYDM+zw==
-----END CERTIFICATE-----
`

func TestWebhookHTTPClientCABundle(t *testing.T) {
    caBundle, caCert := testCABundle(t)

    testCases := map[string]struct {
        spec  *Spec
        check func(*testing.T, *http.Client, *x509.Certificate)
    }{
        "NTLM": {
            spec: &Spec{
                CABundle: caBundle,
                Auth: &AuthorizationProtocol{
                    NTLM: &NTLMProtocol{
                        UserName: esmeta.SecretKeySelector{
                            Name: "dummy",
                            Key:  "userName",
                        },
                        Password: esmeta.SecretKeySelector{
                            Name: "dummy",
                            Key:  "password",
                        },
                    },
                },
            },
            check: func(t *testing.T, client *http.Client, want *x509.Certificate) {
                ntlmTransport, ok := client.Transport.(*ntlmssp.Negotiator)
                if !ok {
                    t.Fatalf("expected *ntlmssp.Negotiator, got %T", client.Transport)
                }

                baseTransport, ok := ntlmTransport.RoundTripper.(*http.Transport)
                if !ok {
                    t.Fatalf("expected *http.Transport, got %T", ntlmTransport.RoundTripper)
                }

                assertTransportHasCA(t, baseTransport, want)
            },
        },
        "Kerberos": {
            spec: &Spec{
                CABundle: caBundle,
                Auth: &AuthorizationProtocol{
                    Kerberos: &KerberosProtocol{
                        UserName: esmeta.SecretKeySelector{
                            Name: "dummy",
                            Key:  "userName",
                        },
                        Password: esmeta.SecretKeySelector{
                            Name: "dummy",
                            Key:  "password",
                        },
                    },
                },
            },
            check: func(t *testing.T, client *http.Client, want *x509.Certificate) {
                krbTransport, ok := client.Transport.(*KrbTransport)
                if !ok {
                    t.Fatalf("expected *KrbTransport, got %T", client.Transport)
                }

                assertTransportHasCA(t, krbTransport.Transport, want)
            },
        },
    }

    for name, tc := range testCases {
        t.Run(name, func(t *testing.T) {
            w := &Webhook{}
            client, err := w.GetHTTPClient(context.Background(), tc.spec)
            if err != nil {
                t.Fatalf("GetHTTPClient failed: %v", err)
            }

            tc.check(t, client, caCert)
        })
    }
}

func testCABundle(t *testing.T) ([]byte, *x509.Certificate) {
    t.Helper()

    caBundle := []byte(testCACertPEM)

    block, _ := pem.Decode(caBundle)
    if block == nil {
        t.Fatal("failed to decode test CA PEM")
    }

    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
        t.Fatalf("failed to parse certificate: %v", err)
    }

    return caBundle, cert
}

func assertTransportHasCA(t *testing.T, transport *http.Transport, want *x509.Certificate) {
    t.Helper()

    if transport == nil {
        t.Fatal("transport is nil")
    }
    if transport.TLSClientConfig == nil {
        t.Fatal("TLSClientConfig is nil")
    }
    if transport.TLSClientConfig.RootCAs == nil {
        t.Fatal("RootCAs is nil")
    }

    subjects := transport.TLSClientConfig.RootCAs.Subjects()
    if len(subjects) == 0 {
        t.Fatal("RootCAs is empty")
    }

    found := false
    for _, subject := range subjects {
        if bytes.Equal(subject, want.RawSubject) {
            found = true
            break
        }
    }
    if !found {
        t.Fatal("expected CA subject was not found in RootCAs")
    }
}