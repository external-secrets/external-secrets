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

package keyvault

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	pkcs12 "software.sslmate.com/src/go-pkcs12"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
	testingfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
)

// Keys and encrypted bundles are generated in memory, never stored as fixtures.
func certificatePasswordFixture(t *testing.T) (*ecdsa.PrivateKey, *x509.Certificate, *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "unit-test-root"},
		NotBefore: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), NotAfter: time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, root, root, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	root, err = x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	leaf := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "unit-test-certificate"},
		NotBefore: root.NotBefore, NotAfter: root.NotAfter, KeyUsage: x509.KeyUsageDigitalSignature,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, root, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err = x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatal(err)
	}
	return key, leaf, root
}

func encodeCertificatePasswordFixture(t *testing.T, key *ecdsa.PrivateKey, leaf *x509.Certificate, chain []*x509.Certificate, password string, encoder *pkcs12.Encoder) []byte {
	t.Helper()
	value, err := encoder.Encode(key, leaf, chain, password)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

type certificateImportRecorder struct {
	getCalls    int
	importCalls int
	password    *string
	value       string
	existingDER []byte
}

type passwordCertificateClient struct {
	SecretClient
	recorder *certificateImportRecorder
}

func (c *passwordCertificateClient) GetCertificate(context.Context, string, string, string) (keyvault.CertificateBundle, error) {
	c.recorder.getCalls++
	if c.recorder.existingDER == nil {
		return keyvault.CertificateBundle{}, autorest.DetailedError{StatusCode: http.StatusNotFound, Message: notFoundMessage}
	}
	return keyvault.CertificateBundle{Cer: &c.recorder.existingDER, Tags: map[string]*string{managedBy: new(managerLabel)}}, nil
}

func (c *passwordCertificateClient) ImportCertificate(_ context.Context, _, _ string, params keyvault.CertificateImportParameters) (keyvault.CertificateBundle, error) {
	c.recorder.importCalls++
	c.recorder.password = params.Password
	c.recorder.value = *params.Base64EncodedCertificate
	return keyvault.CertificateBundle{}, nil
}

type certificatePasswordTransport struct {
	recorder *certificateImportRecorder
}

func (tr *certificatePasswordTransport) Do(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Authorization") == "" {
		return &http.Response{
			StatusCode: http.StatusUnauthorized, Request: req, Body: http.NoBody,
			Header: http.Header{"Www-Authenticate": {`Bearer authorization="https://login.microsoftonline.com/unit-test" resource="https://vault.azure.net"`}},
		}, nil
	}
	status := http.StatusOK
	body := `{"id":"https://unit-test.vault.azure.net/certificates/example/version"}`
	switch req.Method {
	case http.MethodGet:
		tr.recorder.getCalls++
		if tr.recorder.existingDER == nil {
			status = http.StatusNotFound
			body = `{"error":{"code":"CertificateNotFound","message":"not found"}}`
		} else {
			body = `{"cer":"` + base64.StdEncoding.EncodeToString(tr.recorder.existingDER) + `","tags":{"managed-by":"external-secrets"}}`
		}
	case http.MethodPost:
		if !strings.HasSuffix(req.URL.Path, "/import") {
			return nil, errors.New("unexpected certificate request")
		}
		var params struct {
			Value    string  `json:"value"`
			Password *string `json:"pwd"`
		}
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil {
			return nil, err
		}
		tr.recorder.importCalls++
		tr.recorder.password = params.Password
		tr.recorder.value = params.Value
	default:
		return nil, errors.New("unexpected HTTP method")
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
}

type certificatePasswordCredential struct{}

func (certificatePasswordCredential) GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "unit-test-only", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

func certificatePasswordClient(t *testing.T, useNewSDK bool, objects []*corev1.Secret, recorder *certificateImportRecorder) *Azure {
	t.Helper()
	builder := fake.NewClientBuilder()
	for _, object := range objects {
		builder = builder.WithObjects(object)
	}
	a := &Azure{
		provider:  &esv1.AzureKVProvider{VaultURL: new("https://unit-test.vault.azure.net"), UseAzureSDK: new(useNewSDK)},
		crClient:  builder.Build(),
		store:     &esv1.ClusterSecretStore{},
		namespace: "store-namespace",
	}
	if useNewSDK {
		var err error
		a.certsClient, err = azcertificates.NewClient(*a.provider.VaultURL, certificatePasswordCredential{}, &azcertificates.ClientOptions{
			Transport: &certificatePasswordTransport{recorder: recorder}, Retry: policy.RetryOptions{MaxRetries: -1},
		})
		if err != nil {
			t.Fatal(err)
		}
	} else {
		a.baseClient = &passwordCertificateClient{recorder: recorder}
	}
	return a
}

func certificatePasswordPushData(t *testing.T, ref *esmeta.SecretKeySelector, remoteKey string) testingfake.PushSecretData {
	t.Helper()
	data := testingfake.PushSecretData{SecretKey: "certificate", RemoteKey: remoteKey}
	if ref != nil {
		// Raw metadata also makes this regression test runnable before the field exists.
		raw, err := json.Marshal(map[string]any{
			"apiVersion": metadata.APIVersion, "kind": metadata.Kind,
			"spec": map[string]any{"passwordSecretRef": ref},
		})
		if err != nil {
			t.Fatal(err)
		}
		data.Metadata = &apiextensionsv1.JSON{Raw: raw}
	}
	return data
}

func TestPushCertificatePasswordReference(t *testing.T) {
	key, cert, root := certificatePasswordFixture(t)
	password := "certificate-test-only"
	encrypted := encodeCertificatePasswordFixture(t, key, cert, nil, password, pkcs12.Modern)
	chained := encodeCertificatePasswordFixture(t, key, cert, []*x509.Certificate{root}, password, pkcs12.Modern)
	legacyChained := encodeCertificatePasswordFixture(t, key, cert, []*x509.Certificate{root}, password, pkcs12.Legacy)
	unprotected := encodeCertificatePasswordFixture(t, key, cert, nil, "", pkcs12.Legacy)
	unprotectedModern := encodeCertificatePasswordFixture(t, key, cert, nil, "", pkcs12.Modern)
	ref := &esmeta.SecretKeySelector{Name: "certificate-password", Key: "password"}
	secret := &corev1.Secret{Name: ref.Name, Namespace: "application", Data: map[string][]byte{ref.Key: []byte(password)}}
	cases := []struct {
		name         string
		value        []byte
		ref          *esmeta.SecretKeySelector
		objects      []*corev1.Secret
		remoteKey    string
		existingDER  []byte
		wantPassword string
		wantError    string
		wantImports  int
	}{
		{name: "encrypted", value: encrypted, ref: ref, objects: []*corev1.Secret{secret}, wantPassword: password, wantImports: 1},
		{name: "encrypted_chain", value: chained, ref: ref, objects: []*corev1.Secret{secret}, wantPassword: password, wantImports: 1},
		{name: "encrypted_legacy_chain", value: legacyChained, ref: ref, objects: []*corev1.Secret{secret}, wantPassword: password, wantImports: 1},
		{name: "same_certificate", value: encrypted, ref: ref, objects: []*corev1.Secret{secret}, existingDER: cert.Raw},
		{
			name:         "explicit_same_namespace",
			value:        encrypted,
			ref:          &esmeta.SecretKeySelector{Name: ref.Name, Key: ref.Key, Namespace: new("application")},
			objects:      []*corev1.Secret{secret},
			wantPassword: password,
			wantImports:  1,
		},
		{
			name:      "wrong_password",
			value:     encrypted,
			ref:       ref,
			objects:   []*corev1.Secret{{ObjectMeta: secret.ObjectMeta, Data: map[string][]byte{ref.Key: []byte("incorrect-test-only")}}},
			wantError: "not a valid certificate",
		},
		{name: "missing_secret", value: encrypted, ref: ref, wantError: "cannot get Kubernetes secret"},
		{
			name:      "missing_key",
			value:     encrypted,
			ref:       ref,
			objects:   []*corev1.Secret{{ObjectMeta: secret.ObjectMeta, Data: map[string][]byte{"other": []byte(password)}}},
			wantError: "cannot find secret data for key",
		},
		{name: "missing_name", value: encrypted, ref: &esmeta.SecretKeySelector{Key: ref.Key}, wantError: "name and key are required"},
		{name: "missing_key_reference", value: encrypted, ref: &esmeta.SecretKeySelector{Name: ref.Name}, wantError: "name and key are required"},
		{
			name:      "cross_namespace_cluster_store",
			value:     encrypted,
			ref:       &esmeta.SecretKeySelector{Name: ref.Name, Key: ref.Key, Namespace: new("other")},
			objects:   []*corev1.Secret{{Name: ref.Name, Namespace: "other", Data: secret.Data}},
			wantError: "same namespace",
		},
		{name: "non_certificate_reference", value: encrypted, ref: ref, remoteKey: "secret/example", objects: []*corev1.Secret{secret}, wantError: "only supported for certificates"},
		{name: "encrypted_without_reference", value: encrypted, wantError: "not a valid certificate"},
		{name: "empty_password_reference", value: unprotected, ref: ref, objects: []*corev1.Secret{{ObjectMeta: secret.ObjectMeta, Data: map[string][]byte{ref.Key: {}}}}, wantImports: 1},
		{name: "legacy_without_password", value: unprotected, wantImports: 1},
		{name: "modern_without_password", value: unprotectedModern, wantImports: 1},
		{name: "der_without_password", value: cert.Raw, wantImports: 1},
		{name: "pem_without_password", value: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), wantImports: 1},
	}
	for _, useNewSDK := range []bool{false, true} {
		sdk := "legacy_sdk"
		if useNewSDK {
			sdk = "new_sdk"
		}
		t.Run(sdk, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					recorder := &certificateImportRecorder{existingDER: tc.existingDER}
					a := certificatePasswordClient(t, useNewSDK, tc.objects, recorder)
					source := &corev1.Secret{Namespace: "application", Data: map[string][]byte{"certificate": tc.value}}
					remoteKey := tc.remoteKey
					if remoteKey == "" {
						remoteKey = "cert/example"
					}
					err := a.PushSecret(context.Background(), source, certificatePasswordPushData(t, tc.ref, remoteKey))
					if tc.wantError != "" {
						if err == nil || !strings.Contains(err.Error(), tc.wantError) {
							t.Fatalf("PushSecret() error = %v, want %q", err, tc.wantError)
						}
						if strings.Contains(err.Error(), password) || strings.Contains(err.Error(), "incorrect-test-only") {
							t.Fatal("error includes the certificate password")
						}
						if recorder.getCalls != 0 {
							t.Fatal("invalid local input reached the certificate API")
						}
					} else if err != nil {
						t.Fatalf("PushSecret() error = %v", err)
					}
					if recorder.importCalls != tc.wantImports {
						t.Fatalf("import calls = %d, want %d", recorder.importCalls, tc.wantImports)
					}
					if tc.wantImports == 0 {
						return
					}
					if tc.wantPassword == "" {
						if recorder.password != nil {
							t.Fatal("password unexpectedly included in the default import request")
						}
					} else if recorder.password == nil || *recorder.password != tc.wantPassword {
						t.Fatal("password from the referenced Secret was not passed to the import API")
					}
					if recorder.value != base64.StdEncoding.EncodeToString(tc.value) {
						t.Fatal("certificate bundle changed before import")
					}
				})
			}
		})
	}
}

