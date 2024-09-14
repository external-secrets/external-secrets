/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package fortanix

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fortanix/sdkms-client-go/sdkms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var (
	securityObjectID   = "id"
	securityObjectName = "securityObjectName"
	securityObjectUser = "user"
)

func newTestClient(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *client {
	const apiKey = "api-key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	return &client{
		sdkms: sdkms.Client{
			HTTPClient: http.DefaultClient,
			Auth:       sdkms.APIKey(apiKey),
			Endpoint:   server.URL,
		},
	}
}

func toJSON(t *testing.T, v any) []byte {
	jsonBytes, err := json.Marshal(v)
	assert.Nil(t, err)
	return jsonBytes
}

type testSecurityObjectValue struct {
	Property string `json:"property"`
}

func TestGetOpaqueSecurityObject(t *testing.T) {
	ctx := context.Background()

	securityObjectValue := toJSON(t, testSecurityObjectValue{
		Property: "value",
	})

	securityObject := sdkms.Sobject{
		Creator: sdkms.Principal{
			User: &securityObjectUser,
		},
		Name:  &securityObjectName,
		Value: &securityObjectValue,
	}

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(securityObject)
		require.NoError(t, err)
	})

	t.Run("get raw secret value from opaque security object", func(t *testing.T) {
		ref := esv1beta1.ExternalSecretDataRemoteRef{
			Key: securityObjectName,
		}

		got, err := client.GetSecret(ctx, ref)

		assert.NoError(t, err)
		assert.Equal(t, securityObjectValue, got)
	})

	t.Run("get inner property value from opaque security object", func(t *testing.T) {
		ref := esv1beta1.ExternalSecretDataRemoteRef{
			Key:      securityObjectName,
			Property: "property",
		}

		got, err := client.GetSecret(ctx, ref)

		assert.NoError(t, err)
		assert.Equal(t, []byte(`value`), got)
	})
}

func TestGetSecretSecurityObject(t *testing.T) {
	ctx := context.Background()

	securityObjectValue := toJSON(t, testSecurityObjectValue{
		Property: "value",
	})

	securityObject := sdkms.Sobject{
		Creator: sdkms.Principal{
			User: &securityObjectUser,
		},
		Name:    &securityObjectName,
		Kid:     &securityObjectID,
		Value:   &securityObjectValue,
		ObjType: sdkms.ObjectTypeSecret,
	}

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(securityObject)
		require.NoError(t, err)
	})

	t.Run("get raw secret value from secret security object", func(t *testing.T) {
		ref := esv1beta1.ExternalSecretDataRemoteRef{
			Key: securityObjectName,
		}

		got, err := client.GetSecret(ctx, ref)

		assert.NoError(t, err)
		assert.Equal(t, securityObjectValue, got)
	})

	t.Run("get inner property value from secret security object", func(t *testing.T) {
		ref := esv1beta1.ExternalSecretDataRemoteRef{
			Key:      securityObjectName,
			Property: "property",
		}

		got, err := client.GetSecret(ctx, ref)

		assert.NoError(t, err)
		assert.Equal(t, []byte(`value`), got)
	})
}

func TestDataFromExtract(t *testing.T) {
	ctx := context.Background()

	securityObjectValue := toJSON(t, testSecurityObjectValue{
		Property: "value",
	})

	securityObject := sdkms.Sobject{
		Creator: sdkms.Principal{
			User: &securityObjectUser,
		},
		Name:    &securityObjectName,
		Kid:     &securityObjectID,
		Value:   &securityObjectValue,
		ObjType: sdkms.ObjectTypeSecret,
	}

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(securityObject)
		require.NoError(t, err)
	})

	t.Run("extract data from secret security object", func(t *testing.T) {
		ref := esv1beta1.ExternalSecretDataRemoteRef{
			Key: securityObjectName,
		}

		got, err := client.GetSecretMap(ctx, ref)

		assert.NoError(t, err)

		for k, v := range got {
			assert.Equal(t, "property", k)
			assert.Equal(t, []byte(`value`), v)
		}
	})
}
