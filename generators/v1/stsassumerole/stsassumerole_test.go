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
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets/runtime/esutils"
)

func TestGenerate(t *testing.T) {
	type args struct {
		ctx        context.Context
		jsonSpec   *apiextensions.JSON
		kube       client.Client
		namespace  string
		assumeRole func(context.Context, *sts.AssumeRoleInput, ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
	}
	tests := []struct {
		name    string
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
				assumeRole: func(ctx context.Context, input *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
					return nil, errors.New("boom")
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(``),
				},
			},
			wantErr: true,
		},
		{
			name: "with role: calls AssumeRole and returns credentials",
			args: args{
				ctx:       context.Background(),
				namespace: "foobar",
				kube: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-aws-creds",
						Namespace: "foobar",
					},
					Data: map[string][]byte{
						"key-id":        []byte("AKIAIOSFODNN7EXAMPLE"),
						"access-secret": []byte("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
					},
				}).Build(),
				assumeRole: func(ctx context.Context, input *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
					exp := time.Unix(9999, 0)
					return &sts.AssumeRoleOutput{
						Credentials: &ststypes.Credentials{
							AccessKeyId:     esutils.Ptr("assumed-access-key-id"),
							SecretAccessKey: esutils.Ptr("assumed-secret-access-key"),
							SessionToken:    esutils.Ptr("assumed-session-token"),
							Expiration:      esutils.Ptr(exp),
						},
					}, nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: STSAssumeRoleToken
spec:
  region: eu-west-1
  role: "arn:aws:iam::123456789012:role/my-role"
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
				"access_key_id":     []byte("assumed-access-key-id"),
				"secret_access_key": []byte("assumed-secret-access-key"),
				"session_token":     []byte("assumed-session-token"),
				"expiration":        []byte("9999"),
			},
		},
		{
			name: "with role and roleAssumptionParameters: passes duration and externalId",
			args: args{
				ctx:       context.Background(),
				namespace: "foobar",
				kube: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-aws-creds",
						Namespace: "foobar",
					},
					Data: map[string][]byte{
						"key-id":        []byte("AKIAIOSFODNN7EXAMPLE"),
						"access-secret": []byte("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
					},
				}).Build(),
				assumeRole: func(ctx context.Context, input *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
					exp := time.Unix(7777, 0)
					return &sts.AssumeRoleOutput{
						Credentials: &ststypes.Credentials{
							AccessKeyId:     esutils.Ptr("key-with-params"),
							SecretAccessKey: esutils.Ptr("secret-with-params"),
							SessionToken:    esutils.Ptr("token-with-params"),
							Expiration:      esutils.Ptr(exp),
						},
					}, nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: STSAssumeRoleToken
spec:
  region: eu-west-1
  role: "arn:aws:iam::123456789012:role/my-role"
  roleAssumptionParameters:
    sessionDuration: 1800
    externalId: "unique-id-for-external-trust"
    roleSessionName: "my-session"
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
				"access_key_id":     []byte("key-with-params"),
				"secret_access_key": []byte("secret-with-params"),
				"session_token":     []byte("token-with-params"),
				"expiration":        []byte("7777"),
			},
		},
		{
			name: "AssumeRole API error propagated",
			args: args{
				ctx:       context.Background(),
				namespace: "foobar",
				kube: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-aws-creds",
						Namespace: "foobar",
					},
					Data: map[string][]byte{
						"key-id":        []byte("AKIAIOSFODNN7EXAMPLE"),
						"access-secret": []byte("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
					},
				}).Build(),
				assumeRole: func(ctx context.Context, input *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
					return nil, errors.New("AccessDenied: not authorized to assume role")
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: STSAssumeRoleToken
spec:
  region: eu-west-1
  role: "arn:aws:iam::123456789012:role/my-role"
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
			wantErr: true,
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
				func(cfg *aws.Config) stsAssumeRoleAPI {
					return &fakeAssumeRole{
						assumeRole: tt.args.assumeRole,
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

type fakeAssumeRole struct {
	assumeRole func(context.Context, *sts.AssumeRoleInput, ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

func (f *fakeAssumeRole) AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	return f.assumeRole(ctx, params, optFns...)
}
