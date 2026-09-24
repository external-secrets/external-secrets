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

package password

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestGenerate(t *testing.T) {
	type args struct {
		jsonSpec *apiextensions.JSON
		passGen  generateFunc
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
			name: "empty spec should return defaults",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					assert.Equal(t, defaultLength, length)
					assert.Equal(t, defaultSymbolChars, symbolCharacters)
					assert.Equal(t, 6, symbols)
					assert.Equal(t, 6, digits)
					assert.Equal(t, false, noUpper)
					assert.Equal(t, false, allowRepeat)
					return "foobar", nil
				},
			},
			want: map[string][]byte{
				"password": []byte(`foobar`),
			},
			wantErr: false,
		},
		{
			name: "spec should override defaults",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"length":48,"digits":2, "symbols":2, "symbolCharacters":"-_.", "noUpper": true, "allowRepeat": true}}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					assert.Equal(t, 48, length)
					assert.Equal(t, "-_.", symbolCharacters)
					assert.Equal(t, 2, symbols)
					assert.Equal(t, 2, digits)
					assert.Equal(t, true, noUpper)
					assert.Equal(t, true, allowRepeat)
					return "foobar", nil
				},
			},
			want: map[string][]byte{
				"password": []byte(`foobar`),
			},
			wantErr: false,
		},
		{
			name: "generator error should be returned",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					return "", errors.New("boom")
				},
			},
			wantErr: true,
		},
		{
			name: "spec with hex encoding should encode password as hex",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"encoding":"hex"}}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					return "test_hex", nil
				},
			},
			want: map[string][]byte{
				"password": []byte(hex.EncodeToString([]byte("test_hex"))),
			},
			wantErr: false,
		},
		{
			name: "spec with raw encoding should return raw password",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"encoding":"raw"}}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					return "test_raw", nil
				},
			},
			want: map[string][]byte{
				"password": []byte(`test_raw`),
			},
			wantErr: false,
		},
		{
			name: "spec with base64 encoding should encode password as base64",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"encoding":"base64"}}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					return "test_base64", nil
				},
			},
			want: map[string][]byte{
				"password": []byte(base64.StdEncoding.EncodeToString([]byte("test_base64"))),
			},
			wantErr: false,
		},
		{
			// "Test>>Pass??word" exercises the URL-safe alphabet (- and _ in
			// place of + and /) and keeps "=" padding (Go's base64.URLEncoding).
			name: "spec with base64url encoding should encode password as url-safe base64 with padding",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"encoding":"base64url"}}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					return "Test>>Pass??word", nil
				},
			},
			want: map[string][]byte{
				"password": []byte(base64.URLEncoding.EncodeToString([]byte("Test>>Pass??word"))),
			},
			wantErr: false,
		},
		{
			name: "secretKeys overrides default output key",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"secretKeys":["custom"]}}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					return "custom-pwd", nil
				},
			},
			want: map[string][]byte{
				"custom": []byte(`custom-pwd`),
			},
			wantErr: false,
		},
		{
			name: "multiple secretKeys generate multiple passwords",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"secretKeys":["first","second"]}}`),
				},
				passGen: func() func(int, int, string, int, bool, bool) (string, error) {
					passwords := []string{"first-pass", "second-pass"}
					idx := 0
					return func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool) (string, error) {
						p := passwords[idx]
						idx++
						return p, nil
					}
				}(),
			},
			want: map[string][]byte{
				"first":  []byte(`first-pass`),
				"second": []byte(`second-pass`),
			},
			wantErr: false,
		},
		{
			name: "prefix is prepended to the password",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"prefix":"GK"}}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					assert.Equal(t, defaultLength, length)
					return "foobar", nil
				},
			},
			want: map[string][]byte{
				"password": []byte(`GKfoobar`),
			},
			wantErr: false,
		},
		{
			name: "suffix is appended to the password",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"suffix":"-v1"}}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					return "foobar", nil
				},
			},
			want: map[string][]byte{
				"password": []byte(`foobar-v1`),
			},
			wantErr: false,
		},
		{
			name: "prefix and suffix are not affected by noUpper",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"prefix":"PRE-","suffix":"-SUF","noUpper":true}}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					return "foobar", nil
				},
			},
			want: map[string][]byte{
				"password": []byte(`PRE-foobar-SUF`),
			},
			wantErr: false,
		},
		{
			name: "prefix and suffix are applied after hex encoding",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"encoding":"hex","prefix":"GK","suffix":"!"}}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					return "test_hex", nil
				},
			},
			want: map[string][]byte{
				"password": []byte("GK" + hex.EncodeToString([]byte("test_hex")) + "!"),
			},
			wantErr: false,
		},
		{
			name: "prefix is applied after base64 encoding",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"encoding":"base64","prefix":"GK"}}`),
				},
				passGen: func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					return "test_base64", nil
				},
			},
			want: map[string][]byte{
				"password": []byte("GK" + base64.StdEncoding.EncodeToString([]byte("test_base64"))),
			},
			wantErr: false,
		},
		{
			name: "prefix and suffix are applied to every secretKey",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"secretKeys":["first","second"],"prefix":"app-","suffix":"-v1"}}`),
				},
				passGen: func() func(int, int, string, int, bool, bool) (string, error) {
					passwords := []string{"first-pass", "second-pass"}
					idx := 0
					return func(length int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool) (string, error) {
						p := passwords[idx]
						idx++
						return p, nil
					}
				}(),
			},
			want: map[string][]byte{
				"first":  []byte(`app-first-pass-v1`),
				"second": []byte(`app-second-pass-v1`),
			},
			wantErr: false,
		},
		{
			name: "empty secretKeys entry should error",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"secretKeys":[""]}}`),
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate secretKeys entry should error",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"secretKeys":["dup","dup"]}}`),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got, _, err := g.generate(tt.args.jsonSpec, tt.args.passGen)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generator.Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Generator.Generate() = %v, want %v", got, tt.want)
			}
		})
	}
}
