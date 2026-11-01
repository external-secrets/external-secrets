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

package mfa

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// RFC 6238 (Appendix B) test seeds, base32 encoded without padding.
const (
	// "12345678901234567890".
	rfc6238SeedSHA1 = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	// "12345678901234567890123456789012".
	rfc6238SeedSHA256 = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZA"
	// "1234567890123456789012345678901234567890123456789012345678901234".
	rfc6238SeedSHA512 = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNA"
)

func TestGenerate(t *testing.T) {
	type args struct {
		jsonSpec *apiextensions.JSON
		client   client.Client
	}
	tests := []struct {
		name    string
		g       *Generator
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "no json spec should result in error",
			args: args{
				jsonSpec: nil,
			},
			wantErr: true,
		},
		{
			name: "invalid json spec should result in error",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`no json`),
				},
			},
			wantErr: true,
		},
		{
			name: "spec with secret should result in valid token",
			args: args{
				jsonSpec: &apiextensions.JSON{
					// time is used to pin the numbers, otherwise, they would keep changing.
					Raw: []byte(`{"spec": {"secret": {"name": "secret", "key": "secret"}, "when": "1998-05-05T05:05:05Z"}}`),
				},
				client: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					Name:      "secret",
					Namespace: "namespace",
					Data: map[string][]byte{
						"secret": []byte("foo"),
					},
				}).Build(),
			},
			want: map[string][]byte{
				"token":    []byte(`674024`),
				"timeLeft": []byte(`25`),
			},
			wantErr: false,
		},
		{
			name: "spec with sha1 algorithm should match the RFC 6238 test vector",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec": {"secret": {"name": "secret", "key": "secret"}, "length": 8, "algorithm": "sha1", "when": "1970-01-01T00:00:59Z"}}`),
				},
				client: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					Name:      "secret",
					Namespace: "namespace",
					Data: map[string][]byte{
						"secret": []byte(rfc6238SeedSHA1),
					},
				}).Build(),
			},
			want: map[string][]byte{
				"token":    []byte(`94287082`),
				"timeLeft": []byte(`1`),
			},
			wantErr: false,
		},
		{
			name: "spec with sha256 algorithm should match the RFC 6238 test vector",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec": {"secret": {"name": "secret", "key": "secret"}, "length": 8, "algorithm": "sha256", "when": "1970-01-01T00:00:59Z"}}`),
				},
				client: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					Name:      "secret",
					Namespace: "namespace",
					Data: map[string][]byte{
						"secret": []byte(rfc6238SeedSHA256),
					},
				}).Build(),
			},
			want: map[string][]byte{
				"token":    []byte(`46119246`),
				"timeLeft": []byte(`1`),
			},
			wantErr: false,
		},
		{
			name: "spec with sha512 algorithm should match the RFC 6238 test vector",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec": {"secret": {"name": "secret", "key": "secret"}, "length": 8, "algorithm": "sha512", "when": "1970-01-01T00:00:59Z"}}`),
				},
				client: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					Name:      "secret",
					Namespace: "namespace",
					Data: map[string][]byte{
						"secret": []byte(rfc6238SeedSHA512),
					},
				}).Build(),
			},
			want: map[string][]byte{
				"token":    []byte(`90693936`),
				"timeLeft": []byte(`1`),
			},
			wantErr: false,
		},
		{
			name: "spec with unknown algorithm should result in error",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec": {"secret": {"name": "secret", "key": "secret"}, "algorithm": "md5", "when": "1970-01-01T00:00:59Z"}}`),
				},
				client: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					Name:      "secret",
					Namespace: "namespace",
					Data: map[string][]byte{
						"secret": []byte(rfc6238SeedSHA1),
					},
				}).Build(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got, _, err := g.Generate(context.Background(), tt.args.jsonSpec, tt.args.client, "namespace")
			if (err != nil) != tt.wantErr {
				t.Errorf("Generator.Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
