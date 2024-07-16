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

package secretsmanager

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
)

type secretsManagerTestCase struct {
	fakeClient     *fakesm.Client
	apiInput       *awssm.GetSecretValueInput
	apiOutput      *awssm.GetSecretValueOutput
	remoteRef      *esv1beta1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string
	// for testing secretmap
	expectedData map[string][]byte
	// for testing caching
	expectedCounter *int
}

const unexpectedErrorString = "[%d] unexpected error: %s, expected: '%s'"
const (
	tagname1  = "tagname1"
	tagvalue1 = "tagvalue1"
	tagname2  = "tagname2"
	tagvalue2 = "tagvalue2"
)

func makeValidSecretsManagerTestCase() *secretsManagerTestCase {
	smtc := secretsManagerTestCase{
		fakeClient:     fakesm.NewClient(),
		apiInput:       makeValidAPIInput(),
		remoteRef:      makeValidRemoteRef(),
		apiOutput:      makeValidAPIOutput(),
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   map[string][]byte{},
	}
	smtc.fakeClient.WithValue(smtc.apiInput, smtc.apiOutput, smtc.apiErr)
	return &smtc
}

func makeValidRemoteRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     "/baz",
		Version: "AWSCURRENT",
	}
}

func makeValidAPIInput() *awssm.GetSecretValueInput {
	return &awssm.GetSecretValueInput{
		SecretId:     aws.String("/baz"),
		VersionStage: aws.String("AWSCURRENT"),
	}
}

func makeValidAPIOutput() *awssm.GetSecretValueOutput {
	return &awssm.GetSecretValueOutput{
		SecretString: aws.String(""),
	}
}

func makeValidSecretsManagerTestCaseCustom(tweaks ...func(smtc *secretsManagerTestCase)) *secretsManagerTestCase {
	smtc := makeValidSecretsManagerTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.fakeClient.WithValue(smtc.apiInput, smtc.apiOutput, smtc.apiErr)
	return smtc
}

// This case can be shared by both GetSecret and GetSecretMap tests.
// bad case: set apiErr.
var setAPIErr = func(smtc *secretsManagerTestCase) {
	smtc.apiErr = fmt.Errorf("oh no")
	smtc.expectError = "oh no"
}

