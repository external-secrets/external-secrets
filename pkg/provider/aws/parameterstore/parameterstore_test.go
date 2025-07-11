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

package parameterstore

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/external-secrets/external-secrets/pkg/utils/metadata"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	fakeps "github.com/external-secrets/external-secrets/pkg/provider/aws/parameterstore/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
)

const (
	errInvalidProperty = "key INVALPROP does not exist in secret"
	invalidProp        = "INVALPROP"
)

var (
	fakeSecretKey = "fakeSecretKey"
	fakeValue     = "fakeValue"
)

type parameterstoreTestCase struct {
	fakeClient     *fakeps.Client
	apiInput       *ssm.GetParameterInput
	apiOutput      *ssm.GetParameterOutput
	remoteRef      *esv1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string
	expectedData   map[string][]byte
	prefix         string
}

func makeValidParameterStoreTestCase() *parameterstoreTestCase {
	return &parameterstoreTestCase{
		fakeClient:     &fakeps.Client{},
		apiInput:       makeValidAPIInput(),
		apiOutput:      makeValidAPIOutput(),
		remoteRef:      makeValidRemoteRef(),
		apiErr:         nil,
		prefix:         "",
		expectError:    "",
		expectedSecret: "",
		expectedData:   make(map[string][]byte),
	}
}

func makeValidAPIInput() *ssm.GetParameterInput {
	return &ssm.GetParameterInput{
		Name:           aws.String("/baz"),
		WithDecryption: aws.Bool(true),
	}
}

func makeValidAPIOutput() *ssm.GetParameterOutput {
	return &ssm.GetParameterOutput{
		Parameter: &ssmtypes.Parameter{
			Value: aws.String("RRRRR"),
		},
	}
}

func makeValidRemoteRef() *esv1.ExternalSecretDataRemoteRef {
	return &esv1.ExternalSecretDataRemoteRef{
		Key: "/baz",
	}
}

func makeValidParameterStoreTestCaseCustom(tweaks ...func(pstc *parameterstoreTestCase)) *parameterstoreTestCase {
	pstc := makeValidParameterStoreTestCase()
	for _, fn := range tweaks {
		fn(pstc)
	}
	pstc.fakeClient.WithValue(pstc.apiInput, pstc.apiOutput, pstc.apiErr)
	return pstc
}

func TestSSMResolver(t *testing.T) {
	endpointEnvKey := SSMEndpointEnv
	endpointURL := "http://ssm.foo"

	t.Setenv(endpointEnvKey, endpointURL)

	f, err := customEndpointResolver{}.ResolveEndpoint(context.Background(), ssm.EndpointParameters{})

	assert.Nil(t, err)
	assert.Equal(t, endpointURL, f.URI.String())
}

