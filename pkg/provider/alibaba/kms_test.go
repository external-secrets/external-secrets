package alibaba

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	kmssdk "github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/alibaba/fake"
)

type keyManagementServiceTestCase struct {
	mockClient     *fakesm.AlibabaMockClient
	apiInput       *kmssdk.GetSecretValueRequest
	apiOutput      *kmssdk.GetSecretValueResponse
	ref            *esv1alpha1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidKMSTestCase() *keyManagementServiceTestCase {
	kmstc := keyManagementServiceTestCase{
		mockClient:     &fakesm.AlibabaMockClient{},
		apiInput:       makeValidAPIInput(),
		ref:            makeValidRef(),
		apiOutput:      makeValidAPIOutput(),
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   make(map[string][]byte),
	}
	kmstc.mockClient.WithValue(kmstc.apiInput, kmstc.apiOutput, kmstc.apiErr)
	return &kmstc
}

func makeValidRef() *esv1alpha1.ExternalSecretDataRemoteRef {
	return &esv1alpha1.ExternalSecretDataRemoteRef{
		Key: "test-example",
	}
}

func makeValidAPIInput() *kmssdk.GetSecretValueRequest {
	return &kmssdk.GetSecretValueRequest{
		SecretName: "test-example",
	}
}

func makeValidAPIOutput() *kmssdk.GetSecretValueResponse {
	kmsresponse := &kmssdk.GetSecretValueResponse{
		BaseResponse:      &responses.BaseResponse{},
		RequestId:         "",
		SecretName:        "test-example",
		VersionId:         "",
		CreateTime:        "",
		SecretData:        "",
		SecretDataType:    "",
		AutomaticRotation: "",
		RotationInterval:  "",
		NextRotationDate:  "",
		ExtendedConfig:    "",
		LastRotationDate:  "",
		SecretType:        "",
		VersionStages:     kmssdk.VersionStagesInGetSecretValue{},
	}
	return kmsresponse
}

func makeValidKMSTestCaseCustom(tweaks ...func(kmstc *keyManagementServiceTestCase)) *keyManagementServiceTestCase {
	kmstc := makeValidKMSTestCase()
	for _, fn := range tweaks {
		fn(kmstc)
	}
	kmstc.mockClient.WithValue(kmstc.apiInput, kmstc.apiOutput, kmstc.apiErr)
	return kmstc
}

var setAPIErr = func(kmstc *keyManagementServiceTestCase) {
	kmstc.apiErr = fmt.Errorf("oh no")
	kmstc.expectError = "oh no"
}

var setNilMockClient = func(kmstc *keyManagementServiceTestCase) {
	kmstc.mockClient = nil
	kmstc.expectError = errUninitalizedAlibabaProvider
}

func TestAlibabaKMSGetSecret(t *testing.T) {
	secretData := make(map[string]interface{})
	secretValue := "changedvalue"
	secretData["payload"] = secretValue
	
	// good case: default version is set
	//key is passed in, output is sent back
	setSecretString := func(kmstc *keyManagementServiceTestCase) {
		kmstc.apiOutput.SecretName = "test-example"
		kmstc.apiOutput.SecretData = "value"
		kmstc.expectedSecret = "value"
	}

	// // good case: custom version set
	setCustomKey := func(kmstc *keyManagementServiceTestCase) {
		kmstc.apiOutput.SecretName = "test-example-other"
		kmstc.ref.Key = "test-example-other"
		kmstc.apiOutput.SecretData = "value"
		kmstc.expectedSecret = "value"
	}

	successCases := []*keyManagementServiceTestCase{
		makeValidKMSTestCaseCustom(setSecretString),
		makeValidKMSTestCaseCustom(setCustomKey),
		makeValidKMSTestCaseCustom(setAPIErr),
		makeValidKMSTestCaseCustom(setNilMockClient),
	}

	sm := KeyManagementService{}
	for k, v := range successCases {
		sm.Client = v.mockClient
		out, err := sm.GetSecret(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if string(out) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out))
		}
	}
}

func TestGetSecretMap(t *testing.T) {
	// good case: default version & deserialization
	setDeserialization := func(kmstc *keyManagementServiceTestCase) {
		kmstc.apiOutput.SecretName = "foo"
		kmstc.expectedData["foo"] = []byte("bar")
		kmstc.apiOutput.SecretData = `{"foo":"bar"}`
	}

	// bad case: invalid json
	setInvalidJSON := func(kmstc *keyManagementServiceTestCase) {
		kmstc.apiOutput.SecretData = "-----------------"
		kmstc.expectError = "unable to unmarshal secret"
	}

	successCases := []*keyManagementServiceTestCase{
		makeValidKMSTestCaseCustom(setDeserialization),
		makeValidKMSTestCaseCustom(setInvalidJSON),
		makeValidKMSTestCaseCustom(setNilMockClient),
		makeValidKMSTestCaseCustom(setAPIErr),
	}

	sm := KeyManagementService{}
	for k, v := range successCases {
		sm.Client = v.mockClient
		out, err := sm.GetSecretMap(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if err == nil && !reflect.DeepEqual(out, v.expectedData) {
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
