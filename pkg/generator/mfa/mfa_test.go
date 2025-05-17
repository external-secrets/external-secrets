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

package mfa

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
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
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "namespace",
					},
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
