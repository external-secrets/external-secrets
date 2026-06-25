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

package decoding

import (
	"reflect"
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	base64DecodedValue    string = "foo%_?bar"
	base64EncodedValue    string = "Zm9vJV8/YmFy"
	base64URLEncodedValue string = "Zm9vJV8_YmFy"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name     string
		strategy esv1.ExternalSecretDecodingStrategy
		in       []byte
		want     []byte
		wantErr  bool
	}{
		{
			name:     "base64 decoded",
			strategy: esv1.ExternalSecretDecodeBase64,
			in:       []byte("YmFy"),
			want:     []byte("bar"),
		},
		{
			name:     "invalid base64",
			strategy: esv1.ExternalSecretDecodeBase64,
			in:       []byte("foo"),
			wantErr:  true,
		},
		{
			name:     "base64url decoded",
			strategy: esv1.ExternalSecretDecodeBase64URL,
			in:       []byte(base64URLEncodedValue),
			want:     []byte(base64DecodedValue),
		},
		{
			name:     "invalid base64url",
			strategy: esv1.ExternalSecretDecodeBase64URL,
			in:       []byte("foo"),
			wantErr:  true,
		},
		{
			name:     "none",
			strategy: esv1.ExternalSecretDecodeNone,
			in:       []byte(base64URLEncodedValue),
			want:     []byte(base64URLEncodedValue),
		},
		{
			name:     "empty strategy defaults to none",
			strategy: "",
			in:       []byte(base64URLEncodedValue),
			want:     []byte(base64URLEncodedValue),
		},
		{
			name:     "auto base64",
			strategy: esv1.ExternalSecretDecodeAuto,
			in:       []byte(base64EncodedValue),
			want:     []byte(base64DecodedValue),
		},
		{
			name:     "auto base64url",
			strategy: esv1.ExternalSecretDecodeAuto,
			in:       []byte(base64URLEncodedValue),
			want:     []byte(base64DecodedValue),
		},
		{
			name:     "auto invalid base64 returns input",
			strategy: esv1.ExternalSecretDecodeAuto,
			in:       []byte("foo"),
			want:     []byte("foo"),
		},
		{
			name:     "unsupported strategy",
			strategy: esv1.ExternalSecretDecodingStrategy("unsupported"),
			in:       []byte("foo"),
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.strategy, tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecodeMap(t *testing.T) {
	type args struct {
		strategy esv1.ExternalSecretDecodingStrategy
		in       map[string][]byte
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "base64 decoded",
			args: args{
				strategy: esv1.ExternalSecretDecodeBase64,
				in: map[string][]byte{
					"foo": []byte("YmFy"),
				},
			},
			want: map[string][]byte{
				"foo": []byte("bar"),
			},
		},
		{
			name: "invalid base64",
			args: args{
				strategy: esv1.ExternalSecretDecodeBase64,
				in: map[string][]byte{
					"foo": []byte("foo"),
				},
			},
			wantErr: true,
		},
		{
			name: "base64url decoded",
			args: args{
				strategy: esv1.ExternalSecretDecodeBase64URL,
				in: map[string][]byte{
					"foo": []byte(base64URLEncodedValue),
				},
			},
			want: map[string][]byte{
				"foo": []byte(base64DecodedValue),
			},
		},
		{
			name: "invalid base64url",
			args: args{
				strategy: esv1.ExternalSecretDecodeBase64URL,
				in: map[string][]byte{
					"foo": []byte("foo"),
				},
			},
			wantErr: true,
		},
		{
			name: "none",
			args: args{
				strategy: esv1.ExternalSecretDecodeNone,
				in: map[string][]byte{
					"foo": []byte(base64URLEncodedValue),
				},
			},
			want: map[string][]byte{
				"foo": []byte(base64URLEncodedValue),
			},
		},
		{
			name: "auto",
			args: args{
				strategy: esv1.ExternalSecretDecodeAuto,
				in: map[string][]byte{
					"b64":        []byte(base64EncodedValue),
					"invalidb64": []byte("foo"),
					"b64url":     []byte(base64URLEncodedValue),
				},
			},
			want: map[string][]byte{
				"b64":        []byte(base64DecodedValue),
				"invalidb64": []byte("foo"),
				"b64url":     []byte(base64DecodedValue),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeMap(tt.args.strategy, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DecodeMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
