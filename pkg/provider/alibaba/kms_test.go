package alibaba

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	kmssdk "github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/alibaba/fake"
)

type keyManagementServiceTestCase struct {
	mockClient     *fakesm.AlibabaMockClient
	apiInput       *kmssdk.GetSecretValueRequest
	apiOutput      *kmssdk.GetSecretValueResponse
	ref            *esv1alpha1.ExternalSecretDataRemoteRef
	projectID      string
	apiErr         error
	expectError    string
	expectedSecret string
	keyID     []byte
	accessKey []byte
	// for testing secretmap
	expectedData map[string]string
}

func makeValidKMSTestCase() *keyManagementServiceTestCase {
	kmstc := keyManagementServiceTestCase{
		mockClient:     &fakesm.AlibabaMockClient{},
		apiInput:       makeValidAPIInput(),
		ref:            makeValidRef(),
		apiOutput:      makeValidAPIOutput(),
		projectID:      "default",
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   make(map[string]string),
	}
	kmstc.mockClient.WithValue(kmstc.apiInput, kmstc.apiOutput, kmstc.apiErr)
	return &kmstc
}

func makeValidRef() *esv1alpha1.ExternalSecretDataRemoteRef {
	return &esv1alpha1.ExternalSecretDataRemoteRef{
		Key:     "/baz",
		Version: "default",
	}
}

func makeValidAPIInput() *kmssdk.GetSecretValueRequest {
	return &kmssdk.GetSecretValueRequest{
		SecretName: "projects/default/secrets//baz/versions/default",
	}
}

func makeValidAPIOutput() *kmssdk.GetSecretValueResponse {
	return &kmssdk.GetSecretValueResponse{}
}

func makeValidKMSTestCaseCustom(tweaks ...func(smtc *keyManagementServiceTestCase)) *keyManagementServiceTestCase {
	kmstc := makeValidKMSTestCase()
	for _, fn := range tweaks {
		fn(kmstc)
	}
	kmstc.mockClient.WithValue(kmstc.apiInput, kmstc.apiOutput, kmstc.apiErr)
	return kmstc
}

var setAPIErr = func(smtc *keyManagementServiceTestCase) {
	smtc.apiErr = fmt.Errorf("oh no")
	smtc.expectError = "oh no"
}

var setNilMockClient = func(smtc *keyManagementServiceTestCase) {
	smtc.mockClient = nil
	smtc.expectError = errUninitalizedAlibabaProvider
}

func TestAlibabaKMSGetSecret(t *testing.T) {
	secretData := make(map[string]interface{})
	secretValue := "changedvalue"
	secretData["payload"] = secretValue
	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(kmstc *keyManagementServiceTestCase) {
	}

	// good case: custom version set
	setCustomKey := func(smtc *keyManagementServiceTestCase) {
	}

	successCases := []*keyManagementServiceTestCase{
		makeValidKMSTestCase(),
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
	setDeserialization := func(smtc *keyManagementServiceTestCase) {
		smtc.apiOutput.SecretData = (`{"foo":"bar"}`)
		smtc.expectedData["foo"] = "bar"
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *keyManagementServiceTestCase) {
		smtc.apiOutput.SecretData = aws.String(`-----------------`)
		pstc.expectError = "unable to unmarshal secret"
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
