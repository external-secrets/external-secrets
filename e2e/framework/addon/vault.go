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

package addon

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	vault "github.com/hashicorp/vault/api"

	// nolint
	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework/util"
)

type Vault struct {
	chart        *HelmChart
	Namespace    string
	PodName      string
	VaultClient  *vault.Client
	VaultURL     string
	VaultMtlsURL string

	RootToken          string
	VaultServerCA      []byte
	ServerCert         []byte
	ServerKey          []byte
	VaultClientCA      []byte
	ClientCert         []byte
	ClientKey          []byte
	JWTPubkey          []byte
	JWTPrivKey         []byte
	JWTToken           string
	JWTRole            string
	JWTPath            string
	JWTK8sPath         string
	KubernetesAuthPath string
	KubernetesAuthRole string

	AppRoleSecret string
	AppRoleID     string
	AppRolePath   string
}

const privatePemType = "RSA PRIVATE KEY"

func NewVault(namespace string) *Vault {
	repo := "hashicorp-" + namespace
	return &Vault{
		chart: &HelmChart{
			Namespace:    namespace,
			ReleaseName:  fmt.Sprintf("vault-%s", namespace), // avoid cluster role collision
			Chart:        fmt.Sprintf("%s/vault", repo),
			ChartVersion: "0.11.0",
			Repo: ChartRepo{
				Name: repo,
				URL:  "https://helm.releases.hashicorp.com",
			},
			Values: []string{"/k8s/vault.values.yaml"},
		},
		Namespace: namespace,
	}
}

type OperatorInitResponse struct {
	UnsealKeysB64 []string `json:"unseal_keys_b64"`
	RootToken     string   `json:"root_token"`
}

func (l *Vault) Install() error {
	ginkgo.By("Installing vault in " + l.Namespace)
	err := l.chart.Install()
	if err != nil {
		return err
	}

	err = l.patchVaultService()
	if err != nil {
		return err
	}

	err = l.initVault()
	if err != nil {
		return err
	}

	err = l.configureVault()
	if err != nil {
		return err
	}

	return nil
}

func (l *Vault) patchVaultService() error {
	serviceName := fmt.Sprintf("vault-%s", l.Namespace)
	servicePatch := []byte(`[{"op": "add", "path": "/spec/ports/-", "value": { "name": "https-mtls", "port": 8210, "protocol": "TCP", "targetPort": 8210 }}]`)
	clientSet := l.chart.config.KubeClientSet
	_, err := clientSet.CoreV1().Services(l.Namespace).
		Patch(context.Background(), serviceName, types.JSONPatchType, servicePatch, metav1.PatchOptions{})
	return err
}

