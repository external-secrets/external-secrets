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
	"reflect"
	"testing"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/ovh/fake"
)

func TestGetSecretMap(t *testing.T) {
	testCases := map[string]struct {
		shouldmap map[string][]byte
		errshould string
		kube      kclient.Client
		ref       esv1.ExternalSecretDataRemoteRef
	}{
		"Valid Secret": {
			shouldmap: map[string][]byte{
				"key": []byte("value"),
			},
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "key",
			},
		},
		"Non-existent Secret": {
			errshould: "Secret does not exist",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "key",
			},
		},
		"Secret without data": {
			errshould: "secret version data is missing",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "key",
			},
		},
		"MetaDataPolicy: Fetch": {
			errshould: "fetch metadata policy not supported",
			ref: esv1.ExternalSecretDataRemoteRef{
				MetadataPolicy: "Fetch",
				Key:            "key",
			},
		},
		"Valid property that gets Nested Json": {
			shouldmap: map[string][]byte{
				"project1": []byte("Name"),
				"project2": []byte("Name"),
			},
			ref: esv1.ExternalSecretDataRemoteRef{
				Property: "projects",
				Key:      "key",
			},
		},
		"Invalid property": {
			errshould: "secret property \"Invalid Property\" not found",
			ref: esv1.ExternalSecretDataRemoteRef{
				Property: "Invalid Property",
				Key:      "key",
			},
		},
		"Empty property": {
			shouldmap: map[string][]byte{
				"key": []byte("value"),
			},
			ref: esv1.ExternalSecretDataRemoteRef{
				Property: "",
				Key:      "key",
			},
		},
		"Secret Version": {
			shouldmap: map[string][]byte{
				"key": []byte("value"),
			},
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "key",
			},
		},
		"Invalid Secret Version": {
			errshould: "ID=\"\", Request-ID:\"\", Code=17125378, System=CCM, Component=Secret Manager, Category=Not Found",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "key",
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			cl := &ovhClient{
				okmsClient: &fake.FakeOkmsClient{
					TestCase: name,
				},
				kube: testCase.kube,
			}
			secret, err := cl.GetSecretMap(ctx, testCase.ref)
			if testCase.errshould != "" {
				if err == nil {
					t.Error()
				} else if err.Error() != testCase.errshould {
					t.Error()
				}
			} else {
				if err != nil {
					t.Error()
				}
				if !reflect.DeepEqual(secret, testCase.shouldmap) {
					t.Error()
				}
			}
		})
	}
}
