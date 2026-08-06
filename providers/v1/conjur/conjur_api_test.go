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

package conjur

import (
	"testing"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"
)

func TestTelemetryValues(t *testing.T) {
	if telemetry.IntegrationName != "external-secrets" {
		t.Errorf("IntegrationName = %q, want %q", telemetry.IntegrationName, "external-secrets")
	}
	if telemetry.IntegrationType != "external-secrets-operator" {
		t.Errorf("IntegrationType = %q, want %q", telemetry.IntegrationType, "external-secrets-operator")
	}
}

// TestClientAPIImplSetsTelemetry checks that a representative subset of ClientAPIImpl's
// NewClientFrom* methods pass the package telemetry through to conjurapi. NewClientFromCert is
// included because it takes a distinct code path (cert parsing) from the others.
func TestClientAPIImplSetsTelemetry(t *testing.T) {
	impl := &ClientAPIImpl{}

	sc, err := impl.NewClientFromKey(conjurapi.Config{
		Account:      "account1",
		ApplianceURL: "https://example.com",
	}, authn.LoginPair{Login: "login", APIKey: "apikey"})
	assertTelemetry(t, sc, err)

	sc, err = impl.NewClientFromCert(conjurapi.Config{
		Account:       "account1",
		ApplianceURL:  "https://example.com",
		AuthnType:     "cert",
		ServiceID:     "authn-cert-service",
		CertHostID:    "host/test",
		ClientCert:    testClientCertPEM,
		ClientCertKey: testClientCertKeyPEM,
	})
	assertTelemetry(t, sc, err)
}

func assertTelemetry(t *testing.T, sc SecretsClient, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	client, ok := sc.(*conjurapi.Client)
	if !ok {
		t.Fatalf("expected *conjurapi.Client, got %T", sc)
	}

	config := client.GetConfig()
	if config.IntegrationName != telemetry.IntegrationName {
		t.Errorf("IntegrationName = %q, want %q", config.IntegrationName, telemetry.IntegrationName)
	}
	if config.IntegrationType != telemetry.IntegrationType {
		t.Errorf("IntegrationType = %q, want %q", config.IntegrationType, telemetry.IntegrationType)
	}
}

// testClientCertPEM/testClientCertKeyPEM are a throwaway self-signed EC keypair used only to
// exercise NewClientFromCert's construction path; they are never used to authenticate anywhere.
const (
	testClientCertPEM = `-----BEGIN CERTIFICATE-----
MIIBCjCBsaADAgECAgEBMAoGCCqGSM49BAMCMA8xDTALBgNVBAMTBHRlc3QwHhcN
MjYwNzIyMTIyNzIwWhcNMjYwNzIyMTMyNzIwWjAPMQ0wCwYDVQQDEwR0ZXN0MFkw
EwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEOcajEc6uTdyLaLMb2a5xm9c6XYJp9IMQ
ZCGpV3YFpAn55DdjJR1z+5+uBHsEn6HaWd1EdXHmWUnoAvrUOHM79DAKBggqhkjO
PQQDAgNIADBFAiBkUfPLJMRol5G7hKP4867cNrt97t4pQsWLOzGZ+jTP8gIhAOJV
x7+Fs6KWKeSCMlJxYvAPQ7B7T/T64DZ/Yw9Ukz4a
-----END CERTIFICATE-----
`
	testClientCertKeyPEM = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIDe8Q9f07duNFHowASLtmaTJtpHdDUTySNEXwYrzYHq/oAoGCCqGSM49
AwEHoUQDQgAEOcajEc6uTdyLaLMb2a5xm9c6XYJp9IMQZCGpV3YFpAn55DdjJR1z
+5+uBHsEn6HaWd1EdXHmWUnoAvrUOHM79A==
-----END EC PRIVATE KEY-----
`
)
