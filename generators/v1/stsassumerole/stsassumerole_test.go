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

package stsassumerole

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeCredsProvider struct {
	creds aws.Credentials
	err   error
}

func (f *fakeCredsProvider) Retrieve(_ context.Context) (aws.Credentials, error) {
	return f.creds, f.err
}

func makeFactory(creds aws.Credentials, err error) credsProviderFactory {
	return func(_ *aws.Config, _ string, _ ...func(*stscreds.AssumeRoleOptions)) aws.CredentialsProvider {
		return &fakeCredsProvider{creds: creds, err: err}
	}
}

func TestGenerate(t *testing.T) {
	exp := time.Unix(9999, 0).UTC()

	type args struct {
		ctx       context.Context
		jsonSpec  *apiextensions.JSON
		kube      client.Client
		namespace string
		factory   credsProviderFactory
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "nil spec returns error",
			args: args{
				ctx:      context.Background(),
				jsonSpec: nil,
			},
			wantErr: true,
		},
		{
			name: "invalid yaml returns error",
			args: args{
				ctx:      context.Background(),
				jsonSpec: &apiextensions.JSON{Raw: []byte(`{invalid`)},
				factory:  makeFactory(aws.Credentials{}, nil),
			},
			wantErr: true,
		},
		{
			name: "missing role returns error",
			args: args{
				ctx:       context.Background(),
				namespace: "ns",
				kube:      clientfake.NewClientBuilder().Build(),
				factory:   makeFactory(aws.Credentials{}, nil),
				jsonSpec: &apiextensions.JSON{Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: STSAssumeRoleToken
spec:
  region: us-east-1`)},
			},
			wantErr: true,
		},
		{
			name: "assume role error is propagated",
			args: args{
				ctx:       context.Background(),
				namespace: "ns",
				kube:      clientfake.NewClientBuilder().Build(),
				factory: func(_ *aws.Config, _ string, _ ...func(*stscreds.AssumeRoleOptions)) aws.CredentialsProvider {
					return &fakeCredsProvider{err: errors.New("access denied")}
				},
				jsonSpec: &apiextensions.JSON{Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: STSAssumeRoleToken
spec:
  region: us-east-1
  role: arn:aws:iam::123456789012:role/my-role`)},
			},
			wantErr: true,
		},
		{
			name: "happy path with secretRef and role",
			args: args{
				ctx:       context.Background(),
				namespace: "testns",
				kube: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws-creds",
						Namespace: "testns",
					},
					Data: map[string][]byte{
						"access-key-id":     []byte("AKIAIOSFODNN7EXAMPLE"),
						"secret-access-key": []byte("wJalrXUtnFEMI/K7MDENG"),
					},
				}).Build(),
				factory: makeFactory(aws.Credentials{
					AccessKeyID:     "ASSUMED-ACCESS-KEY",
					SecretAccessKey: "ASSUMED-SECRET",
					SessionToken:    "ASSUMED-SESSION-TOKEN",
					Expires:         exp,
				}, nil),
				jsonSpec: &apiextensions.JSON{Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: STSAssumeRoleToken
spec:
  region: eu-west-1
  role: arn:aws:iam::123456789012:role/my-role
  auth:
    secretRef:
      accessKeyIDSecretRef:
        name: aws-creds
        key: access-key-id
      secretAccessKeySecretRef:
        name: aws-creds
        key: secret-access-key`)},
			},
			want: map[string][]byte{
				"access_key_id":     []byte("ASSUMED-ACCESS-KEY"),
				"secret_access_key": []byte("ASSUMED-SECRET"),
				"session_token":     []byte("ASSUMED-SESSION-TOKEN"),
				"expiration":        []byte("9999"),
			},
		},
		{
			name: "happy path with requestParameters",
			args: args{
				ctx:       context.Background(),
				namespace: "testns",
				kube:      clientfake.NewClientBuilder().Build(),
				factory: makeFactory(aws.Credentials{
					AccessKeyID:     "KEY",
					SecretAccessKey: "SECRET",
					SessionToken:    "TOKEN",
					Expires:         exp,
				}, nil),
				jsonSpec: &apiextensions.JSON{Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: STSAssumeRoleToken
spec:
  region: us-east-1
  role: arn:aws:iam::123456789012:role/cross-account
  requestParameters:
    sessionDuration: 7200
    externalID: my-external-id`)},
			},
			want: map[string][]byte{
				"access_key_id":     []byte("KEY"),
				"secret_access_key": []byte("SECRET"),
				"session_token":     []byte("TOKEN"),
				"expiration":        []byte("9999"),
			},
		},
		{
			name: "no expiration when expires is zero",
			args: args{
				ctx:       context.Background(),
				namespace: "testns",
				kube:      clientfake.NewClientBuilder().Build(),
				factory: makeFactory(aws.Credentials{
					AccessKeyID:     "KEY",
					SecretAccessKey: "SECRET",
					SessionToken:    "TOKEN",
				}, nil),
				jsonSpec: &apiextensions.JSON{Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: STSAssumeRoleToken
spec:
  region: us-east-1
  role: arn:aws:iam::123456789012:role/my-role`)},
			},
			want: map[string][]byte{
				"access_key_id":     []byte("KEY"),
				"secret_access_key": []byte("SECRET"),
				"session_token":     []byte("TOKEN"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got, _, err := g.generate(tt.args.ctx, tt.args.jsonSpec, tt.args.kube, tt.args.namespace, tt.args.factory)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generator.generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Generator.generate() = %v, want %v", got, tt.want)
			}
		})
	}
}