// test the sm<->aws interface
// make sure correct values are passed and errors are handled accordingly.
func TestSecretsManagerGetSecret(t *testing.T) {
	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String("testtesttest")
		smtc.expectedSecret = "testtesttest"
	}

	// good case: extract property
	// Testing that the property exists in the SecretString
	setRemoteRefPropertyExistsInKey := func(smtc *secretsManagerTestCase) {
		smtc.remoteRef.Property = "/shmoo"
		smtc.apiOutput.SecretString = aws.String(`{"/shmoo": "bang"}`)
		smtc.expectedSecret = "bang"
	}

	// bad case: missing property
	setRemoteRefMissingProperty := func(smtc *secretsManagerTestCase) {
		smtc.remoteRef.Property = "INVALPROP"
		smtc.expectError = "key INVALPROP does not exist in secret"
	}

	// bad case: extract property failure due to invalid json
	setRemoteRefMissingPropertyInvalidJSON := func(smtc *secretsManagerTestCase) {
		smtc.remoteRef.Property = "INVALPROP"
		smtc.apiOutput.SecretString = aws.String(`------`)
		smtc.expectError = "key INVALPROP does not exist in secret"
	}

	// good case: set .SecretString to nil but set binary with value
	setSecretBinaryNotSecretString := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretBinary = []byte("yesplease")
		// needs to be set as nil, empty quotes ("") is considered existing
		smtc.apiOutput.SecretString = nil
		smtc.expectedSecret = "yesplease"
	}

	// bad case: both .SecretString and .SecretBinary are nil
	setSecretBinaryAndSecretStringToNil := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretBinary = nil
		smtc.apiOutput.SecretString = nil
		smtc.expectError = "no secret string nor binary for key"
	}
	// good case: secretOut.SecretBinary JSON parsing
	setNestedSecretValueJSONParsing := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = nil
		smtc.apiOutput.SecretBinary = []byte(`{"foobar":{"baz":"nestedval"}}`)
		smtc.remoteRef.Property = "foobar.baz"
		smtc.expectedSecret = "nestedval"
	}
	// good case: secretOut.SecretBinary no JSON parsing if name on key
	setSecretValueWithDot := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = nil
		smtc.apiOutput.SecretBinary = []byte(`{"foobar.baz":"nestedval"}`)
		smtc.remoteRef.Property = "foobar.baz"
		smtc.expectedSecret = "nestedval"
	}

	// good case: custom version stage set
	setCustomVersionStage := func(smtc *secretsManagerTestCase) {
		smtc.apiInput.VersionStage = aws.String("1234")
		smtc.remoteRef.Version = "1234"
		smtc.apiOutput.SecretString = aws.String("FOOBA!")
		smtc.expectedSecret = "FOOBA!"
	}

	// good case: custom version id set
	setCustomVersionID := func(smtc *secretsManagerTestCase) {
		smtc.apiInput.VersionStage = nil
		smtc.apiInput.VersionId = aws.String("1234-5678")
		smtc.remoteRef.Version = "uuid/1234-5678"
		smtc.apiOutput.SecretString = aws.String("myvalue")
		smtc.expectedSecret = "myvalue"
	}

	fetchMetadata := func(smtc *secretsManagerTestCase) {
		smtc.remoteRef.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		describeSecretOutput := &awssm.DescribeSecretOutput{
			Tags: getTagSlice(),
		}
		smtc.fakeClient.DescribeSecretWithContextFn = fakesm.NewDescribeSecretWithContextFn(describeSecretOutput, nil)
		jsonTags, _ := util.SecretTagsToJSONString(getTagSlice())
		smtc.apiOutput.SecretString = &jsonTags
		smtc.expectedSecret = jsonTags
	}

	fetchMetadataProperty := func(smtc *secretsManagerTestCase) {
		smtc.remoteRef.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		describeSecretOutput := &awssm.DescribeSecretOutput{
			Tags: getTagSlice(),
		}
		smtc.fakeClient.DescribeSecretWithContextFn = fakesm.NewDescribeSecretWithContextFn(describeSecretOutput, nil)
		smtc.remoteRef.Property = tagname2
		jsonTags, _ := util.SecretTagsToJSONString(getTagSlice())
		smtc.apiOutput.SecretString = &jsonTags
		smtc.expectedSecret = tagvalue2
	}

	failMetadataWrongProperty := func(smtc *secretsManagerTestCase) {
		smtc.remoteRef.MetadataPolicy = esv1beta1.ExternalSecretMetadataPolicyFetch
		describeSecretOutput := &awssm.DescribeSecretOutput{
			Tags: getTagSlice(),
		}
		smtc.fakeClient.DescribeSecretWithContextFn = fakesm.NewDescribeSecretWithContextFn(describeSecretOutput, nil)
		smtc.remoteRef.Property = "fail"
		jsonTags, _ := util.SecretTagsToJSONString(getTagSlice())
		smtc.apiOutput.SecretString = &jsonTags
		smtc.expectError = "key fail does not exist in secret /baz"
	}

	successCases := []*secretsManagerTestCase{
		makeValidSecretsManagerTestCase(),
		makeValidSecretsManagerTestCaseCustom(setSecretString),
		makeValidSecretsManagerTestCaseCustom(setRemoteRefPropertyExistsInKey),
		makeValidSecretsManagerTestCaseCustom(setRemoteRefMissingProperty),
		makeValidSecretsManagerTestCaseCustom(setRemoteRefMissingPropertyInvalidJSON),
		makeValidSecretsManagerTestCaseCustom(setSecretBinaryNotSecretString),
		makeValidSecretsManagerTestCaseCustom(setSecretBinaryAndSecretStringToNil),
		makeValidSecretsManagerTestCaseCustom(setNestedSecretValueJSONParsing),
		makeValidSecretsManagerTestCaseCustom(setSecretValueWithDot),
		makeValidSecretsManagerTestCaseCustom(setCustomVersionStage),
		makeValidSecretsManagerTestCaseCustom(setCustomVersionID),
		makeValidSecretsManagerTestCaseCustom(setAPIErr),
		makeValidSecretsManagerTestCaseCustom(fetchMetadata),
		makeValidSecretsManagerTestCaseCustom(fetchMetadataProperty),
		makeValidSecretsManagerTestCaseCustom(failMetadataWrongProperty),
	}

	for k, v := range successCases {
		sm := SecretsManager{
			cache:  make(map[string]*awssm.GetSecretValueOutput),
			client: v.fakeClient,
		}
		out, err := sm.GetSecret(context.Background(), *v.remoteRef)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(unexpectedErrorString, k, err.Error(), v.expectError)
		}
		if err == nil && string(out) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out))
		}
	}
}
func TestCaching(t *testing.T) {
	fakeClient := fakesm.NewClient()

	// good case: first call, since we are using the same key, results should be cached and the counter should not go
	// over 1
	firstCall := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`{"foo":"bar", "bar":"vodka"}`)
		smtc.remoteRef.Property = "foo"
		smtc.expectedSecret = "bar"
		smtc.expectedCounter = aws.Int(1)
		smtc.fakeClient = fakeClient
	}
	secondCall := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`{"foo":"bar", "bar":"vodka"}`)
		smtc.remoteRef.Property = "bar"
		smtc.expectedSecret = "vodka"
		smtc.expectedCounter = aws.Int(1)
		smtc.fakeClient = fakeClient
	}
	notCachedCall := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`{"sheldon":"bazinga", "bar":"foo"}`)
		smtc.remoteRef.Property = "sheldon"
		smtc.expectedSecret = "bazinga"
		smtc.expectedCounter = aws.Int(2)
		smtc.fakeClient = fakeClient
		smtc.apiInput.SecretId = aws.String("xyz")
		smtc.remoteRef.Key = "xyz" // it should reset the cache since the key is different
	}

	cachedCases := []*secretsManagerTestCase{
		makeValidSecretsManagerTestCaseCustom(firstCall),
		makeValidSecretsManagerTestCaseCustom(firstCall),
		makeValidSecretsManagerTestCaseCustom(secondCall),
		makeValidSecretsManagerTestCaseCustom(notCachedCall),
	}
	sm := SecretsManager{
		cache: make(map[string]*awssm.GetSecretValueOutput),
	}
	for k, v := range cachedCases {
		sm.client = v.fakeClient
		out, err := sm.GetSecret(context.Background(), *v.remoteRef)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(unexpectedErrorString, k, err.Error(), v.expectError)
		}
		if err == nil && string(out) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out))
		}
		if v.expectedCounter != nil && v.fakeClient.ExecutionCounter != *v.expectedCounter {
			t.Errorf("[%d] unexpected counter value: expected %d, got %d", k, v.expectedCounter, v.fakeClient.ExecutionCounter)
		}
	}
}

