/*
Copyright © 2025 ESO Maintainer Team

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

package beyondtrustsecretsdynamic

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets/fake"
	btsutil "github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets/util"
)

type args struct {
	jsonSpec     *apiextensions.JSON
	kube         kclient.Client
	btsClientFn  func(server, token string) (btsutil.Client, error)
	generateMock func(ctx context.Context, name string, folderPath *string) (*btsutil.GeneratedSecret, error)
}

type want struct {
	val        map[string][]byte
	partialVal map[string][]byte
	err        error
}

type testCase struct {
	reason string
	args   args
	want   want
}

func TestBeyondtrustSecretsDynamicSecretGenerator(t *testing.T) {
	namespace := "test-namespace"

	cases := map[string]testCase{
		"NilSpec": {
			reason: "Raise an error with empty spec.",
			args: args{
				jsonSpec: nil,
			},
			want: want{
				err: errors.New("no config spec provided"),
			},
		},
		"InvalidSpec": {
			reason: "Raise an error with invalid spec.",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(``),
				},
			},
			want: want{
				err: errors.New("no beyondtrustsecrets provider config in spec"),
			},
		},
		"MissingFolderPath": {
			reason: "Raise error if path is not provided.",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(specMissingFolderPath),
				},
				kube: clientfake.NewClientBuilder().Build(),
			},
			want: want{
				err: errors.New("path is required in spec"),
			},
		},
		"MissingAuth": {
			reason: "Raise error if auth is not provided.",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(specMissingAuth),
				},
				kube: clientfake.NewClientBuilder().Build(),
			},
			want: want{
				err: errors.New("failed to create BeyondtrustSecrets client: failed to load credentials: missing or invalid BeyondtrustSecrets API Token in BeyondtrustSecrets SecretStore"),
			},
		},
		"SecretNotFound": {
			reason: "Raise error if secret containing api token does not exist.",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(specSecretNotFound),
				},
				kube: clientfake.NewClientBuilder().Build(),
			},
			want: want{
				err: errors.New(
					"failed to create BeyondtrustSecrets client: failed to load credentials: " +
						"cannot get Kubernetes secret \"nonexistent-secret\" from namespace \"test-namespace\": " +
						"secrets \"nonexistent-secret\" not found",
				),
			},
		},
		"SuccessfulGeneration": {
			reason: "Successfully generate dynamic secret.",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(validDynamicSecretSpec),
				},
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "beyondtrustsecrets-token",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"token": []byte("test-token"),
					},
				}).Build(),
				btsClientFn: func(server, token string) (btsutil.Client, error) {
					client := &fake.BeyondtrustSecretsClient{}
					client.WithValues(context.Background(), nil, nil, nil, nil, nil, nil)
					return client, nil
				},
				generateMock: func(ctx context.Context, name string, folderPath *string) (*btsutil.GeneratedSecret, error) {
					return &btsutil.GeneratedSecret{
						AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
						SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
						SessionToken:    "FwoGZXIvYXdzEBYaD...",
						Expiration:      "2025-12-08T12:00:00Z",
						LeaseID:         "aws/creds/example/abc123",
					}, nil
				},
			},
			want: want{
				val: map[string][]byte{
					"accessKeyId":     []byte("AKIAIOSFODNN7EXAMPLE"),
					"secretAccessKey": []byte("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
					"sessionToken":    []byte("FwoGZXIvYXdzEBYaD..."),
					"expiration":      []byte("2025-12-08T12:00:00Z"),
					"leaseId":         []byte("aws/creds/example/abc123"),
				},
			},
		},
		"SuccessfulGenerationAtRootFolder": {
			reason: "Successfully generate dynamic secret at root folder path.",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(validDynamicSecretSpecNoFolder),
				},
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "beyondtrustsecrets-token",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"token": []byte("test-token"),
					},
				}).Build(),
				btsClientFn: func(server, token string) (btsutil.Client, error) {
					client := &fake.BeyondtrustSecretsClient{}
					client.WithValues(context.Background(), nil, nil, nil, nil, nil, nil)
					return client, nil
				},
				generateMock: func(ctx context.Context, name string, folderPath *string) (*btsutil.GeneratedSecret, error) {
					if folderPath != nil {
						return nil, errors.New("expected nil folder path")
					}
					// For generic secrets, we'll just return empty fields
					return &btsutil.GeneratedSecret{
						AccessKeyID:     "admin",
						SecretAccessKey: "secret123",
					}, nil
				},
			},
			want: want{
				partialVal: map[string][]byte{
					"accessKeyId":     []byte("admin"),
					"secretAccessKey": []byte("secret123"),
				},
			},
		},
		"GenerationError": {
			reason: "Raise error when generation fails.",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(validDynamicSecretSpecWithFolder),
				},
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "beyondtrustsecrets-token",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"token": []byte("test-token"),
					},
				}).Build(),
				btsClientFn: func(server, token string) (btsutil.Client, error) {
					client := &fake.BeyondtrustSecretsClient{}
					client.WithValues(context.Background(), nil, nil, nil, nil, nil, nil)
					return client, nil
				},
				generateMock: func(ctx context.Context, name string, folderPath *string) (*btsutil.GeneratedSecret, error) {
					return nil, errors.New("API error: insufficient permissions")
				},
			},
			want: want{
				err: errors.New("unable to generate dynamic secret: API error: insufficient permissions"),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Create client factory function with mock setup
			newClientFn := func(server, token string) (btsutil.Client, error) {
				client := &fake.BeyondtrustSecretsClient{}
				client.WithValues(context.Background(), nil, nil, nil, nil, nil, nil)

				// Set up the generate mock if provided
				if tc.args.generateMock != nil {
					client.WithGenerateDynamicSecret(tc.args.generateMock)
				}

				return client, nil
			}

			if tc.args.btsClientFn != nil {
				// If a custom client function is provided, wrap it to add the generate mock
				customFn := tc.args.btsClientFn
				newClientFn = func(server, token string) (btsutil.Client, error) {
					client, err := customFn(server, token)
					if err != nil {
						return nil, err
					}

					// If it's a fake client, set up the generate mock
					if fakeClient, ok := client.(*fake.BeyondtrustSecretsClient); ok && tc.args.generateMock != nil {
						fakeClient.WithGenerateDynamicSecret(tc.args.generateMock)
					}

					return client, nil
				}
			}

			// Create generator with injected client factory
			gen := &Generator{
				NewBeyondtrustSecretsClient: newClientFn,
			}

			// Call gen.Generate() for all test cases
			val, _, err := gen.Generate(context.Background(), tc.args.jsonSpec, tc.args.kube, namespace)

			// Assertions
			if tc.want.err != nil {
				if err != nil {
					if diff := cmp.Diff(tc.want.err.Error(), err.Error()); diff != "" {
						t.Errorf("\n%s\nbeyondtrustsecrets.Generate(...): -want error, +got error:\n%s", tc.reason, diff)
					}
				} else {
					t.Errorf("\n%s\nbeyondtrustsecrets.Generate(...): -want error, +got val:\n%v", tc.reason, val)
				}
			} else if tc.want.partialVal != nil {
				for k, v := range tc.want.partialVal {
					if diff := cmp.Diff(v, val[k]); diff != "" {
						t.Errorf("\n%s\nbeyondtrustsecrets.Generate(...) -> %s: -want partial, +got partial:\n%s", tc.reason, k, diff)
					}
				}
			} else {
				if diff := cmp.Diff(tc.want.val, val); diff != "" {
					t.Errorf("\n%s\nbeyondtrustsecrets.Generate(...): -want val, +got val:\n%s", tc.reason, diff)
				}
			}
		})
	}
}

func TestPathParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantFolder     *string
		wantSecretName string
	}{
		{
			name:           "Path with folder",
			input:          "test/subfolder/secret-name",
			wantFolder:     stringPtr("test/subfolder"),
			wantSecretName: "secret-name",
		},
		{
			name:           "Path without folder",
			input:          "secret-name",
			wantFolder:     nil,
			wantSecretName: "secret-name",
		},
		{
			name:           "Path with single folder",
			input:          "folder/secret",
			wantFolder:     stringPtr("folder"),
			wantSecretName: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFolder, gotSecretName := parsePath(tt.input)

			if tt.wantFolder == nil && gotFolder != nil {
				t.Errorf("parsePath() gotFolder = %v, want nil", *gotFolder)
			} else if tt.wantFolder != nil && gotFolder == nil {
				t.Errorf("parsePath() gotFolder = nil, want %v", *tt.wantFolder)
			} else if tt.wantFolder != nil && gotFolder != nil && *gotFolder != *tt.wantFolder {
				t.Errorf("parsePath() gotFolder = %v, want %v", *gotFolder, *tt.wantFolder)
			}

			if gotSecretName != tt.wantSecretName {
				t.Errorf("parsePath() gotSecretName = %v, want %v", gotSecretName, tt.wantSecretName)
			}
		})
	}
}

func TestConvertToByteMap(t *testing.T) {
	tests := []struct {
		name  string
		input *btsutil.GeneratedSecret
		want  map[string][]byte
	}{
		{
			name: "Valid string values",
			input: &btsutil.GeneratedSecret{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SessionToken:    "FwoGZXIvYXdzEBYaD...",
				LeaseID:         "aws/creds/example/abc123",
				Expiration:      "2025-12-08T12:00:00Z",
			},
			want: map[string][]byte{
				"accessKeyId":     []byte("AKIAIOSFODNN7EXAMPLE"),
				"secretAccessKey": []byte("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
				"sessionToken":    []byte("FwoGZXIvYXdzEBYaD..."),
				"leaseId":         []byte("aws/creds/example/abc123"),
				"expiration":      []byte("2025-12-08T12:00:00Z"),
			},
		},
		{
			name: "Empty session token",
			input: &btsutil.GeneratedSecret{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				LeaseID:         "aws/creds/example/abc123",
				Expiration:      "2025-12-08T12:00:00Z",
			},
			want: map[string][]byte{
				"accessKeyId":     []byte("AKIAIOSFODNN7EXAMPLE"),
				"secretAccessKey": []byte("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
				"leaseId":         []byte("aws/creds/example/abc123"),
				"expiration":      []byte("2025-12-08T12:00:00Z"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToByteMap(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("convertToByteMap() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
