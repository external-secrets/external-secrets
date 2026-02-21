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

package fake

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"testing"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestResolve(t *testing.T) {
	fr := &FakeSecretKeyResolver{}

	// TESTING FAKE PRIVATE KEY GENERATION
	t.Run("Fake Private Key Generation", func(t *testing.T) {
		privKeyPEM, err := fr.Resolve(context.Background(), nil, "", "", esmeta.SecretKeySelector{Name: "Valid mtls client key"})
		if err != nil {
			t.Errorf("Failed to generate fake private key: %v", err)
		}

		if err := decodePEM(privKeyPEM, "RSA PRIVATE KEY"); err != nil {
			t.Error(err)
		}
	})

	// TESTING FAKE CLIENT CERTIFICATE GENERATION
	t.Run("Fake Client Certificate Generation", func(t *testing.T) {
		certPEM, err := fr.Resolve(context.Background(), nil, "", "", esmeta.SecretKeySelector{Name: "Valid mtls client certificate"})
		if err != nil {
			t.Errorf("Failed to generate fake client certificate: %v", err)
		}

		if err := decodePEM(certPEM, "CERTIFICATE"); err != nil {
			t.Error(err)
		}
	})
}

func decodePEM(pemStr, expectedType string) error {
	pemBlock, _ := pem.Decode([]byte(pemStr))
	if pemBlock == nil {
		return errors.New("Failed to decode PEM (client certificate): got nil")
	}

	if pemBlock.Type != expectedType {
		return fmt.Errorf("Unexpected PEM type: got %s, want %s", pemBlock.Type, expectedType)
	}

	return nil
}
