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
	"reflect"
	"testing"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/ovh/fake"
)

func TestGetSecretMap(t *testing.T) {
	mySecretRemoteKey := "mysecret"
	mySecret2RemoteKey := "mysecret2"
	myNestedSecretRemoteKey := "nested-secret"
	nonExistentSecretRemoteKey := "non-existent-secret"
	emptySecretRemoteKey := "empty-secret"
	nilSecretRemoteKey := "nil-secret"

	property := "keys"
	nestedProperty := "users.alice"
	scalarValueProperty := "users.alice.age"
	invalidProperty := "invalid-property"

	testCases := map[string]struct {
		should     map[string][]byte
		errshould  string
		kube       kclient.Client
		okmsClient fake.FakeOkmsClient
		ref        esv1.ExternalSecretDataRemoteRef
	}{
		"Valid Secret": {
			should: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			},
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: mySecretRemoteKey,
			},
		},
		"Non-existent Secret": {
			errshould: "failed to parse the following okms error: Secret does not exist",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: nonExistentSecretRemoteKey,
			},
		},
		"Secret with nil data": {
			errshould: fmt.Sprintf("failed to retrieve secret at path %q: secret version data is missing", nilSecretRemoteKey),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: nilSecretRemoteKey,
			},
		},
		"Secret without empty data": {
			errshould: fmt.Sprintf("failed to retrieve secret at path %q: secret version data is missing", emptySecretRemoteKey),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: emptySecretRemoteKey,
			},
		},
		"Fetch MetaDataPolicy": {
			errshould: fmt.Sprintf("failed to retrieve secret at path %q: fetch metadata policy not supported", mySecretRemoteKey),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:            mySecretRemoteKey,
				MetadataPolicy: "Fetch",
			},
		},
		"Property": {
			should: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			},
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      mySecret2RemoteKey,
				Property: property,
			},
		},
		"Nested Property": {
			should: map[string][]byte{
				"age": []byte("23"),
			},
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      myNestedSecretRemoteKey,
				Property: nestedProperty,
			},
		},
		"Scalar Value Property": {
			errshould: fmt.Sprintf("failed to retrieve secret at path %q: json: cannot unmarshal number into Go value of type map[string]interface {}", myNestedSecretRemoteKey),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      myNestedSecretRemoteKey,
				Property: scalarValueProperty,
			},
		},
		"Invalid Property": {
			errshould: fmt.Sprintf("failed to retrieve secret at path %q: secret property %q not found", mySecretRemoteKey, invalidProperty),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      mySecretRemoteKey,
				Property: invalidProperty,
			},
		},
		"Error case": {
			errshould: fmt.Sprintf("failed to retrieve secret at path %q: failed to parse the following okms error: custom error", mySecretRemoteKey),
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      mySecretRemoteKey,
				Property: invalidProperty,
			},
			okmsClient: fake.FakeOkmsClient{
				GetSecretV2Fn: fake.NewGetSecretV2Fn(mySecretRemoteKey, errors.New("custom error")),
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			cl := &ovhClient{
				okmsClient: testCase.okmsClient,
				kube:       testCase.kube,
			}
			secret, err := cl.GetSecretMap(ctx, testCase.ref)
			if testCase.errshould != "" {
				if err == nil {
					t.Errorf("\nexpected value: %s\nactual value:   <nil>\n\n", testCase.errshould)
				} else if err.Error() != testCase.errshould {
					t.Errorf("\nexpected value: %s\nactual value:   %v\n\n", testCase.errshould, err)
				}
				return
			}
			if !reflect.DeepEqual(testCase.should, secret) {
				t.Errorf("\nexpected value: %v\nactual value:   %v\n\n", convertByteMapToStringMap(testCase.should), convertByteMapToStringMap(secret))
			}
		})
	}
}
