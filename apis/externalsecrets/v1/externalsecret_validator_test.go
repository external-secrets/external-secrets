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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			name: "deletion policy delete with create or merge",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Target: ExternalSecretTarget{
						DeletionPolicy: DeletionPolicyDelete,
						CreationPolicy: CreatePolicyCreateOrMerge,
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
		{
			name: "templateFrom type target is rejected",
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
							Metadata: ExternalSecretTemplateMetadata{
								Annotations: map[string]string{
									corev1.ServiceAccountNameKey: "victim-sa",
								},
							},
							TemplateFrom: []TemplateFrom{
								{Target: "type"},
							},
						},
					},
				},
			},
			expectedErr: `templateFrom target="type" is not allowed when targeting a Secret, must be one of "Data", "Annotations" or "Labels"`,
		},
		{
			name: "templateFrom nested metadata.annotations target is rejected",
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
								{Target: "metadata.annotations"},
							},
						},
					},
				},
			},
			expectedErr: `templateFrom target="metadata.annotations" is not allowed when targeting a Secret, must be one of "Data", "Annotations" or "Labels"`,
		},
		{
			name: "templateFrom mixed case nested annotations target is rejected",
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
								{Target: "Metadata.Annotations.kubernetes.io/service-account.name"},
							},
						},
					},
				},
			},
			expectedErr: `templateFrom target="Metadata.Annotations.kubernetes.io/service-account.name" is not allowed when targeting a Secret, must be one of "Data", "Annotations" or "Labels"`,
		},
		{
			name: "templateFrom immutable target is rejected",
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
							TemplateFrom: []TemplateFrom{
								{Target: "immutable"},
							},
						},
					},
				},
			},
			expectedErr: `templateFrom target="immutable" is not allowed when targeting a Secret, must be one of "Data", "Annotations" or "Labels"`,
		},
		{
			name: "templateFrom well-known targets are allowed in any case",
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
							TemplateFrom: []TemplateFrom{
								{Target: ""},
								{Target: "data"},
								{Target: TemplateTargetData},
								{Target: "labels"},
								{Target: TemplateTargetLabels},
								{Target: "annotations"},
								{Target: TemplateTargetAnnotations},
							},
						},
					},
				},
			},
		},
		{
			name: "templateFrom nested target is allowed for a custom resource manifest",
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
						Manifest: &ManifestReference{
							APIVersion: "argoproj.io/v1alpha1",
							Kind:       "Application",
						},
						Template: &ExternalSecretTemplate{
							TemplateFrom: []TemplateFrom{
								{Target: "spec.database.config"},
							},
						},
					},
				},
			},
		},
		{
			// Previously this reported only the templateFrom error. The manifest itself is
			// now rejected, and a manifest target is never a Secret, so the Secret-only
			// templateFrom restriction no longer applies to it.
			name: "manifest naming a Secret is rejected, with a nested templateFrom target",
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
						Manifest: &ManifestReference{
							APIVersion: "v1",
							Kind:       "Secret",
						},
						Template: &ExternalSecretTemplate{
							TemplateFrom: []TemplateFrom{
								{Target: "type"},
							},
						},
					},
				},
			},
			expectedErr: ErrManifestSecretTarget.Error(),
		},
		{
			name: "manifest naming a Secret is rejected with no template at all",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Data: []ExternalSecretData{{}},
					Target: ExternalSecretTarget{
						Manifest: &ManifestReference{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
				},
			},
			expectedErr: ErrManifestSecretTarget.Error(),
		},
		{
			name: "manifest naming a Secret is rejected with an ordinary valid Secret template",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Data: []ExternalSecretData{{}},
					Target: ExternalSecretTarget{
						Manifest: &ManifestReference{
							APIVersion: "v1",
							Kind:       "Secret",
						},
						Template: &ExternalSecretTemplate{
							Type: corev1.SecretTypeOpaque,
							TemplateFrom: []TemplateFrom{
								{Target: TemplateTargetData},
							},
						},
					},
				},
			},
			expectedErr: ErrManifestSecretTarget.Error(),
		},
		{
			// "/v1" parses to the same core group/version as "v1", so it must not be a
			// way around the ban.
			name: "manifest naming a Secret through an explicit empty group is rejected",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Data: []ExternalSecretData{{}},
					Target: ExternalSecretTarget{
						Manifest: &ManifestReference{
							APIVersion: "/v1",
							Kind:       "Secret",
						},
					},
				},
			},
			expectedErr: ErrManifestSecretTarget.Error(),
		},
		{
			name: "manifest naming a Secret kind in a custom group is allowed",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Data: []ExternalSecretData{{}},
					Target: ExternalSecretTarget{
						Manifest: &ManifestReference{
							APIVersion: "example.io/v1",
							Kind:       "Secret",
						},
					},
				},
			},
		},
		{
			name: "default Secret target stays supported",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Data:   []ExternalSecretData{{}},
					Target: ExternalSecretTarget{},
				},
			},
		},
		{
			name: "ConfigMap manifest target stays supported",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Data: []ExternalSecretData{{}},
					Target: ExternalSecretTarget{
						Manifest: &ManifestReference{
							APIVersion: "v1",
							Kind:       "ConfigMap",
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

// TestValidateManifestSecretTargetOnCreateAndUpdate goes through the admission entry points
// rather than validateExternalSecret, because both must refuse the target: rejecting only on
// create would leave an existing ExternalSecret free to switch to a Secret manifest.
func TestValidateManifestSecretTargetOnCreateAndUpdate(t *testing.T) {
	validator := &ExternalSecretValidator{}
	secretManifest := func(apiVersion, kind string) *ExternalSecret {
		return &ExternalSecret{
			Spec: ExternalSecretSpec{
				Data: []ExternalSecretData{{}},
				Target: ExternalSecretTarget{
					Manifest: &ManifestReference{APIVersion: apiVersion, Kind: kind},
				},
			},
		}
	}

	tests := []struct {
		name       string
		obj        *ExternalSecret
		wantReject bool
	}{
		{"core Secret manifest", secretManifest("v1", "Secret"), true},
		{"core Secret manifest with an explicit empty group", secretManifest("/v1", "Secret"), true},
		{"custom group Secret kind", secretManifest("example.io/v1", "Secret"), false},
		{"ConfigMap manifest", secretManifest("v1", "ConfigMap"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, createErr := validator.ValidateCreate(context.Background(), tt.obj)
			_, updateErr := validator.ValidateUpdate(context.Background(), tt.obj, tt.obj)

			if tt.wantReject {
				require.ErrorIs(t, createErr, ErrManifestSecretTarget)
				require.ErrorIs(t, updateErr, ErrManifestSecretTarget)
				return
			}
			require.NotErrorIs(t, createErr, ErrManifestSecretTarget)
			require.NotErrorIs(t, updateErr, ErrManifestSecretTarget)
		})
	}
}

func TestValidateUpdateAllowsLegacyFinalization(t *testing.T) {
	for _, tt := range []struct {
		name    string
		mutate  func(oldObj, newObj *ExternalSecret)
		wantErr bool
	}{
		{name: "remove one finalizer, preserve another"},
		{
			name: "remove the last finalizer",
			mutate: func(_, newObj *ExternalSecret) {
				newObj.Finalizers = nil
			},
		},
		{
			name: "active object remains invalid",
			mutate: func(oldObj, newObj *ExternalSecret) {
				oldObj.DeletionTimestamp = nil
				newObj.DeletionTimestamp = nil
			},
			wantErr: true,
		},
		{
			name: "new deletion timestamp does not bypass validation",
			mutate: func(oldObj, _ *ExternalSecret) {
				oldObj.DeletionTimestamp = nil
			},
			wantErr: true,
		},
		{
			name: "changed spec remains invalid",
			mutate: func(_, newObj *ExternalSecret) {
				newObj.Spec.Target.Name = "another-target"
			},
			wantErr: true,
		},
		{
			name: "adding a finalizer while removing others remains invalid",
			mutate: func(_, newObj *ExternalSecret) {
				newObj.Finalizers = []string{"new.example.com/finalizer"}
			},
			wantErr: true,
		},
		{
			name: "ordinary update remains invalid",
			mutate: func(oldObj, newObj *ExternalSecret) {
				newObj.Finalizers = oldObj.Finalizers
			},
			wantErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oldObj := &ExternalSecret{
				DeletionTimestamp: new(metav1.Now()),
				Finalizers: []string{
					"externalsecrets.external-secrets.io/externalsecret-cleanup",
					"other.example.com/finalizer",
				},
				Spec: ExternalSecretSpec{
					Data: []ExternalSecretData{{}},
					Target: ExternalSecretTarget{
						Manifest: &ManifestReference{APIVersion: "v1", Kind: "Secret"},
					},
				},
			}
			newObj := oldObj.DeepCopy()
			newObj.Finalizers = newObj.Finalizers[1:]
			if tt.mutate != nil {
				tt.mutate(oldObj, newObj)
			}

			validator := &ExternalSecretValidator{}
			_, err := validator.ValidateUpdate(context.Background(), oldObj, newObj)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrManifestSecretTarget)
			} else {
				require.NoError(t, err)
			}
			_, err = validator.ValidateCreate(context.Background(), newObj)
			require.ErrorIs(t, err, ErrManifestSecretTarget, "finalization must not bypass create validation")
		})
	}
}