func (l *Vault) initVault() error {
	sec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vault-tls-config",
			Namespace: l.Namespace,
		},
		Data: map[string][]byte{},
	}

	// vault-config contains vault init config and policies
	files, err := os.ReadDir("/k8s/vault-config")
	if err != nil {
		return err
	}
	for _, f := range files {
		name := f.Name()
		data := mustReadFile(fmt.Sprintf("/k8s/vault-config/%s", name))
		sec.Data[name] = data
	}

	// gen certificates and put them into the secret
	serverRootPem, serverPem, serverKeyPem, clientRootPem, clientPem, clientKeyPem, err := genVaultCertificates(l.Namespace)
	if err != nil {
		return fmt.Errorf("unable to gen vault certs: %w", err)
	}
	jwtPrivkey, jwtPubkey, jwtToken, err := genVaultJWTKeys()
	if err != nil {
		return fmt.Errorf("unable to generate vault jwt keys: %w", err)
	}

	// pass certs to secret
	sec.Data["vault-server-ca.pem"] = serverRootPem
	sec.Data["server-cert.pem"] = serverPem
	sec.Data["server-cert-key.pem"] = serverKeyPem
	sec.Data["vault-client-ca.pem"] = clientRootPem
	sec.Data["es-client.pem"] = clientPem
	sec.Data["es-client-key.pem"] = clientKeyPem
	sec.Data["jwt-pubkey.pem"] = jwtPubkey

	// make certs available to the struct
	// so it can be used by the provider
	l.VaultServerCA = serverRootPem
	l.ServerCert = serverPem
	l.ServerKey = serverKeyPem
	l.VaultClientCA = clientRootPem
	l.ClientCert = clientPem
	l.ClientKey = clientKeyPem
	l.JWTPrivKey = jwtPrivkey
	l.JWTPubkey = jwtPubkey
	l.JWTToken = jwtToken
	l.JWTPath = "myjwt"                                // see configure-vault.sh
	l.JWTK8sPath = "myjwtk8s"                          // see configure-vault.sh
	l.JWTRole = "external-secrets-operator"            // see configure-vault.sh
	l.KubernetesAuthPath = "mykubernetes"              // see configure-vault.sh
	l.KubernetesAuthRole = "external-secrets-operator" // see configure-vault.sh

	ginkgo.By("Creating vault TLS secret")
	err = l.chart.config.CRClient.Create(context.Background(), sec)
	if err != nil {
		return err
	}

	ginkgo.By("Waiting for vault pods to be running")
	pl, err := util.WaitForPodsRunning(l.chart.config.KubeClientSet, 1, l.Namespace, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=vault",
	})
	if err != nil {
		return fmt.Errorf("error waiting for vault to be running: %w", err)
	}
	l.PodName = pl.Items[0].Name

	ginkgo.By("Initializing vault")
	out, err := util.ExecCmd(
		l.chart.config.KubeClientSet,
		l.chart.config.KubeConfig,
		l.PodName, l.Namespace, "vault operator init --format=json")
	if err != nil {
		return fmt.Errorf("error initializing vault: %w", err)
	}

	ginkgo.By("Parsing init response")
	var res OperatorInitResponse
	err = json.Unmarshal([]byte(out), &res)
	if err != nil {
		return err
	}
	l.RootToken = res.RootToken

	ginkgo.By("Unsealing vault")
	for _, k := range res.UnsealKeysB64 {
		_, err = util.ExecCmd(
			l.chart.config.KubeClientSet,
			l.chart.config.KubeConfig,
			l.PodName, l.Namespace, "vault operator unseal "+k)
		if err != nil {
			return fmt.Errorf("unable to unseal vault: %w", err)
		}
	}

	// vault becomes ready after it has been unsealed
	err = util.WaitForPodsReady(l.chart.config.KubeClientSet, 1, l.Namespace, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=vault",
	})
	if err != nil {
		return fmt.Errorf("error waiting for vault to be ready: %w", err)
	}
	serverCA := l.VaultServerCA
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(serverCA)
	if !ok {
		panic("unable to append server ca cert")
	}
	cfg := vault.DefaultConfig()
	l.VaultURL = fmt.Sprintf("https://vault-%s.%s.svc.cluster.local:8200", l.Namespace, l.Namespace)
	l.VaultMtlsURL = fmt.Sprintf("https://vault-%s.%s.svc.cluster.local:8210", l.Namespace, l.Namespace)
	cfg.Address = l.VaultURL
	cfg.HttpClient.Transport.(*http.Transport).TLSClientConfig.RootCAs = caCertPool
	l.VaultClient, err = vault.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("unable to create vault client: %w", err)
	}
	l.VaultClient.SetToken(l.RootToken)

	return nil
}

func (l *Vault) configureVault() error {
	ginkgo.By("configuring vault")
	cmd := `sh /etc/vault-config/configure-vault.sh %s`
	_, err := util.ExecCmd(
		l.chart.config.KubeClientSet,
		l.chart.config.KubeConfig,
		l.PodName, l.Namespace, fmt.Sprintf(cmd, l.RootToken))
	if err != nil {
		return fmt.Errorf("unable to configure vault: %w", err)
	}

	// configure appRole
	l.AppRolePath = "myapprole"
	req := l.VaultClient.NewRequest(http.MethodGet, fmt.Sprintf("/v1/auth/%s/role/eso-e2e-role/role-id", l.AppRolePath))
	res, err := l.VaultClient.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer res.Body.Close()
	sec, err := vault.ParseSecret(res.Body)
	if err != nil {
		return err
	}

	l.AppRoleID = sec.Data["role_id"].(string)

	// parse role id
	req = l.VaultClient.NewRequest(http.MethodPost, fmt.Sprintf("/v1/auth/%s/role/eso-e2e-role/secret-id", l.AppRolePath))
	res, err = l.VaultClient.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer res.Body.Close()
	sec, err = vault.ParseSecret(res.Body)
	if err != nil {
		return err
	}
	l.AppRoleSecret = sec.Data["secret_id"].(string)
	return nil
}

