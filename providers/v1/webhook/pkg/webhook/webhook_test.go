/*
Copyright © The ESO Authors
Copyright © 2026 Apple Inc.

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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// generateTestCert returns a fresh self-signed certificate and matching private key
// in PEM form, suitable for exercising tls.X509KeyPair.
func generateTestCert(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "webhook-client-tls-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}

func TestIncludeClientTLS(t *testing.T) {
	const (
		ns         = "test-ns"
		certSecret = "client-cert"
		keySecret  = "client-key"
	)

	certPEM, keyPEM := generateTestCert(t)
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}

	makeKube := func(objs ...runtime.Object) *clientfake.ClientBuilder {
		return clientfake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...)
	}

	validCertObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: certSecret, Namespace: ns},
		Data:       map[string][]byte{"tls.crt": certPEM},
	}
	validKeyObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: keySecret, Namespace: ns},
		Data:       map[string][]byte{"tls.key": keyPEM},
	}

	clientTLSWith := func() *ClientTLS {
		return &ClientTLS{
			CertSecretRef: &esmeta.SecretKeySelector{Name: certSecret, Key: "tls.crt"},
			KeySecretRef:  &esmeta.SecretKeySelector{Name: keySecret, Key: "tls.key"},
		}
	}

	tests := []struct {
		name      string
		spec      *Spec
		objs      []runtime.Object
		wantErr   string
		wantCerts int
	}{
		{
			name:      "no clientTLS configured is a no-op",
			spec:      &Spec{},
			wantCerts: 0,
		},
		{
			name: "missing certSecretRef returns config error",
			spec: &Spec{ClientTLS: &ClientTLS{
				KeySecretRef: &esmeta.SecretKeySelector{Name: keySecret, Key: "tls.key"},
			}},
			wantErr: "clientTLS requires both certSecretRef and keySecretRef",
		},
		{
			name: "missing keySecretRef returns config error",
			spec: &Spec{ClientTLS: &ClientTLS{
				CertSecretRef: &esmeta.SecretKeySelector{Name: certSecret, Key: "tls.crt"},
			}},
			wantErr: "clientTLS requires both certSecretRef and keySecretRef",
		},
		{
			name:    "missing cert secret surfaces resolver error",
			spec:    &Spec{ClientTLS: clientTLSWith()},
			objs:    []runtime.Object{validKeyObj},
			wantErr: "cannot get Kubernetes secret",
		},
		{
			name:    "invalid cert/key pair is rejected",
			spec:    &Spec{ClientTLS: clientTLSWith()},
			objs:    []runtime.Object{validCertObj, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: keySecret, Namespace: ns}, Data: map[string][]byte{"tls.key": []byte("not-a-key")}}},
			wantErr: "tls:",
		},
		{
			name:      "valid cert and key are appended to tls config",
			spec:      &Spec{ClientTLS: clientTLSWith()},
			objs:      []runtime.Object{validCertObj, validKeyObj},
			wantCerts: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := &Webhook{
				Kube:      makeKube(tc.objs...).Build(),
				Namespace: ns,
			}
			tlsConf := &tls.Config{MinVersion: tls.VersionTLS12}
			err := w.includeClientTLS(context.Background(), tlsConf, tc.spec)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("got err=%v, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got := len(tlsConf.Certificates); got != tc.wantCerts {
				t.Fatalf("Certificates: got %d, want %d", got, tc.wantCerts)
			}
		})
	}
}