func TestDeleteSecret(t *testing.T) {
	fakeClient := fakeps.Client{}
	parameterName := "parameter"
	managedBy := "managed-by"
	manager := "external-secrets"
	ssmTag := ssmtypes.Tag{
		Key:   &managedBy,
		Value: &manager,
	}
	type args struct {
		client                fakeps.Client
		getParameterOutput    *ssm.GetParameterOutput
		listTagsOutput        *ssm.ListTagsForResourceOutput
		deleteParameterOutput *ssm.DeleteParameterOutput
		getParameterError     error
		listTagsError         error
		deleteParameterError  error
	}

	type want struct {
		err error
	}

	type testCase struct {
		args   args
		want   want
		reason string
	}
	tests := map[string]testCase{
		"Deletes Successfully": {
			args: args{
				client: fakeClient,
				getParameterOutput: &ssm.GetParameterOutput{
					Parameter: &ssmtypes.Parameter{
						Name: &parameterName,
					},
				},
				listTagsOutput: &ssm.ListTagsForResourceOutput{
					TagList: []ssmtypes.Tag{ssmTag},
				},
				deleteParameterOutput: nil,
				getParameterError:     nil,
				listTagsError:         nil,
				deleteParameterError:  nil,
			},
			want: want{
				err: nil,
			},
			reason: "",
		},
		"Secret Not Found": {
			args: args{
				client:                fakeClient,
				getParameterOutput:    nil,
				listTagsOutput:        nil,
				deleteParameterOutput: nil,
				getParameterError: &ssmtypes.ParameterNotFound{
					Message: aws.String("not here, sorry dude"),
				},
				listTagsError:        nil,
				deleteParameterError: nil,
			},
			want: want{
				err: nil,
			},
			reason: "",
		},
		"No permissions to get secret": {
			args: args{
				client:                fakeClient,
				getParameterOutput:    nil,
				listTagsOutput:        nil,
				deleteParameterOutput: nil,
				getParameterError:     errors.New("no permissions"),
				listTagsError:         nil,
				deleteParameterError:  nil,
			},
			want: want{
				err: errors.New("no permissions"),
			},
			reason: "",
		},
		"No permissions to get tags": {
			args: args{
				client: fakeClient,
				getParameterOutput: &ssm.GetParameterOutput{
					Parameter: &ssmtypes.Parameter{
						Name: &parameterName,
					},
				},
				listTagsOutput:        nil,
				deleteParameterOutput: nil,
				getParameterError:     nil,
				listTagsError:         errors.New("no permissions"),
				deleteParameterError:  nil,
			},
			want: want{
				err: errors.New("no permissions"),
			},
			reason: "",
		},
		"Secret Not Managed by External Secrets": {
			args: args{
				client: fakeClient,
				getParameterOutput: &ssm.GetParameterOutput{
					Parameter: &ssmtypes.Parameter{
						Name: &parameterName,
					},
				},
				listTagsOutput: &ssm.ListTagsForResourceOutput{
					TagList: []ssmtypes.Tag{},
				},
				deleteParameterOutput: nil,
				getParameterError:     nil,
				listTagsError:         nil,
				deleteParameterError:  nil,
			},
			want: want{
				err: nil,
			},
			reason: "",
		},
		"No permissions delete secret": {
			args: args{
				client: fakeClient,
				getParameterOutput: &ssm.GetParameterOutput{
					Parameter: &ssmtypes.Parameter{
						Name: &parameterName,
					},
				},
				listTagsOutput: &ssm.ListTagsForResourceOutput{
					TagList: []ssmtypes.Tag{ssmTag},
				},
				deleteParameterOutput: nil,
				getParameterError:     nil,
				listTagsError:         nil,
				deleteParameterError:  errors.New("no permissions"),
			},
			want: want{
				err: errors.New("no permissions"),
			},
			reason: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ref := fake.PushSecretData{RemoteKey: remoteKey}
			ps := ParameterStore{
				client: &tc.args.client,
			}
			tc.args.client.GetParameterFn = fakeps.NewGetParameterFn(tc.args.getParameterOutput, tc.args.getParameterError)
			tc.args.client.ListTagsForResourceFn = fakeps.NewListTagsForResourceFn(tc.args.listTagsOutput, tc.args.listTagsError)
			tc.args.client.DeleteParameterFn = fakeps.NewDeleteParameterFn(tc.args.deleteParameterOutput, tc.args.deleteParameterError)
			err := ps.DeleteSecret(context.TODO(), ref)

			// Error nil XOR tc.want.err nil
			if ((err == nil) || (tc.want.err == nil)) && !((err == nil) && (tc.want.err == nil)) {
				t.Errorf("\nTesting SetSecret:\nName: %v\nReason: %v\nWant error: %v\nGot error: %v", name, tc.reason, tc.want.err, err)
			}

			// if errors are the same type but their contents do not match
			if err != nil && tc.want.err != nil {
				if !strings.Contains(err.Error(), tc.want.err.Error()) {
					t.Errorf("\nTesting SetSecret:\nName: %v\nReason: %v\nWant error: %v\nGot error got nil", name, tc.reason, tc.want.err)
				}
			}
		})
	}
}

const remoteKey = "fake-key"

