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

package addon

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	infisicalSdk "github.com/infisical/go-sdk"
	. "github.com/onsi/ginkgo/v2"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	infisicalNamespace   = "infisical"
	infisicalReleaseName = "infisical"
	// infisicalServiceName must match infisical.fullnameOverride in
	// infisical.values.yaml so we can port-forward and target it by DNS.
	infisicalServiceName = "infisical-backend"
	infisicalAPIPort     = 8080
	infisicalChartVer    = "1.8.0"
	infisicalSecretName  = "infisical-secrets"

	// Bootstrap identity used to provision the instance. The password meets
	// Infisical's complexity policy (length + mixed character classes).
	infisicalAdminEmail    = "admin@example.com"
	infisicalAdminPassword = "E2eAdminPassw0rd!23"
	infisicalOrgName       = "eso-e2e"
	infisicalProjectName   = "eso-e2e"
	infisicalProjectSlug   = "eso-e2e-tests"

	// InfisicalKubernetesAuthServiceAccount is the only ServiceAccount name the
	// Kubernetes auth identity accepts, so a login with the controller's own
	// token fails.
	InfisicalKubernetesAuthServiceAccount = "infisical-e2e"
	// InfisicalKubernetesAuthAudience is the only token audience the
	// Kubernetes auth identity accepts.
	InfisicalKubernetesAuthAudience = "infisical-e2e"
	// infisicalTokenReviewer names the ServiceAccount, and its
	// ClusterRoleBinding, that Infisical uses to review login tokens.
	infisicalTokenReviewer = "infisical-token-reviewer"
	// The FQDN, because Infisical resolves the host without search domains.
	infisicalKubernetesHost = "https://kubernetes.default.svc.cluster.local"
)

// Infisical deploys a self-hosted Infisical into the cluster and provisions a
// project plus a Universal Auth and a Kubernetes auth machine identity for the
// e2e suite to use.
type Infisical struct {
	chart         *HelmChart
	Namespace     string
	portForwarder *PortForward
	httpClient    *http.Client
	cancelRefresh context.CancelFunc

	// HostAPI is the in-cluster API URL ESO uses from the SecretStore.
	HostAPI string
	// localBaseURL is the port-forwarded API URL the test process uses.
	localBaseURL string

	ClientID        string
	ClientSecret    string
	ProjectID       string
	ProjectSlug     string
	EnvironmentSlug string

	// KubernetesIdentityID is a second machine identity, which logs in with
	// Kubernetes auth as InfisicalKubernetesAuthServiceAccount.
	KubernetesIdentityID string

	// SDKClient is logged in via Universal Auth and used by the suite to seed
	// and remove backend secrets.
	SDKClient infisicalSdk.InfisicalClientInterface
}

func NewInfisical() *Infisical {
	repo := "infisical-helm-charts"
	return &Infisical{
		Namespace:  infisicalNamespace,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		chart: &HelmChart{
			Namespace:    infisicalNamespace,
			ReleaseName:  infisicalReleaseName,
			Chart:        fmt.Sprintf("%s/infisical-standalone", repo),
			ChartVersion: infisicalChartVer,
			Repo: ChartRepo{
				Name: repo,
				URL:  "https://dl.cloudsmith.io/public/infisical/helm-charts/helm/charts/",
			},
			Values: []string{filepath.Join(AssetDir(), "infisical.values.yaml")},
		},
	}
}

func (l *Infisical) Setup(cfg *Config) error {
	return l.chart.Setup(cfg)
}

