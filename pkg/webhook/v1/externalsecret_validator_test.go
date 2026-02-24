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

package v1

import (
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errExtractFindGenerator = "extract, find, or generatorRef cannot be set at the same time"
)

func TestValidateExternalSecret(t *testing.T) {
	tests := []struct {
		name        string
		obj         *esv1.ExternalSecret
		expectedErr string
	}{
		{
			name:        "nil",
			obj:         nil,
			expectedErr: "external secret cannot be nil during validation",
		},
		{
			name: "deletion policy delete",
			obj: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						DeletionPolicy: esv1.DeletionPolicyDelete,
						CreationPolicy: esv1.CreatePolicyMerge,
					},
					Data: []esv1.ExternalSecretData{
						{},
					},
				},
			},
			expectedErr: "deletionPolicy=Delete must not be used when the controller doesn't own the secret. Please set creationPolicy=Owner",
		},
		{
			name: "deletion policy merge",
			obj: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						DeletionPolicy: esv1.DeletionPolicyMerge,
						CreationPolicy: esv1.CreatePolicyNone,
					},
					Data: []esv1.ExternalSecretData{
						{},
					},
				},
			},
			expectedErr: "deletionPolicy=Merge must not be used with creationPolicy=None. There is no Secret to merge with",
		},
		{
			name: "both data and data_from are empty",
			obj: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{},
			},
			expectedErr: "either data or dataFrom should be specified",
		},
		{
			name: "find with extract",
			obj: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
						{
							Find:    &esv1.ExternalSecretFind{},
							Extract: &esv1.ExternalSecretDataRemoteRef{},
						},
					},
				},
			},
			expectedErr: errExtractFindGenerator,
		},
		{
			name: "generator with find",
			obj: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
						{
							Find: &esv1.ExternalSecretFind{},
							SourceRef: &esv1.StoreGeneratorSourceRef{
								GeneratorRef: &esv1.GeneratorRef{},
							},
						},
					},
				},
			},
			expectedErr: errExtractFindGenerator,
		},
		{
			name: "generator with extract",
			obj: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
						{
							Extract: &esv1.ExternalSecretDataRemoteRef{},
							SourceRef: &esv1.StoreGeneratorSourceRef{
								GeneratorRef: &esv1.GeneratorRef{},
							},
						},
					},
				},
			},
			expectedErr: errExtractFindGenerator,
		},
		{
			name: "empty dataFrom",
			obj: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
						{},
					},
				},
			},
			expectedErr: "either extract, find, or sourceRef must be set to dataFrom",
		},
		{
			name: "empty sourceRef",
			obj: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &esv1.StoreGeneratorSourceRef{},
						},
					},
				},
			},
			expectedErr: "generatorRef or storeRef must be set when using sourceRef in dataFrom",
		},
		{
			name: "multiple errors",
			obj: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						DeletionPolicy: esv1.DeletionPolicyMerge,
						CreationPolicy: esv1.CreatePolicyNone,
					},
				},
			},
			expectedErr: `deletionPolicy=Merge must not be used with creationPolicy=None. There is no Secret to merge with
either data or dataFrom should be specified`,
		},
		{
			name: "valid",
			obj: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					DataFrom: []esv1.ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &esv1.StoreGeneratorSourceRef{
								GeneratorRef: &esv1.GeneratorRef{},
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate secretKeys",
			obj: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						DeletionPolicy: esv1.DeletionPolicyRetain,
					},
					Data: []esv1.ExternalSecretData{
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
			obj: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						DeletionPolicy: esv1.DeletionPolicyRetain,
					},
					Data: []esv1.ExternalSecretData{
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
