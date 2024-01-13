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

package gcr

import (
	"context"
	"reflect"
	"testing"
	"time"

	"golang.org/x/oauth2"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

func TestGenerate(t *testing.T) {
	type args struct {
		ctx             context.Context
		jsonSpec        *apiextensions.JSON
		kube            client.Client
		namespace       string
		fakeTokenSource tokenSourceFunc
	}
	tests := []struct {
		name    string
		g       *Generator
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "nil spec",
			args: args{
				jsonSpec: nil,
			},
			wantErr: true,
		},
		{
			name: "full spec",
			args: args{
				namespace: "foobar",
				kube: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example",
						Namespace: "foobar",
					},
					Data: map[string][]byte{
						"foo": []byte("bar"),
					},
				}).Build(),
				fakeTokenSource: func(ctx context.Context, auth v1beta1.GCPSMAuth, projectID string, storeKind string, kube client.Client, namespace string) (oauth2.TokenSource, error) {
					return oauth2.StaticTokenSource(&oauth2.Token{
						AccessToken: "1234",
						Expiry:      time.Unix(5555, 0),
					}), nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: GCRAccessToken
spec:
  projectID: "foobar"
  auth:
    secretRef:
      secretAccessKeySecretRef:
        name: "example"
        key: "foo"
`),
				},
			},
			want: map[string][]byte{
				"username": []byte(defaultLoginUsername),
				"password": []byte("1234"),
				"expiry":   []byte(`5555`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got, err := g.generate(
				tt.args.ctx,
				tt.args.jsonSpec,
				tt.args.kube,
				tt.args.namespace,
				tt.args.fakeTokenSource)
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