func TestGetSecretMap(t *testing.T) {
	// good case: default version & deserialization
	setDeserialization := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`{"foo":"bar"}`)
		smtc.expectedData["foo"] = []byte("bar")
	}

	// good case: nested json
	setNestedJSON := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`{"foobar":{"baz":"nestedval"}}`)
		smtc.expectedData["foobar"] = []byte("{\"baz\":\"nestedval\"}")
	}

	// good case: caching
	cachedMap := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`{"foo":"bar", "plus": "one"}`)
		smtc.expectedData["foo"] = []byte("bar")
		smtc.expectedData["plus"] = []byte("one")
		smtc.expectedCounter = aws.Int(1)
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`-----------------`)
		smtc.expectError = "unable to unmarshal secret"
	}

	successCases := []*secretsManagerTestCase{
		makeValidSecretsManagerTestCaseCustom(setDeserialization),
		makeValidSecretsManagerTestCaseCustom(setNestedJSON),
		makeValidSecretsManagerTestCaseCustom(setAPIErr),
		makeValidSecretsManagerTestCaseCustom(setInvalidJSON),
		makeValidSecretsManagerTestCaseCustom(cachedMap),
	}

	for k, v := range successCases {
		sm := SecretsManager{
			cache:  make(map[string]*awssm.GetSecretValueOutput),
			client: v.fakeClient,
		}
		out, err := sm.GetSecretMap(context.Background(), *v.remoteRef)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(unexpectedErrorString, k, err.Error(), v.expectError)
		}
		if err == nil && !cmp.Equal(out, v.expectedData) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, v.expectedData, out)
		}
		if v.expectedCounter != nil && v.fakeClient.ExecutionCounter != *v.expectedCounter {
			t.Errorf("[%d] unexpected counter value: expected %d, got %d", k, v.expectedCounter, v.fakeClient.ExecutionCounter)
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

func TestSetSecret(t *testing.T) {
	managedBy := managedBy
	notManagedBy := "not-managed-by"
	secretKey := "fake-secret-key"
	secretValue := []byte("fake-value")
	fakeSecret := &corev1.Secret{
		Data: map[string][]byte{
			secretKey: secretValue,
		},
	}
	externalSecrets := externalSecrets
	noPermission := errors.New("no permission")
	arn := "arn:aws:secretsmanager:us-east-1:702902267788:secret:foo-bar5-Robbgh"

	getSecretCorrectErr := awssm.ResourceNotFoundException{}
	getSecretWrongErr := awssm.InvalidRequestException{}

	secretOutput := &awssm.CreateSecretOutput{
		ARN: &arn,
	}

	externalSecretsTag := []*awssm.Tag{
		{
			Key:   &managedBy,
			Value: &externalSecrets,
		},
	}

	externalSecretsTagFaulty := []*awssm.Tag{
		{
			Key:   &notManagedBy,
			Value: &externalSecrets,
		},
	}

	tagSecretOutput := &awssm.DescribeSecretOutput{
		ARN:  &arn,
		Tags: externalSecretsTag,
	}

	tagSecretOutputFaulty := &awssm.DescribeSecretOutput{
		ARN:  &arn,
		Tags: externalSecretsTagFaulty,
	}

	initialVersion := "00000000-0000-0000-0000-000000000001"
	defaultVersion := "00000000-0000-0000-0000-000000000002"
	defaultUpdatedVersion := "00000000-0000-0000-0000-000000000003"
	randomUUIDVersion := "c2812e8d-84ce-4858-abec-7163d8ab312b"
	randomUUIDVersionIncremented := "c2812e8d-84ce-4858-abec-7163d8ab312c"
	unparsableVersion := "IAM UNPARSABLE"

	secretValueOutput := &awssm.GetSecretValueOutput{
		ARN:       &arn,
		VersionId: &defaultVersion,
	}

	secretValueOutput2 := &awssm.GetSecretValueOutput{
		ARN:          &arn,
		SecretBinary: secretValue,
		VersionId:    &defaultVersion,
	}

	type params struct {
		s       string
		b       []byte
		version *string
	}
	secretValueOutputFrom := func(params params) *awssm.GetSecretValueOutput {
		var version *string
		if params.version == nil {
			version = &defaultVersion
		} else {
			version = params.version
		}

		return &awssm.GetSecretValueOutput{
			ARN:          &arn,
			SecretString: &params.s,
			SecretBinary: params.b,
			VersionId:    version,
		}
	}
	blankSecretValueOutput := &awssm.GetSecretValueOutput{}

	putSecretOutput := &awssm.PutSecretValueOutput{
		ARN: &arn,
	}

	pushSecretDataWithoutProperty := fake.PushSecretData{SecretKey: secretKey, RemoteKey: "fake-key", Property: ""}
	pushSecretDataWithMetadata := fake.PushSecretData{SecretKey: secretKey, RemoteKey: "fake-key", Property: "", Metadata: &apiextensionsv1.JSON{
		Raw: []byte(`{"secretPushFormat": "string"}`),
	}}
	pushSecretDataWithProperty := fake.PushSecretData{SecretKey: secretKey, RemoteKey: "fake-key", Property: "other-fake-property"}

	type args struct {
		store          *esv1beta1.AWSProvider
		client         fakesm.Client
		pushSecretData fake.PushSecretData
	}

	type want struct {
		err error
	}
	tests := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"SetSecretSucceedsWithExistingSecret": {
			reason: "a secret can be pushed to aws secrets manager when it already exists",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutput, nil),
					CreateSecretWithContextFn:   fakesm.NewCreateSecretWithContextFn(secretOutput, nil),
					PutSecretValueWithContextFn: fakesm.NewPutSecretValueWithContextFn(putSecretOutput, nil),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(tagSecretOutput, nil),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretSucceedsWithExistingSecretAndStringFormat": {
			reason: "a secret can be pushed to aws secrets manager when it already exists",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutput, nil),
					CreateSecretWithContextFn:   fakesm.NewCreateSecretWithContextFn(secretOutput, nil),
					PutSecretValueWithContextFn: fakesm.NewPutSecretValueWithContextFn(putSecretOutput, nil),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(tagSecretOutput, nil),
				},
				pushSecretData: pushSecretDataWithMetadata,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretSucceedsWithNewSecret": {
			reason: "a secret can be pushed to aws secrets manager if it doesn't already exist",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(blankSecretValueOutput, &getSecretCorrectErr),
					CreateSecretWithContextFn:   fakesm.NewCreateSecretWithContextFn(secretOutput, nil),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithPropertySucceedsWithNewSecret": {
			reason: "if a new secret is pushed to aws sm and a pushSecretData property is specified, create a json secret with the pushSecretData property as a key",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(blankSecretValueOutput, &getSecretCorrectErr),
					CreateSecretWithContextFn:   fakesm.NewCreateSecretWithContextFn(secretOutput, nil, []byte(`{"other-fake-property":"fake-value"}`)),
				},
				pushSecretData: pushSecretDataWithProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithPropertySucceedsWithExistingSecretAndNewPropertyBinary": {
			reason: "when a pushSecretData property is specified, this property will be added to the sm secret if it is currently absent (sm secret is binary)",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutputFrom(params{b: []byte((`{"fake-property":"fake-value"}`))}), nil),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(tagSecretOutput, nil),
					PutSecretValueWithContextFn: fakesm.NewPutSecretValueWithContextFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`{"fake-property":"fake-value","other-fake-property":"fake-value"}`),
						Version:      &defaultUpdatedVersion,
					}),
				},
				pushSecretData: pushSecretDataWithProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithPropertySucceedsWithExistingSecretAndRandomUUIDVersion": {
			reason: "When a secret version is not specified, the client sets a random uuid by default. We should treat a version that can't be parsed to an int as not having a version",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutputFrom(params{
						b:       []byte((`{"fake-property":"fake-value"}`)),
						version: &randomUUIDVersion,
					}), nil),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(tagSecretOutput, nil),
					PutSecretValueWithContextFn: fakesm.NewPutSecretValueWithContextFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`{"fake-property":"fake-value","other-fake-property":"fake-value"}`),
						Version:      &randomUUIDVersionIncremented,
					}),
				},
				pushSecretData: pushSecretDataWithProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithPropertySucceedsWithExistingSecretAndVersionThatCantBeParsed": {
			reason: "A manually set secret version doesn't have to be a UUID, we only support UUID secret versions though",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutputFrom(params{
						b:       []byte((`{"fake-property":"fake-value"}`)),
						version: &unparsableVersion,
					}), nil),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(tagSecretOutput, nil),
					PutSecretValueWithContextFn: fakesm.NewPutSecretValueWithContextFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`{"fake-property":"fake-value","other-fake-property":"fake-value"}`),
						Version:      &initialVersion,
					}),
				},
				pushSecretData: pushSecretDataWithProperty,
			},
			want: want{
				err: fmt.Errorf("expected secret version in AWS SSM to be a UUID but got '%s'", unparsableVersion),
			},
		},
		"SetSecretWithPropertySucceedsWithExistingSecretAndAbsentVersion": {
			reason: "When a secret version is not specified, set it to 1",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(&awssm.GetSecretValueOutput{
						ARN:          &arn,
						SecretBinary: []byte((`{"fake-property":"fake-value"}`)),
					}, nil),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(tagSecretOutput, nil),
					PutSecretValueWithContextFn: fakesm.NewPutSecretValueWithContextFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`{"fake-property":"fake-value","other-fake-property":"fake-value"}`),
						Version:      &initialVersion,
					}),
				},
				pushSecretData: pushSecretDataWithProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithPropertySucceedsWithExistingSecretAndNewPropertyString": {
			reason: "when a pushSecretData property is specified, this property will be added to the sm secret if it is currently absent (sm secret is a string)",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutputFrom(params{s: `{"fake-property":"fake-value"}`}), nil),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(tagSecretOutput, nil),
					PutSecretValueWithContextFn: fakesm.NewPutSecretValueWithContextFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`{"fake-property":"fake-value","other-fake-property":"fake-value"}`),
						Version:      &defaultUpdatedVersion,
					}),
				},
				pushSecretData: pushSecretDataWithProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithPropertySucceedsWithExistingSecretAndNewPropertyWithDot": {
			reason: "when a pushSecretData property is specified, this property will be added to the sm secret if it is currently absent (pushSecretData property is a sub-object)",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutputFrom(params{s: `{"fake-property":{"fake-property":"fake-value"}}`}), nil),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(tagSecretOutput, nil),
					PutSecretValueWithContextFn: fakesm.NewPutSecretValueWithContextFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`{"fake-property":{"fake-property":"fake-value","other-fake-property":"fake-value"}}`),
						Version:      &defaultUpdatedVersion,
					}),
				},
				pushSecretData: fake.PushSecretData{SecretKey: secretKey, RemoteKey: "fake-key", Property: "fake-property.other-fake-property"},
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithPropertyFailsExistingNonJsonSecret": {
			reason: "setting a pushSecretData property is only supported for json secrets",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutputFrom(params{s: `non-json-secret`}), nil),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(tagSecretOutput, nil),
				},
				pushSecretData: pushSecretDataWithProperty,
			},
			want: want{
				err: errors.New("PushSecret for aws secrets manager with a pushSecretData property requires a json secret"),
			},
		},
		"SetSecretCreateSecretFails": {
			reason: "CreateSecretWithContext returns an error if it fails",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(blankSecretValueOutput, &getSecretCorrectErr),
					CreateSecretWithContextFn:   fakesm.NewCreateSecretWithContextFn(nil, noPermission),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: noPermission,
			},
		},
		"SetSecretGetSecretFails": {
			reason: "GetSecretValueWithContext returns an error if it fails",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(blankSecretValueOutput, noPermission),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: noPermission,
			},
		},
		"SetSecretWillNotPushSameSecret": {
			reason: "secret with the same value will not be pushed",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutput2, nil),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(tagSecretOutput, nil),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretPutSecretValueFails": {
			reason: "PutSecretValueWithContext returns an error if it fails",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutput, nil),
					PutSecretValueWithContextFn: fakesm.NewPutSecretValueWithContextFn(nil, noPermission),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(tagSecretOutput, nil),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: noPermission,
			},
		},
		"SetSecretWrongGetSecretErrFails": {
			reason: "GetSecretValueWithContext errors out when anything except awssm.ErrCodeResourceNotFoundException",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(blankSecretValueOutput, &getSecretWrongErr),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: &getSecretWrongErr,
			},
		},
		"SetSecretDescribeSecretFails": {
			reason: "secret cannot be described",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutput, nil),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(nil, noPermission),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: noPermission,
			},
		},
		"SetSecretDoesNotOverwriteUntaggedSecret": {
			reason: "secret cannot be described",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutput, nil),
					DescribeSecretWithContextFn: fakesm.NewDescribeSecretWithContextFn(tagSecretOutputFaulty, nil),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: fmt.Errorf("secret not managed by external-secrets"),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			sm := SecretsManager{
				client: &tc.args.client,
			}
			err := sm.PushSecret(context.Background(), fakeSecret, tc.args.pushSecretData)

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

