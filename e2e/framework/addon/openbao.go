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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework/util"
)

type OpenBao struct {
	chart     *HelmChart
	Namespace string

	InClusterURL string
	LocalURL     string
	RootToken    string

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
	if err := l.chart.Install(); err != nil {
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

	l.InClusterURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:8200", l.chart.ReleaseName, l.Namespace)
	l.LocalURL = fmt.Sprintf("http://localhost:%d", l.portForwarder.localPort)

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
