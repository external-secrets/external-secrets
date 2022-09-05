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
	expectedData   map[string]string
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
		expectedData:   make(map[string]string),
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
	invalidPerameters := errors.New(ssm.ErrCodeInvalidParameters)

	putParameterOutput := &ssm.PutParameterOutput{}

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
					PutParameterWithContextFn: fakeps.NewPutParameterWithContextFn(putParameterOutput, nil),
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
					PutParameterWithContextFn: fakeps.NewPutParameterWithContextFn(putParameterOutput, invalidPerameters),
				},
			},
			want: want{
				err: invalidPerameters,
			},
		},
	}

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
	setDeserialization := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter.Value = aws.String(`{"foo":"bar"}`)
		pstc.expectedData["foo"] = "bar"
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
		makeValidParameterStoreTestCaseCustom(setDeserialization),
		makeValidParameterStoreTestCaseCustom(setAPIError),
		makeValidParameterStoreTestCaseCustom(setInvalidJSON),
	}

	ps := ParameterStore{}
	for k, v := range successCases {
		ps.client = v.fakeClient
		out, err := ps.GetSecretMap(context.Background(), *v.remoteRef)
		if !ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if cmp.Equal(out, v.expectedData) {
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