func TestDeleteSecret(t *testing.T) {
	fakeClient := fakesm.Client{}
	managed := managedBy
	manager := externalSecrets
	secretTag := awssm.Tag{
		Key:   &managed,
		Value: &manager,
	}
	type args struct {
		client               fakesm.Client
		config               esv1beta1.SecretsManager
		getSecretOutput      *awssm.GetSecretValueOutput
		describeSecretOutput *awssm.DescribeSecretOutput
		deleteSecretOutput   *awssm.DeleteSecretOutput
		getSecretErr         error
		describeSecretErr    error
		deleteSecretErr      error
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

				client:          fakeClient,
				config:          esv1beta1.SecretsManager{},
				getSecretOutput: &awssm.GetSecretValueOutput{},
				describeSecretOutput: &awssm.DescribeSecretOutput{
					Tags: []*awssm.Tag{&secretTag},
				},
				deleteSecretOutput: &awssm.DeleteSecretOutput{},
				getSecretErr:       nil,
				describeSecretErr:  nil,
				deleteSecretErr:    nil,
			},
			want: want{
				err: nil,
			},
			reason: "",
		},
		"Deletes Successfully with ForceDeleteWithoutRecovery": {
			args: args{

				client: fakeClient,
				config: esv1beta1.SecretsManager{
					ForceDeleteWithoutRecovery: true,
				},
				getSecretOutput: &awssm.GetSecretValueOutput{},
				describeSecretOutput: &awssm.DescribeSecretOutput{
					Tags: []*awssm.Tag{&secretTag},
				},
				deleteSecretOutput: &awssm.DeleteSecretOutput{
					DeletionDate: aws.Time(time.Now()),
				},
				getSecretErr:      nil,
				describeSecretErr: nil,
				deleteSecretErr:   nil,
			},
			want: want{
				err: nil,
			},
			reason: "",
		},
		"Not Managed by ESO": {
			args: args{

				client:          fakeClient,
				config:          esv1beta1.SecretsManager{},
				getSecretOutput: &awssm.GetSecretValueOutput{},
				describeSecretOutput: &awssm.DescribeSecretOutput{
					Tags: []*awssm.Tag{},
				},
				deleteSecretOutput: &awssm.DeleteSecretOutput{},
				getSecretErr:       nil,
				describeSecretErr:  nil,
				deleteSecretErr:    nil,
			},
			want: want{
				err: nil,
			},
			reason: "",
		},
		"Invalid Recovery Window": {
			args: args{

				client: fakesm.Client{},
				config: esv1beta1.SecretsManager{
					RecoveryWindowInDays: 1,
				},
				getSecretOutput: &awssm.GetSecretValueOutput{},
				describeSecretOutput: &awssm.DescribeSecretOutput{
					Tags: []*awssm.Tag{&secretTag},
				},
				deleteSecretOutput: &awssm.DeleteSecretOutput{},
				getSecretErr:       nil,
				describeSecretErr:  nil,
				deleteSecretErr:    nil,
			},
			want: want{
				err: errors.New("invalid DeleteSecretInput: RecoveryWindowInDays must be between 7 and 30 days"),
			},
			reason: "",
		},
		"RecoveryWindowInDays is supplied with ForceDeleteWithoutRecovery": {
			args: args{

				client: fakesm.Client{},
				config: esv1beta1.SecretsManager{
					RecoveryWindowInDays:       7,
					ForceDeleteWithoutRecovery: true,
				},
				getSecretOutput: &awssm.GetSecretValueOutput{},
				describeSecretOutput: &awssm.DescribeSecretOutput{
					Tags: []*awssm.Tag{&secretTag},
				},
				deleteSecretOutput: &awssm.DeleteSecretOutput{},
				getSecretErr:       nil,
				describeSecretErr:  nil,
				deleteSecretErr:    nil,
			},
			want: want{
				err: errors.New("invalid DeleteSecretInput: ForceDeleteWithoutRecovery conflicts with RecoveryWindowInDays"),
			},
			reason: "",
		},
		"Failed to get Tags": {
			args: args{

				client:               fakeClient,
				config:               esv1beta1.SecretsManager{},
				getSecretOutput:      &awssm.GetSecretValueOutput{},
				describeSecretOutput: nil,
				deleteSecretOutput:   nil,
				getSecretErr:         nil,
				describeSecretErr:    errors.New("failed to get tags"),
				deleteSecretErr:      nil,
			},
			want: want{
				err: errors.New("failed to get tags"),
			},
			reason: "",
		},
		"Secret Not Found": {
			args: args{
				client:               fakeClient,
				config:               esv1beta1.SecretsManager{},
				getSecretOutput:      nil,
				describeSecretOutput: nil,
				deleteSecretOutput:   nil,
				getSecretErr:         awserr.New(awssm.ErrCodeResourceNotFoundException, "not here, sorry dude", nil),
				describeSecretErr:    nil,
				deleteSecretErr:      nil,
			},
			want: want{
				err: nil,
			},
		},
		"Not expected AWS error": {
			args: args{
				client:               fakeClient,
				config:               esv1beta1.SecretsManager{},
				getSecretOutput:      nil,
				describeSecretOutput: nil,
				deleteSecretOutput:   nil,
				getSecretErr:         awserr.New(awssm.ErrCodeEncryptionFailure, "aws unavailable", nil),
				describeSecretErr:    nil,
				deleteSecretErr:      nil,
			},
			want: want{
				err: errors.New("aws unavailable"),
			},
		},
		"unexpected error": {
			args: args{
				client:               fakeClient,
				config:               esv1beta1.SecretsManager{},
				getSecretOutput:      nil,
				describeSecretOutput: nil,
				deleteSecretOutput:   nil,
				getSecretErr:         errors.New("timeout"),
				describeSecretErr:    nil,
				deleteSecretErr:      nil,
			},
			want: want{
				err: errors.New("timeout"),
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ref := fake.PushSecretData{RemoteKey: "fake-key"}
			sm := SecretsManager{
				client: &tc.args.client,
				config: &tc.args.config,
			}
			tc.args.client.GetSecretValueWithContextFn = fakesm.NewGetSecretValueWithContextFn(tc.args.getSecretOutput, tc.args.getSecretErr)
			tc.args.client.DescribeSecretWithContextFn = fakesm.NewDescribeSecretWithContextFn(tc.args.describeSecretOutput, tc.args.describeSecretErr)
			tc.args.client.DeleteSecretWithContextFn = fakesm.NewDeleteSecretWithContextFn(tc.args.deleteSecretOutput, tc.args.deleteSecretErr)
			err := sm.DeleteSecret(context.TODO(), ref)
			t.Logf("DeleteSecret error: %v", err)

			// Error nil XOR tc.want.err nil
			if ((err == nil) || (tc.want.err == nil)) && !((err == nil) && (tc.want.err == nil)) {
				t.Errorf("\nTesting DeleteSecret:\nName: %v\nReason: %v\nWant error: %v\nGot error: %v", name, tc.reason, tc.want.err, err)
			}

			// if errors are the same type but their contents do not match
			if err != nil && tc.want.err != nil {
				if !strings.Contains(err.Error(), tc.want.err.Error()) {
					t.Errorf("\nTesting DeleteSecret:\nName: %v\nReason: %v\nWant error: %v\nGot error got nil", name, tc.reason, tc.want.err)
				}
			}
		})
	}
}
func makeValidSecretStore() *esv1beta1.SecretStore {
	return &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-secret-store",
			Namespace: "default",
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				AWS: &esv1beta1.AWSProvider{
					Service: esv1beta1.AWSServiceSecretsManager,
					Region:  "eu-west-2",
				},
			},
		},
	}
}

