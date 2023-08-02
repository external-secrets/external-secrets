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

package ecr

import (
	"context"
	"encoding/base64"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecr/ecriface"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenerate(t *testing.T) {
	type args struct {
		ctx           context.Context
		jsonSpec      *apiextensions.JSON
		kube          client.Client
		namespace     string
		authTokenFunc func(*ecr.GetAuthorizationTokenInput) (*ecr.GetAuthorizationTokenOutput, error)
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
			name: "invalid json",
			args: args{
				authTokenFunc: func(gati *ecr.GetAuthorizationTokenInput) (*ecr.GetAuthorizationTokenOutput, error) {
					return nil, errors.New("boom")
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(``),
				},
			},
			wantErr: true,
		},
		{
			name: "full spec",
			args: args{
				namespace: "foobar",
				kube: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-aws-creds",
						Namespace: "foobar",
					},
					Data: map[string][]byte{
						"key-id":        []byte("foo"),
						"access-secret": []byte("bar"),
					},
				}).Build(),
				authTokenFunc: func(in *ecr.GetAuthorizationTokenInput) (*ecr.GetAuthorizationTokenOutput, error) {
					t := time.Unix(1234, 0)
					return &ecr.GetAuthorizationTokenOutput{
						AuthorizationData: []*ecr.AuthorizationData{
							{
								AuthorizationToken: utilpointer.To(base64.StdEncoding.EncodeToString([]byte("uuser:pass"))),
								ProxyEndpoint:      utilpointer.To("foo"),
								ExpiresAt:          &t,
							},
						},
					}, nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: ECRAuthorizationToken
spec:
  region: eu-west-1
  role: "my-role"
  auth:
    secretRef:
      accessKeyIDSecretRef:
        name: "my-aws-creds"
        key: "key-id"
      secretAccessKeySecretRef:
        name: "my-aws-creds"
        key: "access-secret"`),
				},
			},
			want: map[string][]byte{
				"username":       []byte("uuser"),
				"password":       []byte("pass"),
				"proxy_endpoint": []byte("foo"),
				"expires_at":     []byte("1234"),
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
				func(aws *session.Session) ecriface.ECRAPI {
					return &FakeECR{
						authTokenFunc: tt.args.authTokenFunc,
					}
				},
			)
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

type FakeECR struct {
	ecriface.ECRAPI
	authTokenFunc func(*ecr.GetAuthorizationTokenInput) (*ecr.GetAuthorizationTokenOutput, error)
}

func (e *FakeECR) GetAuthorizationToken(in *ecr.GetAuthorizationTokenInput) (*ecr.GetAuthorizationTokenOutput, error) {
	return e.authTokenFunc(in)
}