func (l *Infisical) Install() (err error) {
	l.HostAPI = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", infisicalServiceName, l.Namespace, infisicalAPIPort)

	// Arm rollback before the first mutating step so any failure (secret,
	// namespace, chart install, or bootstrap) tears down what was created and a
	// re-run does not trip over a half-installed instance.
	defer func() {
		if err != nil {
			l.cleanup()
		}
	}()

	if err := l.createInstanceSecret(); err != nil {
		return fmt.Errorf("unable to create infisical secret: %w", err)
	}

	if err := l.chart.Install(); err != nil {
		return err
	}

	pf, err := NewPortForward(l.chart.config.KubeClientSet, l.chart.config.KubeConfig, infisicalServiceName, l.Namespace, infisicalAPIPort)
	if err != nil {
		return err
	}
	if err := pf.Start(); err != nil {
		return err
	}
	l.portForwarder = pf
	l.localBaseURL = fmt.Sprintf("http://localhost:%d", pf.localPort)

	if err := l.waitForAPI(); err != nil {
		return fmt.Errorf("infisical API never became ready: %w", err)
	}

	if err := l.bootstrap(); err != nil {
		return fmt.Errorf("unable to bootstrap infisical: %w", err)
	}

	// Own the SDK client context so its token-refresh goroutine lives for the
	// whole suite (not just this BeforeAll node) and is canceled on teardown.
	ctx, cancel := context.WithCancel(context.Background())
	l.cancelRefresh = cancel
	client := infisicalSdk.NewInfisicalClient(ctx, infisicalSdk.Config{SiteUrl: l.localBaseURL})
	if _, err := client.Auth().UniversalAuthLogin(l.ClientID, l.ClientSecret); err != nil {
		return fmt.Errorf("unable to log in with universal auth: %w", err)
	}
	l.SDKClient = client

	return nil
}

// cleanup best-effort removes everything Install created. It runs when a step
// after the chart install fails, so repeated local runs do not collide with a
// leftover release or namespace.
func (l *Infisical) cleanup() {
	if l.cancelRefresh != nil {
		l.cancelRefresh()
		l.cancelRefresh = nil
	}
	if l.portForwarder != nil {
		l.portForwarder.Close()
		l.portForwarder = nil
	}
	_ = l.chart.Uninstall()
	_ = l.deleteTokenReviewerBinding()
	_ = l.chart.config.KubeClientSet.CoreV1().Namespaces().Delete(GinkgoT().Context(), l.Namespace, metav1.DeleteOptions{})
}

// createInstanceSecret creates the Secret the chart mounts as envFrom. Infisical
// requires a 16-byte hex ENCRYPTION_KEY and a base64 AUTH_SECRET.
func (l *Infisical) createInstanceSecret() error {
	encKey := make([]byte, 16)
	if _, err := rand.Read(encKey); err != nil {
		return err
	}
	authSecret := make([]byte, 32)
	if _, err := rand.Read(authSecret); err != nil {
		return err
	}

	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: l.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(GinkgoT().Context(), l.chart.config.CRClient, ns, func() error { return nil }); err != nil {
		return err
	}

	sec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      infisicalSecretName,
			Namespace: l.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(GinkgoT().Context(), l.chart.config.CRClient, sec, func() error {
		sec.StringData = map[string]string{
			"ENCRYPTION_KEY": hex.EncodeToString(encKey),
			"AUTH_SECRET":    base64.StdEncoding.EncodeToString(authSecret),
			"SITE_URL":       l.HostAPI,
			// Kubernetes auth reviews tokens at the in-cluster API server,
			// which has a private IP that Infisical refuses by default.
			"ALLOW_INTERNAL_IP_CONNECTIONS": "true",
		}
		return nil
	})
	return err
}

func (l *Infisical) waitForAPI() error {
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		resp, err := l.httpClient.Get(l.localBaseURL + "/api/status")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("timed out waiting for %s/api/status", l.localBaseURL)
}

