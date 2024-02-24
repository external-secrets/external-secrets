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
	"crypto/rand"
	"encoding/base64"
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
			Namespace:    namespace,
			ReleaseName:  fmt.Sprintf("conjur-%s", namespace), // avoid cluster role collision
			Chart:        fmt.Sprintf("%s/conjur-oss", repo),
			ChartVersion: "2.0.7",
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
	// TODO: This will be used for the JWT tests
	return nil
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
