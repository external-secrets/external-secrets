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

package secretsmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/external-secrets/external-secrets/pkg/esutils/metadata"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
)

type secretsManagerTestCase struct {
	fakeClient     *fakesm.Client
	apiInput       *awssm.GetSecretValueInput
	apiOutput      *awssm.GetSecretValueOutput
	remoteRef      *esv1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string
	// for testing secretmap
	expectedData map[string][]byte
	// for testing caching
	expectedCounter *int
	prefix          string
}

const unexpectedErrorString = "[%d] unexpected error: %s, expected: '%s'"
const (
	tagname1  = "tagname1"
	tagvalue1 = "tagvalue1"
	tagname2  = "tagname2"
	tagvalue2 = "tagvalue2"
	fakeKey   = "fake-key"
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

func makeValidRemoteRef() *esv1.ExternalSecretDataRemoteRef {
	return &esv1.ExternalSecretDataRemoteRef{
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
	smtc.apiErr = errors.New("oh no")
	smtc.expectError = "oh no"
}

func TestSecretsManagerResolver(t *testing.T) {
	endpointEnvKey := SecretsManagerEndpointEnv
	endpointURL := "http://sm.foo"

	t.Setenv(endpointEnvKey, endpointURL)

	f, err := customEndpointResolver{}.ResolveEndpoint(context.Background(), awssm.EndpointParameters{})

	assert.Nil(t, err)
	assert.Equal(t, endpointURL, f.URI.String())
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

	// good case: key is passed in with prefix
	setSecretStringWithPrefix := func(smtc *secretsManagerTestCase) {
		smtc.remoteRef.Key = "secret-key"
		smtc.apiInput = &awssm.GetSecretValueInput{
			SecretId:     aws.String("my-prefix/secret-key"),
			VersionStage: aws.String("AWSCURRENT"),
		}
		smtc.prefix = "my-prefix/"
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
		smtc.remoteRef.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		describeSecretOutput := &awssm.DescribeSecretOutput{
			Tags: getTagSlice(),
		}
		smtc.fakeClient.DescribeSecretFn = fakesm.NewDescribeSecretFn(describeSecretOutput, nil)
		jsonTags, _ := awsutil.SecretTagsToJSONString(getTagSlice())
		smtc.apiOutput.SecretString = &jsonTags
		smtc.expectedSecret = jsonTags
	}

	fetchMetadataProperty := func(smtc *secretsManagerTestCase) {
		smtc.remoteRef.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		describeSecretOutput := &awssm.DescribeSecretOutput{
			Tags: getTagSlice(),
		}
		smtc.fakeClient.DescribeSecretFn = fakesm.NewDescribeSecretFn(describeSecretOutput, nil)
		smtc.remoteRef.Property = tagname2
		jsonTags, _ := awsutil.SecretTagsToJSONString(getTagSlice())
		smtc.apiOutput.SecretString = &jsonTags
		smtc.expectedSecret = tagvalue2
	}

	failMetadataWrongProperty := func(smtc *secretsManagerTestCase) {
		smtc.remoteRef.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		describeSecretOutput := &awssm.DescribeSecretOutput{
			Tags: getTagSlice(),
		}
		smtc.fakeClient.DescribeSecretFn = fakesm.NewDescribeSecretFn(describeSecretOutput, nil)
		smtc.remoteRef.Property = "fail"
		jsonTags, _ := awsutil.SecretTagsToJSONString(getTagSlice())
		smtc.apiOutput.SecretString = &jsonTags
		smtc.expectError = "key fail does not exist in secret /baz"
	}

	successCases := []*secretsManagerTestCase{
		makeValidSecretsManagerTestCase(),
		makeValidSecretsManagerTestCaseCustom(setSecretString),
		makeValidSecretsManagerTestCaseCustom(setSecretStringWithPrefix),
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
			prefix: v.prefix,
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

	getSecretCorrectErr := types.ResourceNotFoundException{}
	getSecretWrongErr := types.InvalidRequestException{}

	secretOutput := &awssm.CreateSecretOutput{
		ARN: &arn,
	}

	externalSecretsTag := []types.Tag{
		{
			Key:   &managedBy,
			Value: &externalSecrets,
		},
		{
			Key:   ptr.To("taname1"),
			Value: ptr.To("tagvalue1"),
		},
	}

	externalSecretsTagFaulty := []types.Tag{
		{
			Key:   &notManagedBy,
			Value: &externalSecrets,
		},
	}

	tagSecretOutputNoVersions := &awssm.DescribeSecretOutput{
		ARN:  &arn,
		Tags: externalSecretsTag,
	}

	defaultVersion := "00000000-0000-0000-0000-000000000002"

	tagSecretOutput := &awssm.DescribeSecretOutput{
		ARN:  &arn,
		Tags: externalSecretsTag,
		VersionIdsToStages: map[string][]string{
			defaultVersion: {"AWSCURRENT"},
		},
	}

	tagSecretOutputFaulty := &awssm.DescribeSecretOutput{
		ARN:  &arn,
		Tags: externalSecretsTagFaulty,
	}

	tagSecretOutputFrom := func(versionId string) *awssm.DescribeSecretOutput {
		return &awssm.DescribeSecretOutput{
			ARN:  &arn,
			Tags: externalSecretsTag,
			VersionIdsToStages: map[string][]string{
				versionId: {"AWSCURRENT"},
			},
		}
	}

	initialVersion := "00000000-0000-0000-0000-000000000001"
	defaultUpdatedVersion := "6c70d57a-f53d-bf4d-9525-3503dd5abe8c"
	randomUUIDVersion := "9d6202c2-c216-433e-a2f0-5836c4f025af"
	randomUUIDVersionIncremented := "4346824b-7da1-4d82-addf-dee197fd5d71"
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
	blankDescribeSecretOutput := &awssm.DescribeSecretOutput{}

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

	putSecretOutput := &awssm.PutSecretValueOutput{
		ARN: &arn,
	}

	pushSecretDataWithoutProperty := fake.PushSecretData{SecretKey: secretKey, RemoteKey: fakeKey, Property: ""}
	pushSecretDataWithoutSecretKey := fake.PushSecretData{RemoteKey: fakeKey, Property: ""}
	pushSecretDataWithMetadata := fake.PushSecretData{SecretKey: secretKey, RemoteKey: fakeKey, Property: "", Metadata: &apiextensionsv1.JSON{
		Raw: []byte(`{
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
					"kind": "PushSecretMetadata",
					"spec": {
						"secretPushFormat": "string"
					}
				}`)}}
	pushSecretDataWithProperty := fake.PushSecretData{SecretKey: secretKey, RemoteKey: fakeKey, Property: "other-fake-property"}

	type args struct {
		store          *esv1.AWSProvider
		client         fakesm.Client
		pushSecretData fake.PushSecretData
		newUUID        string
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
					GetSecretValueFn:       fakesm.NewGetSecretValueFn(secretValueOutput, nil),
					PutSecretValueFn:       fakesm.NewPutSecretValueFn(putSecretOutput, nil),
					DescribeSecretFn:       fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
					TagResourceFn:          fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn:        fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretSucceedsWithExistingSecretButNoSecretVersionsWithoutProperty": {
			reason: "a secret can be pushed to aws secrets manager when it already exists but has no secret versions",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutputNoVersions, nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`fake-value`),
						Version:      aws.String(initialVersion),
					}),
					TagResourceFn:   fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn: fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretSucceedsWithExistingSecretButNoSecretVersionsWithProperty": {
			reason: "a secret can be pushed to aws secrets manager when it already exists but has no secret versions",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutputNoVersions, nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`{"other-fake-property":"fake-value"}`),
						Version:      aws.String(initialVersion),
					}),
					TagResourceFn:   fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn: fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
				},
				pushSecretData: pushSecretDataWithProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretSucceedsWithoutSecretKey": {
			reason: "a secret can be pushed to aws secrets manager without secret key",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutput, nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
					TagResourceFn:    fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn:  fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
				},
				pushSecretData: pushSecretDataWithoutSecretKey,
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
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutput, nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
					TagResourceFn:    fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn:  fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
				},
				pushSecretData: pushSecretDataWithMetadata,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretSucceedsWithExistingSecretAndKMSKeyAndDescription": {
			reason: "a secret can be pushed to aws secrets manager when it already exists",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutput, &getSecretCorrectErr),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
				},
				pushSecretData: fake.PushSecretData{SecretKey: secretKey, RemoteKey: fakeKey, Property: "", Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{
							"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
							"kind": "PushSecretMetadata",
							"spec": {
								"kmsKeyID": "bb123123-b2b0-4f60-ac3a-44a13f0e6b6c",
								"description": "this is a description"
							}
						}`)}},
			},
			want: want{
				err: &getSecretCorrectErr,
			},
		},
		"SetSecretSucceedsWithExistingSecretAndAdditionalTags": {
			reason: "a secret can be pushed to aws secrets manager when it already exists",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutput, nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
					TagResourceFn:    fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn:  fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
				},
				pushSecretData: fake.PushSecretData{SecretKey: secretKey, RemoteKey: fakeKey, Property: "", Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{
							"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
							"kind": "PushSecretMetadata",
							"spec": {
								"tags": {"tagname12": "tagvalue1"}
							}
						}`)}},
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
					DescribeSecretFn: fakesm.NewDescribeSecretFn(blankDescribeSecretOutput, &getSecretCorrectErr),
					CreateSecretFn:   fakesm.NewCreateSecretFn(secretOutput, nil),
					PutResourcePolicyFn: fakesm.NewPutResourcePolicyFn(&awssm.PutResourcePolicyOutput{}, nil),
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
					DescribeSecretFn: fakesm.NewDescribeSecretFn(blankDescribeSecretOutput, &getSecretCorrectErr),
					CreateSecretFn:   fakesm.NewCreateSecretFn(secretOutput, nil, []byte(`{"other-fake-property":"fake-value"}`)),
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
				newUUID: defaultUpdatedVersion,
				store:   makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutputFrom(params{b: []byte((`{"fake-property":"fake-value"}`))}), nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`{"fake-property":"fake-value","other-fake-property":"fake-value"}`),
						Version:      &defaultUpdatedVersion,
					}),
					TagResourceFn:   fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn: fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
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
				store:   makeValidSecretStore().Spec.Provider.AWS,
				newUUID: randomUUIDVersionIncremented,
				client: fakesm.Client{
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutputFrom(params{
						b:       []byte((`{"fake-property":"fake-value"}`)),
						version: &randomUUIDVersion,
					}), nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutputFrom(randomUUIDVersion), nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`{"fake-property":"fake-value","other-fake-property":"fake-value"}`),
						Version:      &randomUUIDVersionIncremented,
					}),
					TagResourceFn:   fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn: fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
				},
				pushSecretData: pushSecretDataWithProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithPropertySucceedsWithExistingSecretAndVersionThatCantBeParsed": {
			reason: "A manually set secret version doesn't have to be a UUID",
			args: args{
				newUUID: unparsableVersion,
				store:   makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutputFrom(params{
						b:       []byte((`{"fake-property":"fake-value"}`)),
						version: &unparsableVersion,
					}), nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte((`fake-value`)),
						Version:      &unparsableVersion,
					}),
					TagResourceFn:   fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn: fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: nil,
			},
		},
		"SetSecretWithPropertySucceedsWithExistingSecretAndAbsentVersion": {
			reason: "When a secret version is not specified, set it to 1",
			args: args{
				newUUID: initialVersion,
				store:   makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueFn: fakesm.NewGetSecretValueFn(&awssm.GetSecretValueOutput{
						ARN:          &arn,
						SecretBinary: []byte((`{"fake-property":"fake-value"}`)),
					}, nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`{"fake-property":"fake-value","other-fake-property":"fake-value"}`),
						Version:      &initialVersion,
					}),
					TagResourceFn:   fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn: fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
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
				newUUID: defaultUpdatedVersion,
				store:   makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutputFrom(params{s: `{"fake-property":"fake-value"}`}), nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`{"fake-property":"fake-value","other-fake-property":"fake-value"}`),
						Version:      &defaultUpdatedVersion,
					}),
					TagResourceFn:   fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn: fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
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
				newUUID: defaultUpdatedVersion,
				store:   makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutputFrom(params{s: `{"fake-property":{"fake-property":"fake-value"}}`}), nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil, fakesm.ExpectedPutSecretValueInput{
						SecretBinary: []byte(`{"fake-property":{"fake-property":"fake-value","other-fake-property":"fake-value"}}`),
						Version:      &defaultUpdatedVersion,
					}),
					TagResourceFn:   fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn: fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
				},
				pushSecretData: fake.PushSecretData{SecretKey: secretKey, RemoteKey: fakeKey, Property: "fake-property.other-fake-property"},
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
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutputFrom(params{s: `non-json-secret`}), nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
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
					DescribeSecretFn: fakesm.NewDescribeSecretFn(blankDescribeSecretOutput, &getSecretCorrectErr),
					CreateSecretFn:   fakesm.NewCreateSecretFn(nil, noPermission),
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
					DescribeSecretFn: fakesm.NewDescribeSecretFn(blankDescribeSecretOutput, noPermission),
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
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutput2, nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
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
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutput, nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(nil, noPermission),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutput, nil),
					TagResourceFn:    fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil),
					UntagResourceFn:  fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: noPermission,
			},
		},
		"SetSecretWrongGetSecretErrFails": {
			reason: "DescribeSecret errors out when anything except awssm.ErrCodeResourceNotFoundException",
			args: args{
				store: makeValidSecretStore().Spec.Provider.AWS,
				client: fakesm.Client{
					DescribeSecretFn: fakesm.NewDescribeSecretFn(blankDescribeSecretOutput, &getSecretWrongErr),
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
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutput, nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(nil, noPermission),
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
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutput, nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(tagSecretOutputFaulty, nil),
				},
				pushSecretData: pushSecretDataWithoutProperty,
			},
			want: want{
				err: errors.New("secret not managed by external-secrets"),
			},
		},
		"PatchSecretTags": {
			reason: "secret key is configured with tags to remove and add",
			args: args{
				store: &esv1.AWSProvider{
					Service: esv1.AWSServiceSecretsManager,
					Region:  "eu-west-2",
				},
				client: fakesm.Client{
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutputFrom(params{s: `{"fake-property":{"fake-property":"fake-value"}}`}), nil),
					DescribeSecretFn: fakesm.NewDescribeSecretFn(&awssm.DescribeSecretOutput{
						ARN: &arn,
						Tags: []types.Tag{
							{Key: &managedBy, Value: &externalSecrets},
							{Key: ptr.To("team"), Value: ptr.To("paradox")},
						},
					}, nil),
					PutSecretValueFn: fakesm.NewPutSecretValueFn(putSecretOutput, nil),
					TagResourceFn: fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil, func(input *awssm.TagResourceInput) {
						assert.Len(t, input.Tags, 2)
						assert.Contains(t, input.Tags, types.Tag{Key: &managedBy, Value: &externalSecrets})
						assert.Contains(t, input.Tags, types.Tag{Key: ptr.To("env"), Value: ptr.To("sandbox")})
					}),
					UntagResourceFn: fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil, func(input *awssm.UntagResourceInput) {
						assert.Len(t, input.TagKeys, 1)
						assert.Equal(t, []string{"team"}, input.TagKeys)
						assert.NotContains(t, input.TagKeys, managedBy)
					}),
					DeleteResourcePolicyFn: fakesm.NewDeleteResourcePolicyFn(&awssm.DeleteResourcePolicyOutput{}, nil),
				},
				pushSecretData: fake.PushSecretData{SecretKey: secretKey, RemoteKey: fakeKey, Property: "", Metadata: &apiextensionsv1.JSON{
					Raw: []byte(`{
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
					"kind": "PushSecretMetadata",
					"spec": {
						"secretPushFormat": "string",
						"tags": {
							"env": "sandbox"
						}
					}
				}`)}},
			},
			want: want{
				err: nil,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			sm := SecretsManager{
				client:  &tc.args.client,
				prefix:  tc.args.store.Prefix,
				newUUID: func() string { return tc.args.newUUID },
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
	secretTag := types.Tag{
		Key:   &managed,
		Value: &manager,
	}
	type args struct {
		client               fakesm.Client
		config               esv1.SecretsManager
		prefix               string
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
				config:          esv1.SecretsManager{},
				getSecretOutput: &awssm.GetSecretValueOutput{},
				describeSecretOutput: &awssm.DescribeSecretOutput{
					Tags: []types.Tag{secretTag},
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
				config: esv1.SecretsManager{
					ForceDeleteWithoutRecovery: true,
				},
				getSecretOutput: &awssm.GetSecretValueOutput{},
				describeSecretOutput: &awssm.DescribeSecretOutput{
					Tags: []types.Tag{secretTag},
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
				config:          esv1.SecretsManager{},
				getSecretOutput: &awssm.GetSecretValueOutput{},
				describeSecretOutput: &awssm.DescribeSecretOutput{
					Tags: []types.Tag{},
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
				config: esv1.SecretsManager{
					RecoveryWindowInDays: 1,
				},
				getSecretOutput: &awssm.GetSecretValueOutput{},
				describeSecretOutput: &awssm.DescribeSecretOutput{
					Tags: []types.Tag{secretTag},
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
				config: esv1.SecretsManager{
					RecoveryWindowInDays:       7,
					ForceDeleteWithoutRecovery: true,
				},
				getSecretOutput: &awssm.GetSecretValueOutput{},
				describeSecretOutput: &awssm.DescribeSecretOutput{
					Tags: []types.Tag{secretTag},
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
				config:               esv1.SecretsManager{},
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
				config:               esv1.SecretsManager{},
				getSecretOutput:      nil,
				describeSecretOutput: nil,
				deleteSecretOutput:   nil,
				getSecretErr:         errors.New("not here, sorry dude"),
				describeSecretErr:    nil,
				deleteSecretErr:      nil,
			},
			want: want{
				err: errors.New("not here, sorry dude"),
			},
		},
		"Not expected AWS error": {
			args: args{
				client:               fakeClient,
				config:               esv1.SecretsManager{},
				getSecretOutput:      nil,
				describeSecretOutput: nil,
				deleteSecretOutput:   nil,
				getSecretErr:         errors.New("aws unavailable"),
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
				config:               esv1.SecretsManager{},
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
		"DeleteWithPrefix": {
			args: args{
				client: fakesm.Client{
					GetSecretValueFn: func(ctx context.Context, input *awssm.GetSecretValueInput, opts ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
						// Verify that the input secret ID has the prefix applied
						if *input.SecretId != "my-prefix-"+fakeKey {
							return nil, fmt.Errorf("expected secret name to be prefixed with 'my-prefix-', got %s", *input.SecretId)
						}
						return &awssm.GetSecretValueOutput{}, nil
					},
					DescribeSecretFn: func(ctx context.Context, input *awssm.DescribeSecretInput, opts ...func(*awssm.Options)) (*awssm.DescribeSecretOutput, error) {
						// Verify that the input secret ID has the prefix applied
						if *input.SecretId != "my-prefix-"+fakeKey {
							return nil, fmt.Errorf("expected secret name to be prefixed with 'my-prefix-', got %s", *input.SecretId)
						}
						return &awssm.DescribeSecretOutput{
							Tags: []types.Tag{secretTag},
						}, nil
					},
					DeleteSecretFn: func(ctx context.Context, input *awssm.DeleteSecretInput, opts ...func(*awssm.Options)) (*awssm.DeleteSecretOutput, error) {
						return &awssm.DeleteSecretOutput{}, nil
					},
				},
				config:               esv1.SecretsManager{},
				prefix:               "my-prefix-",
				getSecretOutput:      nil,
				describeSecretOutput: nil,
				deleteSecretOutput:   nil,
				getSecretErr:         nil,
				describeSecretErr:    nil,
				deleteSecretErr:      nil,
			},
			want: want{
				err: nil,
			},
			reason: "Verifies that the prefix is correctly applied when deleting a secret",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ref := fake.PushSecretData{RemoteKey: fakeKey}
			sm := SecretsManager{
				client: &tc.args.client,
				config: &tc.args.config,
				prefix: tc.args.prefix,
			}

			if tc.args.client.GetSecretValueFn == nil {
				tc.args.client.GetSecretValueFn = fakesm.NewGetSecretValueFn(tc.args.getSecretOutput, tc.args.getSecretErr)
			}
			if tc.args.client.DescribeSecretFn == nil {
				tc.args.client.DescribeSecretFn = fakesm.NewDescribeSecretFn(tc.args.describeSecretOutput, tc.args.describeSecretErr)
			}
			if tc.args.client.DeleteSecretFn == nil {
				tc.args.client.DeleteSecretFn = fakesm.NewDeleteSecretFn(tc.args.deleteSecretOutput, tc.args.deleteSecretErr)
			}

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
func makeValidSecretStore() *esv1.SecretStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-secret-store",
			Namespace: "default",
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AWS: &esv1.AWSProvider{
					Service: esv1.AWSServiceSecretsManager,
					Region:  "eu-west-2",
				},
			},
		},
	}
}

func getTagSlice() []types.Tag {
	tagKey1 := tagname1
	tagValue1 := tagvalue1
	tagKey2 := tagname2
	tagValue2 := tagvalue2

	return []types.Tag{
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
		name                  string
		ref                   esv1.ExternalSecretFind
		secretName            string
		secretVersion         string
		secretValue           string
		batchGetSecretValueFn func(context.Context, *awssm.BatchGetSecretValueInput, ...func(*awssm.Options)) (*awssm.BatchGetSecretValueOutput, error)
		listSecretsFn         func(context.Context, *awssm.ListSecretsInput, ...func(*awssm.Options)) (*awssm.ListSecretsOutput, error)
		expectedData          map[string][]byte
		expectedError         string
	}{
		{
			name: "Matching secrets found",
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: secretName,
				},
				Path: ptr.To(secretPath),
			},
			secretName:    secretName,
			secretVersion: secretVersion,
			secretValue:   secretValue,
			batchGetSecretValueFn: func(_ context.Context, input *awssm.BatchGetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.BatchGetSecretValueOutput, error) {
				assert.Len(t, input.Filters, 1)
				assert.Equal(t, "name", string(input.Filters[0].Key))
				assert.Equal(t, secretPath, input.Filters[0].Values[0])
				return &awssm.BatchGetSecretValueOutput{
					SecretValues: []types.SecretValueEntry{
						{
							Name:          ptr.To(secretName),
							VersionStages: []string{secretVersion},
							SecretBinary:  []byte(secretValue),
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
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: secretName,
				},
				Path: ptr.To(secretPath),
			},
			secretName:    secretName,
			secretVersion: secretVersion,
			secretValue:   secretValue,
			batchGetSecretValueFn: func(_ context.Context, input *awssm.BatchGetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.BatchGetSecretValueOutput, error) {
				return &awssm.BatchGetSecretValueOutput{
					SecretValues: []types.SecretValueEntry{
						{
							Name: ptr.To(secretName),
						},
					},
				}, errBoom
			},
			expectedData:  nil,
			expectedError: errBoom.Error(),
		},
		{
			name: "regexp: error occurred while listing secrets",
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: secretName,
				},
			},
			listSecretsFn: func(_ context.Context, input *awssm.ListSecretsInput, _ ...func(*awssm.Options)) (*awssm.ListSecretsOutput, error) {
				return nil, errBoom
			},
			expectedData:  nil,
			expectedError: errBoom.Error(),
		},
		{
			name: "regep: no matching secrets found",
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: secretName,
				},
			},
			listSecretsFn: func(_ context.Context, input *awssm.ListSecretsInput, _ ...func(*awssm.Options)) (*awssm.ListSecretsOutput, error) {
				return &awssm.ListSecretsOutput{
					SecretList: []types.SecretListEntry{
						{
							Name: ptr.To("other-secret"),
						},
					},
				}, nil
			},
			batchGetSecretValueFn: func(_ context.Context, input *awssm.BatchGetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.BatchGetSecretValueOutput, error) {
				return &awssm.BatchGetSecretValueOutput{
					SecretValues: []types.SecretValueEntry{
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
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{
					RegExp: "[",
				},
			},
			expectedData:  nil,
			expectedError: "could not compile find.name.regexp [[]: error parsing regexp: missing closing ]: `[`",
		},

		{
			name: "tags: Matching secrets found",
			ref: esv1.ExternalSecretFind{
				Tags: secretTags,
			},
			secretName:    secretName,
			secretVersion: secretVersion,
			secretValue:   secretValue,
			batchGetSecretValueFn: func(_ context.Context, input *awssm.BatchGetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.BatchGetSecretValueOutput, error) {
				assert.Len(t, input.Filters, 2)
				assert.Equal(t, "tag-key", string(input.Filters[0].Key))
				assert.Equal(t, "foo", input.Filters[0].Values[0])
				assert.Equal(t, "tag-value", string(input.Filters[1].Key))
				assert.Equal(t, "bar", input.Filters[1].Values[0])
				return &awssm.BatchGetSecretValueOutput{
					SecretValues: []types.SecretValueEntry{
						{
							Name:          ptr.To(secretName),
							VersionStages: []string{secretVersion},
							SecretBinary:  []byte(secretValue),
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
			ref: esv1.ExternalSecretFind{
				Tags: secretTags,
			},
			secretName:    secretName,
			secretVersion: secretVersion,
			secretValue:   secretValue,
			batchGetSecretValueFn: func(_ context.Context, input *awssm.BatchGetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.BatchGetSecretValueOutput, error) {
				return &awssm.BatchGetSecretValueOutput{
					SecretValues: []types.SecretValueEntry{
						{
							Name:          ptr.To(secretName),
							VersionStages: []string{secretVersion},
							SecretBinary:  []byte(secretValue),
						},
					},
				}, errBoom
			},
			expectedData:  nil,
			expectedError: errBoom.Error(),
		},
		{
			name: "tags: error occurred while listing secrets",
			ref: esv1.ExternalSecretFind{
				Tags: secretTags,
			},
			batchGetSecretValueFn: func(_ context.Context, input *awssm.BatchGetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.BatchGetSecretValueOutput, error) {
				return nil, errBoom
			},
			expectedData:  nil,
			expectedError: errBoom.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fc := fakesm.NewClient()
			fc.BatchGetSecretValueFn = tc.batchGetSecretValueFn
			fc.ListSecretsFn = tc.listSecretsFn
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
		cfg          *aws.Config
		referentAuth bool
	}

	validConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider(
			"fake",
			"fake",
			"fake",
		),
	}

	invalidConfig := &aws.Config{
		Credentials: &FakeCredProvider{
			retrieveFunc: func() (aws.Credentials, error) {
				return aws.Credentials{}, errors.New("invalid credentials")
			},
		},
	}

	tests := []struct {
		name    string
		fields  fields
		want    esv1.ValidationResult
		wantErr bool
	}{
		{
			name: "ReferentAuth should always return unknown",
			fields: fields{
				referentAuth: true,
			},
			want: esv1.ValidationResultUnknown,
		},
		{
			name: "Valid credentials should return ready",
			fields: fields{
				cfg: validConfig,
			},
			want: esv1.ValidationResultReady,
		},
		{
			name: "Invalid credentials should return error",
			fields: fields{
				cfg: invalidConfig,
			},
			want:    esv1.ValidationResultError,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SecretsManager{
				cfg:          tt.fields.cfg,
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

	getSecretCorrectErr := types.ResourceNotFoundException{}
	getSecretWrongErr := types.InvalidRequestException{}

	pushSecretDataWithoutProperty := fake.PushSecretData{SecretKey: "fake-secret-key", RemoteKey: fakeKey, Property: ""}

	type args struct {
		store          *esv1.AWSProvider
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
					GetSecretValueFn: fakesm.NewGetSecretValueFn(secretValueOutput, nil),
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
					GetSecretValueFn: fakesm.NewGetSecretValueFn(blankSecretValueOutput, &getSecretCorrectErr),
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
					GetSecretValueFn: fakesm.NewGetSecretValueFn(blankSecretValueOutput, &getSecretWrongErr),
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
					"secretPushFormat":"string",
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
					Description:      "test description",
					SecretPushFormat: "string",
					KMSKeyID:         "custom-kms-key",
					Tags: map[string]string{
						"customKey": "customValue",
						managedBy:   externalSecrets,
					},
				},
			},
		},
		{
			name:  "Empty metadata, defaults applied",
			input: nil,
			expected: &metadata.PushSecretMetadata[PushSecretMetadataSpec]{
				Spec: PushSecretMetadataSpec{
					Description:      fmt.Sprintf("secret '%s:%s'", managedBy, externalSecrets),
					SecretPushFormat: "binary",
					KMSKeyID:         "alias/aws/secretsmanager",
					Tags: map[string]string{
						managedBy: externalSecrets,
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
					"tags": {
                        "managed-by": "external-secrets",
						"customKey": "customValue"
					},
				}
			}`)},
			expected:    nil,
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
			result, err := (&SecretsManager{}).constructMetadataWithDefaults(tt.input)

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
		expected []types.Tag
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
			expected: []types.Tag{
				{Key: ptr.To("key1"), Value: ptr.To("value1")},
				{Key: ptr.To("key2"), Value: ptr.To("value2")},
			},
			modified: false,
		},
		{
			name: "No tags to update as managed-by tag is ignored",
			tags: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			metaTags: map[string]string{
				"key1":    "value1",
				"key2":    "value2",
				managedBy: externalSecrets,
			},
			expected: []types.Tag{
				{Key: ptr.To("key1"), Value: ptr.To("value1")},
				{Key: ptr.To("key2"), Value: ptr.To("value2")},
				{Key: ptr.To(managedBy), Value: ptr.To(externalSecrets)},
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
			expected: []types.Tag{
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
			expected: []types.Tag{
				{Key: ptr.To("key1"), Value: ptr.To("newValue")},
			},
			modified: true,
		},
		{
			name:     "Empty tags and metaTags",
			tags:     map[string]string{},
			metaTags: map[string]string{},
			expected: []types.Tag{},
			modified: false,
		},
		{
			name: "Empty tags with non-empty metaTags",
			tags: map[string]string{},
			metaTags: map[string]string{
				"key1": "value1",
			},
			expected: []types.Tag{
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

func TestPatchTags(t *testing.T) {
	type call struct {
		untagCalled bool
		tagCalled   bool
	}
	tests := []struct {
		name         string
		existingTags map[string]string
		metaTags     map[string]string
		expectUntag  bool
		expectTag    bool
		assertsTag   func(input *awssm.TagResourceInput)
		assertsUntag func(input *awssm.UntagResourceInput)
	}{
		{
			name:         "no changes",
			existingTags: map[string]string{"a": "1"},
			metaTags:     map[string]string{"a": "1"},
			expectUntag:  false,
			expectTag:    false,
			assertsTag: func(input *awssm.TagResourceInput) {
				assert.Fail(t, "Expected TagResource to not be called")
			},
			assertsUntag: func(input *awssm.UntagResourceInput) {
				assert.Fail(t, "Expected UntagResource to not be called")
			},
		},
		{
			name:         "update tag value",
			existingTags: map[string]string{"a": "1"},
			metaTags:     map[string]string{"a": "2"},
			expectUntag:  false,
			expectTag:    true,
			assertsTag: func(input *awssm.TagResourceInput) {
				assert.Contains(t, input.Tags, types.Tag{Key: ptr.To(managedBy), Value: ptr.To(externalSecrets)})
				assert.Contains(t, input.Tags, types.Tag{Key: ptr.To("a"), Value: ptr.To("2")})
			},
			assertsUntag: func(input *awssm.UntagResourceInput) {
				assert.Fail(t, "Expected UntagResource to not be called")
			},
		},
		{
			name:         "remove tag",
			existingTags: map[string]string{"a": "1", "b": "2"},
			metaTags:     map[string]string{"a": "1"},
			expectUntag:  true,
			expectTag:    false,
			assertsTag: func(input *awssm.TagResourceInput) {
				assert.Fail(t, "Expected TagResource to not be called")
			},
			assertsUntag: func(input *awssm.UntagResourceInput) {
				assert.Equal(t, []string{"b"}, input.TagKeys)
			},
		},
		{
			name:         "add tags",
			existingTags: map[string]string{"a": "1"},
			metaTags:     map[string]string{"a": "1", "b": "2"},
			expectUntag:  false,
			expectTag:    true,
			assertsTag: func(input *awssm.TagResourceInput) {
				assert.Contains(t, input.Tags, types.Tag{Key: ptr.To(managedBy), Value: ptr.To(externalSecrets)})
				assert.Contains(t, input.Tags, types.Tag{Key: ptr.To("a"), Value: ptr.To("1")})
				assert.Contains(t, input.Tags, types.Tag{Key: ptr.To("b"), Value: ptr.To("2")})
			},
			assertsUntag: func(input *awssm.UntagResourceInput) {
				assert.Fail(t, "Expected UntagResource to not be called")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := call{}
			fakeClient := &fakesm.Client{
				TagResourceFn: fakesm.NewTagResourceFn(&awssm.TagResourceOutput{}, nil, func(input *awssm.TagResourceInput) {
					tt.assertsTag(input)
					calls.tagCalled = true
				}),
				UntagResourceFn: fakesm.NewUntagResourceFn(&awssm.UntagResourceOutput{}, nil, func(input *awssm.UntagResourceInput) {
					tt.assertsUntag(input)
					calls.untagCalled = true
				}),
			}

			sm := &SecretsManager{client: fakeClient}
			metaMap := map[string]interface{}{
				"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
				"kind":       "PushSecretMetadata",
				"spec": map[string]interface{}{
					"description": "adding managed-by tag explicitly",
					"tags":        tt.metaTags,
				},
			}
			raw, err := json.Marshal(metaMap)
			require.NoError(t, err)
			meta := &apiextensionsv1.JSON{Raw: raw}

			secretId := "secret"
			err = sm.patchTags(context.Background(), meta, &secretId, tt.existingTags)
			require.NoError(t, err)
			assert.Equal(t, tt.expectUntag, calls.untagCalled)
			assert.Equal(t, tt.expectTag, calls.tagCalled)
		})
	}
}

// FakeCredProvider implements the AWS credentials.Provider interface
// It is used to inject an error into the AWS config to cause a
// validation error.
type FakeCredProvider struct {
	retrieveFunc func() (aws.Credentials, error)
}

func (f *FakeCredProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return f.retrieveFunc()
}

func (f *FakeCredProvider) IsExpired() bool {
	return true
}
