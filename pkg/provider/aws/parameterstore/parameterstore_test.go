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
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	fakeps "github.com/external-secrets/external-secrets/pkg/provider/aws/parameterstore/fake"
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
}

type fakeRef struct {
	key string
}

func (f fakeRef) GetRemoteKey() string {
	return f.key
}

func makeValidParameterStoreTestCase() *parameterstoreTestCase {
	return &parameterstoreTestCase{
		fakeClient:     &fakeps.Client{},
		apiInput:       makeValidAPIInput(),
		apiOutput:      makeValidAPIOutput(),
		remoteRef:      makeValidRemoteRef(),
		apiErr:         nil,
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

func TestPushSecret(t *testing.T) {
	invalidParameters := errors.New(ssm.ErrCodeInvalidParameters)
	alreadyExistsError := errors.New(ssm.ErrCodeAlreadyExistsException)

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

	type args struct {
		store  *esv1beta1.AWSProvider
		client fakeps.Client
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
	}
	//
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ref := fakeRef{key: "fake-key"}
			ps := ParameterStore{
				client: &tc.args.client,
			}
			err := ps.SetSecret(context.TODO(), []byte("fakeValue"), ref)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\nTesting SetSecret:\nName: %v\nReason: %v\nWant error: %v\nGot error: %v", name, tc.reason, tc.want.err, diff)
			}
		})
	}
}

// test the ssm<->aws interface
// make sure correct values are passed and errors are handled accordingly.
func TestGetSecret(t *testing.T) {
	// good case: key is passed in, output is sent back
	setSecretString := func(pstc *parameterstoreTestCase) {
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
		pstc.remoteRef.Property = "INVALPROP"
		pstc.expectError = "key INVALPROP does not exist in secret"
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

	successCases := []*parameterstoreTestCase{
		makeValidParameterStoreTestCaseCustom(setSecretString),
		makeValidParameterStoreTestCaseCustom(setExtractProperty),
		makeValidParameterStoreTestCaseCustom(setMissingProperty),
		makeValidParameterStoreTestCaseCustom(setPropertyFail),
		makeValidParameterStoreTestCaseCustom(setParameterValueNil),
		makeValidParameterStoreTestCaseCustom(setAPIError),
		makeValidParameterStoreTestCaseCustom(setExtractPropertyWithDot),
		makeValidParameterStoreTestCaseCustom(setParameterValueNotFound),
	}

	ps := ParameterStore{}
	for k, v := range successCases {
		ps.client = v.fakeClient
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
