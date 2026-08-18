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

package webhook

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const testSAToken = "sa-token-123"

// makeTokenClient returns a fake client that serves TokenRequest calls for
// service accounts, mimicking the kube-apiserver's token subresource.
func makeTokenClient(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().WithObjects(objs...).WithInterceptorFuncs(interceptor.Funcs{
		SubResourceCreate: func(_ context.Context, _ client.Client, subResourceName string, _ client.Object, subResource client.Object, _ ...client.SubResourceCreateOption) error {
			if subResourceName != "token" {
				return errors.New("unexpected subresource")
			}
			tokenRequest, ok := subResource.(*authenticationv1.TokenRequest)
			if !ok {
				return errors.New("subresource is not a TokenRequest")
			}
			tokenRequest.Status.Token = testSAToken
			return nil
		},
	}).Build()
}

func makeServiceAccount(name, namespace string, labels map[string]string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

func makeServiceAccountStore(kind, url string, ref esmeta.ServiceAccountSelector) esv1.GenericStore {
	spec := esv1.SecretStoreSpec{
		Provider: &esv1.SecretStoreProvider{
			Webhook: &esv1.WebhookProvider{
				URL: url,
				Headers: map[string]string{
					"Authorization": "Bearer {{ .auth.token }}",
				},
				Secrets: []esv1.WebhookSecret{
					{
						Name:              "auth",
						ServiceAccountRef: &ref,
					},
				},
			},
		},
	}
	objectMeta := metav1.ObjectMeta{Name: "webhook-store", Namespace: "default"}
	if kind == esv1.ClusterSecretStoreKind {
		return &esv1.ClusterSecretStore{
			TypeMeta:   metav1.TypeMeta{Kind: kind},
			ObjectMeta: objectMeta,
			Spec:       spec,
		}
	}
	return &esv1.SecretStore{
		TypeMeta:   metav1.TypeMeta{Kind: kind},
		ObjectMeta: objectMeta,
		Spec:       spec,
	}
}

// authHeaderEchoServer replies with the Authorization header it received.
func authHeaderEchoServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.Header.Get("Authorization")))
	}))
}

func TestWebhookServiceAccountToken(t *testing.T) {
	webhookLabels := map[string]string{"external-secrets.io/type": "webhook"}
	saNamespace := "sa-namespace"

	tests := []struct {
		name      string
		storeKind string
		saRef     esmeta.ServiceAccountSelector
		sa        *corev1.ServiceAccount
		want      string
		wantErr   string
	}{
		{
			name:      "SecretStore resolves token from own namespace",
			storeKind: esv1.SecretStoreKind,
			saRef:     esmeta.ServiceAccountSelector{Name: "webhook-sa"},
			sa:        makeServiceAccount("webhook-sa", "default", webhookLabels),
			want:      "Bearer " + testSAToken,
		},
		{
			name:      "ClusterSecretStore resolves token from referenced namespace",
			storeKind: esv1.ClusterSecretStoreKind,
			saRef:     esmeta.ServiceAccountSelector{Name: "webhook-sa", Namespace: &saNamespace},
			sa:        makeServiceAccount("webhook-sa", saNamespace, webhookLabels),
			want:      "Bearer " + testSAToken,
		},
		{
			name:      "ClusterSecretStore requires namespace",
			storeKind: esv1.ClusterSecretStoreKind,
			saRef:     esmeta.ServiceAccountSelector{Name: "webhook-sa"},
			sa:        makeServiceAccount("webhook-sa", saNamespace, webhookLabels),
			wantErr:   "no namespace on ClusterScoped webhook service account",
		},
		{
			name:      "service account must be labeled",
			storeKind: esv1.SecretStoreKind,
			saRef:     esmeta.ServiceAccountSelector{Name: "webhook-sa"},
			sa:        makeServiceAccount("webhook-sa", "default", nil),
			wantErr:   "service account does not contain needed label",
		},
		{
			name:      "missing service account",
			storeKind: esv1.SecretStoreKind,
			saRef:     esmeta.ServiceAccountSelector{Name: "no-such-sa"},
			sa:        makeServiceAccount("webhook-sa", "default", webhookLabels),
			wantErr:   "failed to get webhook service account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := authHeaderEchoServer()
			defer server.Close()

			store := makeServiceAccountStore(tt.storeKind, server.URL, tt.saRef)
			kubeClient := makeTokenClient(tt.sa)

			provider := &Provider{}
			whClient, err := provider.NewClient(context.Background(), store, kubeClient, "default")
			if err != nil {
				t.Fatalf("error creating client: %s", err)
			}

			result, err := whClient.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "dummy"})
			errStr := ""
			if err != nil {
				errStr = err.Error()
			}
			if (tt.wantErr == "") != (errStr == "") || !strings.Contains(errStr, tt.wantErr) {
				t.Fatalf("unexpected error: %q (expected %q)", errStr, tt.wantErr)
			}
			if err == nil && string(result) != tt.want {
				t.Errorf("unexpected response: %q (expected %q)", result, tt.want)
			}
		})
	}
}

func TestValidateStoreWebhookSecrets(t *testing.T) {
	secretRef := &esmeta.SecretKeySelector{Name: "webhook-credentials"}
	serviceAccountRef := &esmeta.ServiceAccountSelector{Name: "webhook-sa"}

	tests := []struct {
		name    string
		secrets []esv1.WebhookSecret
	}{
		{
			name:    "secretRef only is valid",
			secrets: []esv1.WebhookSecret{{Name: "auth", SecretRef: secretRef}},
		},
		{
			name:    "serviceAccountRef only is valid",
			secrets: []esv1.WebhookSecret{{Name: "auth", ServiceAccountRef: serviceAccountRef}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &esv1.SecretStore{
				TypeMeta:   metav1.TypeMeta{Kind: esv1.SecretStoreKind},
				ObjectMeta: metav1.ObjectMeta{Name: "webhook-store", Namespace: "default"},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Webhook: &esv1.WebhookProvider{
							URL:     "http://localhost",
							Secrets: tt.secrets,
						},
					},
				},
			}
			provider := &Provider{}
			_, err := provider.ValidateStore(store)
			if err != nil {
				t.Fatalf("unexpected error: %q", err.Error())
			}
		})
	}
}
