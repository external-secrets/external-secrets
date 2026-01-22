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

package ovh

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets/providers/v1/ovh/fake"
	testingfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
)

func TestPushSecret(t *testing.T) {
	testCases := map[string]struct {
		should string
		secret *v1.Secret
		data   testingfake.PushSecretData
	}{
		"Nil Secret": {
			should: "nil secret",
			secret: nil,
			data: testingfake.PushSecretData{
				SecretKey: "secretKey",
				RemoteKey: "remoteKey",
				Property:  "property",
			},
		},
		"Nil Secret Data": {
			should: "cannot push empty secret",
			secret: &v1.Secret{
				Data: nil,
			},
			data: testingfake.PushSecretData{
				SecretKey: "secretKey",
				RemoteKey: "remoteKey",
				Property:  "property",
			},
		},
		"Empty Secret Data": {
			should: "cannot push empty secret",
			secret: &v1.Secret{
				Data: map[string][]byte{},
			},
			data: testingfake.PushSecretData{
				SecretKey: "secretKey",
				RemoteKey: "remoteKey",
				Property:  "property",
			},
		},
		"Empty Remote Key": {
			should: "spec.data.remoteRef.key cannot be empty",
			secret: &v1.Secret{
				Data: map[string][]byte{
					"key": []byte("value"),
				},
			},
			data: testingfake.PushSecretData{
				RemoteKey: "",
			},
		},
		"Empty Secret Key / Empty Property / Existing Remote Key (Equal Data)": {
			should: "",
			secret: &v1.Secret{
				Data: map[string][]byte{
					"test4": []byte(`"value4"`),
				},
			},
			data: testingfake.PushSecretData{
				SecretKey: "",
				RemoteKey: "pattern2/test/test-secret",
				Property:  "",
			},
		},
		"Empty Secret Key / Property / Existing Remote Key (Equal Data)": {
			should: `{"property":{"test4":"value4"}}`,
			secret: &v1.Secret{
				Data: map[string][]byte{
					"test4": []byte(`"value4"`),
				},
			},
			data: testingfake.PushSecretData{
				SecretKey: "",
				RemoteKey: "pattern2/test/test-secret",
				Property:  "property",
			},
		},
		"Empty Secret Key / Empty Property / Existing Remote Key (Non-Equal Data)": {
			should: `{"new-test4":"new-value4"}`,
			secret: &v1.Secret{
				Data: map[string][]byte{
					"new-test4": []byte(`"new-value4"`),
				},
			},
			data: testingfake.PushSecretData{
				SecretKey: "",
				RemoteKey: "pattern2/test/test-secret",
				Property:  "",
			},
		},
		"Empty Secret Key / Property / Existing Remote Key (Non-Equal Data)": {
			should: `{"property":{"new-test4":"new-value4"}}`,
			secret: &v1.Secret{
				Data: map[string][]byte{
					"new-test4": []byte(`"new-value4"`),
				},
			},
			data: testingfake.PushSecretData{
				SecretKey: "",
				RemoteKey: "pattern2/test/test-secret",
				Property:  "property",
			},
		},
		"Empty Secret Key / Empty Property / Non-Existent Remote Key": {
			should: `{"root":{"sub1":{"value":"string"},"sub2":"Name"},"test":"value","test1":"value1"}`,
			secret: &v1.Secret{
				Data: map[string][]byte{
					"root":  []byte(`{"sub1":{"value":"string"},"sub2":"Name"}`),
					"test":  []byte(`"value"`),
					"test1": []byte(`"value1"`),
				},
			},
			data: testingfake.PushSecretData{
				SecretKey: "",
				RemoteKey: "non-existent",
				Property:  "",
			},
		},
		"Empty Secret Key / Property / Non-Existent Remote Key": {
			should: `{"property":{"root":{"sub1":{"value":"string"},"sub2":"Name"},"test":"value","test1":"value1"}}`,
			secret: &v1.Secret{
				Data: map[string][]byte{
					"root":  []byte(`{"sub1":{"value":"string"},"sub2":"Name"}`),
					"test":  []byte(`"value"`),
					"test1": []byte(`"value1"`),
				},
			},
			data: testingfake.PushSecretData{
				SecretKey: "",
				RemoteKey: "non-existent",
				Property:  "property",
			},
		},
		"Secret Key / Empty Property / Existing Remote Key": {
			should: `{"test":"value"}`,
			secret: &v1.Secret{
				Data: map[string][]byte{
					"root":  []byte(`{"sub1":{"value":"string"},"sub2":"Name"}`),
					"test":  []byte(`"value"`),
					"test1": []byte(`"value1"`),
				},
			},
			data: testingfake.PushSecretData{
				SecretKey: "test",
				RemoteKey: "pattern1/path3",
				Property:  "",
			},
		},
		"Secret Key / Property / Existing Remote Key": {
			should: `{"property":{"test":"value"}}`,
			secret: &v1.Secret{
				Data: map[string][]byte{
					"root":  []byte(`{"sub1":{"value":"string"},"sub2":"Name"}`),
					"test":  []byte(`"value"`),
					"test1": []byte(`"value1"`),
				},
			},
			data: testingfake.PushSecretData{
				SecretKey: "test",
				RemoteKey: "pattern1/path3",
				Property:  "property",
			},
		},
		"Secret Key / Property / Non-Existent Remote Key": {
			should: `{"property":{"test":"value"}}`,
			secret: &v1.Secret{
				Data: map[string][]byte{
					"root":  []byte(`{"sub1":{"value":"string"},"sub2":"Name"}`),
					"test":  []byte(`"value"`),
					"test1": []byte(`"value1"`),
				},
			},
			data: testingfake.PushSecretData{
				SecretKey: "test",
				RemoteKey: "non-existent",
				Property:  "property",
			},
		},
		"Secret Key / Empty Property / Non-Existent Remote Key": {
			should: `{"test":"value"}`,
			secret: &v1.Secret{
				Data: map[string][]byte{
					"root":  []byte(`{"sub1":{"value":"string"},"sub2":"Name"}`),
					"test":  []byte(`"value"`),
					"test1": []byte(`"value1"`),
				},
			},
			data: testingfake.PushSecretData{
				SecretKey: "test",
				RemoteKey: "non-existent",
				Property:  "",
			},
		},
	}

	ctx := context.Background()
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			cl := ovhClient{
				okmsClient: &fake.FakeOkmsClient{
					TestCase: name,
				},
			}
			err := cl.PushSecret(ctx, testCase.secret, testCase.data)
			if err != nil && testCase.should != err.Error() {
				t.Error()
			} else if err == nil && testCase.should != "" {
				t.Error()
			}
		})
	}
}
