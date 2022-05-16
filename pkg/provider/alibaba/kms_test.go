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

package alibaba

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	kmssdk "github.com/aliyun/alibaba-cloud-sdk-go/services/kms"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/alibaba/fake"
)

const (
	secretName  = "test-example"
	secretValue = "value"
)

type keyManagementServiceTestCase struct {
	mockClient     *fakesm.AlibabaMockClient
	apiInput       *kmssdk.GetSecretValueRequest
	apiOutput      *kmssdk.GetSecretValueResponse
	ref            *esv1beta1.ExternalSecretDataRemoteRef
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

func makeValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key: secretName,
	}
}

func makeValidAPIInput() *kmssdk.GetSecretValueRequest {
	return &kmssdk.GetSecretValueRequest{
		SecretName: secretName,
	}
}

func makeValidAPIOutput() *kmssdk.GetSecretValueResponse {
	kmsresponse := &kmssdk.GetSecretValueResponse{
		BaseResponse:      &responses.BaseResponse{},
		RequestId:         "",
		SecretName:        secretName,
		VersionId:         "",
		CreateTime:        "",
		SecretData:        secretValue,
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
	// key is passed in, output is sent back
	setSecretString := func(kmstc *keyManagementServiceTestCase) {
		kmstc.apiOutput.SecretName = secretName
		kmstc.apiOutput.SecretData = secretValue
		kmstc.expectedSecret = secretValue
	}

	// good case: custom version set
	setCustomKey := func(kmstc *keyManagementServiceTestCase) {
		kmstc.apiOutput.SecretName = "test-example-other"
		kmstc.ref.Key = "test-example-other"
		kmstc.apiOutput.SecretData = secretValue
		kmstc.expectedSecret = secretValue
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

func TestValidateStore(t *testing.T) {
	kms := KeyManagementService{}

	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Alibaba: &esv1beta1.AlibabaProvider{
					RegionID: "region-1",
					Auth: &esv1beta1.AlibabaAuth{
						SecretRef: esv1beta1.AlibabaAuthSecretRef{
							AccessKeyID: esmeta.SecretKeySelector{
								Name: "accessKeyID",
								Key:  "key-1",
							},
							AccessKeySecret: esmeta.SecretKeySelector{
								Name: "accessKeySecret",
								Key:  "key-1",
							},
						},
					},
				},
			},
		},
	}

	err := kms.ValidateStore(store)
	if err != nil {
		t.Errorf(err.Error())
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
