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

package crds

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	failedCreateCaCerts     = "could not create ca certificates:%v"
	failedCreateCaChain     = "could not create the intermediate certificates:%v"
	failedCreateServerCerts = "could not create server certificates:%v"
	invalidCerts            = "generated certificates are invalid:%v,%v"
	dnsName                 = "foobar"
	tlscrt                  = "/tmp/tls"
	tlskey                  = "/tmp/key"
	cacrt                   = "/tmp/ca"
)

func newReconciler() Reconciler {
	return Reconciler{
		CrdResources:    []string{"one", "two", "three"},
		SvcName:         "foo",
		SvcNamespace:    "default",
		SecretName:      "foo",
		SecretNamespace: "default",
	}
}

func newService() corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Labels:    map[string]string{"foo": "bar"},
		},
	}
}
func newSecret() corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Labels:    map[string]string{"foo": "bar"},
		},
	}
}

func newCRD() apiextensionsv1.CustomResourceDefinition {
	return apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "one",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Conversion: &apiextensionsv1.CustomResourceConversion{
				Strategy: "Webhook",
				Webhook: &apiextensionsv1.WebhookConversion{
					ConversionReviewVersions: []string{"v1"},
					ClientConfig: &apiextensionsv1.WebhookClientConfig{
						CABundle: []byte("test"),
						Service: &apiextensionsv1.ServiceReference{
							Name:      "wrong",
							Namespace: "wrong",
						},
					},
				},
			},
		},
	}
}

func TestUpdateCRD(t *testing.T) {
	rec := newReconciler()
	svc := newService()
	secret := newSecret()
	crd := newCRD()
	c := client.NewClientBuilder().WithObjects(&svc, &secret, &crd).Build()
	rec.Client = c
	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: "one",
		},
	}
	err := rec.updateCRD(ctx, req)
	if err != nil {
		t.Errorf("Failed updating CRD: %v", err)
	}
}

func TestInjectSvcToConversionWebhook(t *testing.T) {
	svc := newService()
	crd := newCRD()
	name := types.NamespacedName{
		Name:      svc.Name,
		Namespace: svc.Namespace,
	}
	err := injectService(&crd, name)
	if err != nil {
		t.Errorf("Failed: error when injecting: %v", err)
	}
	val := crd.Spec.Conversion.Webhook.ClientConfig.Service.Name
	if val != "foo" {
		t.Errorf("Wrong service name injected: %v", val)
	}
	val = crd.Spec.Conversion.Webhook.ClientConfig.Service.Namespace
	if val != "default" {
		t.Errorf("Wrong service namespace injected: %v", val)
	}
}

func TestInjectCertToConversionWebhook(t *testing.T) {
	certPEM := []byte("foobar")
	crd := newCRD()
	err := injectCert(&crd, certPEM)
	if err != nil {
		t.Errorf("Failed: error when injecting: %v", err)
	}
	if string(crd.Spec.Conversion.Webhook.ClientConfig.CABundle) != "foobar" {
		t.Errorf("Wrong certificate name injected: %v", string(crd.Spec.Conversion.Webhook.ClientConfig.CABundle))
	}
}
func TestPopulateSecret(t *testing.T) {
	secret := newSecret()
	caArtifacts := KeyPairArtifacts{
		Cert:    &x509.Certificate{},
		Key:     &rsa.PrivateKey{},
		CertPEM: []byte("foobarca"),
		KeyPEM:  []byte("foobarcakey"),
	}
	cert := []byte("foobarcert")
	key := []byte("foobarkey")
	populateSecret(cert, key, &caArtifacts, &secret)
	if !bytes.Equal(secret.Data["tls.crt"], cert) {
		t.Errorf("secret value for tls.crt is wrong:%v", cert)
	}
	if !bytes.Equal(secret.Data["tls.key"], key) {
		t.Errorf("secret value for tls.key is wrong:%v", cert)
	}
	if !bytes.Equal(secret.Data["ca.crt"], caArtifacts.CertPEM) {
		t.Errorf("secret value for ca.crt is wrong:%v", cert)
	}
	if !bytes.Equal(secret.Data["ca.key"], caArtifacts.KeyPEM) {
		t.Errorf("secret value for ca.key is wrong:%v", cert)
	}
}

func TestCreateCACert(t *testing.T) {
	rec := newReconciler()
	caArtifacts, err := rec.CreateCACert(time.Now(), time.Now().AddDate(1, 0, 0))
	if err != nil {
		t.Errorf(failedCreateCaCerts, err)
	}
	if !rec.validCACert(caArtifacts.CertPEM, caArtifacts.KeyPEM) {
		t.Errorf(invalidCerts, caArtifacts.CertPEM, caArtifacts.KeyPEM)
	}
}

