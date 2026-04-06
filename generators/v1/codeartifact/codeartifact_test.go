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

package codeartifact

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codeartifact"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenerate(t *testing.T) {
	type args struct {
		ctx           context.Context
		jsonSpec      *apiextensions.JSON
		kube          client.Client
		namespace     string
		authTokenFunc func(*codeartifact.GetAuthorizationTokenInput) (*codeartifact.GetAuthorizationTokenOutput, error)
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
				ctx:      context.Background(),
				jsonSpec: nil,
			},
			wantErr: true,
		},
		{
			name: "invalid json",
			args: args{
				ctx: context.Background(),
				authTokenFunc: func(in *codeartifact.GetAuthorizationTokenInput) (*codeartifact.GetAuthorizationTokenOutput, error) {
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
				ctx:       context.Background(),
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
				authTokenFunc: func(in *codeartifact.GetAuthorizationTokenInput) (*codeartifact.GetAuthorizationTokenOutput, error) {
					expiry := time.Unix(1234, 0)
					return &codeartifact.GetAuthorizationTokenOutput{
						AuthorizationToken: aws.String("my-secret-token"),
						Expiration:         &expiry,
					}, nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: CodeArtifactAuthorizationToken
spec:
  region: us-east-1
  role: "my-role"
  domain: "my-domain"
  domainOwner: "123456789012"
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
				"authorizationToken": []byte("my-secret-token"),
				"expiration":         []byte("1234"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got, _, err := g.generate(
				tt.args.ctx,
				tt.args.jsonSpec,
				tt.args.kube,
				tt.args.namespace,
				func(cfg *aws.Config) codeArtifactAPI {
					return &FakeCodeArtifact{
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

type FakeCodeArtifact struct {
	authTokenFunc func(*codeartifact.GetAuthorizationTokenInput) (*codeartifact.GetAuthorizationTokenOutput, error)
}

func (f *FakeCodeArtifact) GetAuthorizationToken(
	ctx context.Context,
	params *codeartifact.GetAuthorizationTokenInput,
	optFns ...func(*codeartifact.Options),
) (*codeartifact.GetAuthorizationTokenOutput, error) {
	return f.authTokenFunc(params)
}
