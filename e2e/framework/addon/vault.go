/*
Copyright © 2025 ESO Maintainer Team

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

package addon

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/golang-jwt/jwt/v4"
	vault "github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework/util"
)

type Vault struct {
	chart         *HelmChart
	Namespace     string
	PodName       string
	VaultClient   *vault.Client
	VaultURL      string
	VaultMtlsURL  string
	portForwarder *PortForward

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

func NewVault() *Vault {
	repo := "hashicorp-vault"
	return &Vault{
		chart: &HelmChart{
			Namespace:    "vault",
			ReleaseName:  "vault",
			Chart:        fmt.Sprintf("%s/vault", repo),
			ChartVersion: "0.30.1",
			Repo: ChartRepo{
				Name: repo,
				URL:  "https://helm.releases.hashicorp.com",
			},
			Args: []string{
				"--create-namespace",
			},
			Values: []string{filepath.Join(AssetDir(), "vault.values.yaml")},
		},
		Namespace: "vault",
	}
}

type OperatorInitResponse struct {
	UnsealKeysB64 []string `json:"unseal_keys_b64"`
	RootToken     string   `json:"root_token"`
}

func (l *Vault) Install() error {
	// From Kubernetes 1.32+ on the oidc endpoint is not available to unauthenticated clients.
	// We create this clusterrole to allow vault to access the oidc endpoint.
	// see: https://github.com/ansible-collections/kubernetes.core/issues/868
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "allow-anon-oidc",
		},
	}
	_, err := controllerutil.CreateOrUpdate(GinkgoT().Context(), l.chart.config.CRClient, crb, func() error {
		crb.Subjects = []rbacv1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     "system:anonymous",
			},
		}
		crb.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "system:service-account-issuer-discovery",
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err = l.chart.Install(); err != nil {
		return err
	}

	if err = l.patchVaultService(); err != nil {
		return err
	}

	if err = l.initVault(); err != nil {
		return err
	}

	if err = l.configureVault(); err != nil {
		return err
	}

	return nil
}

func (l *Vault) patchVaultService() error {
	serviceName := l.chart.ReleaseName
	servicePatch := []byte(`[{"op": "add", "path": "/spec/ports/-", "value": { "name": "https-mtls", "port": 8210, "protocol": "TCP", "targetPort": 8210 }}]`)
	clientSet := l.chart.config.KubeClientSet
	_, err := clientSet.CoreV1().Services(l.Namespace).
		Patch(GinkgoT().Context(), serviceName, types.JSONPatchType, servicePatch, metav1.PatchOptions{})
	return err
}

func (l *Vault) initVault() error {
	// gen certificates and put them into the secret
	serverRootPem, serverPem, serverKeyPem, clientRootPem, clientPem, clientKeyPem, err := genVaultCertificates(l.Namespace, l.chart.ReleaseName)
	if err != nil {
		return fmt.Errorf("unable to gen vault certs: %w", err)
	}
	jwtPrivkey, jwtPubkey, jwtToken, err := genVaultJWTKeys()
	if err != nil {
		return fmt.Errorf("unable to generate vault jwt keys: %w", err)
	}

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

	// vault-config contains vault init config and policies
	files, err := os.ReadDir(fmt.Sprintf("%s/vault-config", AssetDir()))
	if err != nil {
		return err
	}
	sec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vault-tls-config",
			Namespace: l.Namespace,
		},
		Data: map[string][]byte{},
	}
	_, err = controllerutil.CreateOrUpdate(GinkgoT().Context(), l.chart.config.CRClient, sec, func() error {
		sec.Data = map[string][]byte{}
		for _, f := range files {
			name := f.Name()
			data := mustReadFile(fmt.Sprintf("%s/vault-config/%s", AssetDir(), name))
			sec.Data[name] = data
		}
		sec.Data["vault-server-ca.pem"] = serverRootPem
		sec.Data["server-cert.pem"] = serverPem
		sec.Data["server-cert-key.pem"] = serverKeyPem
		sec.Data["vault-client-ca.pem"] = clientRootPem
		sec.Data["es-client.pem"] = clientPem
		sec.Data["es-client-key.pem"] = clientKeyPem
		sec.Data["jwt-pubkey.pem"] = jwtPubkey

		return nil
	})
	if err != nil {
		return err
	}

	pl, err := util.WaitForPodsRunning(l.chart.config.KubeClientSet, 1, l.Namespace, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=vault",
	})
	if err != nil {
		return fmt.Errorf("error waiting for vault to be running: %w", err)
	}
	l.PodName = pl.Items[0].Name

	out, err := util.ExecCmd(
		l.chart.config.KubeClientSet,
		l.chart.config.KubeConfig,
		l.PodName, l.Namespace, "vault operator init --format=json")
	if err != nil {
		return fmt.Errorf("error initializing vault: %w", err)
	}

	var res OperatorInitResponse
	err = json.Unmarshal([]byte(out), &res)
	if err != nil {
		return err
	}
	l.RootToken = res.RootToken

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

	// This e2e test provider uses a local port-forwarded to talk to the vault API instead
	// of using the kubernetes service. This allows us to run the e2e test suite locally.
	l.portForwarder, err = NewPortForward(l.chart.config.KubeClientSet, l.chart.config.KubeConfig, "vault", l.chart.Namespace, 8200)
	if err != nil {
		return err
	}
	if err := l.portForwarder.Start(); err != nil {
		return err
	}

	serverCA := l.VaultServerCA
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(serverCA)
	if !ok {
		panic("unable to append server ca cert")
	}
	cfg := vault.DefaultConfig()
	l.VaultURL = fmt.Sprintf("https://%s.%s.svc.cluster.local:8200", l.chart.ReleaseName, l.Namespace)
	l.VaultMtlsURL = fmt.Sprintf("https://%s.%s.svc.cluster.local:8210", l.chart.ReleaseName, l.Namespace)
	cfg.Address = fmt.Sprintf("https://localhost:%d", l.portForwarder.localPort)
	cfg.HttpClient.Transport.(*http.Transport).TLSClientConfig.RootCAs = caCertPool
	l.VaultClient, err = vault.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("unable to create vault client: %w", err)
	}
	l.VaultClient.SetToken(l.RootToken)

	return nil
}

func (l *Vault) configureVault() error {
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
	defer func() {
		_ = res.Body.Close()
	}()
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
	defer func() {
		_ = res.Body.Close()
	}()
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
	if l.portForwarder != nil {
		l.portForwarder.Close()
		l.portForwarder = nil
	}
	if err := l.chart.Uninstall(); err != nil {
		return err
	}
	return l.chart.config.KubeClientSet.CoreV1().Namespaces().Delete(GinkgoT().Context(), l.chart.Namespace, metav1.DeleteOptions{})
}

func (l *Vault) Setup(cfg *Config) error {
	return l.chart.Setup(cfg)
}

// nolint:gocritic
func genVaultCertificates(namespace, serviceName string) ([]byte, []byte, []byte, []byte, []byte, []byte, error) {
	// gen server ca + certs
	serverRootCert, serverRootPem, serverRootKey, err := genCARoot()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("unable to generate ca cert: %w", err)
	}
	serverPem, serverKey, err := genPeerCert(serverRootCert, serverRootKey, "vault", []string{
		"localhost",
		serviceName,
		fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace)})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, errors.New("unable to generate vault server cert")
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
		return nil, nil, nil, nil, nil, nil, errors.New("unable to generate vault server cert")
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
