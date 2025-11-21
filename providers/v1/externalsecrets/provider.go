// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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

// Package externalsecrets provides an ExternalSecrets provider implementation.
package externalsecrets

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"time"

	"k8s.io/client-go/kubernetes"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	utils "github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// Provider satisfies the provider interface.
type Provider struct{}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient instantiates a new ExternalSecrets client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	storeKind := store.GetObjectKind().GroupVersionKind().Kind
	spec := storeSpec.Provider.ExternalSecrets
	if spec.Auth.Kubernetes == nil {
		return nil, errors.New("only kubernetes auth supported")
	}
	var remoteCaRef *x509.CertPool
	var localCaRef []byte
	var err error
	if spec.Server.CaRef != nil {
		remoteCaRef, err = getCertificate(ctx, kube, spec.Server.CaRef, storeKind, namespace)
		if err != nil {
			return nil, err
		}
	}
	localCaRef, err = getCertificateBytes(ctx, kube, &spec.Auth.Kubernetes.CaCertRef, storeKind, namespace)
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	if remoteCaRef != nil {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    remoteCaRef,
				MinVersion: tls.VersionTLS12,
			},
		}
		httpClient.Transport = transport
	}
	// TODO: move this to auth/kubernetes.go so that we support multiple token fetch schemes
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	token, err := resolvers.GenerateToken(ctx, clientset.CoreV1(), storeKind, namespace, spec.Auth.Kubernetes.ServiceAccountRef, nil, 600)
	if err != nil {
		return nil, err
	}
	ans := &Client{
		httpClient:      httpClient,
		kclient:         kube,
		serverURL:       spec.Server.URL,
		localCaRef:      localCaRef,
		token:           token,
		secretStoreName: *spec.Target.ClusterSecretStoreName,
	}

	return ans, nil
}
func getCertificateBytes(ctx context.Context, kube kclient.Client, ref *esv1.ExternalSecretsCARef, storeKind, namespace string) ([]byte, error) {
	if ref.Bundle != nil {
		return ref.Bundle, nil
	}
	if ref.SecretRef != nil {
		value, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, ref.SecretRef)
		if err != nil {
			return nil, fmt.Errorf("could not get secret for certs:%w", err)
		}
		return []byte(value), nil
	}
	if ref.ConfigMapRef != nil {
		value, err := resolvers.ConfigMapKeyRef(ctx, kube, namespace, ref.ConfigMapRef)
		if err != nil {
			return nil, fmt.Errorf("could not get secret for certs:%w", err)
		}
		return []byte(value), nil
	}
	return nil, errors.New("no cert ref found")
}

func getCertificate(ctx context.Context, kube kclient.Client, ref *esv1.ExternalSecretsCARef, storeKind, namespace string) (*x509.CertPool, error) {
	certBytes, err := getCertificateBytes(ctx, kube, ref, storeKind, namespace)
	if err != nil {
		return nil, err
	}
	return convertCertificate(certBytes)
}

func convertCertificate(certBytes []byte) (*x509.CertPool, error) {
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return nil, errors.New("failed to decode certificate")
	}
	certs, err := x509.ParseCertificates(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	pool := x509.NewCertPool()
	for _, cert := range certs {
		pool.AddCert(cert)
	}
	return pool, nil
}

// ValidateStore validates the store configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	spec := storeSpec.Provider.ExternalSecrets
	if spec.Auth.Kubernetes != nil {
		err := utils.ValidateServiceAccountSelector(store, spec.Auth.Kubernetes.ServiceAccountRef)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func init() {
	esv1.Register(&Provider{}, &esv1.SecretStoreProvider{
		ExternalSecrets: &esv1.ExternalSecretsProvider{},
	}, esv1.MaintenanceStatusMaintained)
}
