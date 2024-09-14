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
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/yaml"
)

// HelmChart installs the specified Chart into the cluster.
type ArgoCDApplication struct {
	Name                 string
	Namespace            string
	DestinationNamespace string
	HelmChart            string
	HelmRepo             string
	HelmRevision         string
	HelmParameters       []string

	config *Config
	dc     *dynamic.DynamicClient
}

var argoApp = schema.GroupVersionResource{
	Group:    "argoproj.io",
	Version:  "v1alpha1",
	Resource: "applications",
}

var argoAppResource = `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: %s
  annotations:
    argocd.argoproj.io/refresh: "hard"
spec:
  project: default
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
  source:
    chart: %s
    repoURL: %s
    targetRevision: %s
    helm:
      releaseName: %s
      parameters: %s
  destination:
    server: https://kubernetes.default.svc
    namespace: %s
`

const (
	// taken from: https://github.com/argoproj/argo-cd/blob/0a8a71e12c5010d5ada1fce37feb0d8add1c61d0/pkg/apis/application/v1alpha1/types.go#LL1351C41-L1351C47
	StatusSynced = "Synced"
)

// Setup stores the config in an internal field
// to get access to the k8s api in orderto fetch logs.
func (c *ArgoCDApplication) Setup(cfg *Config) error {
	c.config = cfg
	dc, err := dynamic.NewForConfig(cfg.KubeConfig)
	if err != nil {
		return err
	}

	c.dc = dc
	return nil
}

// Install adds the chart repo and installs the helm chart.
func (c *ArgoCDApplication) Install() error {
	// construct helm parameters
	var helmParams string
	for _, param := range c.HelmParameters {
		args := strings.Split(param, "=")
		helmParams += fmt.Sprintf(`
      - name: "%s"
        value: %s`, args[0], args[1])
	}
	jsonBytes, err := yaml.YAMLToJSON([]byte(fmt.Sprintf(argoAppResource, c.Name, c.Namespace, c.HelmChart, c.HelmRepo, c.HelmRevision, c.Name, helmParams, c.DestinationNamespace)))
	if err != nil {
		return fmt.Errorf("unable to decode argo app yaml to json: %w", err)
	}
	us := &unstructured.Unstructured{}
	err = us.UnmarshalJSON(jsonBytes)
	if err != nil {
		return fmt.Errorf("unable to unmarshal json into unstructured: %w", err)
	}
	_, err = c.dc.Resource(argoApp).Namespace(c.Namespace).Create(context.Background(), us, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create argo app: %w", err)
	}

	// wait for app to become ready
	err = wait.PollImmediate(time.Second*5, time.Minute*10, func() (bool, error) {
		us, err = c.dc.Resource(argoApp).Namespace(c.Namespace).Get(context.Background(), c.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		syncStatus, _, err := unstructured.NestedString(us.Object, "status", "sync", "status")
		if err != nil {
			return false, nil
		}
		return syncStatus == StatusSynced, nil
	})
	if err != nil {
		return fmt.Errorf("failed waiting for argo app to become ready: %w", err)
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
	err := c.dc.Resource(argoApp).Namespace(c.Namespace).Delete(context.Background(), c.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	err = uninstallCRDs(c.config)
	if err != nil {
		return err
	}
	return nil
}

func (c *ArgoCDApplication) Logs() error {
	return nil
}