func TestCreateCertPEM(t *testing.T) {
	rec := newReconciler()
	caArtifacts, err := rec.CreateCACert(time.Now(), time.Now().AddDate(1, 0, 0))
	if err != nil {
		t.Fatalf(failedCreateCaCerts, err)
	}
	certPEM, keyPEM, err := rec.CreateCertPEM(caArtifacts, time.Now(), time.Now().AddDate(1, 0, 0))
	if err != nil {
		t.Errorf(failedCreateServerCerts, err)
	}
	if !rec.validServerCert(caArtifacts.CertPEM, certPEM, keyPEM) {
		t.Errorf(invalidCerts, certPEM, keyPEM)
	}
}
func TestValidCert(t *testing.T) {
	rec := newReconciler()
	rec.dnsName = dnsName
	caArtifacts, err := rec.CreateCACert(time.Now(), time.Now().AddDate(1, 0, 0))
	if err != nil {
		t.Fatalf(failedCreateCaCerts, err)
	}
	certPEM, keyPEM, err := rec.CreateCertPEM(caArtifacts, time.Now(), time.Now().AddDate(1, 0, 0))
	if err != nil {
		t.Errorf(failedCreateServerCerts, err)
	}
	ok, err := ValidCert(caArtifacts.CertPEM, certPEM, keyPEM, dnsName, time.Now())
	if err != nil {
		t.Errorf("error validating cert: %v", err)
	}
	if !ok {
		t.Errorf("certificate is invalid")
	}
}

func TestRefreshCertIfNeeded(t *testing.T) {
	rec := newReconciler()
	secret := newSecret()
	c := client.NewClientBuilder().WithObjects(&secret).Build()
	rec.Client = c
	rec.dnsName = dnsName
	caArtifacts, err := rec.CreateCACert(time.Now().AddDate(-1, 0, 0), time.Now().AddDate(0, -1, 0))
	if err != nil {
		t.Fatalf(failedCreateCaCerts, err)
	}
	certPEM, keyPEM, err := rec.CreateCertPEM(caArtifacts, time.Now(), time.Now().AddDate(1, 0, 0))
	if err != nil {
		t.Errorf(failedCreateServerCerts, err)
	}
	populateSecret(certPEM, keyPEM, caArtifacts, &secret)
	ok, err := rec.refreshCertIfNeeded(&secret)
	if err != nil {
		t.Errorf("could not verify refresh need: %v", err)
	}
	if !ok {
		t.Error("expected refresh true. got false")
	}
	ok, err = rec.refreshCertIfNeeded(&secret)
	if err != nil {
		t.Errorf("could not verify refresh need: %v", err)
	}
	if !ok {
		t.Error("expected refresh false. got true")
	}
}

func TestCheckCerts(t *testing.T) {
	rec := newReconciler()
	rec.dnsName = dnsName
	caArtifacts, err := rec.CreateCACert(time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 2))
	if err != nil {
		t.Fatalf(failedCreateCaCerts, err)
	}
	certPEM, keyPEM, err := rec.CreateCertPEM(caArtifacts, time.Now(), time.Now().AddDate(0, 0, 1))
	if err != nil {
		t.Errorf(failedCreateServerCerts, err)
	}
	os.WriteFile(cacrt, caArtifacts.CertPEM, 0644)
	os.WriteFile(tlscrt, certPEM, 0644)
	os.WriteFile(tlskey, keyPEM, 0644)
	cert := CertInfo{
		CertDir:  "/tmp",
		CertName: "tls",
		CAName:   "ca",
		KeyName:  "key",
	}
	err = CheckCerts(cert, rec.dnsName, time.Now())
	if err != nil {
		t.Errorf("error checking valid cert: %v", err)
	}
	err = CheckCerts(cert, rec.dnsName, time.Now().AddDate(-1, 0, 0))
	if err == nil {
		t.Error("expected failure due to expired certificate, got success")
	}
	err = CheckCerts(cert, "wrong", time.Now())
	if err == nil {
		t.Error("expected failure due to dns name got, success")
	}
	cert.CAName = "wrong"
	err = CheckCerts(cert, rec.dnsName, time.Now())
	if err == nil {
		t.Error("expected failure due to wrong certificate name, got success")
	}
}

func TestCheckCertChain(t *testing.T) {
	rec := newReconciler()
	rec.dnsName = dnsName
	caArtifacts, err := rec.CreateCACert(time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 2))
	if err != nil {
		t.Fatalf(failedCreateCaCerts, err)
	}
	chainArtifacts, err := rec.CreateCAChain(caArtifacts, time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 2))
	if err != nil {
		t.Fatalf(failedCreateCaChain, err)
	}
	certPEM, keyPEM, err := rec.CreateCertPEM(chainArtifacts, time.Now(), time.Now().AddDate(0, 0, 1))
	if err != nil {
		t.Errorf(failedCreateServerCerts, err)
	}
	os.WriteFile(cacrt, caArtifacts.CertPEM, 0644)
	os.WriteFile(tlscrt, certPEM, 0644)
	f, _ := os.OpenFile(tlscrt, os.O_APPEND|os.O_WRONLY, 0644)
	defer f.Close()
	if _, err = f.Write(chainArtifacts.CertPEM); err != nil {
		t.Errorf(failedCreateCaChain, err)
	}
	os.WriteFile(tlskey, keyPEM, 0644)
	cert := CertInfo{
		CertDir:  "/tmp",
		CertName: "tls",
		CAName:   "ca",
		KeyName:  "key",
	}
	err = CheckCerts(cert, rec.dnsName, time.Now())
	if err != nil {
		t.Errorf("error checking valid cert: %v", err)
	}
	err = CheckCerts(cert, rec.dnsName, time.Now().AddDate(-1, 0, 0))
	if err == nil {
		t.Error("expected failure due to expired certificate, got success")
	}
	err = CheckCerts(cert, "wrong", time.Now())
	if err == nil {
		t.Error("expected failure due to dns name got, success")
	}
	cert.CAName = "wrong"
	err = CheckCerts(cert, rec.dnsName, time.Now())
	if err == nil {
		t.Error("expected failure due to wrong certificate name, got success")
	}
}
