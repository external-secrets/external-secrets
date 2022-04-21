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

	argoapp "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

// HelmChart installs the specified Chart into the cluster.
type ArgoCDApplication struct {
	Name                 string
	Namespace            string
	DestinationNamespace string
	HelmChart            string
	HelmRepo             string
	HelmRevision         string
	HelmValues           string

	config *Config
}

// Setup stores the config in an internal field
// to get access to the k8s api in orderto fetch logs.
func (c *ArgoCDApplication) Setup(cfg *Config) error {
	c.config = cfg
	return nil
}

// Install adds the chart repo and installs the helm chart.
func (c *ArgoCDApplication) Install() error {
	app := &argoapp.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
			Annotations: map[string]string{
				"argocd.argoproj.io/refresh": "hard",
			},
		},
		Spec: argoapp.ApplicationSpec{
			Project: "default",
			SyncPolicy: &argoapp.SyncPolicy{
				Automated: &argoapp.SyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
				SyncOptions: argoapp.SyncOptions{
					"CreateNamespace=true",
				},
			},
			Source: argoapp.ApplicationSource{
				Chart:          c.HelmChart,
				RepoURL:        c.HelmRepo,
				TargetRevision: c.HelmRevision,
				Helm: &argoapp.ApplicationSourceHelm{
					ReleaseName: c.Name,
					Values:      c.HelmValues,
				},
			},
			Destination: argoapp.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: c.DestinationNamespace,
			},
		},
	}
	err := c.config.CRClient.Create(context.Background(), app)
	if err != nil {
		return err
	}

	// wait for app to become ready
	err = wait.PollImmediate(time.Second*5, time.Minute*10, func() (bool, error) {
		var app argoapp.Application
		err := c.config.CRClient.Get(context.Background(), types.NamespacedName{
			Name:      "external-secrets",
			Namespace: "argocd",
		}, &app)
		if err != nil {
			return false, nil
		}
		return app.Status.Sync.Status == argoapp.SyncStatusCodeSynced, nil
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
func (c *ArgoCDApplication) Uninstall() error {
	return c.config.CRClient.Delete(context.Background(), &argoapp.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
		},
	})
}

func (c *ArgoCDApplication) Logs() error {
	return nil
}