func TestPushSecret(t *testing.T) {
	invalidParameters := &ssmtypes.InvalidParameters{}
	alreadyExistsError := &ssmtypes.AlreadyExistsException{}
	fakeSecret := &corev1.Secret{
		Data: map[string][]byte{
			fakeSecretKey: []byte(fakeValue),
		},
	}

	managedByESO := ssmtypes.Tag{
		Key:   &managedBy,
		Value: &externalSecrets,
	}

	putParameterOutput := &ssm.PutParameterOutput{}
	getParameterOutput := &ssm.GetParameterOutput{}
	describeParameterOutput := &ssm.DescribeParametersOutput{}
	validListTagsForResourceOutput := &ssm.ListTagsForResourceOutput{
		TagList: []ssmtypes.Tag{managedByESO},
	}
	noTagsResourceOutput := &ssm.ListTagsForResourceOutput{}

	validGetParameterOutput := &ssm.GetParameterOutput{
		Parameter: &ssmtypes.Parameter{},
	}

	sameGetParameterOutput := &ssm.GetParameterOutput{
		Parameter: &ssmtypes.Parameter{
			Value: &fakeValue,
		},
	}

	type args struct {
		store    *esv1.AWSProvider
		metadata *apiextensionsv1.JSON
		client   fakeps.Client
	}

	type want struct {
		err error
	}

	tests := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"PutParameterSucceeds": {
			reason: "a parameter can be successfully pushed to aws parameter store",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				client: fakeps.Client{
					PutParameterFn: fakeps.NewPutParameterFn(putParameterOutput, nil, func(input *ssm.PutParameterInput) {
						assert.Len(t, input.Tags, 1)
						assert.Contains(t, input.Tags, managedByESO)
					}),
					GetParameterFn:       fakeps.NewGetParameterFn(getParameterOutput, nil),
					DescribeParametersFn: fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil, func(input *ssm.ListTagsForResourceInput) {
						assert.Equal(t, "/external-secrets/parameters/fake-key", input.ResourceId)
					}),
					RemoveTagsFromResourceFn: fakeps.NewRemoveTagsFromResourceFn(&ssm.RemoveTagsFromResourceOutput{}, nil),
					AddTagsToResourceFn:      fakeps.NewAddTagsToResourceFn(&ssm.AddTagsToResourceOutput{}, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"SetParameterFailsWhenNoNameProvided": {
			reason: "test push secret with no name gives error",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				client: fakeps.Client{
					PutParameterFn:        fakeps.NewPutParameterFn(putParameterOutput, nil),
					GetParameterFn:        fakeps.NewGetParameterFn(getParameterOutput, invalidParameters),
					DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil),
				},
			},
			want: want{
				err: invalidParameters,
			},
		},
		"SetSecretWhenAlreadyExists": {
			reason: "test push secret with secret that already exists gives error",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				client: fakeps.Client{
					PutParameterFn:        fakeps.NewPutParameterFn(putParameterOutput, alreadyExistsError),
					GetParameterFn:        fakeps.NewGetParameterFn(getParameterOutput, nil),
					DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil),
				},
			},
			want: want{
				err: alreadyExistsError,
			},
		},
		"GetSecretWithValidParameters": {
			reason: "Get secret with valid parameters",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				client: fakeps.Client{
					PutParameterFn:        fakeps.NewPutParameterFn(putParameterOutput, nil),
					GetParameterFn:        fakeps.NewGetParameterFn(validGetParameterOutput, nil),
					DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretNotManagedByESO": {
			reason: "SetSecret to the parameter store but tags are not managed by ESO",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				client: fakeps.Client{
					PutParameterFn:        fakeps.NewPutParameterFn(putParameterOutput, nil),
					GetParameterFn:        fakeps.NewGetParameterFn(validGetParameterOutput, nil),
					DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(noTagsResourceOutput, nil),
				},
			},
			want: want{
				err: errors.New("secret not managed by external-secrets"),
			},
		},
		"SetSecretGetTagsError": {
			reason: "SetSecret to the parameter store returns error while obtaining tags",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				client: fakeps.Client{
					PutParameterFn:        fakeps.NewPutParameterFn(putParameterOutput, nil),
					GetParameterFn:        fakeps.NewGetParameterFn(validGetParameterOutput, nil),
					DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(nil, errors.New("you shall not tag")),
				},
			},
			want: want{
				err: errors.New("you shall not tag"),
			},
		},
		"SetSecretContentMatches": {
			reason: "No ops",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				client: fakeps.Client{
					PutParameterFn:        fakeps.NewPutParameterFn(putParameterOutput, nil),
					GetParameterFn:        fakeps.NewGetParameterFn(sameGetParameterOutput, nil),
					DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithValidMetadata": {
			reason: "test push secret with valid parameterStoreType metadata",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{
						"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
						"kind": "PushSecretMetadata",
						"spec": {
							"secretType": "SecureString",
							"kmsKeyID": "arn:aws:kms:sa-east-1:00000000000:key/bb123123-b2b0-4f60-ac3a-44a13f0e6b6c"
						}
					}`),
				},
				client: fakeps.Client{
					PutParameterFn:        fakeps.NewPutParameterFn(putParameterOutput, nil),
					GetParameterFn:        fakeps.NewGetParameterFn(sameGetParameterOutput, nil),
					DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithValidMetadataListString": {
			reason: "test push secret with valid parameterStoreType metadata and unused parameterStoreKeyID",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{
						"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
						"kind": "PushSecretMetadata",
						"spec": {
							"secretType": "StringList"
						}
					}`),
				},
				client: fakeps.Client{
					PutParameterFn:        fakeps.NewPutParameterFn(putParameterOutput, nil),
					GetParameterFn:        fakeps.NewGetParameterFn(sameGetParameterOutput, nil),
					DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil),
				},
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithInvalidMetadata": {
			reason: "test push secret with invalid metadata structure",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{ fakeMetadataKey: "" }`),
				},
				client: fakeps.Client{
					PutParameterFn:        fakeps.NewPutParameterFn(putParameterOutput, nil),
					GetParameterFn:        fakeps.NewGetParameterFn(sameGetParameterOutput, nil),
					DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil),
				},
			},
			want: want{
				err: errors.New(`failed to parse metadata: failed to parse kubernetes.external-secrets.io/v1alpha1 PushSecretMetadata: error unmarshaling JSON: while decoding JSON: json: unknown field "fakeMetadataKey"`),
			},
		},
		"GetRemoteSecretWithoutDecryption": {
			reason: "test if push secret's get remote source is encrypted for valid comparison",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{
						"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
						"kind": "PushSecretMetadata",
						"spec": {
							"secretType": "SecureString",
							"kmsKeyID": "arn:aws:kms:sa-east-1:00000000000:key/bb123123-b2b0-4f60-ac3a-44a13f0e6b6c"
						}
					}`),
				},
				client: fakeps.Client{
					PutParameterFn: fakeps.NewPutParameterFn(putParameterOutput, nil),
					GetParameterFn: fakeps.NewGetParameterFn(&ssm.GetParameterOutput{
						Parameter: &ssmtypes.Parameter{
							Type:  ssmtypes.ParameterTypeSecureString,
							Value: aws.String("sensitive"),
						},
					}, nil),
					DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil),
				},
			},
			want: want{
				err: errors.New("unable to compare 'sensitive' result, ensure to request a decrypted value"),
			},
		},
		"SecretWithAdvancedTier": {
			reason: "test if we can provide advanced tier policies",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{
						"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
						"kind": "PushSecretMetadata",
						"spec": {
							"secretType": "SecureString",
							"kmsKeyID": "arn:aws:kms:sa-east-1:00000000000:key/bb123123-b2b0-4f60-ac3a-44a13f0e6b6c",
							"tier": {
								"type": "Advanced",
								"policies": [
										{
												"type": "Expiration",
												"version": "1.0",
												"attributes": {
														"timestamp": "2024-12-02T21:34:33.000Z"
												}
										},
										{
												"type": "ExpirationNotification",
												"version": "1.0",
												"attributes": {
														"before": "2",
														"unit": "Days"
												}
										}
								]
							}
						}
					}`),
				},
				client: fakeps.Client{
					PutParameterFn: fakeps.NewPutParameterFn(putParameterOutput, nil),
					GetParameterFn: fakeps.NewGetParameterFn(&ssm.GetParameterOutput{
						Parameter: &ssmtypes.Parameter{
							Type:  ssmtypes.ParameterTypeSecureString,
							Value: aws.String("sensitive"),
						},
					}, nil),
					DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil),
				},
			},
			want: want{
				err: errors.New("unable to compare 'sensitive' result, ensure to request a decrypted value"),
			},
		},
		"SecretPatchTags": {
			reason: "test if we can configure tags for the secret",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{
						"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
						"kind": "PushSecretMetadata",
						"spec": {
							"tags": {
								"env": "sandbox",
								"rotation": "1h"
							},
						}
					}`),
				},
				client: fakeps.Client{
					PutParameterFn: fakeps.NewPutParameterFn(putParameterOutput, nil, func(input *ssm.PutParameterInput) {
						assert.Len(t, input.Tags, 0)
					}),
					GetParameterFn: fakeps.NewGetParameterFn(&ssm.GetParameterOutput{
						Parameter: &ssmtypes.Parameter{
							Value: aws.String("some-value"),
						},
					}, nil),
					DescribeParametersFn: fakeps.NewDescribeParametersFn(&ssm.DescribeParametersOutput{}, nil),
					ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(&ssm.ListTagsForResourceOutput{
						TagList: []ssmtypes.Tag{managedByESO,
							{Key: ptr.To("team"), Value: ptr.To("no-longer-needed")},
							{Key: ptr.To("rotation"), Value: ptr.To("10m")},
						},
					}, nil),
					RemoveTagsFromResourceFn: fakeps.NewRemoveTagsFromResourceFn(&ssm.RemoveTagsFromResourceOutput{}, nil, func(input *ssm.RemoveTagsFromResourceInput) {
						assert.Len(t, input.TagKeys, 1)
						assert.Equal(t, []string{"team"}, input.TagKeys)
					}),
					AddTagsToResourceFn: fakeps.NewAddTagsToResourceFn(&ssm.AddTagsToResourceOutput{}, nil, func(input *ssm.AddTagsToResourceInput) {
						assert.Len(t, input.Tags, 3)
						assert.Contains(t, input.Tags, ssmtypes.Tag{Key: &managedBy, Value: &externalSecrets})
						assert.Contains(t, input.Tags, ssmtypes.Tag{Key: ptr.To("env"), Value: ptr.To("sandbox")})
						assert.Contains(t, input.Tags, ssmtypes.Tag{Key: ptr.To("rotation"), Value: ptr.To("1h")})
					}),
				},
			},
			want: want{
				err: nil,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			psd := fake.PushSecretData{SecretKey: fakeSecretKey, RemoteKey: remoteKey}
			if tc.args.metadata != nil {
				psd.Metadata = tc.args.metadata
			}
			ps := ParameterStore{
				client: &tc.args.client,
			}
			err := ps.PushSecret(context.TODO(), fakeSecret, psd)

			// Error nil XOR tc.want.err nil
			if ((err == nil) || (tc.want.err == nil)) && !((err == nil) && (tc.want.err == nil)) {
				t.Errorf("\nTesting SetSecret:\nName: %v\nReason: %v\nWant error: %v\nGot error: %v", name, tc.reason, tc.want.err, err)
			}

			// if errors are the same type but their contents do not match
			if err != nil && tc.want.err != nil {
				if !strings.Contains(err.Error(), tc.want.err.Error()) {
					t.Errorf("\nTesting SetSecret:\nName: %v\nReason: %v\nWant error: %v\nGot error: %s", name, tc.reason, tc.want.err, err)
				}
			}
		})
	}
}

