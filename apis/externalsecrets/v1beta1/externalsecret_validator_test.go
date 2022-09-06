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

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestValidateExternalSecret(t *testing.T) {
	tests := []struct {
		name    string
		obj     runtime.Object
		wantErr bool
	}{
		{
			name:    "nil",
			obj:     nil,
			wantErr: true,
		},
		{
			name: "deletion policy delete",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Target: ExternalSecretTarget{
						DeletionPolicy: DeletionPolicyDelete,
						CreationPolicy: CreatePolicyMerge,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "deletion policy merge",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					Target: ExternalSecretTarget{
						DeletionPolicy: DeletionPolicyMerge,
						CreationPolicy: CreatePolicyNone,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "generator with find",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							Find: &ExternalSecretFind{},
							SourceRef: &SourceRef{
								Generator: &v1.JSON{},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "generator with extract",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{
						{
							Extract: &ExternalSecretDataRemoteRef{},
							SourceRef: &SourceRef{
								Generator: &v1.JSON{},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			obj: &ExternalSecret{
				Spec: ExternalSecretSpec{
					DataFrom: []ExternalSecretDataFromRemoteRef{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateExternalSecret(tt.obj); (err != nil) != tt.wantErr {
				t.Errorf("validateExternalSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
