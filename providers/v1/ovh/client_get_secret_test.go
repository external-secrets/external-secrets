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

	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/ovh/fake"
)

func TestGetSecret(t *testing.T) {
	testCases := map[string]struct {
		should    string
		errshould string
		kube      kclient.Client
		ref       esv1.ExternalSecretDataRemoteRef
	}{
		"Valid Secret": {
			should: "{\"key\":\"value\"}",
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
				Key:            "key",
				MetadataPolicy: "Fetch",
			},
		},
		"Valid property that gets Nested Json": {
			should: "{\"project1\":\"Name\",\"project2\":\"Name\"}",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "key",
				Property: "projects",
			},
		},
		"Valid property that gets non_Nested Json": {
			should: "Name",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "key",
				Property: "projects.project1",
			},
		},
		"Invalid property": {
			errshould: "secret property \"Invalid Property\" not found",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "key",
				Property: "Invalid Property",
			},
		},
		"Empty property": {
			should: "{\"key\":\"value\"}",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "key",
				Property: "",
			},
		},
		"Secret Version": {
			should: "{\"key\":\"value\"}",
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
			secret, err := cl.GetSecret(ctx, testCase.ref)
			if testCase.errshould != "" && err != nil && err.Error() != testCase.errshould {
				t.Error()
			} else if testCase.should != "" && string(secret) != testCase.should {
				t.Error()
			}
		})
	}
}
