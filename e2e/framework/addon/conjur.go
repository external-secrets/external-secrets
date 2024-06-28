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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// nolint
	ginkgo "github.com/onsi/ginkgo/v2"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"
	"github.com/external-secrets/external-secrets-e2e/framework/util"
)

type Conjur struct {
	chart        *HelmChart
	dataKey      string
	Namespace    string
	PodName      string
	ConjurClient *conjurapi.Client
	ConjurURL    string

	AdminApiKey    string
	ConjurServerCA []byte
}

func NewConjur(namespace string) *Conjur {
	repo := "conjur-" + namespace
	dataKey := generateConjurDataKey()

	return &Conjur{
		dataKey: dataKey,
		chart: &HelmChart{
			Namespace:   namespace,
			ReleaseName: fmt.Sprintf("conjur-%s", namespace), // avoid cluster role collision
			Chart:       fmt.Sprintf("%s/conjur-oss", repo),
			// Use latest version of Conjur OSS. To pin to a specific version, uncomment the following line.
			// ChartVersion: "2.0.7",
			Repo: ChartRepo{
				Name: repo,
				URL:  "https://cyberark.github.io/helm-charts",
			},
			Values: []string{"/k8s/conjur.values.yaml"},
			Vars: []StringTuple{
				{
					Key:   "dataKey",
					Value: dataKey,
				},
			},
		},
		Namespace: namespace,
	}
}

func (l *Conjur) Install() error {
	ginkgo.By("Installing conjur in " + l.Namespace)
	err := l.chart.Install()
	if err != nil {
		return err
	}

	err = l.initConjur()
	if err != nil {
		return err
	}

	err = l.configureConjur()
	if err != nil {
		return err
	}

	return nil
}

func (l *Conjur) initConjur() error {
	ginkgo.By("Waiting for conjur pods to be running")
	pl, err := util.WaitForPodsRunning(l.chart.config.KubeClientSet, 1, l.Namespace, metav1.ListOptions{
		LabelSelector: "app=conjur-oss",
	})
	if err != nil {
		return fmt.Errorf("error waiting for conjur to be running: %w", err)
	}
	l.PodName = pl.Items[0].Name

	ginkgo.By("Initializing conjur")
	// Get the auto generated certificates from the K8s secrets
	caCertSecret, err := util.GetKubeSecret(l.chart.config.KubeClientSet, l.Namespace, fmt.Sprintf("%s-conjur-ssl-ca-cert", l.chart.ReleaseName))
	if err != nil {
		return fmt.Errorf("error getting conjur ca cert: %w", err)
	}
	l.ConjurServerCA = caCertSecret.Data["tls.crt"]

	// Create "default" account
	_, err = util.ExecCmdWithContainer(
		l.chart.config.KubeClientSet,
		l.chart.config.KubeConfig,
		l.PodName, "conjur-oss", l.Namespace, "conjurctl account create default")
	if err != nil {
		return fmt.Errorf("error initializing conjur: %w", err)
	}

	// Retrieve the admin API key
	apiKey, err := util.ExecCmdWithContainer(
		l.chart.config.KubeClientSet,
		l.chart.config.KubeConfig,
		l.PodName, "conjur-oss", l.Namespace, "conjurctl role retrieve-key default:user:admin")
	if err != nil {
		return fmt.Errorf("error fetching admin API key: %w", err)
	}

	// TODO: ExecCmdWithContainer includes the StdErr output with a warning about config directory.
	// Therefore we need to split the output and only use the first line.
	l.AdminApiKey = strings.Split(apiKey, "\n")[0]

	l.ConjurURL = fmt.Sprintf("https://conjur-%s-conjur-oss.%s.svc.cluster.local", l.Namespace, l.Namespace)
	cfg := conjurapi.Config{
		Account:      "default",
		ApplianceURL: l.ConjurURL,
		SSLCert:      string(l.ConjurServerCA),
	}

	l.ConjurClient, err = conjurapi.NewClientFromKey(cfg, authn.LoginPair{
		Login:  "admin",
		APIKey: l.AdminApiKey,
	})
	if err != nil {
		return fmt.Errorf("unable to create conjur client: %w", err)
	}

	return nil
}

