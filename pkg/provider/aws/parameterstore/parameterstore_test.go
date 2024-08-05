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
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
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
	remoteRef      *esv1beta1.ExternalSecretDataRemoteRef
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
		Parameter: &ssm.Parameter{
			Value: aws.String("RRRRR"),
		},
	}
}

func makeValidRemoteRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
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

func TestDeleteSecret(t *testing.T) {
	fakeClient := fakeps.Client{}
	parameterName := "parameter"
	managedBy := "managed-by"
	manager := "external-secrets"
	ssmTag := ssm.Tag{
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
					Parameter: &ssm.Parameter{
						Name: &parameterName,
					},
				},
				listTagsOutput: &ssm.ListTagsForResourceOutput{
					TagList: []*ssm.Tag{&ssmTag},
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
				getParameterError:     awserr.New(ssm.ErrCodeParameterNotFound, "not here, sorry dude", nil),
				listTagsError:         nil,
				deleteParameterError:  nil,
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
					Parameter: &ssm.Parameter{
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
					Parameter: &ssm.Parameter{
						Name: &parameterName,
					},
				},
				listTagsOutput: &ssm.ListTagsForResourceOutput{
					TagList: []*ssm.Tag{},
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
					Parameter: &ssm.Parameter{
						Name: &parameterName,
					},
				},
				listTagsOutput: &ssm.ListTagsForResourceOutput{
					TagList: []*ssm.Tag{&ssmTag},
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
			tc.args.client.GetParameterWithContextFn = fakeps.NewGetParameterWithContextFn(tc.args.getParameterOutput, tc.args.getParameterError)
			tc.args.client.ListTagsForResourceWithContextFn = fakeps.NewListTagsForResourceWithContextFn(tc.args.listTagsOutput, tc.args.listTagsError)
			tc.args.client.DeleteParameterWithContextFn = fakeps.NewDeleteParameterWithContextFn(tc.args.deleteParameterOutput, tc.args.deleteParameterError)
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
	invalidParameters := errors.New(ssm.ErrCodeInvalidParameters)
	alreadyExistsError := errors.New(ssm.ErrCodeAlreadyExistsException)
	fakeSecret := &corev1.Secret{
		Data: map[string][]byte{
			fakeSecretKey: []byte(fakeValue),
		},
	}

	managedByESO := ssm.Tag{
		Key:   &managedBy,
		Value: &externalSecrets,
	}

	putParameterOutput := &ssm.PutParameterOutput{}
	getParameterOutput := &ssm.GetParameterOutput{}
	describeParameterOutput := &ssm.DescribeParametersOutput{}
	validListTagsForResourceOutput := &ssm.ListTagsForResourceOutput{
		TagList: []*ssm.Tag{&managedByESO},
	}
	noTagsResourceOutput := &ssm.ListTagsForResourceOutput{}

	validGetParameterOutput := &ssm.GetParameterOutput{
		Parameter: &ssm.Parameter{
			ARN:              nil,
			DataType:         nil,
			LastModifiedDate: nil,
			Name:             nil,
			Selector:         nil,
			SourceResult:     nil,
			Type:             nil,
			Value:            nil,
			Version:          nil,
		},
	}

	sameGetParameterOutput := &ssm.GetParameterOutput{
		Parameter: &ssm.Parameter{
			Value: &fakeValue,
		},
	}

	type args struct {
		store    *esv1beta1.AWSProvider
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
					PutParameterWithContextFn:        fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
					GetParameterWithContextFn:        fakeps.NewGetParameterWithContextFn(getParameterOutput, nil),
					DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
					ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(validListTagsForResourceOutput, nil),
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
					PutParameterWithContextFn:        fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
					GetParameterWithContextFn:        fakeps.NewGetParameterWithContextFn(getParameterOutput, invalidParameters),
					DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
					ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(validListTagsForResourceOutput, nil),
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
					PutParameterWithContextFn:        fakeps.NewPutParameterWithContextFn(putParameterOutput, alreadyExistsError),
					GetParameterWithContextFn:        fakeps.NewGetParameterWithContextFn(getParameterOutput, nil),
					DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
					ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(validListTagsForResourceOutput, nil),
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
					PutParameterWithContextFn:        fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
					GetParameterWithContextFn:        fakeps.NewGetParameterWithContextFn(validGetParameterOutput, nil),
					DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
					ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(validListTagsForResourceOutput, nil),
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
					PutParameterWithContextFn:        fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
					GetParameterWithContextFn:        fakeps.NewGetParameterWithContextFn(validGetParameterOutput, nil),
					DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
					ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(noTagsResourceOutput, nil),
				},
			},
			want: want{
				err: fmt.Errorf("secret not managed by external-secrets"),
			},
		},
		"SetSecretGetTagsError": {
			reason: "SetSecret to the parameter store returns error while obtaining tags",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				client: fakeps.Client{
					PutParameterWithContextFn:        fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
					GetParameterWithContextFn:        fakeps.NewGetParameterWithContextFn(validGetParameterOutput, nil),
					DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
					ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(nil, fmt.Errorf("you shall not tag")),
				},
			},
			want: want{
				err: fmt.Errorf("you shall not tag"),
			},
		},
		"SetSecretContentMatches": {
			reason: "No ops",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				client: fakeps.Client{
					PutParameterWithContextFn:        fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
					GetParameterWithContextFn:        fakeps.NewGetParameterWithContextFn(sameGetParameterOutput, nil),
					DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
					ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(validListTagsForResourceOutput, nil),
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
					Raw: []byte(`
					{
						"parameterStoreType": "SecureString", 
						"parameterStoreKeyID": "arn:aws:kms:sa-east-1:00000000000:key/bb123123-b2b0-4f60-ac3a-44a13f0e6b6c"
					}
					`),
				},
				client: fakeps.Client{
					PutParameterWithContextFn:        fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
					GetParameterWithContextFn:        fakeps.NewGetParameterWithContextFn(sameGetParameterOutput, nil),
					DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
					ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(validListTagsForResourceOutput, nil),
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
					Raw: []byte(`{"parameterStoreType": "StringList", "parameterStoreKeyID": "alias/aws/ssm"}`),
				},
				client: fakeps.Client{
					PutParameterWithContextFn:        fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
					GetParameterWithContextFn:        fakeps.NewGetParameterWithContextFn(sameGetParameterOutput, nil),
					DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
					ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(validListTagsForResourceOutput, nil),
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
					PutParameterWithContextFn:        fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
					GetParameterWithContextFn:        fakeps.NewGetParameterWithContextFn(sameGetParameterOutput, nil),
					DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
					ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(validListTagsForResourceOutput, nil),
				},
			},
			want: want{
				err: fmt.Errorf("failed to parse metadata: failed to parse JSON raw data: invalid character 'f' looking for beginning of object key string"),
			},
		},
		"GetRemoteSecretWithoutDecryption": {
			reason: "test if push secret's get remote source is encrypted for valid comparison",
			args: args{
				store: makeValidParameterStore().Spec.Provider.AWS,
				metadata: &apiextensionsv1.JSON{
					Raw: []byte(`
					{
						"parameterStoreType": "SecureString",
						"parameterStoreKeyID": "arn:aws:kms:sa-east-1:00000000000:key/bb123123-b2b0-4f60-ac3a-44a13f0e6b6c"
					}
					`),
				},
				client: fakeps.Client{
					PutParameterWithContextFn: fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
					GetParameterWithContextFn: fakeps.NewGetParameterWithContextFn(&ssm.GetParameterOutput{
						Parameter: &ssm.Parameter{
							Type:  aws.String("SecureString"),
							Value: aws.String("sensitive"),
						},
					}, nil),
					DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
					ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(validListTagsForResourceOutput, nil),
				},
			},
			want: want{
				err: fmt.Errorf("unable to compare 'sensitive' result, ensure to request a decrypted value"),
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
					t.Errorf("\nTesting SetSecret:\nName: %v\nReason: %v\nWant error: %v\nGot error got nil", name, tc.reason, tc.want.err)
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
	managedByESO := ssm.Tag{
		Key:   &managedBy,
		Value: &externalSecrets,
	}
	putParameterOutput := &ssm.PutParameterOutput{}
	getParameterOutput := &ssm.GetParameterOutput{}
	describeParameterOutput := &ssm.DescribeParametersOutput{}
	validListTagsForResourceOutput := &ssm.ListTagsForResourceOutput{
		TagList: []*ssm.Tag{&managedByESO},
	}

	client := fakeps.Client{
		PutParameterWithContextFn:        fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
		GetParameterWithContextFn:        fakeps.NewGetParameterWithContextFn(getParameterOutput, nil),
		DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
		ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(validListTagsForResourceOutput, nil),
	}

	psd := fake.PushSecretData{SecretKey: fakeSecretKey, RemoteKey: remoteKey}
	ps := ParameterStore{
		client: &client,
		prefix: "/test/this/thing/",
	}
	err := ps.PushSecret(context.TODO(), fakeSecret, psd)
	require.NoError(t, err)

	input := client.PutParameterWithContextFnCalledWith[0][0]
	assert.Equal(t, "/test/this/thing/fake-key", *input.Name)
}

func TestPushSecretCalledOnlyOnce(t *testing.T) {
	fakeSecret := &corev1.Secret{
		Data: map[string][]byte{
			fakeSecretKey: []byte(fakeValue),
		},
	}

	managedByESO := ssm.Tag{
		Key:   &managedBy,
		Value: &externalSecrets,
	}

	putParameterOutput := &ssm.PutParameterOutput{}
	validGetParameterOutput := &ssm.GetParameterOutput{
		Parameter: &ssm.Parameter{
			Value: &fakeValue,
		},
	}
	describeParameterOutput := &ssm.DescribeParametersOutput{}
	validListTagsForResourceOutput := &ssm.ListTagsForResourceOutput{
		TagList: []*ssm.Tag{&managedByESO},
	}

	client := fakeps.Client{
		PutParameterWithContextFn:        fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
		GetParameterWithContextFn:        fakeps.NewGetParameterWithContextFn(validGetParameterOutput, nil),
		DescribeParametersWithContextFn:  fakeps.NewDescribeParametersWithContextFn(describeParameterOutput, nil),
		ListTagsForResourceWithContextFn: fakeps.NewListTagsForResourceWithContextFn(validListTagsForResourceOutput, nil),
	}

	psd := fake.PushSecretData{SecretKey: fakeSecretKey, RemoteKey: remoteKey}
	ps := ParameterStore{
		client: &client,
	}

	require.NoError(t, ps.PushSecret(context.TODO(), fakeSecret, psd))

	assert.Equal(t, 0, client.PutParameterWithContextCalledN)
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
		pstc.apiErr = esv1beta1.NoSecretErr
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
		pstc.apiErr = fmt.Errorf("oh no")
		pstc.expectError = "oh no"
	}

	// good case: metadata returned
	setMetadataString := func(pstc *parameterstoreTestCase) {
		pstc.remoteRef.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		output := ssm.ListTagsForResourceOutput{
			TagList: getTagSlice(),
		}
		pstc.fakeClient.ListTagsForResourceWithContextFn = fakeps.NewListTagsForResourceWithContextFn(&output, nil)
		pstc.expectedSecret, _ = util.ParameterTagsToJSONString(getTagSlice())
	}

	// good case: metadata property returned
	setMetadataProperty := func(pstc *parameterstoreTestCase) {
		pstc.remoteRef.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		output := ssm.ListTagsForResourceOutput{
			TagList: getTagSlice(),
		}
		pstc.fakeClient.ListTagsForResourceWithContextFn = fakeps.NewListTagsForResourceWithContextFn(&output, nil)
		pstc.remoteRef.Property = "tagname2"
		pstc.expectedSecret = "tagvalue2"
	}

	// bad case: metadata property not found
	setMetadataMissingProperty := func(pstc *parameterstoreTestCase) {
		pstc.remoteRef.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		output := ssm.ListTagsForResourceOutput{
			TagList: getTagSlice(),
		}
		pstc.fakeClient.ListTagsForResourceWithContextFn = fakeps.NewListTagsForResourceWithContextFn(&output, nil)
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
		pstc.apiOutput.Parameter = &ssm.Parameter{}
		pstc.expectError = "some api err"
		pstc.apiErr = fmt.Errorf("some api err")
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

func makeValidParameterStore() *esv1beta1.SecretStore {
	return &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-parameterstore",
			Namespace: "default",
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				AWS: &esv1beta1.AWSProvider{
					Service: esv1beta1.AWSServiceParameterStore,
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

func getTagSlice() []*ssm.Tag {
	tagKey1 := "tagname1"
	tagValue1 := "tagvalue1"
	tagKey2 := "tagname2"
	tagValue2 := "tagvalue2"

	return []*ssm.Tag{
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
