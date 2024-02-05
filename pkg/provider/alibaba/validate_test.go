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
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"testing"
)

func TestValidateStore(t *testing.T) {
	const validRegion = "eu-central-1"
	type args struct {
		store esv1beta1.GenericStore
	}

	tests := []struct {
		name     string
		args     args
		expected bool
	}{

		{
			name:     "Invalid store with missing region should fail validation",
			expected: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							Alibaba: &esv1beta1.AlibabaProvider{
								RegionID: "No region",
							},
						},
					},
				},
			},
		},
		{
			name:     "Valid store should pass validation",
			expected: false,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							Alibaba: &esv1beta1.AlibabaProvider{
								RegionID: validRegion,
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kms := &KeyManagementService{}
			if err := kms.ValidateStore(tc.args.store); (err != nil) != tc.expected {
				t.Errorf("Provider.ValidateStore() error = %v,expected %v", err, tc.expected)
			}
		})
	}
}
