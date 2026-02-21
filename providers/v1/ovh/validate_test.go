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
	"errors"
	"testing"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets/providers/v1/ovh/fake"
)

func TestValidate(t *testing.T) {
	testCases := map[string]struct {
		kube       kclient.Client
		okmsClient fake.FakeOkmsClient
		errshould  string
	}{
		"Error case": {
			errshould: "failed to validate secret store: custom error",
			okmsClient: fake.FakeOkmsClient{
				ListSecretV2Fn: fake.NewListSecretV2Fn(errors.New("custom error")),
			},
		},
		"Valid case": {
			errshould: "",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			cl := ovhClient{
				kube:       testCase.kube,
				okmsClient: testCase.okmsClient,
			}
			_, err := cl.Validate()
			if testCase.errshould != "" {
				if err != nil && testCase.errshould != err.Error() {
					t.Errorf("\nexpected error: %s\nactual error:   %v\n\n", testCase.errshould, err)
				}
				if err == nil {
					t.Errorf("\nexpected error: %s\nactual error:   <nil>\n\n", testCase.errshould)
				}
			} else if err != nil {
				t.Errorf("\nunexpected error: %v\n\n", err)
			}
		})
	}
}
