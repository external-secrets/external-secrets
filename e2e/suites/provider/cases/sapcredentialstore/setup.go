//go:build e2e_sapcredentialstore

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

package sapcredentialstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

	//nolint
	. "github.com/onsi/ginkgo/v2"
	//nolint
	. "github.com/onsi/gomega"
)

// SAPCSTestConfig holds connection parameters read from environment variables.
type SAPCSTestConfig struct {
	ServiceURL   string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Namespace    string
}

// NewConfigFromEnv reads test config from environment variables.
// Skips the suite via Ginkgo Skip if any required variable is absent.
func NewConfigFromEnv() SAPCSTestConfig {
	cfg := SAPCSTestConfig{
		ServiceURL:   os.Getenv("SAPCS_SERVICE_URL"),
		TokenURL:     os.Getenv("SAPCS_TOKEN_URL"),
		ClientID:     os.Getenv("SAPCS_CLIENT_ID"),
		ClientSecret: os.Getenv("SAPCS_CLIENT_SECRET"),
		Namespace:    os.Getenv("SAPCS_NAMESPACE"),
	}
	missing := []string{}
	if cfg.ServiceURL == "" {
		missing = append(missing, "SAPCS_SERVICE_URL")
	}
	if cfg.TokenURL == "" {
		missing = append(missing, "SAPCS_TOKEN_URL")
	}
	if cfg.ClientID == "" {
		missing = append(missing, "SAPCS_CLIENT_ID")
	}
	if cfg.ClientSecret == "" {
		missing = append(missing, "SAPCS_CLIENT_SECRET")
	}
	if cfg.Namespace == "" {
		missing = append(missing, "SAPCS_NAMESPACE")
	}
	if len(missing) > 0 {
		Skip(fmt.Sprintf("skipping SAP CS e2e suite: missing env vars: %v", missing))
	}
	return cfg
}

// CreateBindingSecret creates a Kubernetes Secret containing BTP-style binding JSON.
func CreateBindingSecret(ctx context.Context, kube client.Client, ns, name string, cfg SAPCSTestConfig) {
	creds, err := json.Marshal(map[string]string{
		"clientid":     cfg.ClientID,
		"clientsecret": cfg.ClientSecret,
		"url":          cfg.ServiceURL,
		"tokenurl":     cfg.TokenURL,
	})
	Expect(err).ToNot(HaveOccurred())

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Data:       map[string][]byte{"credentials": creds},
	}
	Expect(kube.Create(ctx, secret)).To(Succeed())
}

// CreateClusterSecretStore creates a ClusterSecretStore with inline OAuth2 credentials.
func CreateClusterSecretStore(ctx context.Context, kube client.Client, name, credSecretNS, credSecretName, sapcsNS string) {
	store := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				SAPCredentialStore: &esv1.SAPCredentialStoreProvider{
					Namespace: sapcsNS,
					ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{
						Name:      credSecretName,
						Namespace: credSecretNS,
					},
				},
			},
		},
	}
	Expect(kube.Create(ctx, store)).To(Succeed())
}

// CreateExternalSecret creates an ExternalSecret referencing a ClusterSecretStore.
func CreateExternalSecret(ctx context.Context, kube client.Client, ns, esName, storeName, targetName, credKey, sapcsNSOverride string) {
	ref := esv1.ExternalSecretDataRemoteRef{
		Key:      credKey,
		Property: "password",
	}
	if sapcsNSOverride != "" {
		ref.Namespace = sapcsNSOverride
	}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{Name: esName, Namespace: ns},
		Spec: esv1.ExternalSecretSpec{
			RefreshInterval: &metav1.Duration{Duration: 0},
			SecretStoreRef: esv1.SecretStoreRef{
				Name: storeName,
				Kind: esv1.ClusterSecretStoreKind,
			},
			Target: esv1.ExternalSecretTarget{Name: targetName},
			Data: []esv1.ExternalSecretData{
				{SecretKey: "value", RemoteRef: ref},
			},
		},
	}
	Expect(kube.Create(ctx, es)).To(Succeed())
}

// waitForSecretSynced polls until the ExternalSecret reaches SecretSynced condition.
func waitForSecretSynced(ctx context.Context, kube client.Client, ns, name string) {
	Eventually(func() bool {
		var es esv1.ExternalSecret
		if err := kube.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, &es); err != nil {
			return false
		}
		for _, c := range es.Status.Conditions {
			if c.Type == "Ready" && c.Status == "True" {
				return true
			}
		}
		return false
	}, wait.ForeverTestTimeout, "5s").Should(BeTrue(), "ExternalSecret %s/%s did not sync", ns, name)
}
