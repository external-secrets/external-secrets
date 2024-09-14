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

package password

import (
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
				passGen: func(len int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					assert.Equal(t, defaultLength, len)
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
				passGen: func(len int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					assert.Equal(t, 48, len)
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
				passGen: func(len int, symbols int, symbolCharacters string, digits int, noUpper bool, allowRepeat bool,
				) (string, error) {
					return "", errors.New("boom")
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got, err := g.generate(tt.args.jsonSpec, tt.args.passGen)
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
