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

	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets/providers/v1/ovh/fake"
	testingfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
)

func TestSecretExists(t *testing.T) {
	mysecretRef := testingfake.PushSecretData{
		RemoteKey: "mysecret",
	}
	nonExistentSecretRef := testingfake.PushSecretData{
		RemoteKey: "non-existent-secret",
	}

	testCases := map[string]struct {
		should     bool
		errshould  string
		remoteRef  testingfake.PushSecretData
		okmsClient fake.FakeOkmsClient
		kube       kclient.Client
	}{
		"Valid Secret": {
			should:    true,
			remoteRef: mysecretRef,
		},
		"Non-existent Secret": {
			should:    false,
			remoteRef: nonExistentSecretRef,
		},
		"Error case": {
			errshould: fmt.Sprintf("failed to check existence of secret %q: failed to parse the following okms error: custom error", mysecretRef.RemoteKey),
			remoteRef: mysecretRef,
			okmsClient: fake.FakeOkmsClient{
				GetSecretV2Fn: fake.NewGetSecretV2Fn(mysecretRef.RemoteKey, errors.New("custom error")),
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			cl := &ovhClient{
				kube:       testCase.kube,
				okmsClient: testCase.okmsClient,
			}
			ctx := context.Background()
			exists, err := cl.SecretExists(ctx, testCase.remoteRef)
			if testCase.errshould != "" {
				if err == nil {
					t.Errorf("\nexpected error: %s\nactual error:   <nil>\n\n", testCase.errshould)
				} else if err.Error() != testCase.errshould {
					t.Errorf("\nexpected error: %s\nactual error:   %v\n\n", testCase.errshould, err)
				}
				return
			} else if err != nil {
				t.Errorf("\nunexpected error: %v\n\n", err)
				return
			}
			if exists != testCase.should {
				t.Errorf("\nexpected value: %t\nactual value:   %t\n\n", testCase.should, exists)
			}
		})
	}
}
