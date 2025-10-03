/*
Copyright © 2025 ESO Maintainer Team

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

package utils

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/oracle/oci-go-sdk/v65/vault"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmetav1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	base64DecodedValue         string = "foo%_?bar"
	base64EncodedValue         string = "Zm9vJV8/YmFy"
	base64URLEncodedValue      string = "Zm9vJV8_YmFy"
	keyWithEmojis              string = "😀foo😁bar😂baz😈bing"
	keyWithInvalidChars        string = "some-array[0].entity"
	keyWithEncodedInvalidChars string = "some-array_U005b_0_U005d_.entity"
)

func TestObjectHash(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{
			name:  "A nil should be still working",
			input: nil,
			want:  "c461202a18e99215f121936fb2452e03843828e448a00a53f285a6fc",
		},
		{
			name:  "We accept a simple scalar value, i.e. string",
			input: "hello there",
			want:  "f78681ec611ebaeea0689bff6c7812a83ff98a7faba986d9af76c999",
		},
		{
			name: "A complex object like a secret is not an issue",
			input: v1.Secret{Data: map[string][]byte{
				"xx": []byte("yyy"),
			}},
			want: "9c717e13e4281db3cdad3f56c6e7faab1d7029c4b4fbbf12fbec9b1e",
		},
		{
			name: "map also works",
			input: map[string][]byte{
				"foo": []byte("value1"),
				"bar": []byte("value2"),
			},
			want: "1bed8bcbcb4547ffe19a19cd47d9078e84aa6598266d86b99f992d64",
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
		val  any
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
		strategy esv1.ExternalSecretConversionStrategy
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
				strategy: esv1.ExternalSecretConversionDefault,
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
				strategy: esv1.ExternalSecretConversionDefault,
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
				strategy: esv1.ExternalSecretConversionDefault,
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
				strategy: esv1.ExternalSecretConversionUnicode,
				in: map[string][]byte{
					keyWithEmojis: []byte(`noop`),
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

func TestReverseKeys(t *testing.T) {
	type args struct {
		encodingStrategy esv1.ExternalSecretConversionStrategy
		decodingStrategy esv1alpha1.PushSecretConversionStrategy
		in               map[string][]byte
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "encoding and decoding strategy are selecting Unicode conversion and reverse unicode, so the in and want should match, this test covers Unicode characters beyond the Basic Multilingual Plane (BMP)",
			args: args{
				encodingStrategy: esv1.ExternalSecretConversionUnicode,
				decodingStrategy: esv1alpha1.PushSecretConversionReverseUnicode,
				in: map[string][]byte{
					keyWithEmojis: []byte(`noop`),
				},
			},
			want: map[string][]byte{
				keyWithEmojis: []byte(`noop`),
			},
		},
		{
			name: "encoding and decoding strategy are selecting Unicode conversion and reverse unicode, so the in and want should match, this test covers Unicode characters in the Basic Multilingual Plane (BMP)",
			args: args{
				encodingStrategy: esv1.ExternalSecretConversionUnicode,
				decodingStrategy: esv1alpha1.PushSecretConversionReverseUnicode,
				in: map[string][]byte{
					keyWithInvalidChars: []byte(`noop`),
				},
			},
			want: map[string][]byte{
				keyWithInvalidChars: []byte(`noop`),
			},
		},
		{
			name: "the encoding strategy is selecting Unicode conversion, but the decoding strategy is none, so we want an encoded representation of the content",
			args: args{
				encodingStrategy: esv1.ExternalSecretConversionUnicode,
				decodingStrategy: esv1alpha1.PushSecretConversionNone,
				in: map[string][]byte{
					keyWithInvalidChars: []byte(`noop`),
				},
			},
			want: map[string][]byte{
				keyWithEncodedInvalidChars: []byte(`noop`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertKeys(tt.args.encodingStrategy, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, err = ReverseKeys(tt.args.decodingStrategy, got)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReverseKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReverseKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecode(t *testing.T) {
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
func TestValidate(t *testing.T) {
	err := NetworkValidate("http://google.com", 10*time.Second)
	if err != nil {
		t.Errorf("Connection problem: %v", err)
	}
}

func TestRewrite(t *testing.T) {
	type args struct {
		operations []esv1.ExternalSecretRewrite
		in         map[string][]byte
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "using double merge",
			args: args{
				operations: []esv1.ExternalSecretRewrite{
					{
						Merge: &esv1.ExternalSecretRewriteMerge{
							Strategy:       esv1.ExternalSecretRewriteMergeStrategyJSON,
							ConflictPolicy: esv1.ExternalSecretRewriteMergeConflictPolicyIgnore,
							Into:           "merged",
							Priority:       []string{"a"},
						},
					},
					{
						Merge: &esv1.ExternalSecretRewriteMerge{
							Strategy:       esv1.ExternalSecretRewriteMergeStrategyExtract,
							ConflictPolicy: esv1.ExternalSecretRewriteMergeConflictPolicyIgnore,
							Priority:       []string{"b"},
						},
					},
				},
				in: map[string][]byte{
					"a": []byte(`{"host": "dba", "pass": "yola", "port": 123}`),
					"b": []byte(`{"host": "dbb", "pass": "yolb"}`),
				},
			},
			want: map[string][]byte{
				"host": []byte("dbb"),
				"pass": []byte("yolb"),
				"port": []byte("123"),
			},
		},
		{
			name: "using regexp and merge",
			args: args{
				operations: []esv1.ExternalSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
							Source: "db/(.*)",
							Target: "$1",
						},
					},
					{
						Merge: &esv1.ExternalSecretRewriteMerge{
							Strategy:       esv1.ExternalSecretRewriteMergeStrategyJSON,
							ConflictPolicy: esv1.ExternalSecretRewriteMergeConflictPolicyIgnore,
							Into:           "merged",
							Priority:       []string{"a"},
						},
					},
				},
				in: map[string][]byte{
					"db/a": []byte(`{"host": "dba.example.com"}`),
					"db/b": []byte(`{"host": "dbb.example.com", "pass": "yolo"}`),
				},
			},
			want: map[string][]byte{
				"a":      []byte(`{"host": "dba.example.com"}`),
				"b":      []byte(`{"host": "dbb.example.com", "pass": "yolo"}`),
				"merged": []byte(`{"host":"dba.example.com","pass":"yolo"}`),
			},
		},
		{
			name: "replace of a single key",
			args: args{
				operations: []esv1.ExternalSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
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
				operations: []esv1.ExternalSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
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
				operations: []esv1.ExternalSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
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
				operations: []esv1.ExternalSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
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
				operations: []esv1.ExternalSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
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
				operations: []esv1.ExternalSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
							Source: "my/(.*?)/bar/(.*)",
							Target: "$1-$2",
						},
					},
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
							Source: "-",
							Target: "_",
						},
					},
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
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
		{
			name: "using transform rewrite operation to create env var format keys",
			args: args{
				operations: []esv1.ExternalSecretRewrite{
					{
						Regexp: &esv1.ExternalSecretRewriteRegexp{
							Source: "my/(.*?)/bar/(.*)",
							Target: "$1-$2",
						},
					},
					{
						Transform: &esv1.ExternalSecretRewriteTransform{
							Template: `{{ .value | upper | replace "-" "_" }}`,
						},
					},
				},
				in: map[string][]byte{
					"my/app/bar/key": []byte("bar"),
				},
			},
			want: map[string][]byte{
				"APP_KEY": []byte("bar"),
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

func TestRewriteMerge(t *testing.T) {
	type args struct {
		operation esv1.ExternalSecretRewriteMerge
		in        map[string][]byte
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "using empty merge",
			args: args{
				operation: esv1.ExternalSecretRewriteMerge{},
				in: map[string][]byte{
					"mongo-credentials": []byte(`{"username": "foz", "password": "baz"}`),
					"redis-credentials": []byte(`{"host": "redis.example.com", "port": "6379"}`),
				},
			},
			want: map[string][]byte{
				"username": []byte("foz"),
				"password": []byte("baz"),
				"host":     []byte("redis.example.com"),
				"port":     []byte("6379"),
			},
			wantErr: false,
		},
		{
			name: "using priority",
			args: args{
				operation: esv1.ExternalSecretRewriteMerge{
					ConflictPolicy: esv1.ExternalSecretRewriteMergeConflictPolicyIgnore,
					Priority:       []string{"mongo-credentials", "redis-credentials"},
				},
				in: map[string][]byte{
					"redis-credentials": []byte(`{"host": "redis.example.com", "port": "6379"}`),
					"mongo-credentials": []byte(`{"username": "foz", "password": "baz"}`),
					"other-credentials": []byte(`{"key": "value", "host": "other.example.com"}`),
				},
			},
			want: map[string][]byte{
				"username": []byte("foz"),
				"password": []byte("baz"),
				"host":     []byte("redis.example.com"),
				"port":     []byte("6379"),
				"key":      []byte("value"),
			},
			wantErr: false,
		},
		{
			name: "using priority with keys not in input (default strict)",
			args: args{
				operation: esv1.ExternalSecretRewriteMerge{
					ConflictPolicy: esv1.ExternalSecretRewriteMergeConflictPolicyIgnore,
					Priority:       []string{"non-existent-key", "another-missing-key", "mongo-credentials"},
				},
				in: map[string][]byte{
					"mongo-credentials": []byte(`{"username": "foz", "password": "baz"}`),
					"redis-credentials": []byte(`{"host": "redis.example.com", "port": "6379"}`),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "using priority with keys not in input and ignore policy",
			args: args{
				operation: esv1.ExternalSecretRewriteMerge{
					ConflictPolicy: esv1.ExternalSecretRewriteMergeConflictPolicyIgnore,
					Priority:       []string{"non-existent-key", "mongo-credentials"},
					PriorityPolicy: esv1.ExternalSecretRewriteMergePriorityPolicyIgnoreNotFound,
				},
				in: map[string][]byte{
					"mongo-credentials": []byte(`{"username": "foz", "password": "baz"}`),
					"redis-credentials": []byte(`{"host": "redis.example.com", "port": "6379"}`),
				},
			},
			want: map[string][]byte{
				"username": []byte("foz"),
				"password": []byte("baz"),
				"host":     []byte("redis.example.com"),
				"port":     []byte("6379"),
			},
			wantErr: false,
		},
		{
			name: "using conflict policy error",
			args: args{
				operation: esv1.ExternalSecretRewriteMerge{
					ConflictPolicy: esv1.ExternalSecretRewriteMergeConflictPolicyError,
				},
				in: map[string][]byte{
					"mongo-credentials": []byte(`{"username": "foz", "password": "baz"}`),
					"redis-credentials": []byte(`{"username": "redis", "port": "6379"}`),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "using JSON strategy",
			args: args{
				operation: esv1.ExternalSecretRewriteMerge{
					Strategy: esv1.ExternalSecretRewriteMergeStrategyJSON,
					Into:     "credentials",
				},
				in: map[string][]byte{
					"mongo-credentials": []byte(`{"username": "foz", "password": "baz"}`),
					"redis-credentials": []byte(`{"host": "redis.example.com", "port": "6379"}`),
				},
			},
			want: map[string][]byte{
				"mongo-credentials": []byte(`{"username": "foz", "password": "baz"}`),
				"redis-credentials": []byte(`{"host": "redis.example.com", "port": "6379"}`),
				"credentials": func() []byte {
					expected := map[string]interface{}{
						"username": "foz",
						"password": "baz",
						"host":     "redis.example.com",
						"port":     "6379",
					}
					b, _ := json.Marshal(expected)
					return b
				}(),
			},
			wantErr: false,
		},
		{
			name: "using JSON strategy without into",
			args: args{
				operation: esv1.ExternalSecretRewriteMerge{
					Strategy: esv1.ExternalSecretRewriteMergeStrategyJSON,
				},
				in: map[string][]byte{
					"mongo-credentials": []byte(`{"username": "foz", "password": "baz"}`),
					"redis-credentials": []byte(`{"host": "redis.example.com", "port": "6379"}`),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "with invalid JSON",
			args: args{
				operation: esv1.ExternalSecretRewriteMerge{},
				in: map[string][]byte{
					"invalid-json": []byte(`{"username": "foz", "password": "baz"`),
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RewriteMerge(tt.args.operation, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("RewriteMerge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RewriteMerge() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestReverse(t *testing.T) {
	type args struct {
		strategy esv1alpha1.PushSecretConversionStrategy
		in       string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "do not change the key when using the None strategy",
			args: args{
				strategy: esv1alpha1.PushSecretConversionNone,
				in:       keyWithEncodedInvalidChars,
			},
			want: keyWithEncodedInvalidChars,
		},
		{
			name: "reverse an unicode encoded key",
			args: args{
				strategy: esv1alpha1.PushSecretConversionReverseUnicode,
				in:       keyWithEncodedInvalidChars,
			},
			want: keyWithInvalidChars,
		},
		{
			name: "do not attempt to decode an invalid unicode representation",
			args: args{
				strategy: esv1alpha1.PushSecretConversionReverseUnicode,
				in:       "_U0xxx_x_U005b_",
			},
			want: "_U0xxx_x[",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reverse(tt.args.strategy, tt.args.in); got != tt.want {
				t.Errorf("reverse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchValueFromMetadata(t *testing.T) {
	type args struct {
		key  string
		data *apiextensionsv1.JSON
		def  any
	}
	type testCase struct {
		name    string
		args    args
		wantT   any
		wantErr bool
	}
	tests := []testCase{
		{
			name: "plain dig for an existing key",
			args: args{
				key: "key",
				data: &apiextensionsv1.JSON{
					Raw: []byte(
						`{"key": "value"}`,
					),
				},
				def: "def",
			},
			wantT:   "value",
			wantErr: false,
		},
		{
			name: "return default if key not found",
			args: args{
				key: "key2",
				data: &apiextensionsv1.JSON{
					Raw: []byte(
						`{"key": "value"}`,
					),
				},
				def: "def",
			},
			wantT:   "def",
			wantErr: false,
		},
		{
			name: "use a different type",
			args: args{
				key: "key",
				data: &apiextensionsv1.JSON{
					Raw: []byte(
						`{"key": 123}`,
					),
				},
				def: 1234,
			},
			wantT:   float64(123), // unmarshal is always float64
			wantErr: false,
		},
		{
			name: "digging deeper",
			args: args{
				key: "key2",
				data: &apiextensionsv1.JSON{
					Raw: []byte(
						`{"key": {"key2": "value"}}`,
					),
				},
				def: "",
			},
			wantT:   "value",
			wantErr: false,
		},
		{
			name: "digging for a slice",
			args: args{
				key: "topics",
				data: &apiextensionsv1.JSON{
					Raw: []byte(
						`{"topics": ["topic1", "topic2"]}`,
					),
				},
				def: []string{},
			},
			wantT:   []any{"topic1", "topic2"}, // we don't have deep type matching so it's not an []string{} but []any.
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotT, err := FetchValueFromMetadata(tt.args.key, tt.args.data, tt.args.def)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchValueFromMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.wantT, gotT)
		})
	}
}

func TestGetByteValue(t *testing.T) {
	type args struct {
		data any
	}
	type testCase struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}
	tests := []testCase{
		{
			name: "string",
			args: args{
				data: "value",
			},
			want:    []byte("value"),
			wantErr: false,
		},
		{
			name: "map of any",
			args: args{
				data: map[string]any{
					"key": "value",
				},
			},
			want:    []byte(`{"key":"value"}`),
			wantErr: false,
		},
		{
			name: "slice of string",
			args: args{
				data: []string{"value1", "value2"},
			},
			want:    []byte("value1\nvalue2"),
			wantErr: false,
		},
		{
			name: "json.RawMessage",
			args: args{
				data: json.RawMessage(`{"key":"value"}`),
			},
			want:    []byte(`{"key":"value"}`),
			wantErr: false,
		},
		{
			name: "float64",
			args: args{
				data: 123.45,
			},
			want:    []byte("123.45"),
			wantErr: false,
		},
		{
			name: "json.Number",
			args: args{
				data: json.Number("123.45"),
			},
			want:    []byte("123.45"),
			wantErr: false,
		},
		{
			name: "slice of any",
			args: args{
				data: []any{"value1", "value2"},
			},
			want:    []byte(`["value1","value2"]`),
			wantErr: false,
		},
		{
			name: "boolean",
			args: args{
				data: true,
			},
			want:    []byte("true"),
			wantErr: false,
		},
		{
			name: "nil",
			args: args{
				data: nil,
			},
			want:    []byte(nil),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetByteValue(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByteValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetByteValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareStringAndByteSlices(t *testing.T) {
	type args struct {
		stringValue    *string
		byteValueSlice []byte
	}
	type testCase struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}
	tests := []testCase{
		{
			name: "same contents",
			args: args{
				stringValue:    aws.String("value"),
				byteValueSlice: []byte("value"),
			},
			want:    true,
			wantErr: true,
		}, {
			name: "different contents",
			args: args{
				stringValue:    aws.String("value89"),
				byteValueSlice: []byte("value"),
			},
			want:    true,
			wantErr: false,
		}, {
			name: "same contents with random",
			args: args{
				stringValue:    aws.String("value89!3#@212"),
				byteValueSlice: []byte("value89!3#@212"),
			},
			want:    true,
			wantErr: true,
		}, {
			name: "check Nil",
			args: args{
				stringValue:    nil,
				byteValueSlice: []byte("value89!3#@212"),
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareStringAndByteSlices(tt.args.stringValue, tt.args.byteValueSlice)
			if got != tt.wantErr {
				t.Errorf("CompareStringAndByteSlices() got = %v, want = %v", got, tt.wantErr)
				return
			}
		})
	}
}

func TestValidateSecretSelector(t *testing.T) {
	tests := []struct {
		desc     string
		store    esv1.GenericStore
		ref      esmetav1.SecretKeySelector
		expected error
	}{
		{
			desc: "cluster secret store with namespace reference",
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.ClusterSecretStoreKind,
				},
			},
			ref: esmetav1.SecretKeySelector{
				Namespace: Ptr("test"),
			},
			expected: nil,
		},
		{
			desc: "secret store without namespace reference",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
			},
			ref:      esmetav1.SecretKeySelector{},
			expected: nil,
		},
		{
			desc: "secret store with the same namespace reference",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
				},
			},
			ref: esmetav1.SecretKeySelector{
				Namespace: Ptr("test"),
			},
			expected: nil,
		},
		{
			desc: "cluster secret store without namespace reference",
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.ClusterSecretStoreKind,
				},
			},
			ref:      esmetav1.SecretKeySelector{},
			expected: errRequireNamespace,
		},
		{
			desc: "secret store with the different namespace reference",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
				},
			},
			ref: esmetav1.SecretKeySelector{
				Namespace: Ptr("different"),
			},
			expected: errNamespaceNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := ValidateSecretSelector(tt.store, tt.ref)
			if !errors.Is(got, tt.expected) {
				t.Errorf("ValidateSecretSelector() got = %v, want = %v", got, tt.expected)
				return
			}
		})
	}
}

func TestValidateReferentSecretSelector(t *testing.T) {
	tests := []struct {
		desc     string
		store    esv1.GenericStore
		ref      esmetav1.SecretKeySelector
		expected error
	}{
		{
			desc: "cluster secret store with namespace reference",
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.ClusterSecretStoreKind,
				},
			},
			ref: esmetav1.SecretKeySelector{
				Namespace: Ptr("test"),
			},
			expected: nil,
		},
		{
			desc: "secret store without namespace reference",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
			},
			ref:      esmetav1.SecretKeySelector{},
			expected: nil,
		},
		{
			desc: "secret store with the same namespace reference",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
				},
			},
			ref: esmetav1.SecretKeySelector{
				Namespace: Ptr("test"),
			},
			expected: nil,
		},
		{
			desc: "secret store with the different namespace reference",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
				},
			},
			ref: esmetav1.SecretKeySelector{
				Namespace: Ptr("different"),
			},
			expected: errNamespaceNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := ValidateReferentSecretSelector(tt.store, tt.ref)
			if !errors.Is(got, tt.expected) {
				t.Errorf("ValidateReferentSecretSelector() got = %v, want = %v", got, tt.expected)
				return
			}
		})
	}
}

func TestValidateServiceAccountSelector(t *testing.T) {
	tests := []struct {
		desc     string
		store    esv1.GenericStore
		ref      esmetav1.ServiceAccountSelector
		expected error
	}{
		{
			desc: "cluster secret store with namespace reference",
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.ClusterSecretStoreKind,
				},
			},
			ref: esmetav1.ServiceAccountSelector{
				Namespace: Ptr("test"),
			},
			expected: nil,
		},
		{
			desc: "secret store without namespace reference",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
			},
			ref:      esmetav1.ServiceAccountSelector{},
			expected: nil,
		},
		{
			desc: "secret store with the same namespace reference",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
				},
			},
			ref: esmetav1.ServiceAccountSelector{
				Namespace: Ptr("test"),
			},
			expected: nil,
		},
		{
			desc: "cluster secret store without namespace reference",
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.ClusterSecretStoreKind,
				},
			},
			ref:      esmetav1.ServiceAccountSelector{},
			expected: errRequireNamespace,
		},
		{
			desc: "secret store with the different namespace reference",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
				},
			},
			ref: esmetav1.ServiceAccountSelector{
				Namespace: Ptr("different"),
			},
			expected: errNamespaceNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := ValidateServiceAccountSelector(tt.store, tt.ref)
			if !errors.Is(got, tt.expected) {
				t.Errorf("ValidateServiceAccountSelector() got = %v, want = %v", got, tt.expected)
				return
			}
		})
	}
}

func TestValidateReferentServiceAccountSelector(t *testing.T) {
	tests := []struct {
		desc     string
		store    esv1.GenericStore
		ref      esmetav1.ServiceAccountSelector
		expected error
	}{
		{
			desc: "cluster secret store with namespace reference",
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.ClusterSecretStoreKind,
				},
			},
			ref: esmetav1.ServiceAccountSelector{
				Namespace: Ptr("test"),
			},
			expected: nil,
		},
		{
			desc: "secret store without namespace reference",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
			},
			ref:      esmetav1.ServiceAccountSelector{},
			expected: nil,
		},
		{
			desc: "secret store with the same namespace reference",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
				},
			},
			ref: esmetav1.ServiceAccountSelector{
				Namespace: Ptr("test"),
			},
			expected: nil,
		},
		{
			desc: "secret store with the different namespace reference",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1.SecretStoreKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
				},
			},
			ref: esmetav1.ServiceAccountSelector{
				Namespace: Ptr("different"),
			},
			expected: errNamespaceNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := ValidateReferentServiceAccountSelector(tt.store, tt.ref)
			if !errors.Is(got, tt.expected) {
				t.Errorf("ValidateReferentServiceAccountSelector() got = %v, want = %v", got, tt.expected)
				return
			}
		})
	}
}

const mockJWTToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiZXhwIjoxNzAwMDAwMDAwfQ.signature"

func TestParseJWTClaims(t *testing.T) {
	// Mock JWT token with known payload
	mockToken := mockJWTToken

	claims, err := ParseJWTClaims(mockToken)
	if err != nil {
		t.Fatalf("Failed to get claims: %v", err)
	}

	if claims["sub"] != "1234567890" {
		t.Errorf("Expected sub claim to be '1234567890', got %v", claims["sub"])
	}
	if claims["name"] != "John Doe" {
		t.Errorf("Expected name claim to be 'John Doe', got %v", claims["name"])
	}
}

func TestExtractJWTExpiration(t *testing.T) {
	// Mock JWT token with known exp claim
	mockToken := mockJWTToken

	exp, err := ExtractJWTExpiration(mockToken)
	if err != nil {
		t.Fatalf("Failed to get token expiration: %v", err)
	}

	if exp != "1700000000" {
		t.Errorf("Expected expiration to be '1700000000', got %s", exp)
	}
}