func (l *Conjur) configureConjur() error {
	ginkgo.By("configuring conjur")
	// Construct Conjur policy for authn-jwt. This uses the token-app-property "sub" to
	// authenticate the host. This means that Conjur will determine which host is authenticating
	// based on the "sub" claim in the JWT token, which is provided by the Kubernetes service account.
	policy := `- !policy
  id: conjur/authn-jwt/eso-tests
  body:
    - !webservice
    - !variable public-keys
    - !variable issuer
    - !variable token-app-property
    - !variable audience`

	_, err := l.ConjurClient.LoadPolicy(conjurapi.PolicyModePost, "root", strings.NewReader(policy))
	if err != nil {
		return fmt.Errorf("unable to load authn-jwt policy: %w", err)
	}

	// Construct Conjur policy for authn-jwt-hostid. This does not use the token-app-property variable
	// and instead uses the HostID passed in the authentication URL to determine which host is authenticating.
	// This is not the recommended way to authenticate, but it is needed for certain use cases where the
	// JWT token does not contain the "sub" claim.
	policy = `- !policy
  id: conjur/authn-jwt/eso-tests-hostid
  body:
    - !webservice
    - !variable public-keys
    - !variable issuer
    - !variable audience`

	_, err = l.ConjurClient.LoadPolicy(conjurapi.PolicyModePost, "root", strings.NewReader(policy))
	if err != nil {
		return fmt.Errorf("unable to load authn-jwt policy: %w", err)
	}

	// Fetch the jwks info from the k8s cluster
	pubKeysJson, issuer, err := l.fetchJWKSandIssuer()
	if err != nil {
		return fmt.Errorf("unable to fetch jwks and issuer: %w", err)
	}

	// Set the variables for the authn-jwt policies
	secrets := map[string]string{
		"conjur/authn-jwt/eso-tests/audience":           l.ConjurURL,
		"conjur/authn-jwt/eso-tests/issuer":             issuer,
		"conjur/authn-jwt/eso-tests/public-keys":        string(pubKeysJson),
		"conjur/authn-jwt/eso-tests/token-app-property": "sub",
		"conjur/authn-jwt/eso-tests-hostid/audience":    l.ConjurURL,
		"conjur/authn-jwt/eso-tests-hostid/issuer":      issuer,
		"conjur/authn-jwt/eso-tests-hostid/public-keys": string(pubKeysJson),
	}

	for secretPath, secretValue := range secrets {
		err := l.ConjurClient.AddSecret(secretPath, secretValue)
		if err != nil {
			return fmt.Errorf("unable to add secret %s: %w", secretPath, err)
		}
	}

	return nil
}

func (l *Conjur) fetchJWKSandIssuer() (pubKeysJson string, issuer string, err error) {
	kc := l.chart.config.KubeClientSet

	// Fetch the openid-configuration
	res, err := kc.CoreV1().RESTClient().Get().AbsPath("/.well-known/openid-configuration").DoRaw(context.Background())
	if err != nil {
		return "", "", fmt.Errorf("unable to fetch openid-configuration: %w", err)
	}
	var openidConfig map[string]any
	json.Unmarshal(res, &openidConfig)
	issuer = openidConfig["issuer"].(string)

	// Fetch the jwks
	jwksJson, err := kc.CoreV1().RESTClient().Get().AbsPath("/openid/v1/jwks").DoRaw(context.Background())
	if err != nil {
		return "", "", fmt.Errorf("unable to fetch jwks: %w", err)
	}
	var jwks map[string]any
	json.Unmarshal(jwksJson, &jwks)

	// Create a JSON object with the jwks that can be used by Conjur
	pubKeysObj := map[string]any{
		"type":  "jwks",
		"value": jwks,
	}
	pubKeysJsonObj, err := json.Marshal(pubKeysObj)
	if err != nil {
		return "", "", fmt.Errorf("unable to marshal jwks: %w", err)
	}

	pubKeysJson = string(pubKeysJsonObj)
	return pubKeysJson, issuer, nil
}

func (l *Conjur) Logs() error {
	return l.chart.Logs()
}

func (l *Conjur) Uninstall() error {
	return l.chart.Uninstall()
}

func (l *Conjur) Setup(cfg *Config) error {
	return l.chart.Setup(cfg)
}

func generateConjurDataKey() string {
	// Generate a 32 byte cryptographically secure random string.
	// Normally this is done by running `conjurctl data-key generate`
	// but for test purposes we can generate it programmatically.
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Errorf("unable to generate random string: %w", err))
	}

	// Encode the bytes as a base64 string
	return base64.StdEncoding.EncodeToString(b)
}
