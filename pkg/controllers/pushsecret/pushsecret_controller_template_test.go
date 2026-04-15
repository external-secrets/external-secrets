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
