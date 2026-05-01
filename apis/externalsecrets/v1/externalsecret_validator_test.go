/*
Copyright © The ESO Authors

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

	corev1 "k8s.io/api/core/v1"
)

const (
	errExtractFindGenerator = "extract, find, or generatorRef cannot be set at the same time"
)

func TestValidateExternalSecret(t *testing.T) {
	tests := []struct {
		name        string
		obj         *ExternalSecret
		expectedErr string
	}{
		{
			name:        "nil",
			obj:         nil,
			expectedErr: "external secret cannot be nil during validation",
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
			expectedErr: errExtractFindGenerator,
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
			expectedErr: errExtractFindGenerator,
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
			expectedErr: errExtractFindGenerator,
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
		{
			name: "service account token template with name annotation is rejected",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &StoreGeneratorSourceRef{
								GeneratorRef: &GeneratorRef{},
							},
						},
					},
					Target: ExternalSecretTarget{
						Template: &ExternalSecretTemplate{
							Type: corev1.SecretTypeServiceAccountToken,
							Metadata: ExternalSecretTemplateMetadata{
								Annotations: map[string]string{
									corev1.ServiceAccountNameKey: "external-secrets",
								},
							},
						},
					},
				},
			},
			expectedErr: `template.type="kubernetes.io/service-account-token" with annotation "kubernetes.io/service-account.name" is not allowed`,
		},
		{
			name: "service account token template without name annotation is allowed",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &StoreGeneratorSourceRef{
								GeneratorRef: &GeneratorRef{},
							},
						},
					},
					Target: ExternalSecretTarget{
						Template: &ExternalSecretTemplate{
							Type: corev1.SecretTypeServiceAccountToken,
						},
					},
				},
			},
		},
		{
			name: "service account token template with templateFrom annotations target is rejected",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &StoreGeneratorSourceRef{
								GeneratorRef: &GeneratorRef{},
							},
						},
					},
					Target: ExternalSecretTarget{
						Template: &ExternalSecretTemplate{
							Type: corev1.SecretTypeServiceAccountToken,
							TemplateFrom: []TemplateFrom{
								{Target: TemplateTargetAnnotations},
							},
						},
					},
				},
			},
			expectedErr: `template.type="kubernetes.io/service-account-token" with templateFrom target="Annotations" is not allowed`,
		},
		{
			name: "service account token template with lowercase templateFrom annotations target is rejected",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &StoreGeneratorSourceRef{
								GeneratorRef: &GeneratorRef{},
							},
						},
					},
					Target: ExternalSecretTarget{
						Template: &ExternalSecretTemplate{
							Type: corev1.SecretTypeServiceAccountToken,
							TemplateFrom: []TemplateFrom{
								{Target: "annotations"},
							},
						},
					},
				},
			},
			expectedErr: `template.type="kubernetes.io/service-account-token" with templateFrom target="Annotations" is not allowed`,
		},
		{
			name: "service account token template with templateFrom data target is allowed",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &StoreGeneratorSourceRef{
								GeneratorRef: &GeneratorRef{},
							},
						},
					},
					Target: ExternalSecretTarget{
						Template: &ExternalSecretTemplate{
							Type: corev1.SecretTypeServiceAccountToken,
							TemplateFrom: []TemplateFrom{
								{Target: TemplateTargetData},
							},
						},
					},
				},
			},
		},
		{
			name: "bootstrap token template is rejected",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &StoreGeneratorSourceRef{
								GeneratorRef: &GeneratorRef{},
							},
						},
					},
					Target: ExternalSecretTarget{
						Template: &ExternalSecretTemplate{
							Type: corev1.SecretTypeBootstrapToken,
						},
					},
				},
			},
			expectedErr: `template.type="bootstrap.kubernetes.io/token" is not allowed`,
		},
		{
			name: "service account name annotation without service account token type is allowed",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							SourceRef: &StoreGeneratorSourceRef{
								GeneratorRef: &GeneratorRef{},
							},
						},
					},
					Target: ExternalSecretTarget{
						Template: &ExternalSecretTemplate{
							Type: corev1.SecretTypeOpaque,
							Metadata: ExternalSecretTemplateMetadata{
								Annotations: map[string]string{
									corev1.ServiceAccountNameKey: "external-secrets",
								},
							},
						},
					},
				},
			},
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