func TestPushSecretWithPrefix(t *testing.T) {
	fakeSecret := &corev1.Secret{
		Data: map[string][]byte{
			fakeSecretKey: []byte(fakeValue),
		},
	}
	managedByESO := ssmtypes.Tag{
		Key:   &managedBy,
		Value: &externalSecrets,
	}
	putParameterOutput := &ssm.PutParameterOutput{}
	getParameterOutput := &ssm.GetParameterOutput{}
	describeParameterOutput := &ssm.DescribeParametersOutput{}
	validListTagsForResourceOutput := &ssm.ListTagsForResourceOutput{
		TagList: []ssmtypes.Tag{managedByESO},
	}

	client := fakeps.Client{
		PutParameterFn:        fakeps.NewPutParameterFn(putParameterOutput, nil),
		GetParameterFn:        fakeps.NewGetParameterFn(getParameterOutput, nil),
		DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
		ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil),
	}

	psd := fake.PushSecretData{SecretKey: fakeSecretKey, RemoteKey: remoteKey}
	ps := ParameterStore{
		client: &client,
		prefix: "/test/this/thing/",
	}
	err := ps.PushSecret(context.TODO(), fakeSecret, psd)
	require.NoError(t, err)

	input := client.PutParameterFnCalledWith[0][0]
	assert.Equal(t, "/test/this/thing/fake-key", *input.Name)
}

