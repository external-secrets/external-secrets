/*
Copyright 2020 The cert-manager Authors.
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
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"time"

	fluxhelm "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/fluxcd/pkg/apis/meta"
	fluxsrc "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const fluxNamespace = "flux-system"

// HelmChart installs the specified Chart into the cluster.
type FluxHelmRelease struct {
	Name            string
	Namespace       string
	TargetNamespace string
	HelmChart       string
	HelmRepo        string
	HelmRevision    string
	HelmValues      string

	config *Config
}

// Setup stores the config in an internal field
// to get access to the k8s api in orderto fetch logs.
func (c *FluxHelmRelease) Setup(cfg *Config) error {
	c.config = cfg
	return nil
}

// Install adds the chart repo and installs the helm chart.
func (c *FluxHelmRelease) Install() error {
	app := &fluxsrc.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: fluxNamespace,
		},
		Spec: fluxsrc.HelmRepositorySpec{
			URL: c.HelmRepo,
		},
	}
	err := c.config.CRClient.Create(context.Background(), app)
	if err != nil {
		return err
	}

	hr := &fluxhelm.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
		},
		Spec: fluxhelm.HelmReleaseSpec{
			ReleaseName:     c.Name,
			TargetNamespace: c.TargetNamespace,
			Values: &v1.JSON{
				Raw: []byte(c.HelmValues),
			},
			Install: &fluxhelm.Install{
				CreateNamespace: true,
				Remediation: &fluxhelm.InstallRemediation{
					Retries: -1,
				},
			},
			Chart: fluxhelm.HelmChartTemplate{
				Spec: fluxhelm.HelmChartTemplateSpec{
					Version: c.HelmRevision,
					Chart:   c.HelmChart,
					SourceRef: fluxhelm.CrossNamespaceObjectReference{
						Kind:      "HelmRepository",
						Name:      c.Name,
						Namespace: fluxNamespace,
					},
				},
			},
		},
	}
	err = c.config.CRClient.Create(context.Background(), hr)
	if err != nil {
		return err
	}

	// wait for app to become ready
	err = wait.PollImmediate(time.Second*5, time.Minute*3, func() (bool, error) {
		var hr fluxhelm.HelmRelease
		err := c.config.CRClient.Get(context.Background(), types.NamespacedName{
			Name:      c.Name,
			Namespace: c.Namespace,
		}, &hr)
		if err != nil {
			return false, nil
		}
		for _, cond := range hr.GetConditions() {
			ginkgo.GinkgoWriter.Printf("check condition: %s=%s: %s\n", cond.Type, cond.Status, cond.Message)
			if cond.Type == meta.ReadyCondition && cond.Status == metav1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	// we have to wait for the webhook to become ready
	tr := &http.Transport{
		// nolint:gosec
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	return wait.PollImmediate(time.Second, time.Minute*5, func() (bool, error) {
		const payload = `{"apiVersion": "apiextensions.k8s.io/v1","kind": "ConversionReview","request": {}}`
		res, err := client.Post("https://external-secrets-webhook.external-secrets.svc.cluster.local/convert", "application/json", bytes.NewBufferString(payload))
		if err != nil {
			return false, nil
		}
		defer res.Body.Close()
		ginkgo.GinkgoWriter.Printf("conversion res: %d", res.StatusCode)
		return res.StatusCode == http.StatusOK, nil
	})
}

// Uninstall removes the chart aswell as the repo.
func (c *FluxHelmRelease) Uninstall() error {
	err := c.config.CRClient.Delete(context.Background(), &fluxhelm.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
		},
	})
	if err != nil {
		return err
	}
	return c.config.CRClient.Delete(context.Background(), &fluxsrc.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: fluxNamespace,
		},
	})
}

func (c *FluxHelmRelease) Logs() error {
	return nil
}
