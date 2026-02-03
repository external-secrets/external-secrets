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
	"errors"
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets/providers/v1/ovh/fake"
	testingfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
)

func TestPushSecret(t *testing.T) {
	secretData := &v1.Secret{
		Data: map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		},
	}
	mySecretRemoteKey := "mysecret"
	mySecret2RemoteKey := "mysecret2"
	nonExistentSecretRemoteKey := "non-existent-secret"
	emptyRemoteKey := ""
	emptySecretRemoteKey := "empty-secret"
	nilSecretRemoteKey := "nil-secret"

	testCases := map[string]struct {
		errshould  string
		secret     *v1.Secret
		data       testingfake.PushSecretData
		okmsClient fake.FakeOkmsClient
	}{
		"Nil Secret": {
			errshould: fmt.Sprintf("failed to push secret at path %q: provided secret is nil", nilSecretRemoteKey),
			secret:    nil,
			data: testingfake.PushSecretData{
				RemoteKey: nilSecretRemoteKey,
			},
			okmsClient: fake.FakeOkmsClient{
				PostSecretV2Fn: fake.NewPostSecretV2Fn(nil),
				PutSecretV2Fn:  fake.NewPutSecretV2Fn(nil),
			},
		},
		"Empty Secret Data": {
			errshould: fmt.Sprintf("failed to push secret at path %q: provided secret is empty", emptySecretRemoteKey),
			secret: &v1.Secret{
				Data: nil,
			},
			data: testingfake.PushSecretData{
				RemoteKey: emptySecretRemoteKey,
			},
			okmsClient: fake.FakeOkmsClient{
				PostSecretV2Fn: fake.NewPostSecretV2Fn(nil),
				PutSecretV2Fn:  fake.NewPutSecretV2Fn(nil),
			},
		},
		"Empty Remote Key": {
			errshould: fmt.Sprintf("failed to push secret at path %q: remote key cannot be empty (spec.data.remoteRef.key)", emptyRemoteKey),
			secret:    secretData,
			data: testingfake.PushSecretData{
				RemoteKey: emptyRemoteKey,
			},
			okmsClient: fake.FakeOkmsClient{
				PostSecretV2Fn: fake.NewPostSecretV2Fn(nil),
				PutSecretV2Fn:  fake.NewPutSecretV2Fn(nil),
			},
		},
		"Non-Existent Remote Key": {
			errshould: "",
			secret:    secretData,
			data: testingfake.PushSecretData{
				RemoteKey: nonExistentSecretRemoteKey,
			},
			okmsClient: fake.FakeOkmsClient{
				PostSecretV2Fn: fake.NewPostSecretV2Fn(nil),
				PutSecretV2Fn:  fake.NewPutSecretV2Fn(nil),
			},
		},
		"Existing Remote Key": {
			errshould: "",
			secret:    secretData,
			data: testingfake.PushSecretData{
				RemoteKey: mySecretRemoteKey,
			},
			okmsClient: fake.FakeOkmsClient{
				PostSecretV2Fn: fake.NewPostSecretV2Fn(nil),
				PutSecretV2Fn:  fake.NewPutSecretV2Fn(nil),
			},
		},
		"Secret Key": {
			errshould: "",
			secret:    secretData,
			data: testingfake.PushSecretData{
				RemoteKey: mySecretRemoteKey,
				SecretKey: "key1",
			},
			okmsClient: fake.FakeOkmsClient{
				PostSecretV2Fn: fake.NewPostSecretV2Fn(nil),
				PutSecretV2Fn:  fake.NewPutSecretV2Fn(nil),
			},
		},
		"Property": {
			errshould: "",
			secret:    secretData,
			data: testingfake.PushSecretData{
				RemoteKey: mySecretRemoteKey,
				Property:  "property",
			},
			okmsClient: fake.FakeOkmsClient{
				PostSecretV2Fn: fake.NewPostSecretV2Fn(nil),
				PutSecretV2Fn:  fake.NewPutSecretV2Fn(nil),
			},
		},
		"Custom PostSecretV2 Error": {
			errshould: fmt.Sprintf("failed to push secret at path %q: could not create remote secret \"mysecret\": custom error", mySecretRemoteKey),
			secret:    secretData,
			data: testingfake.PushSecretData{
				RemoteKey: mySecretRemoteKey,
			},
			okmsClient: fake.FakeOkmsClient{
				// A non-existent secret is referenced to trigger Post instead of Put
				GetSecretV2Fn:  fake.NewGetSecretV2Fn(nonExistentSecretRemoteKey, nil),
				PostSecretV2Fn: fake.NewPostSecretV2Fn(errors.New("custom error")),
				PutSecretV2Fn:  fake.NewPutSecretV2Fn(nil),
			},
		},
		"Custom PutSecretV2 Error": {
			errshould: fmt.Sprintf("failed to push secret at path %q: could not update remote secret \"mysecret\": custom error", mySecretRemoteKey),
			secret:    secretData,
			data: testingfake.PushSecretData{
				RemoteKey: mySecretRemoteKey,
			},
			okmsClient: fake.FakeOkmsClient{
				// An existing secret is referenced to trigger Put instead of Post
				GetSecretV2Fn:  fake.NewGetSecretV2Fn(mySecret2RemoteKey, nil),
				PostSecretV2Fn: fake.NewPostSecretV2Fn(nil),
				PutSecretV2Fn:  fake.NewPutSecretV2Fn(errors.New("custom error")),
			},
		},
		"Custom GetSecretV2 Error": {
			errshould: fmt.Sprintf("failed to push secret at path %q: failed to parse the following okms error: custom error", mySecretRemoteKey),
			secret:    secretData,
			data: testingfake.PushSecretData{
				RemoteKey: mySecretRemoteKey,
			},
			okmsClient: fake.FakeOkmsClient{
				GetSecretV2Fn:  fake.NewGetSecretV2Fn(mySecretRemoteKey, errors.New("custom error")),
				PostSecretV2Fn: fake.NewPostSecretV2Fn(nil),
				PutSecretV2Fn:  fake.NewPutSecretV2Fn(nil),
			},
		},
	}

	ctx := context.Background()
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			cl := ovhClient{
				okmsClient: testCase.okmsClient,
			}
			err := cl.PushSecret(ctx, testCase.secret, testCase.data)
			if testCase.errshould != "" {
				if err == nil {
					t.Errorf("\nexpected error: %s\nactual error:   <nil>\n\n", testCase.errshould)
				} else if err.Error() != testCase.errshould {
					t.Errorf("\nexpected error: %s\nactual error:   %v\n\n", testCase.errshould, err)
				}
			}
		})
	}
}
