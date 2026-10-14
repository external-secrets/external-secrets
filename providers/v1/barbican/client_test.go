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

package barbican

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	thclient "github.com/gophercloud/gophercloud/v2/testhelper/client"
	"github.com/stretchr/testify/assert"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestExtractUUIDFromRef(t *testing.T) {
	testCases := []struct {
		name         string
		secretRef    string
		expectedUUID string
	}{
		{
			name:         "valid barbican secret ref",
			secretRef:    "https://barbican.example.com/v1/secrets/12345678-1234-1234-1234-123456789abc",
			expectedUUID: "12345678-1234-1234-1234-123456789abc",
		},
		{
			name:         "secret ref without protocol",
			secretRef:    "barbican.example.com/v1/secrets/87654321-4321-4321-4321-cba987654321",
			expectedUUID: "87654321-4321-4321-4321-cba987654321",
		},
		{
			name:         "empty string",
			secretRef:    "",
			expectedUUID: "",
		},
		{
			name:         "trailing slash",
			secretRef:    "https://barbican.example.com/v1/secrets/12345678-1234-1234-1234-123456789abc/",
			expectedUUID: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			uuid := extractUUIDFromRef(tc.secretRef)
			assert.Equal(t, tc.expectedUUID, uuid)
		})
	}
}

func TestGetSecretPayloadProperty(t *testing.T) {
	testPayload := []byte(`{"username":"admin","password":"secret123","nested":{"key":"value"}}`)

	testCases := []struct {
		name         string
		payload      []byte
		property     string
		expectError  bool
		errorMessage string
		expectedData []byte
	}{
		{
			name:         "empty property returns full payload",
			payload:      testPayload,
			property:     "",
			expectError:  false,
			expectedData: testPayload,
		},
		{
			name:         "valid property extraction",
			payload:      testPayload,
			property:     "username",
			expectError:  false,
			expectedData: []byte("admin"),
		},
		{
			name:         "nested property extraction",
			payload:      testPayload,
			property:     "nested",
			expectError:  false,
			expectedData: []byte(`{"key":"value"}`),
		},
		{
			name:         "property not found",
			payload:      testPayload,
			property:     "nonexistent",
			expectError:  true,
			errorMessage: "property nonexistent not found in secret payload",
		},
		{
			name:         "invalid JSON",
			payload:      []byte("invalid-json"),
			property:     "username",
			expectError:  true,
			errorMessage: "barbican client",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := getSecretPayloadProperty(tc.payload, tc.property)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMessage)
				assert.Nil(t, data)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedData, data)
			}
		})
	}
}

func TestGetSecret(t *testing.T) {
	const uuid = "12345678-1234-1234-1234-123456789abc"
	payload := `{"username":"admin","port":8080,"nested":{"key":"value"},"weird.key":"literal","bignum":123456789012345678}`

	fakeServer := th.SetupHTTP()
	defer fakeServer.Teardown()

	fakeServer.Mux.HandleFunc("/secrets/"+uuid+"/payload", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	})

	client := &Client{keyManager: thclient.ServiceClient(fakeServer)}

	testCases := []struct {
		name         string
		property     string
		expectError  bool
		expectedData []byte
	}{
		{
			name:         "no property returns full payload",
			property:     "",
			expectedData: []byte(payload),
		},
		{
			name:         "string value is unquoted",
			property:     "username",
			expectedData: []byte("admin"),
		},
		{
			name:         "number is returned as-is",
			property:     "port",
			expectedData: []byte("8080"),
		},
		{
			name:         "large integer keeps precision",
			property:     "bignum",
			expectedData: []byte("123456789012345678"),
		},
		{
			name:         "object stays as json",
			property:     "nested",
			expectedData: []byte(`{"key":"value"}`),
		},
		{
			name:         "nested path is followed",
			property:     "nested.key",
			expectedData: []byte("value"),
		},
		{
			name:         "dotted key matched as a literal",
			property:     "weird.key",
			expectedData: []byte("literal"),
		},
		{
			name:        "missing property errors",
			property:    "nonexistent",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := client.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
				Key:      uuid,
				Property: tc.property,
			})

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, data)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedData, data)
			}
		})
	}
}