func TestPushSecretWithoutKeyAndEncodedAsDecodedTrue(t *testing.T) {
	fakeSecret := &corev1.Secret{
		Data: map[string][]byte{
			fakeSecretKey: []byte(fakeValue),
		},
	}
	managedByESO := ssmtypes.Tag{
		Key:   &managedBy,
		Value: &externalSecrets,
	}
	putParameterOutput := &ssm.PutParameterOutput{}
	getParameterOutput := &ssm.GetParameterOutput{}
	describeParameterOutput := &ssm.DescribeParametersOutput{}
	validListTagsForResourceOutput := &ssm.ListTagsForResourceOutput{
		TagList: []ssmtypes.Tag{managedByESO},
	}

	client := fakeps.Client{
		PutParameterFn:        fakeps.NewPutParameterFn(putParameterOutput, nil),
		GetParameterFn:        fakeps.NewGetParameterFn(getParameterOutput, nil),
		DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
		ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil),
	}

	psd := fake.PushSecretData{RemoteKey: remoteKey, Metadata: &apiextensionsv1.JSON{Raw: []byte(`
apiVersion: kubernetes.external-secrets.io/v1alpha1
kind: PushSecretMetadata
spec:
  encodeAsDecoded: true
`)}}
	ps := ParameterStore{
		client: &client,
		prefix: "/test/this/thing/",
	}
	err := ps.PushSecret(context.TODO(), fakeSecret, psd)
	require.NoError(t, err)

	input := client.PutParameterFnCalledWith[0][0]
	assert.Equal(t, "{\"fakeSecretKey\":\"fakeValue\"}", *input.Value)
}

func TestPushSecretCalledOnlyOnce(t *testing.T) {
	fakeSecret := &corev1.Secret{
		Data: map[string][]byte{
			fakeSecretKey: []byte(fakeValue),
		},
	}

	managedByESO := ssmtypes.Tag{
		Key:   &managedBy,
		Value: &externalSecrets,
	}

	putParameterOutput := &ssm.PutParameterOutput{}
	validGetParameterOutput := &ssm.GetParameterOutput{
		Parameter: &ssmtypes.Parameter{
			Value: &fakeValue,
		},
	}
	describeParameterOutput := &ssm.DescribeParametersOutput{}
	validListTagsForResourceOutput := &ssm.ListTagsForResourceOutput{
		TagList: []ssmtypes.Tag{managedByESO},
	}

	client := fakeps.Client{
		PutParameterFn:        fakeps.NewPutParameterFn(putParameterOutput, nil),
		GetParameterFn:        fakeps.NewGetParameterFn(validGetParameterOutput, nil),
		DescribeParametersFn:  fakeps.NewDescribeParametersFn(describeParameterOutput, nil),
		ListTagsForResourceFn: fakeps.NewListTagsForResourceFn(validListTagsForResourceOutput, nil),
	}

	psd := fake.PushSecretData{SecretKey: fakeSecretKey, RemoteKey: remoteKey}
	ps := ParameterStore{
		client: &client,
	}

	require.NoError(t, ps.PushSecret(context.TODO(), fakeSecret, psd))

	assert.Equal(t, 0, client.PutParameterCalledN)
}

