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

package fake

import (
	"context"
	"reflect"
	"testing"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGenerate(t *testing.T) {
	type args struct {
		ctx       context.Context
		jsonSpec  *apiextensions.JSON
		kube      client.Client
		namespace string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "no spec",
			args: args{
				jsonSpec: nil,
			},
			wantErr: true,
		},
		{
			name: "invalid json",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(``),
				},
			},
			wantErr: true,
		},
		{
			name: "empty json produces empty map",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{}`),
				},
			},
			want:    make(map[string][]byte),
			wantErr: false,
		},
		{
			name: "spec with values produces valus",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`{"spec":{"data":{"foo":"bar","num":"42"}}}`),
				},
			},
			want: map[string][]byte{
				"foo": []byte(`bar`),
				"num": []byte(`42`),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got, err := g.Generate(tt.args.ctx, tt.args.jsonSpec, tt.args.kube, tt.args.namespace)
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
