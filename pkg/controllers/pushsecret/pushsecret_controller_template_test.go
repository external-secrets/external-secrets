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

package pushsecret

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

func TestApplyTemplatePreservesSourceSecretTypeWhenTemplateTypeUnset(t *testing.T) {
	r := &Reconciler{}
	secret := &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}
	ps := &esv1alpha1.PushSecret{
		Spec: esv1alpha1.PushSecretSpec{
			Template: &esv1.ExternalSecretTemplate{
				EngineVersion: esv1.TemplateEngineV2,
				Data: map[string]string{
					"key": "{{ .key | upper }}",
				},
			},
		},
	}

	if err := r.applyTemplate(context.Background(), ps, secret); err != nil {
		t.Fatalf("applyTemplate() error = %v", err)
	}
	if secret.Type != corev1.SecretTypeOpaque {
		t.Fatalf("applyTemplate() changed secret type to %q", secret.Type)
	}
	if got := string(secret.Data["key"]); got != "VALUE" {
		t.Fatalf("applyTemplate() templated value = %q, want %q", got, "VALUE")
	}
}
