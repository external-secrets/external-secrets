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

package externalsecret

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestApplyTemplateRejectsPathStyleTemplateFromTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{name: "type", target: "type"},
		{name: "nested annotations path", target: "metadata.annotations"},
		{name: "mixed case nested annotations path", target: "Metadata.Annotations"},
		{name: "immutable", target: "immutable"},
		{name: "owner references", target: "metadata.ownerReferences"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = esv1.AddToScheme(scheme.Scheme)
			r := &Reconciler{
				Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
				Scheme: scheme.Scheme,
			}

			literal := "kubernetes.io/service-account-token"
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{Name: "test-es", Namespace: "default"},
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Name: "test-secret",
						Template: &esv1.ExternalSecretTemplate{
							EngineVersion: esv1.TemplateEngineV2,
							Metadata: esv1.ExternalSecretTemplateMetadata{
								Annotations: map[string]string{
									v1.ServiceAccountNameKey: "victim-sa",
								},
							},
							TemplateFrom: []esv1.TemplateFrom{
								{Literal: &literal, Target: tt.target},
							},
						},
					},
				},
			}

			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "test-secret", Namespace: "default"},
			}

			err := r.ApplyTemplate(context.Background(), es, secret, map[string][]byte{})

			require.Error(t, err)
			assert.Contains(t, err.Error(), "is not allowed when targeting a Secret")
			assert.NotEqual(t, v1.SecretTypeServiceAccountToken, secret.Type)
			assert.NotContains(t, secret.Annotations, v1.ServiceAccountNameKey)
		})
	}
}

func TestApplyTemplateRejectsPrivilegedTemplate(t *testing.T) {
	literal := "irrelevant"
	tests := []struct {
		name     string
		template *esv1.ExternalSecretTemplate
		wantErr  string
	}{
		{
			name: "service account token type with a service account annotation",
			template: &esv1.ExternalSecretTemplate{
				EngineVersion: esv1.TemplateEngineV2,
				Type:          v1.SecretTypeServiceAccountToken,
				Metadata: esv1.ExternalSecretTemplateMetadata{
					Annotations: map[string]string{
						v1.ServiceAccountNameKey: "kube-system-admin-sa",
					},
				},
			},
			wantErr: `template.type="kubernetes.io/service-account-token" with annotation "kubernetes.io/service-account.name" is not allowed`,
		},
		{
			name: "service account token type with a templateFrom annotations target",
			template: &esv1.ExternalSecretTemplate{
				EngineVersion: esv1.TemplateEngineV2,
				Type:          v1.SecretTypeServiceAccountToken,
				TemplateFrom: []esv1.TemplateFrom{
					{Literal: &literal, Target: esv1.TemplateTargetAnnotations},
				},
			},
			wantErr: `template.type="kubernetes.io/service-account-token" with templateFrom target="Annotations" is not allowed`,
		},
		{
			name: "bootstrap token type",
			template: &esv1.ExternalSecretTemplate{
				EngineVersion: esv1.TemplateEngineV2,
				Type:          v1.SecretTypeBootstrapToken,
			},
			wantErr: `template.type="bootstrap.kubernetes.io/token" is not allowed`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = esv1.AddToScheme(scheme.Scheme)
			r := &Reconciler{
				Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
				Scheme: scheme.Scheme,
			}

			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{Name: "test-es", Namespace: "default"},
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Name:     "test-secret",
						Template: tt.template,
					},
				},
			}

			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "test-secret", Namespace: "default"},
			}

			err := r.ApplyTemplate(context.Background(), es, secret, map[string][]byte{})

			require.EqualError(t, err, tt.wantErr)
			assert.Empty(t, secret.Type)
			assert.NotContains(t, secret.Annotations, v1.ServiceAccountNameKey)
		})
	}
}

func TestApplyTemplateAllowsWellKnownTemplateFromTargets(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	r := &Reconciler{
		Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
		Scheme: scheme.Scheme,
	}

	dataLiteral := "greeting: hello"
	annotationLiteral := "team: platform"
	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{Name: "test-es", Namespace: "default"},
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{
				Name: "test-secret",
				Template: &esv1.ExternalSecretTemplate{
					EngineVersion: esv1.TemplateEngineV2,
					TemplateFrom: []esv1.TemplateFrom{
						{Literal: &dataLiteral, Target: "data"},
						{Literal: &annotationLiteral, Target: esv1.TemplateTargetAnnotations},
					},
				},
			},
		},
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "test-secret", Namespace: "default"},
	}

	require.NoError(t, r.ApplyTemplate(context.Background(), es, secret, map[string][]byte{}))
	assert.Equal(t, []byte("hello"), secret.Data["greeting"])
	assert.Equal(t, "platform", secret.Annotations["team"])
}
