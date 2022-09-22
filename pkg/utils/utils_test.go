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

package utils

import (
	"reflect"
	"testing"
	"time"

	vault "github.com/oracle/oci-go-sdk/v56/vault"
	v1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	base64DecodedValue    string = "foo%_?bar"
	base64EncodedValue    string = "Zm9vJV8/YmFy"
	base64URLEncodedValue string = "Zm9vJV8_YmFy"
)

func TestObjectHash(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{
			name:  "A nil should be still working",
			input: nil,
			want:  "60046f14c917c18a9a0f923e191ba0dc",
		},
		{
			name:  "We accept a simple scalar value, i.e. string",
			input: "hello there",
			want:  "161bc25962da8fed6d2f59922fb642aa",
		},
		{
			name: "A complex object like a secret is not an issue",
			input: v1.Secret{Data: map[string][]byte{
				"xx": []byte("yyy"),
			}},
			want: "7b6fb27a17a8e4a4be8fda6cb499f3d5",
		},
		{
			name: "map also works",
			input: map[string][]byte{
				"foo": []byte("value1"),
				"bar": []byte("value2"),
			},
			want: "caa0155759a6a9b3b6ada5a6883ee2bb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ObjectHash(tt.input); got != tt.want {
				t.Errorf("ObjectHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNil(t *testing.T) {
	tbl := []struct {
		name string
		val  interface{}
		exp  bool
	}{
		{
			name: "simple nil val",
			val:  nil,
			exp:  true,
		},
		{
			name: "nil slice",
			val:  (*[]struct{})(nil),
			exp:  true,
		},
		{
			name: "struct pointer",
			val:  &testing.T{},
			exp:  false,
		},
		{
			name: "struct",
			val:  testing.T{},
			exp:  false,
		},
		{
			name: "slice of struct",
			val:  []struct{}{{}},
			exp:  false,
		},
		{
			name: "slice of ptr",
			val:  []*testing.T{nil},
			exp:  false,
		},
		{
			name: "slice",
			val:  []struct{}(nil),
			exp:  false,
		},
		{
			name: "int default value",
			val:  0,
			exp:  false,
		},
		{
			name: "empty str",
			val:  "",
			exp:  false,
		},
		{
			name: "oracle vault",
			val:  vault.VaultsClient{},
			exp:  false,
		},
		{
			name: "func",
			val: func() {
				// noop for testing and to make linter happy
			},
			exp: false,
		},
		{
			name: "channel",
			val:  make(chan struct{}),
			exp:  false,
		},
		{
			name: "map",
			val:  map[string]string{},
			exp:  false,
		},
	}

	for _, row := range tbl {
		t.Run(row.name, func(t *testing.T) {
			res := IsNil(row.val)
			if res != row.exp {
				t.Errorf("IsNil(%#v)=%t, expected %t", row.val, res, row.exp)
			}
		})
	}
}

func TestConvertKeys(t *testing.T) {
	type args struct {
		strategy esv1beta1.ExternalSecretConversionStrategy
		in       map[string][]byte
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "convert with special chars",
			args: args{
				strategy: esv1beta1.ExternalSecretConversionDefault,
				in: map[string][]byte{
					"foo$bar%baz*bing": []byte(`noop`),
				},
			},
			want: map[string][]byte{
				"foo_bar_baz_bing": []byte(`noop`),
			},
		},
		{
			name: "error on collision",
			args: args{
				strategy: esv1beta1.ExternalSecretConversionDefault,
				in: map[string][]byte{
					"foo$bar%baz*bing": []byte(`noop`),
					"foo_bar_baz$bing": []byte(`noop`),
				},
			},
			wantErr: true,
		},
		{
			name: "convert path",
			args: args{
				strategy: esv1beta1.ExternalSecretConversionDefault,
				in: map[string][]byte{
					"/foo/bar/baz/bing": []byte(`noop`),
					"foo/bar/baz/bing/": []byte(`noop`),
				},
			},
			want: map[string][]byte{
				"_foo_bar_baz_bing": []byte(`noop`),
				"foo_bar_baz_bing_": []byte(`noop`),
			},
		},
		{
			name: "convert unicode",
			args: args{
				strategy: esv1beta1.ExternalSecretConversionUnicode,
				in: map[string][]byte{
					"😀foo😁bar😂baz😈bing": []byte(`noop`),
				},
			},
			want: map[string][]byte{
				"_U1f600_foo_U1f601_bar_U1f602_baz_U1f608_bing": []byte(`noop`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertKeys(tt.args.strategy, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	type args struct {
		strategy esv1beta1.ExternalSecretDecodingStrategy
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
				strategy: esv1beta1.ExternalSecretDecodeBase64,
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
				strategy: esv1beta1.ExternalSecretDecodeBase64,
				in: map[string][]byte{
					"foo": []byte("foo"),
				},
			},
			wantErr: true,
		},
		{
			name: "base64url decoded",
			args: args{
				strategy: esv1beta1.ExternalSecretDecodeBase64URL,
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
				strategy: esv1beta1.ExternalSecretDecodeBase64URL,
				in: map[string][]byte{
					"foo": []byte("foo"),
				},
			},
			wantErr: true,
		},
		{
			name: "none",
			args: args{
				strategy: esv1beta1.ExternalSecretDecodeNone,
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
				strategy: esv1beta1.ExternalSecretDecodeAuto,
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
func TestValidate(t *testing.T) {
	err := NetworkValidate("http://google.com", 10*time.Second)
	if err != nil {
		t.Errorf("Connection problem: %v", err)
	}
}

func TestRewriteRegexp(t *testing.T) {
	type args struct {
		operations []esv1beta1.ExternalSecretRewrite
		in         map[string][]byte
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "replace of a single key",
			args: args{
				operations: []esv1beta1.ExternalSecretRewrite{
					{
						Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
							Source: "-",
							Target: "_",
						},
					},
				},
				in: map[string][]byte{
					"foo-bar": []byte("bar"),
				},
			},
			want: map[string][]byte{
				"foo_bar": []byte("bar"),
			},
		},
		{
			name: "no operation",
			args: args{
				operations: []esv1beta1.ExternalSecretRewrite{
					{
						Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
							Source: "hello",
							Target: "world",
						},
					},
				},
				in: map[string][]byte{
					"foo": []byte("bar"),
				},
			},
			want: map[string][]byte{
				"foo": []byte("bar"),
			},
		},
		{
			name: "removing prefix from keys",
			args: args{
				operations: []esv1beta1.ExternalSecretRewrite{
					{
						Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
							Source: "^my/initial/path/",
							Target: "",
						},
					},
				},
				in: map[string][]byte{
					"my/initial/path/foo": []byte("bar"),
				},
			},
			want: map[string][]byte{
				"foo": []byte("bar"),
			},
		},
		{
			name: "using un-named capture groups",
			args: args{
				operations: []esv1beta1.ExternalSecretRewrite{
					{
						Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
							Source: "f(.*)o",
							Target: "a_new_path_$1",
						},
					},
				},
				in: map[string][]byte{
					"foo":      []byte("bar"),
					"foodaloo": []byte("barr"),
				},
			},
			want: map[string][]byte{
				"a_new_path_o":      []byte("bar"),
				"a_new_path_oodalo": []byte("barr"),
			},
		},
		{
			name: "using named and numbered capture groups",
			args: args{
				operations: []esv1beta1.ExternalSecretRewrite{
					{
						Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
							Source: "f(?P<content>.*)o",
							Target: "a_new_path_${content}_${1}",
						},
					},
				},
				in: map[string][]byte{
					"foo":  []byte("bar"),
					"floo": []byte("barr"),
				},
			},
			want: map[string][]byte{
				"a_new_path_o_o":   []byte("bar"),
				"a_new_path_lo_lo": []byte("barr"),
			},
		},
		{
			name: "using sequenced rewrite operations",
			args: args{
				operations: []esv1beta1.ExternalSecretRewrite{
					{
						Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
							Source: "my/(.*?)/bar/(.*)",
							Target: "$1-$2",
						},
					},
					{
						Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
							Source: "-",
							Target: "_",
						},
					},
					{
						Regexp: &esv1beta1.ExternalSecretRewriteRegexp{
							Source: "ass",
							Target: "***",
						},
					},
				},
				in: map[string][]byte{
					"my/app/bar/key":      []byte("bar"),
					"my/app/bar/password": []byte("barr"),
				},
			},
			want: map[string][]byte{
				"app_key":      []byte("bar"),
				"app_p***word": []byte("barr"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RewriteMap(tt.args.operations, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("RewriteMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RewriteMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
