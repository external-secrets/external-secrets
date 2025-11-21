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
package awsiam

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenerate(t *testing.T) {
	type args struct {
		jsonSpec      *apiextensions.JSON
		kube          client.Client
		namespace     string
		createKeyFunc func(ctx context.Context, in *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
		listKeyFunc   func(ctx context.Context, in *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error)
		deleteKeyfunc func(ctx context.Context, in *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error)
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
				listKeyFunc: func(ctx context.Context, in *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
					return nil, errors.New("boom")
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(``),
				},
			},
			wantErr: true,
		},
		{
			name: "create no delete",
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
				listKeyFunc: func(ctx context.Context, in *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
					return &iam.ListAccessKeysOutput{
						AccessKeyMetadata: []types.AccessKeyMetadata{},
					}, nil
				},
				createKeyFunc: func(ctx context.Context, in *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
					t := time.Unix(1234, 0)
					return &iam.CreateAccessKeyOutput{
						AccessKey: &types.AccessKey{
							AccessKeyId:     utilpointer.To("uuser"),
							SecretAccessKey: utilpointer.To("pass"),
							CreateDate:      &t,
						},
					}, nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: AWSIAMKey
spec:
  region: eu-west-1
  role: "my-role"
  iamRef:
    username: foo
    maxKeys: 1
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
				"access_key_id":     []byte("uuser"),
				"secret_access_key": []byte("pass"),
			},
		},
		{
			name: "delete all create one",
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
				listKeyFunc: func(ctx context.Context, in *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
					return &iam.ListAccessKeysOutput{
						AccessKeyMetadata: []types.AccessKeyMetadata{
							{
								AccessKeyId: utilpointer.To("dead"),
								CreateDate:  utilpointer.To(time.Unix(1234, 0)),
							},
							{
								AccessKeyId: utilpointer.To("beef"),
								CreateDate:  utilpointer.To(time.Unix(1234, 0)),
							},
						},
					}, nil
				},
				deleteKeyfunc: func(ctx context.Context, in *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
					return &iam.DeleteAccessKeyOutput{}, nil
				},
				createKeyFunc: func(ctx context.Context, in *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
					t := time.Unix(1234, 0)
					return &iam.CreateAccessKeyOutput{
						AccessKey: &types.AccessKey{
							AccessKeyId:     utilpointer.To("uuser"),
							SecretAccessKey: utilpointer.To("pass"),
							CreateDate:      &t,
						},
					}, nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: AWSIAMKey
spec:
  region: eu-west-1
  role: "my-role"
  iamRef:
    username: foo
    maxKeys: 1
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
				"access_key_id":     []byte("uuser"),
				"secret_access_key": []byte("pass"),
			},
		},
		{
			name: "must delete oldest keys",
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
				listKeyFunc: func(ctx context.Context, in *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
					return &iam.ListAccessKeysOutput{
						AccessKeyMetadata: []types.AccessKeyMetadata{
							{
								AccessKeyId: utilpointer.To("uuser"),
								CreateDate:  utilpointer.To(time.Unix(1234, 0)),
							},
							{
								AccessKeyId: utilpointer.To("uuser2"),
								CreateDate:  utilpointer.To(time.Unix(1235, 0)),
							},
							{
								AccessKeyId: utilpointer.To("uuser3"),
								CreateDate:  utilpointer.To(time.Unix(1236, 0)),
							},
						},
					}, nil
				},
				deleteKeyfunc: func(ctx context.Context, in *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
					if in.AccessKeyId != nil && *in.AccessKeyId == "uuser3" {
						return nil, errors.New("target wrong key")
					}
					return &iam.DeleteAccessKeyOutput{}, nil
				},
				createKeyFunc: func(ctx context.Context, in *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
					t := time.Unix(1237, 0)
					return &iam.CreateAccessKeyOutput{
						AccessKey: &types.AccessKey{
							AccessKeyId:     utilpointer.To("uuser"),
							SecretAccessKey: utilpointer.To("pass"),
							CreateDate:      &t,
						},
					}, nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: AWSIAMKey
spec:
  region: eu-west-1
  role: "my-role"
  iamRef:
    username: foo
    maxKeys: 2
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
				"access_key_id":     []byte("uuser"),
				"secret_access_key": []byte("pass"),
			},
		},
		{
			name: "maxKeys is greater than current keys",
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
				listKeyFunc: func(ctx context.Context, in *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
					return &iam.ListAccessKeysOutput{
						AccessKeyMetadata: []types.AccessKeyMetadata{
							{
								AccessKeyId: utilpointer.To("uuser"),
								CreateDate:  utilpointer.To(time.Unix(1234, 0)),
							},
						},
					}, nil
				},
				deleteKeyfunc: func(ctx context.Context, in *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
					return nil, errors.New("should not be called")
				},
				createKeyFunc: func(ctx context.Context, in *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
					t := time.Unix(1234, 0)
					return &iam.CreateAccessKeyOutput{
						AccessKey: &types.AccessKey{
							AccessKeyId:     utilpointer.To("uuser"),
							SecretAccessKey: utilpointer.To("pass"),
							CreateDate:      &t,
						},
					}, nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: AWSIAMKey
spec:
  region: eu-west-1
  role: "my-role"
  iamRef:
    username: foo
    maxKeys: 2
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
				"access_key_id":     []byte("uuser"),
				"secret_access_key": []byte("pass"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got, _, err := g.generate(
				context.TODO(),
				tt.args.jsonSpec,
				tt.args.kube,
				tt.args.namespace,
				func(cfg *aws.Config) iamAPI {
					return &FakeIAM{
						createAccessKeyFunc: tt.args.createKeyFunc,
						listAccessKeyFunc:   tt.args.listKeyFunc,
						deleteAccessKeyFunc: tt.args.deleteKeyfunc,
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

type FakeIAM struct {
	listAccessKeyFunc   func(ctx context.Context, in *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error)
	deleteAccessKeyFunc func(ctx context.Context, in *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error)
	createAccessKeyFunc func(ctx context.Context, in *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
}

func (i *FakeIAM) CreateAccessKey(ctx context.Context, in *iam.CreateAccessKeyInput, optFns ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
	return i.createAccessKeyFunc(ctx, in)
}

func (i *FakeIAM) ListAccessKeys(ctx context.Context, in *iam.ListAccessKeysInput, optFns ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
	return i.listAccessKeyFunc(ctx, in)
}

func (i *FakeIAM) DeleteAccessKey(ctx context.Context, in *iam.DeleteAccessKeyInput, optFns ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
	return i.deleteAccessKeyFunc(ctx, in)
}
