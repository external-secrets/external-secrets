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

package rdsiam

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenerate(t *testing.T) {
	fixedNow := time.Unix(1000, 0)
	type args struct {
		ctx        context.Context
		jsonSpec   *apiextensions.JSON
		kube       client.Client
		namespace  string
		buildToken buildAuthTokenFunc
		now        nowFunc
	}
	tests := []struct {
		name            string
		args            args
		want            map[string][]byte
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "nil spec",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "invalid yaml",
			args: args{
				ctx: context.Background(),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`spec: [`),
				},
			},
			wantErr: true,
		},
		{
			name: "missing region",
			args: args{
				ctx: context.Background(),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  hostname: database.example.com
  port: 5432
  username: db_user`),
				},
			},
			wantErr:         true,
			wantErrContains: "region must be specified",
		},
		{
			name: "blank region",
			args: args{
				ctx: context.Background(),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  region: "   "
  hostname: database.example.com
  port: 5432
  username: db_user`),
				},
			},
			wantErr:         true,
			wantErrContains: "region must be specified",
		},
		{
			name: "missing hostname",
			args: args{
				ctx: context.Background(),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  region: ap-southeast-2
  port: 5432
  username: db_user`),
				},
			},
			wantErr:         true,
			wantErrContains: "hostname must be specified",
		},
		{
			name: "blank hostname",
			args: args{
				ctx: context.Background(),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  region: ap-southeast-2
  hostname: "   "
  port: 5432
  username: db_user`),
				},
			},
			wantErr:         true,
			wantErrContains: "hostname must be specified",
		},
		{
			name: "invalid port below range",
			args: args{
				ctx: context.Background(),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  region: ap-southeast-2
  hostname: database.example.com
  port: 0
  username: db_user`),
				},
			},
			wantErr:         true,
			wantErrContains: "port must be between 1 and 65535, got 0",
		},
		{
			name: "invalid port above range",
			args: args{
				ctx: context.Background(),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  region: ap-southeast-2
  hostname: database.example.com
  port: 65536
  username: db_user`),
				},
			},
			wantErr:         true,
			wantErrContains: "port must be between 1 and 65535, got 65536",
		},
		{
			name: "missing username",
			args: args{
				ctx: context.Background(),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  region: ap-southeast-2
  hostname: database.example.com
  port: 5432`),
				},
			},
			wantErr:         true,
			wantErrContains: "username must be specified",
		},
		{
			name: "blank username",
			args: args{
				ctx: context.Background(),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  region: ap-southeast-2
  hostname: database.example.com
  port: 5432
  username: "   "`),
				},
			},
			wantErr:         true,
			wantErrContains: "username must be specified",
		},
		{
			name: "successful token",
			args: args{
				ctx:       context.Background(),
				namespace: "foobar",
				kube:      clientfake.NewClientBuilder().Build(),
				now: func() time.Time {
					return fixedNow
				},
				buildToken: func(ctx context.Context, endpoint, region, username string, creds aws.CredentialsProvider) (string, error) {
					if endpoint != "database.example.com:5432" {
						t.Fatalf("endpoint = %s, want database.example.com:5432", endpoint)
					}
					if region != "ap-southeast-2" {
						t.Fatalf("region = %s, want ap-southeast-2", region)
					}
					if username != "db_user" {
						t.Fatalf("username = %s, want db_user", username)
					}
					return "rds-token", nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  controller: rds-iam
  region: ap-southeast-2
  hostname: database.example.com
  port: 5432
  username: db_user`),
				},
			},
			want: map[string][]byte{
				"username":   []byte("db_user"),
				"password":   []byte("rds-token"),
				"token":      []byte("rds-token"),
				"hostname":   []byte("database.example.com"),
				"port":       []byte("5432"),
				"endpoint":   []byte("database.example.com:5432"),
				"expires_at": []byte("1900"),
			},
		},
		{
			name: "uses static credentials",
			args: args{
				ctx:       context.Background(),
				namespace: "foobar",
				kube: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-aws-creds",
						Namespace: "foobar",
					},
					Data: map[string][]byte{
						"key-id":        []byte("access-key-id"),
						"access-secret": []byte("secret-access-key"),
					},
				}).Build(),
				now: func() time.Time {
					return fixedNow
				},
				buildToken: func(ctx context.Context, endpoint, region, username string, creds aws.CredentialsProvider) (string, error) {
					gotCreds, err := creds.Retrieve(ctx)
					if err != nil {
						t.Fatalf("Retrieve() error = %v", err)
					}
					if gotCreds.AccessKeyID != "access-key-id" {
						t.Fatalf("AccessKeyID = %s, want access-key-id", gotCreds.AccessKeyID)
					}
					if gotCreds.SecretAccessKey != "secret-access-key" {
						t.Fatalf("SecretAccessKey = %s, want secret-access-key", gotCreds.SecretAccessKey)
					}
					return "static-token", nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  region: ap-southeast-2
  hostname: database.example.com
  port: 5432
  username: db_user
  auth:
    secretRef:
      accessKeyIDSecretRef:
        name: my-aws-creds
        key: key-id
      secretAccessKeySecretRef:
        name: my-aws-creds
        key: access-secret`),
				},
			},
			want: map[string][]byte{
				"username":   []byte("db_user"),
				"password":   []byte("static-token"),
				"token":      []byte("static-token"),
				"hostname":   []byte("database.example.com"),
				"port":       []byte("5432"),
				"endpoint":   []byte("database.example.com:5432"),
				"expires_at": []byte("1900"),
			},
		},
		{
			name: "normalizes string fields",
			args: args{
				ctx:       context.Background(),
				namespace: "foobar",
				kube:      clientfake.NewClientBuilder().Build(),
				now: func() time.Time {
					return fixedNow
				},
				buildToken: func(ctx context.Context, endpoint, region, username string, creds aws.CredentialsProvider) (string, error) {
					if endpoint != "database.example.com:5432" {
						t.Fatalf("endpoint = %s, want database.example.com:5432", endpoint)
					}
					if region != "ap-southeast-2" {
						t.Fatalf("region = %s, want ap-southeast-2", region)
					}
					if username != "db_user" {
						t.Fatalf("username = %s, want db_user", username)
					}
					return "normalized-token", nil
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  region: " ap-southeast-2 "
  hostname: " database.example.com "
  port: 5432
  username: " db_user "`),
				},
			},
			want: map[string][]byte{
				"username":   []byte("db_user"),
				"password":   []byte("normalized-token"),
				"token":      []byte("normalized-token"),
				"hostname":   []byte("database.example.com"),
				"port":       []byte("5432"),
				"endpoint":   []byte("database.example.com:5432"),
				"expires_at": []byte("1900"),
			},
		},
		{
			name: "aws auth failure",
			args: args{
				ctx:       context.Background(),
				namespace: "foobar",
				kube:      clientfake.NewClientBuilder().Build(),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  region: ap-southeast-2
  hostname: database.example.com
  port: 5432
  username: db_user
  auth:
    secretRef:
      accessKeyIDSecretRef:
        name: missing
        key: key-id
      secretAccessKeySecretRef:
        name: missing
        key: access-secret`),
				},
			},
			wantErr: true,
		},
		{
			name: "token builder failure",
			args: args{
				ctx:       context.Background(),
				namespace: "foobar",
				kube:      clientfake.NewClientBuilder().Build(),
				buildToken: func(ctx context.Context, endpoint, region, username string, creds aws.CredentialsProvider) (string, error) {
					return "", errors.New("boom")
				},
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthToken
spec:
  region: ap-southeast-2
  hostname: database.example.com
  port: 5432
  username: db_user`),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			if tt.args.now == nil {
				tt.args.now = time.Now
			}
			if tt.args.buildToken == nil {
				tt.args.buildToken = buildAuthToken
			}
			got, _, err := g.generate(tt.args.ctx, tt.args.jsonSpec, tt.args.kube, tt.args.namespace, tt.args.buildToken, tt.args.now)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generator.Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("Generator.Generate() error = %v, want error containing %q", err, tt.wantErrContains)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Generator.Generate() = %v, want %v", got, tt.want)
			}
		})
	}
}
