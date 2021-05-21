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
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	fake "github.com/external-secrets/external-secrets/pkg/provider/aws/parameterstore/fake"
	sess "github.com/external-secrets/external-secrets/pkg/provider/aws/session"
)

func TestConstructor(t *testing.T) {
	s, err := sess.New("1111", "2222", "foo", "", nil)
	assert.Nil(t, err)
	c, err := New(s)
	assert.Nil(t, err)
	assert.NotNil(t, c.client)
}

type parameterstoreTestCase struct {
	fakeClient     *fake.Client
	apiInput       *ssm.GetParameterInput
	apiOutput      *ssm.GetParameterOutput
	remoteRef      *esv1alpha1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string
	expectedData   map[string]string
}

func makeValidParameterStoreTestCase() *parameterstoreTestCase {
	return &parameterstoreTestCase{
		fakeClient:     &fake.Client{},
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

func makeValidRemoteRef() *esv1alpha1.ExternalSecretDataRemoteRef {
	return &esv1alpha1.ExternalSecretDataRemoteRef{
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

// test the ssm<->aws interface
// make sure correct values are passed and errors are handled accordingly.
func TestGetSecret(t *testing.T) {
	f := &fake.Client{}
	p := &ParameterStore{
		client: f,
	}
	for i, row := range []struct {
		apiInput       *ssm.GetParameterInput
		apiOutput      *ssm.GetParameterOutput
		rr             esv1alpha1.ExternalSecretDataRemoteRef
		apiErr         error
		expectError    string
		expectedSecret string
	}{
		{
			// good case: key is passed in, output is sent back
			apiInput: &ssm.GetParameterInput{
				Name:           aws.String("/baz"),
				WithDecryption: aws.Bool(true),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key: "/baz",
			},
			apiOutput: &ssm.GetParameterOutput{
				Parameter: &ssm.Parameter{
					Value: aws.String("RRRRR"),
				},
			},
			apiErr:         nil,
			expectError:    "",
			expectedSecret: "RRRRR",
		},
		{
			// good case: extract property
			apiInput: &ssm.GetParameterInput{
				Name:           aws.String("/baz"),
				WithDecryption: aws.Bool(true),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key:      "/baz",
				Property: "/shmoo",
			},
			apiOutput: &ssm.GetParameterOutput{
				Parameter: &ssm.Parameter{
					Value: aws.String(`{"/shmoo": "bang"}`),
				},
			},
			apiErr:         nil,
			expectError:    "",
			expectedSecret: "bang",
		},
		{
			// bad case: missing property
			apiInput: &ssm.GetParameterInput{
				Name:           aws.String("/baz"),
				WithDecryption: aws.Bool(true),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key:      "/baz",
				Property: "INVALPROP",
			},
			apiOutput: &ssm.GetParameterOutput{
				Parameter: &ssm.Parameter{
					Value: aws.String(`{"/shmoo": "bang"}`),
				},
			},
			apiErr:         nil,
			expectError:    "key INVALPROP does not exist in secret",
			expectedSecret: "",
		},
		{
			// bad case: extract property failure due to invalid json
			apiInput: &ssm.GetParameterInput{
				Name:           aws.String("/baz"),
				WithDecryption: aws.Bool(true),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key:      "/baz",
				Property: "INVALPROP",
			},
			apiOutput: &ssm.GetParameterOutput{
				Parameter: &ssm.Parameter{
					Value: aws.String(`------`),
				},
			},
			apiErr:         nil,
			expectError:    "key INVALPROP does not exist in secret",
			expectedSecret: "",
		},
		{
			// case: parameter.Value may be nil but binary is set
			apiInput: &ssm.GetParameterInput{
				Name:           aws.String("/baz"),
				WithDecryption: aws.Bool(true),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key: "/baz",
			},
			apiOutput: &ssm.GetParameterOutput{
				Parameter: &ssm.Parameter{
					Value: nil,
				},
			},
			apiErr:         nil,
			expectError:    "parameter value is nil for key",
			expectedSecret: "",
		},
		{
			// should return err
			apiInput: &ssm.GetParameterInput{
				Name:           aws.String("/foo/bar"),
				WithDecryption: aws.Bool(true),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key: "/foo/bar",
			},
			apiOutput:   &ssm.GetParameterOutput{},
			apiErr:      fmt.Errorf("oh no"),
			expectError: "oh no",
		},
	} {
		f.WithValue(row.apiInput, row.apiOutput, row.apiErr)
		out, err := p.GetSecret(context.Background(), row.rr)
		if !ErrorContains(err, row.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", i, err.Error(), row.expectError)
		}
		if string(out) != row.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", i, row.expectedSecret, string(out))
		}
	}
}

func TestGetSecretMap(t *testing.T) {
	// good case: default version & deserialization
	setDeserialization := func(pstc *parameterstoreTestCase) {
		pstc.apiOutput.Parameter = &ssm.Parameter{
			Value: aws.String(`{"foo":"bar"}`),
		}
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
		pstc.apiOutput.Parameter = &ssm.Parameter{
			Value: aws.String(`-----------------`),
		}
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

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}
