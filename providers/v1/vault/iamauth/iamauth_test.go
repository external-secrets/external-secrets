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
		name              string
		endpoint          string
		region            string
		useGlobalEndpoint bool
		wantHost          string
		wantSigningRegion string
		wantErr           bool
	}{
		{
			name:              "regional resolver without override",
			region:            "us-east-1",
			wantHost:          "sts.us-east-1.amazonaws.com",
			wantSigningRegion: "us-east-1",
		},
		{
			name:              "global endpoint for a classic region signs with us-east-1 scope",
			region:            "us-west-2",
			useGlobalEndpoint: true,
			wantHost:          "sts.amazonaws.com",
			wantSigningRegion: "us-east-1",
		},
		{
			name:              "global endpoint falls back to regional for a post-2019 region",
			region:            "ap-east-1",
			useGlobalEndpoint: true,
			wantHost:          "sts.ap-east-1.amazonaws.com",
			wantSigningRegion: "ap-east-1",
		},
		{
			name:              "global endpoint falls back to regional for the China partition",
			region:            "cn-north-1",
			useGlobalEndpoint: true,
			wantHost:          "sts.cn-north-1.amazonaws.com.cn",
			wantSigningRegion: "cn-north-1",
		},
		{
			name:              "override wins over the global endpoint and signs with the requested region",
			endpoint:          "https://sts.internal.example.com",
			region:            "us-west-2",
			useGlobalEndpoint: true,
			wantHost:          "sts.internal.example.com",
			wantSigningRegion: "us-west-2",
		},
		{
			name:     "override without a scheme is rejected",
			endpoint: "sts.internal.example.com",
			region:   "us-east-1",
			wantErr:  true,
		},
		{
			name:     "unparsable override is rejected",
			endpoint: "https://sts.internal example.com",
			region:   "us-east-1",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(STSEndpointEnv, tt.endpoint)
			ep, signingRegion, err := ResolveSTSEndpoint(t.Context(), tt.region, tt.useGlobalEndpoint)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantHost, ep.URI.Host)
			assert.Equal(t, tt.wantSigningRegion, signingRegion)
		})
	}
}
