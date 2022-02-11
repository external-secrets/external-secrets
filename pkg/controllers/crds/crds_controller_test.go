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
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newReconciler() Reconciler {
	return Reconciler{
		CrdResources: []string{"one", "two", "three"},
		SvcLabels:    map[string]string{"foo": "bar"},
		SecretLabels: map[string]string{"foo": "bar"},
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
func TestConvertToWebhookInfo(t *testing.T) {
	rec := newReconciler()
	info := rec.ConvertToWebhookInfo()
	if len(info) != 3 {
		t.Errorf("Convert to WebhookInfo failed. Total resources:%d", len(info))
	}
	for _, v := range info {
		if v.Type != CRDConversion {
			t.Errorf("Convert to WebhookInfo failed. wrong type:%v", v.Type)
		}
		if v.Name != "one" && v.Name != "two" && v.Name != "three" {
			t.Errorf("Convert to WebhookInfo failed. wrong name:%v", v.Name)
		}
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
		t.Errorf("Failed updating CRD:%v", err)
	}
}

func TestInjectSvcToConversionWebhook(t *testing.T) {
	svc := newService()
	crd := newCRD()
	crdunmarshalled := make(map[string]interface{})
	crdJSON, err := json.Marshal(crd)
	if err != nil {
		t.Fatal("Could not setup test")
	}
	err = json.Unmarshal(crdJSON, &crdunmarshalled)
	if err != nil {
		t.Fatal("Could not setup test")
	}
	u := unstructured.Unstructured{
		Object: crdunmarshalled,
	}
	err = injectSvcToConversionWebhook(&u, &svc)
	if err != nil {
		t.Errorf("Failed: error when injecting: %v", err)
	}
	val, found, err := unstructured.NestedString(u.Object, "spec", "conversion", "webhook", "clientConfig", "service", "name")
	if err != nil {
		t.Error("Error when searching for field")
	}
	if !found {
		t.Error("Field not found, mutation failed")
	}
	if val != "foo" {
		t.Errorf("Wrong service name injected: %v", val)
	}

	val, found, err = unstructured.NestedString(u.Object, "spec", "conversion", "webhook", "clientConfig", "service", "namespace")
	if err != nil {
		t.Error("Error when searching for field")
	}
	if !found {
		t.Error("Field not found, mutation failed")
	}
	if val != "default" {
		t.Errorf("Wrong service namespace injected: %v", val)
	}
}

func TestInjectCertToConversionWebhook(t *testing.T) {
	certPEM := []byte("foobar")
	crd := newCRD()
	crdunmarshalled := make(map[string]interface{})
	crdJson, err := json.Marshal(crd)
	if err != nil {
		t.Fatal("Could not setup test")
	}
	err = json.Unmarshal(crdJson, &crdunmarshalled)
	if err != nil {
		t.Fatal("Could not setup test")
	}
	u := unstructured.Unstructured{
		Object: crdunmarshalled,
	}
	err = injectCertToConversionWebhook(&u, certPEM)
	if err != nil {
		t.Errorf("Failed: error when injecting: %v", err)
	}
	val, found, err := unstructured.NestedString(u.Object, "spec", "conversion", "webhook", "clientConfig", "caBundle")
	if err != nil {
		t.Error("Error when searching for field")
	}
	if !found {
		t.Error("Field not found, mutation failed")
	}
	if val != "Zm9vYmFy" {
		t.Errorf("Wrong certificate name injected: %v", val)
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
	if string(secret.Data["tls.crt"]) != string(cert) {
		t.Errorf("secret value for tls.crt is wrong:%v", cert)
	}
	if string(secret.Data["tls.key"]) != string(key) {
		t.Errorf("secret value for tls.key is wrong:%v", cert)
	}
	if string(secret.Data["ca.crt"]) != string(caArtifacts.CertPEM) {
		t.Errorf("secret value for ca.crt is wrong:%v", cert)
	}
	if string(secret.Data["ca.key"]) != string(caArtifacts.KeyPEM) {
		t.Errorf("secret value for ca.key is wrong:%v", cert)
	}
}

func TestCreateCACert(t *testing.T) {
	rec := newReconciler()
	caArtifacts, err := rec.CreateCACert(time.Now(), time.Now().AddDate(1, 0, 0))
	if err != nil {
		t.Errorf("could not create certificates:%v", err)
	}
	if !rec.validCACert(caArtifacts.CertPEM, caArtifacts.KeyPEM) {
		t.Errorf("generated certificates are invalid:%v", caArtifacts)
	}
}

func TestCreateCertPEM(t *testing.T) {
	rec := newReconciler()
	caArtifacts, err := rec.CreateCACert(time.Now(), time.Now().AddDate(1, 0, 0))
	if err != nil {
		t.Fatalf("could not create ca certificates:%v", err)
	}
	certPEM, keyPEM, err := rec.CreateCertPEM(caArtifacts, time.Now(), time.Now().AddDate(1, 0, 0))
	if err != nil {
		t.Errorf("could not create server certificates: %v", err)
	}
	if !rec.validServerCert(caArtifacts.CertPEM, certPEM, keyPEM) {
		t.Errorf("generated certificates are invalid:%v,%v", certPEM, keyPEM)
	}
}
func TestValidCert(t *testing.T) {
	rec := newReconciler()
	rec.dnsName = "foobar"
	caArtifacts, err := rec.CreateCACert(time.Now(), time.Now().AddDate(1, 0, 0))
	if err != nil {
		t.Fatalf("could not create ca certificates:%v", err)
	}
	certPEM, keyPEM, err := rec.CreateCertPEM(caArtifacts, time.Now(), time.Now().AddDate(1, 0, 0))
	if err != nil {
		t.Errorf("could not create server certificates: %v", err)
	}
	ok, err := ValidCert(caArtifacts.CertPEM, certPEM, keyPEM, "foobar", time.Now())
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
	rec.dnsName = "foobar"
	caArtifacts, err := rec.CreateCACert(time.Now().AddDate(-1, 0, 0), time.Now().AddDate(0, -1, 0))
	if err != nil {
		t.Fatalf("could not create ca certificates:%v", err)
	}
	certPEM, keyPEM, err := rec.CreateCertPEM(caArtifacts, time.Now(), time.Now().AddDate(1, 0, 0))
	if err != nil {
		t.Errorf("could not create server certificates: %v", err)
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