// test the ssm<->aws interface
// make sure correct values are passed and errors are handled accordingly.
func TestGetSecret(t *testing.T) {
	// good case: key is passed in, output is sent back
	setSecretString := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter.Value = aws.String("RRRRR")
		pstc.expectedSecret = "RRRRR"
	}

	// good case: key is passed in and prefix is set, output is sent back
	setSecretStringWithPrefix := func(pstc *parameterstoreTestCase) {
		pstc.apiInput = &ssm.GetParameterInput{
			Name:           aws.String("/test/this/baz"),
			WithDecryption: aws.Bool(true),
		}
		pstc.prefix = "/test/this"
		pstc.apiOutput.Parameter.Value = aws.String("RRRRR")
		pstc.expectedSecret = "RRRRR"
	}

	// good case: extract property
	setExtractProperty := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter.Value = aws.String(`{"/shmoo": "bang"}`)
		pstc.expectedSecret = "bang"
		pstc.remoteRef.Property = "/shmoo"
	}

	// good case: extract property with `.`
	setExtractPropertyWithDot := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter.Value = aws.String(`{"/shmoo.boom": "bang"}`)
		pstc.expectedSecret = "bang"
		pstc.remoteRef.Property = "/shmoo.boom"
	}

	// bad case: missing property
	setMissingProperty := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter.Value = aws.String(`{"/shmoo": "bang"}`)
		pstc.remoteRef.Property = "INVALPROP"
		pstc.expectError = "key INVALPROP does not exist in secret"
	}

	// bad case: parameter.Value not found
	setParameterValueNotFound := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter.Value = aws.String("NONEXISTENT")
		pstc.apiErr = esv1.NoSecretErr
		pstc.expectError = "Secret does not exist"
	}

	// bad case: extract property failure due to invalid json
	setPropertyFail := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter.Value = aws.String(`------`)
		pstc.remoteRef.Property = invalidProp
		pstc.expectError = errInvalidProperty
	}

	// bad case: parameter.Value may be nil but binary is set
	setParameterValueNil := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter.Value = nil
		pstc.expectError = "parameter value is nil for key"
	}

	// base case: api output return error
	setAPIError := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput = &ssm.GetParameterOutput{}
		pstc.apiErr = errors.New("oh no")
		pstc.expectError = "oh no"
	}

	// good case: metadata returned
	setMetadataString := func(pstc *parameterstoreTestCase) {
		pstc.remoteRef.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		output := ssm.ListTagsForResourceOutput{
			TagList: getTagSlice(),
		}
		pstc.fakeClient.ListTagsForResourceFn = fakeps.NewListTagsForResourceFn(&output, nil)
		pstc.expectedSecret, _ = util.ParameterTagsToJSONString(normaliseTags(getTagSlice()))
	}

	// good case: metadata property returned
	setMetadataProperty := func(pstc *parameterstoreTestCase) {
		pstc.remoteRef.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		output := ssm.ListTagsForResourceOutput{
			TagList: getTagSlice(),
		}
		pstc.fakeClient.ListTagsForResourceFn = fakeps.NewListTagsForResourceFn(&output, nil)
		pstc.remoteRef.Property = "tagname2"
		pstc.expectedSecret = "tagvalue2"
	}

	// bad case: metadata property not found
	setMetadataMissingProperty := func(pstc *parameterstoreTestCase) {
		pstc.remoteRef.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		output := ssm.ListTagsForResourceOutput{
			TagList: getTagSlice(),
		}
		pstc.fakeClient.ListTagsForResourceFn = fakeps.NewListTagsForResourceFn(&output, nil)
		pstc.remoteRef.Property = invalidProp
		pstc.expectError = errInvalidProperty
	}

	successCases := []*parameterstoreTestCase{
		makeValidParameterStoreTestCaseCustom(setSecretStringWithPrefix),
		makeValidParameterStoreTestCaseCustom(setSecretString),
		makeValidParameterStoreTestCaseCustom(setExtractProperty),
		makeValidParameterStoreTestCaseCustom(setMissingProperty),
		makeValidParameterStoreTestCaseCustom(setPropertyFail),
		makeValidParameterStoreTestCaseCustom(setParameterValueNil),
		makeValidParameterStoreTestCaseCustom(setAPIError),
		makeValidParameterStoreTestCaseCustom(setExtractPropertyWithDot),
		makeValidParameterStoreTestCaseCustom(setParameterValueNotFound),
		makeValidParameterStoreTestCaseCustom(setMetadataString),
		makeValidParameterStoreTestCaseCustom(setMetadataProperty),
		makeValidParameterStoreTestCaseCustom(setMetadataMissingProperty),
	}

	ps := ParameterStore{}
	for k, v := range successCases {
		ps.client = v.fakeClient
		ps.prefix = v.prefix
		out, err := ps.GetSecret(context.Background(), *v.remoteRef)
		if !ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if cmp.Equal(out, v.expectedSecret) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, v.expectedSecret, out)
		}
	}
}

