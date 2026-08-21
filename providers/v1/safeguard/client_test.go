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

package safeguard

import (
	"context"
	"testing"

	sg "github.com/OneIdentity/safeguard-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type fakePushSecretData struct {
	metadata  *apiextensionsv1.JSON
	secretKey string
	remoteKey string
	property  string
}

func (f fakePushSecretData) GetMetadata() *apiextensionsv1.JSON { return f.metadata }
func (f fakePushSecretData) GetSecretKey() string               { return f.secretKey }
func (f fakePushSecretData) GetRemoteKey() string               { return f.remoteKey }
func (f fakePushSecretData) GetProperty() string                { return f.property }

func TestGetSecretMapAPIKey(t *testing.T) {
	fake := &fakeA2A{
		apiKeys: map[string][]sg.APIKey{
			"lookup-key": {
				{
					ID:           42,
					Name:         "oauth",
					Description:  "primary oauth key",
					ClientID:     "client-id",
					ClientSecret: sg.NewSecretString("client-secret"),
				},
			},
		},
	}
	client := &secretsClient{a2a: fake}

	data, err := client.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "lookup-key",
		Property: "apiKey",
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("42"), data["id"])
	assert.Equal(t, []byte("oauth"), data["name"])
	assert.Equal(t, []byte("primary oauth key"), data["description"])
	assert.Equal(t, []byte("client-id"), data["clientId"])
	assert.Equal(t, []byte("client-secret"), data["clientSecret"])
}

func TestGetSecretRejectsInvalidAPIKeyProperty(t *testing.T) {
	client := &secretsClient{a2a: &fakeA2A{}}

	_, err := client.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "lookup-key",
		Property: "apiKeys",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported property "apiKeys"`)
}

func TestParseCredentialPropertyRequiresExactPrefix(t *testing.T) {
	tests := map[string]struct {
		property string
		wantType string
		wantErr  bool
	}{
		"apikey exact": {
			property: "apiKey",
			wantType: credentialTypeAPIKey,
		},
		"apikey sub property": {
			property: "apiKey.oauth",
			wantType: credentialTypeAPIKey,
		},
		"privatekey exact": {
			property: "privateKey",
			wantType: credentialTypePrivateKey,
		},
		"privatekey format": {
			property: "privateKey.ssh2",
			wantType: credentialTypePrivateKey,
		},
		"apikeys typo": {
			property: "apiKeys",
			wantErr:  true,
		},
		"privatekeyssh typo": {
			property: "privateKeySsh",
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			credType, _, _ := parseCredentialProperty(tc.property)
			if tc.wantErr {
				assert.NotEqual(t, tc.wantType, credType)
				return
			}
			assert.Equal(t, tc.wantType, credType)
		})
	}
}

func TestPushSecretMetadataParseError(t *testing.T) {
	client := &secretsClient{a2a: &fakeA2A{}}

	err := client.PushSecret(context.Background(), &corev1.Secret{
		Data: map[string][]byte{"password": []byte("new-password")},
	}, fakePushSecretData{
		remoteKey: "lookup-key",
		secretKey: "password",
		metadata: &apiextensionsv1.JSON{
			Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":`),
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse push secret metadata")
}

func TestPushLookupOptionsUsesMetadataFilter(t *testing.T) {
	opts, err := pushLookupOptions(fakePushSecretData{
		metadata: &apiextensionsv1.JSON{
			Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"filter":"AccountName ieq 'dbadmin'"}}`),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, opts)
	assert.Equal(t, "AccountName ieq 'dbadmin'", opts.filter)
}