func TestPushCertificatePasswordRotation(t *testing.T) {
	key, cert, _ := certificatePasswordFixture(t)
	for _, useNewSDK := range []bool{false, true} {
		name := "legacy_sdk"
		if useNewSDK {
			name = "new_sdk"
		}
		t.Run(name, func(t *testing.T) {
			ref := &esmeta.SecretKeySelector{Name: "certificate-password", Key: "password"}
			secret := &corev1.Secret{Name: ref.Name, Namespace: "application", Data: map[string][]byte{ref.Key: []byte("first-test-only")}}
			recorder := &certificateImportRecorder{}
			a := certificatePasswordClient(t, useNewSDK, []*corev1.Secret{secret}, recorder)
			ctx := context.Background()
			data := certificatePasswordPushData(t, ref, "cert/example")
			for _, password := range []string{"first-test-only", "second-test-only"} {
				current := &corev1.Secret{}
				if err := a.crClient.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: secret.Namespace}, current); err != nil {
					t.Fatal(err)
				}
				current.Data[ref.Key] = []byte(password)
				if err := a.crClient.Update(ctx, current); err != nil {
					t.Fatal(err)
				}
				value := encodeCertificatePasswordFixture(t, key, cert, nil, password, pkcs12.Modern)
				source := &corev1.Secret{Namespace: secret.Namespace, Data: map[string][]byte{"certificate": value}}
				if err := a.PushSecret(ctx, source, data); err != nil {
					t.Fatal(err)
				}
				if recorder.password == nil || *recorder.password != password {
					t.Fatal("import used a stale password")
				}
			}
			if recorder.importCalls != 2 {
				t.Fatalf("import calls = %d, want 2", recorder.importCalls)
			}
		})
	}
}