// bootstrap initializes a fresh instance and provisions everything the suite
// needs: an admin identity, a project, a Universal Auth machine identity, and
// a Kubernetes auth machine identity.
func (l *Infisical) bootstrap() error {
	var boot struct {
		Identity struct {
			Credentials struct {
				Token string `json:"token"`
			} `json:"credentials"`
		} `json:"identity"`
		Organization struct {
			ID string `json:"id"`
		} `json:"organization"`
	}
	if err := l.apiCall(http.MethodPost, "/api/v1/admin/bootstrap", "", map[string]any{
		"email":        infisicalAdminEmail,
		"password":     infisicalAdminPassword,
		"organization": infisicalOrgName,
	}, &boot); err != nil {
		return fmt.Errorf("admin bootstrap: %w", err)
	}
	token := boot.Identity.Credentials.Token

	var project struct {
		Project struct {
			ID           string `json:"id"`
			Slug         string `json:"slug"`
			Environments []struct {
				Slug string `json:"slug"`
			} `json:"environments"`
		} `json:"project"`
	}
	if err := l.apiCall(http.MethodPost, "/api/v1/projects", token, map[string]any{
		"projectName":             infisicalProjectName,
		"slug":                    infisicalProjectSlug,
		"shouldCreateDefaultEnvs": true,
	}, &project); err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	l.ProjectID = project.Project.ID
	l.ProjectSlug = project.Project.Slug
	l.EnvironmentSlug = pickEnvironment(project.Project.Environments)

	identityID, err := l.createProjectIdentity(token, boot.Organization.ID, "eso-e2e")
	if err != nil {
		return err
	}

	trustedIPs := []map[string]string{{"ipAddress": "0.0.0.0/0"}, {"ipAddress": "::/0"}}
	var ua struct {
		IdentityUniversalAuth struct {
			ClientID string `json:"clientId"`
		} `json:"identityUniversalAuth"`
	}
	if err := l.apiCall(http.MethodPost, "/api/v1/auth/universal-auth/identities/"+identityID, token, map[string]any{
		"accessTokenTTL":          2592000,
		"accessTokenMaxTTL":       2592000,
		"accessTokenNumUsesLimit": 0,
		"clientSecretTrustedIps":  trustedIPs,
		"accessTokenTrustedIps":   trustedIPs,
	}, &ua); err != nil {
		return fmt.Errorf("attach universal auth: %w", err)
	}
	l.ClientID = ua.IdentityUniversalAuth.ClientID

	var secret struct {
		ClientSecret string `json:"clientSecret"`
	}
	if err := l.apiCall(http.MethodPost, "/api/v1/auth/universal-auth/identities/"+identityID+"/client-secrets", token, map[string]any{
		"description":  "eso-e2e",
		"numUsesLimit": 0,
		"ttl":          0,
	}, &secret); err != nil {
		return fmt.Errorf("create client secret: %w", err)
	}
	l.ClientSecret = secret.ClientSecret

	return l.bootstrapKubernetesAuth(token, boot.Organization.ID, trustedIPs)
}

// createProjectIdentity creates a machine identity, then grants it admin
// access to the project. The org role alone does not confer project access,
// so without the membership the identity gets a 403 on secret operations.
func (l *Infisical) createProjectIdentity(token, orgID, name string) (string, error) {
	var identity struct {
		Identity struct {
			ID string `json:"id"`
		} `json:"identity"`
	}
	if err := l.apiCall(http.MethodPost, "/api/v1/identities", token, map[string]any{
		"name":           name,
		"organizationId": orgID,
		"role":           "member",
	}, &identity); err != nil {
		return "", fmt.Errorf("create identity %s: %w", name, err)
	}
	identityID := identity.Identity.ID

	if err := l.apiCall(http.MethodPost, "/api/v2/workspace/"+l.ProjectID+"/identity-memberships/"+identityID, token, map[string]any{
		"role": "admin",
	}, nil); err != nil {
		return "", fmt.Errorf("add identity %s to project: %w", name, err)
	}
	return identityID, nil
}