func getTagSlice() []*awssm.Tag {
	tagKey1 := tagname1
	tagValue1 := tagvalue1
	tagKey2 := tagname2
	tagValue2 := tagvalue2

	return []*awssm.Tag{
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
func TestSecretsManagerGetAllSecrets(t *testing.T) {
	ctx := context.Background()

	errBoom := errors.New("boom")
	secretName := "my-secret"
	secretVersion := "AWSCURRENT"
	secretPath := "/path/to/secret"
	secretValue := "secret value"
	secretTags := map[string]string{
		"foo": "bar",
	}
	// Test cases
	testCases := []struct {
		name string
		ref  esv1beta1.ExternalSecretFind

		secretName    string
		secretVersion string
		secretValue   string
		fetchError    error
		listSecretsFn func(ctx context.Context, input *awssm.ListSecretsInput, opts ...request.Option) (*awssm.ListSecretsOutput, error)

		expectedData  map[string][]byte
		expectedError string
	}{
		{
			name: "Matching secrets found",
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{
					RegExp: secretName,
				},
				Path: ptr.To(secretPath),
			},
			secretName:    secretName,
			secretVersion: secretVersion,
			secretValue:   secretValue,
			listSecretsFn: func(ctx context.Context, input *awssm.ListSecretsInput, opts ...request.Option) (*awssm.ListSecretsOutput, error) {
				assert.Len(t, input.Filters, 1)
				assert.Equal(t, "name", *input.Filters[0].Key)
				assert.Equal(t, secretPath, *input.Filters[0].Values[0])
				return &awssm.ListSecretsOutput{
					SecretList: []*awssm.SecretListEntry{
						{
							Name: ptr.To(secretName),
						},
					},
				}, nil
			},
			expectedData: map[string][]byte{
				secretName: []byte(secretValue),
			},
			expectedError: "",
		},
		{
			name: "Error occurred while fetching secret value",
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{
					RegExp: secretName,
				},
				Path: ptr.To(secretPath),
			},
			secretName:    secretName,
			secretVersion: secretVersion,
			secretValue:   secretValue,
			fetchError:    errBoom,
			listSecretsFn: func(ctx context.Context, input *awssm.ListSecretsInput, opts ...request.Option) (*awssm.ListSecretsOutput, error) {
				return &awssm.ListSecretsOutput{
					SecretList: []*awssm.SecretListEntry{
						{
							Name: ptr.To(secretName),
						},
					},
				}, nil
			},
			expectedData:  nil,
			expectedError: errBoom.Error(),
		},
		{
			name: "regexp: error occurred while listing secrets",
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{
					RegExp: secretName,
				},
			},
			listSecretsFn: func(ctx context.Context, input *awssm.ListSecretsInput, opts ...request.Option) (*awssm.ListSecretsOutput, error) {
				return nil, errBoom
			},
			expectedData:  nil,
			expectedError: errBoom.Error(),
		},
		{
			name: "regep: no matching secrets found",
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{
					RegExp: secretName,
				},
			},
			listSecretsFn: func(ctx context.Context, input *awssm.ListSecretsInput, opts ...request.Option) (*awssm.ListSecretsOutput, error) {
				return &awssm.ListSecretsOutput{
					SecretList: []*awssm.SecretListEntry{
						{
							Name: ptr.To("other-secret"),
						},
					},
				}, nil
			},
			expectedData:  make(map[string][]byte),
			expectedError: "",
		},
		{
			name: "invalid regexp",
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{
					RegExp: "[",
				},
			},
			expectedData:  nil,
			expectedError: "could not compile find.name.regexp [[]: error parsing regexp: missing closing ]: `[`",
		},

		{
			name: "tags: Matching secrets found",
			ref: esv1beta1.ExternalSecretFind{
				Tags: secretTags,
			},
			secretName:    secretName,
			secretVersion: secretVersion,
			secretValue:   secretValue,
			listSecretsFn: func(ctx context.Context, input *awssm.ListSecretsInput, opts ...request.Option) (*awssm.ListSecretsOutput, error) {
				assert.Len(t, input.Filters, 2)
				assert.Equal(t, "tag-key", *input.Filters[0].Key)
				assert.Equal(t, "foo", *input.Filters[0].Values[0])
				assert.Equal(t, "tag-value", *input.Filters[1].Key)
				assert.Equal(t, "bar", *input.Filters[1].Values[0])
				return &awssm.ListSecretsOutput{
					SecretList: []*awssm.SecretListEntry{
						{
							Name: ptr.To(secretName),
						},
					},
				}, nil
			},
			expectedData: map[string][]byte{
				secretName: []byte(secretValue),
			},
			expectedError: "",
		},
		{
			name: "tags: error occurred while fetching secret value",
			ref: esv1beta1.ExternalSecretFind{
				Tags: secretTags,
			},
			secretName:    secretName,
			secretVersion: secretVersion,
			secretValue:   secretValue,
			fetchError:    errBoom,
			listSecretsFn: func(ctx context.Context, input *awssm.ListSecretsInput, opts ...request.Option) (*awssm.ListSecretsOutput, error) {
				return &awssm.ListSecretsOutput{
					SecretList: []*awssm.SecretListEntry{
						{
							Name: ptr.To(secretName),
						},
					},
				}, nil
			},
			expectedData:  nil,
			expectedError: errBoom.Error(),
		},
		{
			name: "tags: error occurred while listing secrets",
			ref: esv1beta1.ExternalSecretFind{
				Tags: secretTags,
			},
			listSecretsFn: func(ctx context.Context, input *awssm.ListSecretsInput, opts ...request.Option) (*awssm.ListSecretsOutput, error) {
				return nil, errBoom
			},
			expectedData:  nil,
			expectedError: errBoom.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fc := fakesm.NewClient()
			fc.ListSecretsFn = tc.listSecretsFn
			fc.WithValue(&awssm.GetSecretValueInput{
				SecretId:     ptr.To(tc.secretName),
				VersionStage: ptr.To(tc.secretVersion),
			}, &awssm.GetSecretValueOutput{
				Name:          ptr.To(tc.secretName),
				VersionStages: []*string{ptr.To(tc.secretVersion)},
				SecretBinary:  []byte(tc.secretValue),
			}, tc.fetchError)
			sm := SecretsManager{
				client: fc,
				cache:  make(map[string]*awssm.GetSecretValueOutput),
			}
			data, err := sm.GetAllSecrets(ctx, tc.ref)
			if err != nil && err.Error() != tc.expectedError {
				t.Errorf("unexpected error: got %v, want %v", err, tc.expectedError)
			}
			if !reflect.DeepEqual(data, tc.expectedData) {
				t.Errorf("unexpected data: got %v, want %v", data, tc.expectedData)
			}
		})
	}
}

