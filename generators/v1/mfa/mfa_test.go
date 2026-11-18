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
	"fmt"
	"strings"
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

// generateWithAlgorithm runs the generator for an RFC 6238 style setup:
// 8 digits, pinned to T=59s so that the token is stable.
func generateWithAlgorithm(algorithm, seed string) (map[string][]byte, error) {
	spec := fmt.Sprintf(`{"spec": {"secret": {"name": "secret", "key": "secret"}, "length": 8, "algorithm": %q, "when": "1970-01-01T00:00:59Z"}}`, algorithm)
	c := clientfake.NewClientBuilder().WithObjects(&v1.Secret{
		Name:      "secret",
		Namespace: "namespace",
		Data: map[string][]byte{
			"secret": []byte(seed),
		},
	}).Build()
	got, _, err := (&Generator{}).Generate(context.Background(), &apiextensions.JSON{Raw: []byte(spec)}, c, "namespace")
	return got, err
}

// TestGenerateAlgorithm is a matrix: every supported algorithm is combined with
// every spelling of its name. To cover a new algorithm, add one row to algorithms.
func TestGenerateAlgorithm(t *testing.T) {
	// Expected values are the RFC 6238 (Appendix B) test vectors at T=59s.
	algorithms := []struct {
		name string
		seed string
		want string
	}{
		{name: "sha1", seed: rfc6238SeedSHA1, want: "94287082"},
		{name: "sha256", seed: rfc6238SeedSHA256, want: "46119246"},
		{name: "sha512", seed: rfc6238SeedSHA512, want: "90693936"},
	}
	// otpauth URIs and provider docs write the names in upper case (SHA256).
	spellings := []struct {
		name  string
		apply func(string) string
	}{
		{name: "lower", apply: strings.ToLower},
		{name: "upper", apply: strings.ToUpper},
		{name: "mixed", apply: func(s string) string { return strings.ToUpper(s[:1]) + s[1:] }},
	}

	for _, alg := range algorithms {
		for _, spelling := range spellings {
			t.Run(alg.name+"/"+spelling.name, func(t *testing.T) {
				got, err := generateWithAlgorithm(spelling.apply(alg.name), alg.seed)
				assert.NoError(t, err)
				assert.Equal(t, map[string][]byte{
					"token":    []byte(alg.want),
					"timeLeft": []byte("1"),
				}, got)
			})
		}
	}

	t.Run("empty algorithm defaults to sha1", func(t *testing.T) {
		got, err := generateWithAlgorithm("", rfc6238SeedSHA1)
		assert.NoError(t, err)
		assert.Equal(t, []byte("94287082"), got["token"])
	})

	t.Run("unknown algorithm is rejected", func(t *testing.T) {
		_, err := generateWithAlgorithm("md5", rfc6238SeedSHA1)
		assert.ErrorContains(t, err, "md5 for hash function is invalid")
	})
}
