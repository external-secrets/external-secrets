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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

const (
	certName             = "tls.crt"
	keyName              = "tls.key"
	caCertName           = "ca.crt"
	caKeyName            = "ca.key"
	certValidityDuration = 10 * 365 * 24 * time.Hour
	LookaheadInterval    = 90 * 24 * time.Hour

	errResNotReady       = "resource not ready: %s"
	errSubsetsNotReady   = "subsets not ready"
	errAddressesNotReady = "addresses not ready"
)

type Reconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	recorder        record.EventRecorder
	SvcName         string
	SvcNamespace    string
	SecretName      string
	SecretNamespace string
	CrdResources    []string
	dnsName         string
	CAName          string
	CAChainName     string
	CAOrganization  string
	RequeueInterval time.Duration

	// the controller is ready when all crds are injected
	// and the controller is elected as leader
	leaderChan       <-chan struct{}
	leaderElected    bool
	readyStatusMapMu *sync.Mutex
	readyStatusMap   map[string]bool
}

func New(k8sClient client.Client, scheme *runtime.Scheme, leaderChan <-chan struct{}, logger logr.Logger,
	interval time.Duration, svcName, svcNamespace, secretName, secretNamespace string, resources []string) *Reconciler {
	return &Reconciler{
		Client:           k8sClient,
		Log:              logger,
		Scheme:           scheme,
		SvcName:          svcName,
		SvcNamespace:     svcNamespace,
		SecretName:       secretName,
		SecretNamespace:  secretNamespace,
		RequeueInterval:  interval,
		CrdResources:     resources,
		CAName:           "external-secrets",
		CAOrganization:   "external-secrets",
		leaderChan:       leaderChan,
		readyStatusMapMu: &sync.Mutex{},
		readyStatusMap:   map[string]bool{},
	}
}

type CertInfo struct {
	CertDir  string
	CertName string
	KeyName  string
	CAName   string
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("CustomResourceDefinition", req.NamespacedName)
	if contains(r.CrdResources, req.NamespacedName.Name) {
		err := r.updateCRD(ctx, req)
		if err != nil {
			log.Error(err, "failed to inject conversion webhook")
			r.readyStatusMapMu.Lock()
			r.readyStatusMap[req.NamespacedName.Name] = false
			r.readyStatusMapMu.Unlock()
			return ctrl.Result{}, err
		}
		r.readyStatusMapMu.Lock()
		r.readyStatusMap[req.NamespacedName.Name] = true
		r.readyStatusMapMu.Unlock()
	}
	return ctrl.Result{RequeueAfter: r.RequeueInterval}, nil
}

// ReadyCheck reviews if all webhook configs have been injected into the CRDs
// and if the referenced webhook service is ready.
func (r *Reconciler) ReadyCheck(_ *http.Request) error {
	// skip readiness check if we're not leader
	// as we depend on caches and being able to reconcile Webhooks
	if !r.leaderElected {
		select {
		case <-r.leaderChan:
			r.leaderElected = true
		default:
			return nil
		}
	}
	if err := r.checkCRDs(); err != nil {
		return err
	}
	return r.checkEndpoints()
}

func (r *Reconciler) checkCRDs() error {
	for _, res := range r.CrdResources {
		r.readyStatusMapMu.Lock()
		rdy := r.readyStatusMap[res]
		r.readyStatusMapMu.Unlock()
		if !rdy {
			return fmt.Errorf(errResNotReady, res)
		}
	}
	return nil
}

func (r *Reconciler) checkEndpoints() error {
	var eps corev1.Endpoints
	err := r.Get(context.TODO(), types.NamespacedName{
		Name:      r.SvcName,
		Namespace: r.SvcNamespace,
	}, &eps)
	if err != nil {
		return err
	}
	if len(eps.Subsets) == 0 {
		return fmt.Errorf(errSubsetsNotReady)
	}
	if len(eps.Subsets[0].Addresses) == 0 {
		return fmt.Errorf(errAddressesNotReady)
	}
	return nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("custom-resource-definition")
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&apiext.CustomResourceDefinition{}).
		Complete(r)
}

func (r *Reconciler) updateCRD(ctx context.Context, req ctrl.Request) error {
	secret := corev1.Secret{}
	secretName := types.NamespacedName{
		Name:      r.SecretName,
		Namespace: r.SecretNamespace,
	}
	err := r.Get(context.Background(), secretName, &secret)
	if err != nil {
		return err
	}
	var updatedResource apiext.CustomResourceDefinition
	if err := r.Get(ctx, req.NamespacedName, &updatedResource); err != nil {
		return err
	}
	svc := types.NamespacedName{
		Name:      r.SvcName,
		Namespace: r.SvcNamespace,
	}
	if err := injectService(&updatedResource, svc); err != nil {
		return err
	}
	r.dnsName = fmt.Sprintf("%v.%v.svc", r.SvcName, r.SvcNamespace)
	need, err := r.refreshCertIfNeeded(&secret)
	if err != nil {
		return err
	}
	if need {
		artifacts, err := buildArtifactsFromSecret(&secret)
		if err != nil {
			return err
		}
		if err := injectCert(&updatedResource, artifacts.CertPEM); err != nil {
			return err
		}
	}
	return r.Update(ctx, &updatedResource)
}