func TestSecretsManagerValidate(t *testing.T) {
	type fields struct {
		sess         *session.Session
		referentAuth bool
	}
	validSession, _ := session.NewSession(aws.NewConfig().WithCredentials(credentials.NewStaticCredentials("fake", "fake", "fake")))
	invalidSession, _ := session.NewSession(aws.NewConfig().WithCredentials(credentials.NewCredentials(&FakeCredProvider{
		retrieveFunc: func() (credentials.Value, error) {
			return credentials.Value{}, errors.New("invalid credentials")
		},
	})))
	tests := []struct {
		name    string
		fields  fields
		want    esv1beta1.ValidationResult
		wantErr bool
	}{
		{
			name: "ReferentAuth should always return unknown",
			fields: fields{
				referentAuth: true,
			},
			want: esv1beta1.ValidationResultUnknown,
		},
		{
			name: "Valid credentials should return ready",
			fields: fields{
				sess: validSession,
			},
			want: esv1beta1.ValidationResultReady,
		},
		{
			name: "Invalid credentials should return error",
			fields: fields{
				sess: invalidSession,
			},
			want:    esv1beta1.ValidationResultError,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SecretsManager{
				sess:         tt.fields.sess,
				referentAuth: tt.fields.referentAuth,
			}
			got, err := sm.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SecretsManager.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SecretsManager.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestSecretExists(t *testing.T) {
	arn := "arn:aws:secretsmanager:us-east-1:702902267788:secret:foo-bar5-Robbgh"
	defaultVersion := "00000000-0000-0000-0000-000000000002"
	secretValueOutput := &awssm.GetSecretValueOutput{
		ARN:       &arn,
		VersionId: &defaultVersion,
	}

	blankSecretValueOutput := &awssm.GetSecretValueOutput{}

	getSecretCorrectErr := awssm.ResourceNotFoundException{}
	getSecretWrongErr := awssm.InvalidRequestException{}

	pushSecretDataWithoutProperty := fake.PushSecretData{SecretKey: "fake-secret-key", RemoteKey: "fake-key", Property: ""}

	type args struct {
		store          *esv1beta1.AWSProvider
		client         fakesm.Client
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
		"SecretExistsReturnsTrueForExistingSecret": {
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(secretValueOutput, nil),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err:       nil,
				wantError: true,
			},
		},
		"SecretExistsReturnsFalseForNonExistingSecret": {
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(blankSecretValueOutput, &getSecretCorrectErr),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err:       nil,
				wantError: false,
			},
		},
		"SecretExistsReturnsFalseForErroredSecret": {
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueWithContextFn: fakesm.NewGetSecretValueWithContextFn(blankSecretValueOutput, &getSecretWrongErr),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err:       &getSecretWrongErr,
				wantError: false,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			sm := &SecretsManager{
				client: &tc.args.client,
			}
			got, err := sm.SecretExists(context.Background(), tc.args.pushSecretData)

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

// FakeCredProvider implements the AWS credentials.Provider interface
// It is used to inject an error into the AWS session to cause a
// validation error.
type FakeCredProvider struct {
	retrieveFunc func() (credentials.Value, error)
}

func (f *FakeCredProvider) Retrieve() (credentials.Value, error) {
	return f.retrieveFunc()
}

func (f *FakeCredProvider) IsExpired() bool {
	return true
}
