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
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"time"

	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/meta"
	fluxsrc "github.com/fluxcd/source-controller/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	err := c.config.CRClient.Create(GinkgoT().Context(), app)
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
			Chart: &fluxhelm.HelmChartTemplate{
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
	err = c.config.CRClient.Create(GinkgoT().Context(), hr)
	if err != nil {
		return err
	}

	// wait for app to become ready
	err = wait.PollUntilContextTimeout(GinkgoT().Context(), time.Second*5, time.Minute*3, true, func(ctx context.Context) (bool, error) {
		var hr fluxhelm.HelmRelease
		err := c.config.CRClient.Get(GinkgoT().Context(), types.NamespacedName{
			Name:      c.Name,
			Namespace: c.Namespace,
		}, &hr)
		if err != nil {
			return false, nil
		}
		for _, cond := range hr.GetConditions() {
			GinkgoWriter.Printf("check condition: %s=%s: %s\n", cond.Type, cond.Status, cond.Message)
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
	return wait.PollUntilContextTimeout(GinkgoT().Context(), time.Second, time.Minute*5, true, func(ctx context.Context) (bool, error) {
		const payload = `{"apiVersion": "admission.k8s.io/v1","kind": "AdmissionReview","request": {"uid": "test","kind": {"group": "external-secrets.io","version": "v1","kind": "ExternalSecret"}, "resource": "external-secrets.io/v1.externalsecrets","dryRun": true, "operation": "CREATE", "userInfo":{"username":"test","uid":"test","groups":[],"extra":{}}}}`
		res, err := client.Post("https://external-secrets-webhook.external-secrets.svc.cluster.local/validate-external-secrets-io-v1-externalsecret", "application/json", bytes.NewBufferString(payload))
		if err != nil {
			return false, nil
		}
		defer func() {
			_ = res.Body.Close()
		}()
		GinkgoWriter.Printf("webhook res: %d", res.StatusCode)
		return res.StatusCode == http.StatusOK, nil
	})
}

// Uninstall removes the chart aswell as the repo.
func (c *FluxHelmRelease) Uninstall() error {
	err := uninstallCRDs(c.config)
	if err != nil {
		return err
	}
	err = c.config.CRClient.Delete(GinkgoT().Context(), &fluxhelm.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
		},
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	Eventually(func() bool {
		var hr fluxhelm.HelmRelease
		err = c.config.CRClient.Get(GinkgoT().Context(), types.NamespacedName{
			Name:      c.Name,
			Namespace: c.Namespace,
		}, &hr)
		if apierrors.IsNotFound(err) {
			return true
		}
		return false
	}).WithPolling(time.Second).WithTimeout(time.Second * 30).Should(BeTrue())

	if err := c.config.CRClient.Delete(GinkgoT().Context(), &fluxsrc.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: fluxNamespace,
		},
	}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (c *FluxHelmRelease) Logs() error {
	return nil
}
