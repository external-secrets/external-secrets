// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Copyright External Secrets Inc. All Rights Reserved

package basic_auth

import (
	"errors"
	"reflect"
	"testing"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/stretchr/testify/assert"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestGenerate(t *testing.T) {
	type args struct {
		jsonSpec *apiextensions.JSON
		userGen  usernameGenerateFunc
		passGen  passwordGenerateFunc
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
				userGen: func(len int, prefix string, sufix string, wordCount int, separator string, includeNumbers bool,
				) (string, error) {
					assert.Equal(t, defaultUsernameLength, len)
					assert.Equal(t, "", prefix)
					assert.Equal(t, "", sufix)
					assert.Equal(t, 1, wordCount)
					assert.Equal(t, defaultSeparator, separator)
					assert.Equal(t, false, includeNumbers)
					return "foo", nil
				},
				passGen: func(passSpec genv1alpha1.PasswordSpec) ([]byte, error) {
					return []byte("bar"), nil
				},
			},
			want: map[string][]byte{
				"username": []byte(`foo`),
				"password": []byte(`bar`),
			},
			wantErr: false,
		},
		{
			name: "spec should override defaults",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{
						"spec": {
							"username": {
								"length": 12,
								"prefix": "dev-",
								"sufix": "-svc",
								"wordCount": 2,
								"separator": ".",
								"includeNumbers": true
							},
							"password": {
								"length": 48,
								"digits": 2,
								"symbols": 2,
								"symbolCharacters": "-_.",
								"noUpper": true,
								"allowRepeat": true
							}
						}
					}`),
				},
				userGen: func(len int, prefix, sufix string, wordCount int, separator string, includeNumbers bool) (string, error) {
					assert.Equal(t, 12, len)
					assert.Equal(t, "dev-", prefix)
					assert.Equal(t, "-svc", sufix)
					assert.Equal(t, 2, wordCount)
					assert.Equal(t, ".", separator)
					assert.True(t, includeNumbers)
					return "dev-word1.word2-svc4254", nil
				},
				passGen: func(passSpec genv1alpha1.PasswordSpec) ([]byte, error) {
					assert.Equal(t, 48, passSpec.Length)
					assert.Equal(t, "-_.", *passSpec.SymbolCharacters)
					assert.Equal(t, 2, *passSpec.Symbols)
					assert.Equal(t, 2, *passSpec.Digits)
					assert.True(t, passSpec.NoUpper)
					assert.True(t, passSpec.AllowRepeat)
					return []byte("securePassword123!_"), nil
				},
			},
			want: map[string][]byte{
				"username": []byte("dev-word1.word2-svc4254"),
				"password": []byte("securePassword123!_"),
			},
			wantErr: false,
		},
		{
			name: "generator error should be returned",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{}`),
				},
				userGen: func(len int, prefix string, sufix string, wordCount int, separator string, includeNumbers bool,
				) (string, error) {
					return "", errors.New("boom")
				},
				passGen: func(passSpec genv1alpha1.PasswordSpec) ([]byte, error) {
					return nil, errors.New("boom")
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got, _, err := g.generate(tt.args.jsonSpec, tt.args.userGen, tt.args.passGen)
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
