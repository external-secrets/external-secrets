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

package v1beta1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestValidateExternalSecret(t *testing.T) {
	tests := []struct {
		name        string
		obj         runtime.Object
		expectedErr string
	}{
		{
			name:        "nil",
			obj:         nil,
			expectedErr: "unexpected type",
		},
		{
			name: "deletion policy delete",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Target: ExternalSecretTarget{
						DeletionPolicy: DeletionPolicyDelete,
						CreationPolicy: CreatePolicyMerge,
					},
					Data: []ExternalSecretData{
						{},
					},
				},
			},
			expectedErr: "deletionPolicy=Delete must not be used when the controller doesn't own the secret. Please set creationPolicy=Owner",
		},
		{
			name: "deletion policy merge",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Target: ExternalSecretTarget{
						DeletionPolicy: DeletionPolicyMerge,
						CreationPolicy: CreatePolicyNone,
					},
					Data: []ExternalSecretData{
						{},
					},
				},
			},
			expectedErr: "deletionPolicy=Merge must not be used with creationPolicy=None. There is no Secret to merge with",
		},
		{
			name: "both data and data_from are empty",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{},
			},
			expectedErr: "either data or dataFrom should be specified",
		},
		{
			name: "find with extract",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							Find:    &ExternalSecretFind{},
							Extract: &ExternalSecretDataRemoteRef{},
						},
					},
				},
			},
			expectedErr: "extract, find, or generatorRef cannot be set at the same time",
		},
		{
			name: "generator with find",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							Find: &ExternalSecretFind{},
							SourceRef: &StoreGeneratorSourceRef{
								GeneratorRef: &GeneratorRef{},
							},
						},
					},
				},
			},
			expectedErr: "extract, find, or generatorRef cannot be set at the same time",
		},
		{
			name: "generator with extract",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							Extract: &ExternalSecretDataRemoteRef{},
							SourceRef: &StoreGeneratorSourceRef{
								GeneratorRef: &GeneratorRef{},
							},
						},
					},
				},
			},
			expectedErr: "extract, find, or generatorRef cannot be set at the same time",
		},
		{
			name: "empty dataFrom",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{},
					},
				},
			},
			expectedErr: "either extract, find, or sourceRef must be set to dataFrom",
		},
		{
			name: "empty sourceRef",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &StoreGeneratorSourceRef{},
						},
					},
				},
			},
			expectedErr: "generatorRef or storeRef must be set when using sourceRef in dataFrom",
		},
		{
			name: "multiple errors",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Target: ExternalSecretTarget{
						DeletionPolicy: DeletionPolicyMerge,
						CreationPolicy: CreatePolicyNone,
					},
				},
			},
			expectedErr: `deletionPolicy=Merge must not be used with creationPolicy=None. There is no Secret to merge with
either data or dataFrom should be specified`,
		},
		{
			name: "valid",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &StoreGeneratorSourceRef{
								GeneratorRef: &GeneratorRef{},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate secretKeys",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Target: ExternalSecretTarget{
						DeletionPolicy: DeletionPolicyRetain,
					},
					Data: []ExternalSecretData{
						{SecretKey: "SERVICE_NAME"},
						{SecretKey: "SERVICE_NAME"},
						{SecretKey: "SERVICE_NAME-2"},
						{SecretKey: "SERVICE_NAME-2"},
						{SecretKey: "NOT_DUPLICATE"},
					},
				},
			},
			expectedErr: "duplicate secretKey found: SERVICE_NAME\nduplicate secretKey found: SERVICE_NAME-2",
		},
		{
			name: "duplicate secretKey",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Target: ExternalSecretTarget{
						DeletionPolicy: DeletionPolicyRetain,
					},
					Data: []ExternalSecretData{
						{SecretKey: "SERVICE_NAME"},
						{SecretKey: "SERVICE_NAME"},
					},
				},
			},
			expectedErr: "duplicate secretKey found: SERVICE_NAME",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateExternalSecret(tt.obj)
			if err != nil {
				if tt.expectedErr == "" {
					t.Fatalf("validateExternalSecret() returned an unexpected error: %v", err)
				}

				if err.Error() != tt.expectedErr {
					t.Fatalf("validateExternalSecret() returned an unexpected error: got: %v, expected: %v", err, tt.expectedErr)
				}
				return
			}
			if tt.expectedErr != "" {
				t.Errorf("validateExternalSecret() should have returned an error but got nil")
			}
		})
	}
}