// bootstrapKubernetesAuth provisions the identity that serviceAccountRef
// stores log in as. Infisical checks each login token with a TokenReview at
// the cluster API server, signed with the token reviewer's own token.
func (l *Infisical) bootstrapKubernetesAuth(token, orgID string, trustedIPs []map[string]string) error {
	identityID, err := l.createProjectIdentity(token, orgID, "eso-e2e-kubernetes")
	if err != nil {
		return err
	}

	reviewerJWT, err := l.createTokenReviewer()
	if err != nil {
		return fmt.Errorf("create token reviewer: %w", err)
	}
	rootCA, err := l.chart.config.KubeClientSet.CoreV1().ConfigMaps(l.Namespace).Get(GinkgoT().Context(), "kube-root-ca.crt", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("read cluster CA: %w", err)
	}

	if err := l.apiCall(http.MethodPost, "/api/v1/auth/kubernetes-auth/identities/"+identityID, token, map[string]any{
		"kubernetesHost":          infisicalKubernetesHost,
		"caCert":                  rootCA.Data["ca.crt"],
		"tokenReviewerJwt":        reviewerJWT,
		"tokenReviewMode":         "api",
		"allowedNamespaces":       "",
		"allowedNames":            InfisicalKubernetesAuthServiceAccount,
		"allowedAudience":         InfisicalKubernetesAuthAudience,
		"accessTokenTTL":          2592000,
		"accessTokenMaxTTL":       2592000,
		"accessTokenNumUsesLimit": 0,
		"accessTokenTrustedIps":   trustedIPs,
	}, nil); err != nil {
		return fmt.Errorf("attach kubernetes auth: %w", err)
	}
	l.KubernetesIdentityID = identityID
	return nil
}

// createTokenReviewer creates a ServiceAccount that may review tokens and
// returns a token for it that outlives the suite.
func (l *Infisical) createTokenReviewer() (string, error) {
	ctx := GinkgoT().Context()
	sa := &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: infisicalTokenReviewer, Namespace: l.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, l.chart.config.CRClient, sa, func() error { return nil }); err != nil {
		return "", err
	}

	crb := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: infisicalTokenReviewer}}
	if _, err := controllerutil.CreateOrUpdate(ctx, l.chart.config.CRClient, crb, func() error {
		crb.Subjects = []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      infisicalTokenReviewer,
				Namespace: l.Namespace,
			},
		}
		crb.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "system:auth-delegator",
		}
		return nil
	}); err != nil {
		return "", err
	}

	tr, err := l.chart.config.KubeClientSet.CoreV1().ServiceAccounts(l.Namespace).CreateToken(ctx, infisicalTokenReviewer, &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{ExpirationSeconds: new(int64(24 * 60 * 60))},
	}, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	return tr.Status.Token, nil
}

// deleteTokenReviewerBinding removes the cluster-scoped binding, which the
// namespace delete does not.
func (l *Infisical) deleteTokenReviewerBinding() error {
	crb := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: infisicalTokenReviewer}}
	return crclient.IgnoreNotFound(l.chart.config.CRClient.Delete(GinkgoT().Context(), crb))
}

// pickEnvironment returns the "dev" environment slug when present, otherwise the
// first environment the project was created with.
func pickEnvironment(envs []struct {
	Slug string `json:"slug"`
}) string {
	for _, e := range envs {
		if e.Slug == "dev" {
			return e.Slug
		}
	}
	if len(envs) > 0 {
		return envs[0].Slug
	}
	return "dev"
}

// apiCall performs a JSON request against the port-forwarded API, optionally
// bearer-authenticated, and decodes the response into out.
func (l *Infisical) apiCall(method, path, token string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(GinkgoT().Context(), method, l.localBaseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s returned %d: %s", method, path, resp.StatusCode, string(data))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

func (l *Infisical) Logs() error {
	return l.chart.Logs()
}

func (l *Infisical) Uninstall() error {
	if l.cancelRefresh != nil {
		l.cancelRefresh()
		l.cancelRefresh = nil
	}
	if l.portForwarder != nil {
		l.portForwarder.Close()
		l.portForwarder = nil
	}
	if err := l.chart.Uninstall(); err != nil {
		return err
	}
	if err := l.deleteTokenReviewerBinding(); err != nil {
		return err
	}
	return l.chart.config.KubeClientSet.CoreV1().Namespaces().Delete(GinkgoT().Context(), l.Namespace, metav1.DeleteOptions{})
}
