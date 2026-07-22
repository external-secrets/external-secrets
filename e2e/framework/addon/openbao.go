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
	"crypto/rand"
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/external-secrets/external-secrets-e2e/framework/util"
)

type OpenBao struct {
	chart     *HelmChart
	Namespace string

	URLs struct {
		InClusterPlainText string
		InClusterTLS       string
		Local              string
	}
	RootToken string
	ServerCA  []byte

	portForwarder *PortForward
}

func NewOpenBao() *OpenBao {
	rootToken := rand.Text()

	repo := "openbao"
	return &OpenBao{
		chart: &HelmChart{
			Namespace:    "openbao",
			ReleaseName:  "openbao",
			Chart:        fmt.Sprintf("%s/openbao", repo),
			ChartVersion: "0.28.3",
			Repo: ChartRepo{
				Name: repo,
				URL:  "https://openbao.github.io/openbao-helm",
			},
			Args: []string{
				"--create-namespace",
			},
			Values: []string{filepath.Join(AssetDir(), "openbao.values.yaml")},
			Vars: []StringTuple{{
				Key:   "server.dev.devRootToken",
				Value: rootToken,
			}},
		},
		Namespace: "openbao",
		RootToken: rootToken,
	}
}

func (l *OpenBao) Install() error {
	serverRootPem, serverPem, serverKeyPem, _, _, _, err := genVaultCertificates(l.Namespace, l.chart.ReleaseName)
	if err != nil {
		return err
	}
	l.ServerCA = serverRootPem

	if err := l.chart.Install(); err != nil {
		return err
	}

	sec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openbao-config",
			Namespace: l.Namespace,
		},
		Data: map[string][]byte{},
	}
	_, err = controllerutil.CreateOrUpdate(GinkgoT().Context(), l.chart.config.CRClient, sec, func() error {
		sec.Data = map[string][]byte{
			"server-cert.pem":     serverPem,
			"server-cert-key.pem": serverKeyPem,
			"config.hcl": []byte(`
				ui = true
				listener "tcp" {
					address = "[::]:8300"
					tls_cert_file = "/etc/bao/server-cert.pem"
					tls_key_file = "/etc/bao/server-cert-key.pem"
				}
			`),
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := l.initBao(); err != nil {
		return err
	}

	return nil
}

func (l *OpenBao) initBao() error {
	err := util.WaitForPodsReady(l.chart.config.KubeClientSet, 1, l.Namespace, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=openbao",
	})
	if err != nil {
		return fmt.Errorf("error waiting for OpenBao to be ready: %w", err)
	}

	// This e2e test provider uses a local port-forwarded to talk to the OpenBao API instead
	// of using the kubernetes service. This allows us to run the e2e test suite locally.
	l.portForwarder, err = NewPortForward(l.chart.config.KubeClientSet, l.chart.config.KubeConfig, "openbao", l.chart.Namespace, 8200)
	if err != nil {
		return err
	}
	if err := l.portForwarder.Start(); err != nil {
		return err
	}

	l.URLs.InClusterTLS = fmt.Sprintf("https://%s.%s.svc.cluster.local:8300", l.chart.ReleaseName, l.Namespace)
	l.URLs.InClusterPlainText = fmt.Sprintf("http://%s.%s.svc.cluster.local:8200", l.chart.ReleaseName, l.Namespace)
	l.URLs.Local = fmt.Sprintf("http://localhost:%d", l.portForwarder.localPort)

	return nil
}

func (l *OpenBao) Logs() error {
	return l.chart.Logs()
}

func (l *OpenBao) Uninstall() error {
	if l.portForwarder != nil {
		l.portForwarder.Close()
		l.portForwarder = nil
	}

	if err := l.chart.Uninstall(); err != nil {
		return err
	}

	return l.chart.config.KubeClientSet.CoreV1().Namespaces().Delete(GinkgoT().Context(), l.chart.Namespace, metav1.DeleteOptions{})
}

func (l *OpenBao) Setup(cfg *Config) error {
	return l.chart.Setup(cfg)
}