func TestGetSecretMap(t *testing.T) {
	// good case: default version & deserialization
	simpleJSON := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter.Value = aws.String(`{"foo":"bar"}`)
		pstc.expectedData["foo"] = []byte("bar")
	}

	// good case: default version & complex json
	complexJSON := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter.Value = aws.String(`{"int": 42, "str": "str", "nested": {"foo":"bar"}}`)
		pstc.expectedData["int"] = []byte("42")
		pstc.expectedData["str"] = []byte("str")
		pstc.expectedData["nested"] = []byte(`{"foo":"bar"}`)
	}

	// bad case: api error returned
	setAPIError := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter = &ssmtypes.Parameter{}
		pstc.expectError = "some api err"
		pstc.apiErr = errors.New("some api err")
	}
	// bad case: invalid json
	setInvalidJSON := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter.Value = aws.String(`-----------------`)
		pstc.expectError = "unable to unmarshal secret"
	}

	successCases := []*parameterstoreTestCase{
		makeValidParameterStoreTestCaseCustom(simpleJSON),
		makeValidParameterStoreTestCaseCustom(complexJSON),
		makeValidParameterStoreTestCaseCustom(setAPIError),
		makeValidParameterStoreTestCaseCustom(setInvalidJSON),
	}

	ps := ParameterStore{}
	for k, v := range successCases {
		ps.client = v.fakeClient
		out, err := ps.GetSecretMap(context.Background(), *v.remoteRef)
		if !ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %q, expected: %q", k, err.Error(), v.expectError)
		}
		if err == nil && !cmp.Equal(out, v.expectedData) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, v.expectedData, out)
		}
	}
}

func makeValidParameterStore() *esv1.SecretStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-parameterstore",
			Namespace: "default",
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AWS: &esv1.AWSProvider{
					Service: esv1.AWSServiceParameterStore,
					Region:  "us-east-1",
				},
			},
		},
	}
}

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

func getTagSlice() []ssmtypes.Tag {
	tagKey1 := "tagname1"
	tagValue1 := "tagvalue1"
	tagKey2 := "tagname2"
	tagValue2 := "tagvalue2"

	return []ssmtypes.Tag{
		{
			Key:   &tagKey1,
			Value: &tagValue1,
		},
		{
			Key:   &tagKey2,
			Value: &tagValue2,
		},
	}
}

func normaliseTags(input []ssmtypes.Tag) map[string]string {
	tags := make(map[string]string, len(input))
	for _, tag := range input {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}
	return tags
}

func TestSecretExists(t *testing.T) {
	parameterOutput := &ssm.GetParameterOutput{
		Parameter: &ssmtypes.Parameter{
			Value: aws.String("sensitive"),
		},
	}

	blankParameterOutput := &ssm.GetParameterOutput{}
	getParameterCorrectErr := ssmtypes.ResourceNotFoundException{}
	getParameterWrongErr := ssmtypes.InvalidParameters{}

	pushSecretDataWithoutProperty := fake.PushSecretData{SecretKey: "fake-secret-key", RemoteKey: fakeSecretKey, Property: ""}

	type args struct {
		store          *esv1.AWSProvider
		client         fakeps.Client
		pushSecretData fake.PushSecretData
	}

	type want struct {
		err       error
		wantError bool
	}

	tests := map[string]struct {
		args args
		want want
	}{
		"SecretExistsReturnsTrueForExistingParameter": {
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				client: fakeps.Client{
					GetParameterFn: fakeps.NewGetParameterFn(parameterOutput, nil),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err:       nil,
				wantError: true,
			},
		},
		"SecretExistsReturnsFalseForNonExistingParameter": {
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				client: fakeps.Client{
					GetParameterFn: fakeps.NewGetParameterFn(blankParameterOutput, &getParameterCorrectErr),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err:       nil,
				wantError: false,
			},
		},
		"SecretExistsReturnsFalseForErroredParameter": {
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				client: fakeps.Client{
					GetParameterFn: fakeps.NewGetParameterFn(blankParameterOutput, &getParameterWrongErr),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err:       &getParameterWrongErr,
				wantError: false,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ps := &ParameterStore{
				client: &tc.args.client,
			}
			got, err := ps.SecretExists(context.Background(), tc.args.pushSecretData)

			assert.Equal(
				t,
				tc.want,
				want{
					err:       err,
					wantError: got,
				})
		})
	}
}