func TestGetSecretMap(t *testing.T) {
	const uuid = "abcdef00-0000-0000-0000-000000000000"
	payload := `{"username":"admin","port":8080,"enabled":true,"nested":{"key":"value"},"bignum":123456789012345678}`

	fakeServer := th.SetupHTTP()
	defer fakeServer.Teardown()

	fakeServer.Mux.HandleFunc("/secrets/"+uuid+"/payload", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	})

	client := &Client{keyManager: thclient.ServiceClient(fakeServer)}

	result, err := client.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: uuid})
	assert.NoError(t, err)
	// String values used to keep their surrounding JSON quotes.
	assert.Equal(t, []byte("admin"), result["username"])
	assert.Equal(t, []byte("8080"), result["port"])
	assert.Equal(t, []byte("true"), result["enabled"])
	assert.Equal(t, []byte(`{"key":"value"}`), result["nested"])
	assert.Equal(t, []byte("123456789012345678"), result["bignum"])
}

func TestUnsupportedOperations(t *testing.T) {
	client := &Client{
		keyManager: &gophercloud.ServiceClient{},
	}

	// Test PushSecret
	err := client.PushSecret(context.Background(), nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support pushing secrets")

	// Test SecretExists
	exists, err := client.SecretExists(context.Background(), nil)
	assert.Error(t, err)
	assert.False(t, exists)
	assert.Contains(t, err.Error(), "does not support checking secret existence")

	// Test DeleteSecret
	err = client.DeleteSecret(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support deleting secrets")
}

func TestValidateAndClose(t *testing.T) {
	client := &Client{
		keyManager: &gophercloud.ServiceClient{},
	}

	// Test Validate
	result, err := client.Validate()
	assert.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultUnknown, result)

	// Test Close
	err = client.Close(context.Background())
	assert.NoError(t, err)
}

func TestGetAllSecretsValidation(t *testing.T) {
	client := &Client{
		keyManager: &gophercloud.ServiceClient{},
	}

	testCases := []struct {
		name         string
		findRef      esv1.ExternalSecretFind
		expectError  bool
		errorMessage string
	}{
		{
			name: "no name specified should return error",
			findRef: esv1.ExternalSecretFind{
				Name: nil,
			},
			expectError:  true,
			errorMessage: "missing field",
		},
		{
			name: "empty name regex should return error",
			findRef: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "",
				},
			},
			expectError:  true,
			errorMessage: "missing field",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.GetAllSecrets(context.Background(), tc.findRef)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMessage)
			} else if err != nil {
				assert.Contains(t, err.Error(), "barbican client")
			}
		})
	}
}

func TestGetAllSecretsRegexpMatch(t *testing.T) {
	type fakeSecret struct {
		name    string
		uuid    string
		payload string
	}
	all := []fakeSecret{
		{name: "db-a", uuid: "11111111-1111-1111-1111-111111111111", payload: "payload-db-a"},
		{name: "db-b", uuid: "22222222-2222-2222-2222-222222222222", payload: "payload-db-b"},
		{name: "web-a", uuid: "33333333-3333-3333-3333-333333333333", payload: "payload-web-a"},
	}

	fakeServer := th.SetupHTTP()
	defer fakeServer.Teardown()

	// Barbican's list endpoint only does exact-name matching, so mirror that
	// here: honor the ?name= query with a literal comparison, like the real
	// service does. A regexp value therefore matches nothing server-side, which
	// is exactly the reported bug.
	fakeServer.Mux.HandleFunc("/secrets", func(w http.ResponseWriter, r *http.Request) {
		nameFilter := r.URL.Query().Get("name")
		type listed struct {
			Name      string `json:"name"`
			SecretRef string `json:"secret_ref"`
		}
		var out []listed
		for _, s := range all {
			if nameFilter != "" && s.name != nameFilter {
				continue
			}
			out = append(out, listed{
				Name:      s.name,
				SecretRef: fmt.Sprintf("http://barbican.example.com/v1/secrets/%s", s.uuid),
			})
		}
		body, _ := json.Marshal(map[string]any{"secrets": out, "total": len(out)})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})

	for _, s := range all {
		payload := s.payload
		fakeServer.Mux.HandleFunc("/secrets/"+s.uuid+"/payload", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(payload))
		})
	}

	client := &Client{keyManager: thclient.ServiceClient(fakeServer)}

	result, err := client.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
		Name: &esv1.FindName{RegExp: "^db-"},
	})
	assert.NoError(t, err)
	// Only the db-* secrets should match the pattern, not web-a and not just a
	// literal secret named "^db-".
	assert.Len(t, result, 2)
	assert.Equal(t, []byte("payload-db-a"), result["11111111-1111-1111-1111-111111111111"])
	assert.Equal(t, []byte("payload-db-b"), result["22222222-2222-2222-2222-222222222222"])
	assert.NotContains(t, result, "33333333-3333-3333-3333-333333333333")
}
