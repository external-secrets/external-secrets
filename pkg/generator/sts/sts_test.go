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

package sts

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets/pkg/utils"
)

func TestGenerate(t *testing.T) {
	type args struct {
		ctx       context.Context
		jsonSpec  *apiextensions.JSON
		kube      client.Client
		namespace string
		tokenFunc func(*sts.GetSessionTokenInput) (*sts.GetSessionTokenOutput, error)
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
				tokenFunc: func(*sts.GetSessionTokenInput) (*sts.GetSessionTokenOutput, error) {
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
				tokenFunc: func(*sts.GetSessionTokenInput) (*sts.GetSessionTokenOutput, error) {
					t := time.Unix(1234, 0)
					return &sts.GetSessionTokenOutput{
						Credentials: &sts.Credentials{
							AccessKeyId:     utils.Ptr("access-key-id"),
							Expiration:      utils.Ptr(t),
							SecretAccessKey: utils.Ptr("secret-access-key"),
							SessionToken:    utils.Ptr("session-token"),
						},
					}, nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: STSSessionToken
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
				"access_key_id":     []byte("access-key-id"),
				"expiration":        []byte("1234"),
				"secret_access_key": []byte("secret-access-key"),
				"session_token":     []byte("session-token"),
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
				func(aws *session.Session) stsiface.STSAPI {
					return &FakeSTS{
						getSessionToken: tt.args.tokenFunc,
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

type FakeSTS struct {
	stsiface.STSAPI
	getSessionToken func(*sts.GetSessionTokenInput) (*sts.GetSessionTokenOutput, error)
}

func (e *FakeSTS) GetSessionToken(in *sts.GetSessionTokenInput) (*sts.GetSessionTokenOutput, error) {
	return e.getSessionToken(in)
}