func TestConstructMetadataWithDefaults(t *testing.T) {
	tests := []struct {
		name        string
		input       *apiextensionsv1.JSON
		expected    *metadata.PushSecretMetadata[PushSecretMetadataSpec]
		expectError bool
	}{
		{
			name: "Valid metadata with multiple fields",
			input: &apiextensionsv1.JSON{Raw: []byte(`{
				"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
				"kind": "PushSecretMetadata",
				"spec": {
 					"description": "test description",
					"tier": {"type": "Advanced"},
					"secretType":"SecureString",
					"kmsKeyID": "custom-kms-key",
					"tags": {
						"customKey": "customValue"
					},
				}
			}`)},
			expected: &metadata.PushSecretMetadata[PushSecretMetadataSpec]{
				APIVersion: "kubernetes.external-secrets.io/v1alpha1",
				Kind:       "PushSecretMetadata",
				Spec: PushSecretMetadataSpec{
					Description: "test description",
					Tier: Tier{
						Type: "Advanced",
					},
					SecretType: "SecureString",
					KMSKeyID:   "custom-kms-key",
					Tags: map[string]string{
						"customKey":  "customValue",
						"managed-by": "external-secrets",
					},
				},
			},
		},
		{
			name:  "Empty metadata, defaults applied",
			input: nil,
			expected: &metadata.PushSecretMetadata[PushSecretMetadataSpec]{
				Spec: PushSecretMetadataSpec{
					Description: "secret 'managed-by:external-secrets'",
					Tier: Tier{
						Type: "Standard",
					},
					SecretType: "String",
					KMSKeyID:   "alias/aws/ssm",
					Tags: map[string]string{
						"managed-by": "external-secrets",
					},
				},
			},
		},
		{
			name: "Added default metadata with 'managed-by' tag",
			input: &apiextensionsv1.JSON{Raw: []byte(`{
				"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
				"kind": "PushSecretMetadata",
				"spec": {
 					"description": "adding managed-by tag explicitly",
					"tags": {
						"managed-by": "external-secrets",
						"customKey": "customValue"
					},
				}
			}`)},
			expectError: true,
		},
		{
			name:        "Invalid metadata format",
			input:       &apiextensionsv1.JSON{Raw: []byte(`invalid-json`)},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Metadata with 'managed-by' tag specified",
			input:       &apiextensionsv1.JSON{Raw: []byte(`{"tags":{"managed-by":"invalid"}}`)},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := (&ParameterStore{}).constructMetadataWithDefaults(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestComputeTagsToUpdate(t *testing.T) {
	tests := []struct {
		name     string
		tags     map[string]string
		metaTags map[string]string
		expected []ssmtypes.Tag
		modified bool
	}{
		{
			name: "No tags to update",
			tags: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			metaTags: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expected: []ssmtypes.Tag{
				{Key: ptr.To("key1"), Value: ptr.To("value1")},
				{Key: ptr.To("key2"), Value: ptr.To("value2")},
			},
			modified: false,
		},
		{
			name: "Add new tag",
			tags: map[string]string{
				"key1": "value1",
			},
			metaTags: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expected: []ssmtypes.Tag{
				{Key: ptr.To("key1"), Value: ptr.To("value1")},
				{Key: ptr.To("key2"), Value: ptr.To("value2")},
			},
			modified: true,
		},
		{
			name: "Update existing tag value",
			tags: map[string]string{
				"key1": "value1",
			},
			metaTags: map[string]string{
				"key1": "newValue",
			},
			expected: []ssmtypes.Tag{
				{Key: ptr.To("key1"), Value: ptr.To("newValue")},
			},
			modified: true,
		},
		{
			name:     "Empty tags and metaTags",
			tags:     map[string]string{},
			metaTags: map[string]string{},
			expected: []ssmtypes.Tag{},
			modified: false,
		},
		{
			name: "Empty tags with non-empty metaTags",
			tags: map[string]string{},
			metaTags: map[string]string{
				"key1": "value1",
			},
			expected: []ssmtypes.Tag{
				{Key: ptr.To("key1"), Value: ptr.To("value1")},
			},
			modified: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, modified := computeTagsToUpdate(tt.tags, tt.metaTags)
			assert.ElementsMatch(t, tt.expected, result)
			assert.Equal(t, tt.modified, modified)
		})
	}
}
