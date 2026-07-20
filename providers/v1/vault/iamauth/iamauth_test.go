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

package iamauth

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/external-secrets/external-secrets/runtime/util/fake"
)

func TestTokenFetcher(t *testing.T) {
	tf := &authTokenFetcher{
		ServiceAccount: "foobar",
		Namespace:      "example",
		Context:        t.Context(),
		k8sClient:      fake.NewCreateTokenMock().WithToken("FAKETOKEN"),
	}
	token, err := tf.GetIdentityToken()
	assert.Nil(t, err)
	assert.Equal(t, []byte("FAKETOKEN"), token)
}

func TestResolveSTSEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantHost string
		wantErr  bool
	}{
		{
			name:     "default resolver without override",
			wantHost: "sts.us-east-1.amazonaws.com",
		},
		{
			name:     "valid override",
			endpoint: "https://sts.internal.example.com",
			wantHost: "sts.internal.example.com",
		},
		{
			name:     "override without a scheme is rejected",
			endpoint: "sts.internal.example.com",
			wantErr:  true,
		},
		{
			name:     "unparsable override is rejected",
			endpoint: "https://sts.internal example.com",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(STSEndpointEnv, tt.endpoint)
			ep, err := ResolveSTSEndpoint(t.Context(), "us-east-1")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantHost, ep.URI.Host)
		})
	}
}