func injectService(crd *apiext.CustomResourceDefinition, svc types.NamespacedName) error {
	if crd.Spec.Conversion == nil ||
		crd.Spec.Conversion.Webhook == nil ||
		crd.Spec.Conversion.Webhook.ClientConfig == nil ||
		crd.Spec.Conversion.Webhook.ClientConfig.Service == nil {
		return fmt.Errorf("unexpected crd conversion webhook config")
	}
	crd.Spec.Conversion.Webhook.ClientConfig.Service.Namespace = svc.Namespace
	crd.Spec.Conversion.Webhook.ClientConfig.Service.Name = svc.Name
	return nil
}

func injectCert(crd *apiext.CustomResourceDefinition, certPem []byte) error {
	if crd.Spec.Conversion == nil ||
		crd.Spec.Conversion.Webhook == nil ||
		crd.Spec.Conversion.Webhook.ClientConfig == nil {
		return fmt.Errorf("unexpected crd conversion webhook config")
	}
	crd.Spec.Conversion.Webhook.ClientConfig.CABundle = certPem
	return nil
}

type KeyPairArtifacts struct {
	Cert    *x509.Certificate
	Key     *rsa.PrivateKey
	CertPEM []byte
	KeyPEM  []byte
}

func populateSecret(cert, key []byte, caArtifacts *KeyPairArtifacts, secret *corev1.Secret) {
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	secret.Data[caCertName] = caArtifacts.CertPEM
	secret.Data[caKeyName] = caArtifacts.KeyPEM
	secret.Data[certName] = cert
	secret.Data[keyName] = key
}

func ValidCert(caCert, cert, key []byte, dnsName string, at time.Time) (bool, error) {
	if len(caCert) == 0 || len(cert) == 0 || len(key) == 0 {
		return false, errors.New("empty cert")
	}

	pool := x509.NewCertPool()
	caDer, _ := pem.Decode(caCert)
	if caDer == nil {
		return false, errors.New("bad CA cert")
	}
	cac, err := x509.ParseCertificate(caDer.Bytes)
	if err != nil {
		return false, err
	}
	pool.AddCert(cac)

	_, err = tls.X509KeyPair(cert, key)
	if err != nil {
		return false, err
	}

	b, rest := pem.Decode(cert)
	if b == nil {
		return false, err
	}
	if len(rest) > 0 {
		intermediate, _ := pem.Decode(rest)
		inter, err := x509.ParseCertificate(intermediate.Bytes)
		if err != nil {
			return false, err
		}
		pool.AddCert(inter)
	}

	crt, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return false, err
	}
	_, err = crt.Verify(x509.VerifyOptions{
		DNSName:     dnsName,
		Roots:       pool,
		CurrentTime: at,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func lookaheadTime() time.Time {
	return time.Now().Add(LookaheadInterval)
}

func (r *Reconciler) validServerCert(caCert, cert, key []byte) bool {
	valid, err := ValidCert(caCert, cert, key, r.dnsName, lookaheadTime())
	if err != nil {
		return false
	}
	return valid
}

func (r *Reconciler) validCACert(cert, key []byte) bool {
	valid, err := ValidCert(cert, cert, key, r.CAName, lookaheadTime())
	if err != nil {
		return false
	}
	return valid
}

func (r *Reconciler) refreshCertIfNeeded(secret *corev1.Secret) (bool, error) {
	if secret.Data == nil || !r.validCACert(secret.Data[caCertName], secret.Data[caKeyName]) {
		if err := r.refreshCerts(true, secret); err != nil {
			return false, err
		}
		return true, nil
	}
	if !r.validServerCert(secret.Data[caCertName], secret.Data[certName], secret.Data[keyName]) {
		if err := r.refreshCerts(false, secret); err != nil {
			return false, err
		}
		return true, nil
	}
	return true, nil
}

func (r *Reconciler) refreshCerts(refreshCA bool, secret *corev1.Secret) error {
	var caArtifacts *KeyPairArtifacts
	now := time.Now()
	begin := now.Add(-1 * time.Hour)
	end := now.Add(certValidityDuration)
	if refreshCA {
		var err error
		caArtifacts, err = r.CreateCACert(begin, end)
		if err != nil {
			return err
		}
	} else {
		var err error
		caArtifacts, err = buildArtifactsFromSecret(secret)
		if err != nil {
			return err
		}
	}
	cert, key, err := r.CreateCertPEM(caArtifacts, begin, end)
	if err != nil {
		return err
	}
	return r.writeSecret(cert, key, caArtifacts, secret)
}

func buildArtifactsFromSecret(secret *corev1.Secret) (*KeyPairArtifacts, error) {
	caPem, ok := secret.Data[caCertName]
	if !ok {
		return nil, fmt.Errorf("cert secret is not well-formed, missing %s", caCertName)
	}
	keyPem, ok := secret.Data[caKeyName]
	if !ok {
		return nil, fmt.Errorf("cert secret is not well-formed, missing %s", caKeyName)
	}
	caDer, _ := pem.Decode(caPem)
	if caDer == nil {
		return nil, errors.New("bad CA cert")
	}
	caCert, err := x509.ParseCertificate(caDer.Bytes)
	if err != nil {
		return nil, err
	}
	keyDer, _ := pem.Decode(keyPem)
	if keyDer == nil {
		return nil, err
	}
	key, err := x509.ParsePKCS1PrivateKey(keyDer.Bytes)
	if err != nil {
		return nil, err
	}
	return &KeyPairArtifacts{
		Cert:    caCert,
		CertPEM: caPem,
		KeyPEM:  keyPem,
		Key:     key,
	}, nil
}

func (r *Reconciler) CreateCACert(begin, end time.Time) (*KeyPairArtifacts, error) {
	templ := &x509.Certificate{
		SerialNumber: big.NewInt(0),
		Subject: pkix.Name{
			CommonName:   r.CAName,
			Organization: []string{r.CAOrganization},
		},
		DNSNames: []string{
			r.CAName,
		},
		NotBefore:             begin,
		NotAfter:              end,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	der, err := x509.CreateCertificate(rand.Reader, templ, templ, key.Public(), key)
	if err != nil {
		return nil, err
	}
	certPEM, keyPEM, err := pemEncode(der, key)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}

	return &KeyPairArtifacts{Cert: cert, Key: key, CertPEM: certPEM, KeyPEM: keyPEM}, nil
}

func (r *Reconciler) CreateCAChain(ca *KeyPairArtifacts, begin, end time.Time) (*KeyPairArtifacts, error) {
	templ := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName:   r.CAChainName,
			Organization: []string{r.CAOrganization},
		},
		DNSNames: []string{
			r.CAChainName,
		},
		NotBefore:             begin,
		NotAfter:              end,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	der, err := x509.CreateCertificate(rand.Reader, templ, ca.Cert, key.Public(), ca.Key)
	if err != nil {
		return nil, err
	}
	certPEM, keyPEM, err := pemEncode(der, key)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}

	return &KeyPairArtifacts{Cert: cert, Key: key, CertPEM: certPEM, KeyPEM: keyPEM}, nil
}