func (l *Vault) Logs() error {
	return l.chart.Logs()
}

func (l *Vault) Uninstall() error {
	return l.chart.Uninstall()
}

func (l *Vault) Setup(cfg *Config) error {
	return l.chart.Setup(cfg)
}

// nolint:gocritic
func genVaultCertificates(namespace string) ([]byte, []byte, []byte, []byte, []byte, []byte, error) {
	// gen server ca + certs
	serverRootCert, serverRootPem, serverRootKey, err := genCARoot()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("unable to generate ca cert: %w", err)
	}
	serverPem, serverKey, err := genPeerCert(serverRootCert, serverRootKey, "vault", []string{
		"localhost",
		"vault-" + namespace,
		fmt.Sprintf("vault-%s.%s.svc.cluster.local", namespace, namespace)})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("unable to generate vault server cert")
	}
	serverKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  privatePemType,
		Bytes: x509.MarshalPKCS1PrivateKey(serverKey)},
	)
	// gen client ca + certs
	clientRootCert, clientRootPem, clientRootKey, err := genCARoot()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("unable to generate ca cert: %w", err)
	}
	clientPem, clientKey, err := genPeerCert(clientRootCert, clientRootKey, "vault-client", nil)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("unable to generate vault server cert")
	}
	clientKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  privatePemType,
		Bytes: x509.MarshalPKCS1PrivateKey(clientKey)},
	)
	return serverRootPem, serverPem, serverKeyPem, clientRootPem, clientPem, clientKeyPem, err
}

func genVaultJWTKeys() ([]byte, []byte, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, "", err
	}
	privPem := pem.EncodeToMemory(&pem.Block{
		Type:  privatePemType,
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	pk, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, nil, "", err
	}
	pubPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pk,
	})

	token := jwt.NewWithClaims(jwt.SigningMethodPS256, jwt.MapClaims{
		"aud":  "vault.client",
		"sub":  "vault@example",
		"iss":  "example.iss",
		"user": "eso",
		"exp":  time.Now().Add(time.Hour).Unix(),
		"nbf":  time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(key)
	if err != nil {
		return nil, nil, "", err
	}

	return privPem, pubPem, tokenString, nil
}

func genCARoot() (*x509.Certificate, []byte, *rsa.PrivateKey, error) {
	tpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:      []string{"/dev/null"},
			Organization: []string{"External Secrets ACME"},
			CommonName:   "External Secrets Vault CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
	}
	pkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, err
	}
	rootCert, rootPEM, err := genCert(&tpl, &tpl, &pkey.PublicKey, pkey)
	return rootCert, rootPEM, pkey, err
}

func genCert(template, parent *x509.Certificate, publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey) (*x509.Certificate, []byte, error) {
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent, publicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	b := pem.Block{Type: "CERTIFICATE", Bytes: certBytes}
	certPEM := pem.EncodeToMemory(&b)

	return cert, certPEM, err
}

func genPeerCert(signingCert *x509.Certificate, signingKey *rsa.PrivateKey, cn string, dnsNames []string) ([]byte, *rsa.PrivateKey, error) {
	pkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	tpl := x509.Certificate{
		Subject: pkix.Name{
			Country:      []string{"/dev/null"},
			Organization: []string{"External Secrets ACME"},
			CommonName:   cn,
		},
		SerialNumber:   big.NewInt(1),
		NotBefore:      time.Now(),
		NotAfter:       time.Now().Add(time.Hour),
		KeyUsage:       x509.KeyUsageCRLSign,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IsCA:           false,
		MaxPathLenZero: true,
		IPAddresses:    []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:       dnsNames,
	}
	_, serverPEM, err := genCert(&tpl, signingCert, &pkey.PublicKey, signingKey)
	return serverPEM, pkey, err
}

func mustReadFile(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}
