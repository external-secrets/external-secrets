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

	"github.com/external-secrets/external-secrets/providers/v1/ovh/fake"
	testingfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
)

func TestSecretExists(t *testing.T) {
	testCases := map[string]struct {
		should    bool
		errshould string
		kube      kclient.Client
		remoteRef testingfake.PushSecretData
	}{
		"Valid Secret": {
			should:    true,
			remoteRef: testingfake.PushSecretData{},
		},
		"Non-existent Secret": {
			should:    false,
			remoteRef: testingfake.PushSecretData{},
		},
		"Error case": {
			errshould: "SecretExists error",
			remoteRef: testingfake.PushSecretData{},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			cl := &ovhClient{
				kube: testCase.kube,
				okmsClient: &fake.FakeOkmsClient{
					TestCase: name,
				},
			}
			ctx := context.Background()
			exists, err := cl.SecretExists(ctx, testCase.remoteRef)
			if testCase.errshould != "" && err != nil && err.Error() != testCase.errshould {
				t.Error()
			} else if (testCase.should && !exists) || (!testCase.should && exists) {
				t.Error()
			}
		})
	}
}