func (r *Reconciler) CreateCertPEM(ca *KeyPairArtifacts, begin, end time.Time) ([]byte, []byte, error) {
	templ := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: r.dnsName,
		},
		DNSNames: []string{
			r.dnsName,
		},
		NotBefore:             begin,
		NotAfter:              end,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	der, err := x509.CreateCertificate(rand.Reader, templ, ca.Cert, key.Public(), ca.Key)
	if err != nil {
		return nil, nil, err
	}
	certPEM, keyPEM, err := pemEncode(der, key)
	if err != nil {
		return nil, nil, err
	}
	return certPEM, keyPEM, nil
}

func pemEncode(certificateDER []byte, key *rsa.PrivateKey) ([]byte, []byte, error) {
	certBuf := &bytes.Buffer{}
	if err := pem.Encode(certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: certificateDER}); err != nil {
		return nil, nil, err
	}
	keyBuf := &bytes.Buffer{}
	if err := pem.Encode(keyBuf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		return nil, nil, err
	}
	return certBuf.Bytes(), keyBuf.Bytes(), nil
}

func (r *Reconciler) writeSecret(cert, key []byte, caArtifacts *KeyPairArtifacts, secret *corev1.Secret) error {
	populateSecret(cert, key, caArtifacts, secret)
	return r.Update(context.Background(), secret)
}

// CheckCerts verifies that certificates exist in a given fs location
// and if they're valid.
func CheckCerts(c CertInfo, dnsName string, at time.Time) error {
	certFile := filepath.Join(c.CertDir, c.CertName)
	_, err := os.Stat(certFile)
	if err != nil {
		return err
	}
	ca, err := os.ReadFile(filepath.Join(c.CertDir, c.CAName))
	if err != nil {
		return err
	}
	cert, err := os.ReadFile(filepath.Join(c.CertDir, c.CertName))
	if err != nil {
		return err
	}
	key, err := os.ReadFile(filepath.Join(c.CertDir, c.KeyName))
	if err != nil {
		return err
	}
	ok, err := ValidCert(ca, cert, key, dnsName, at)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("certificate is not valid")
	}
	return nil
}
