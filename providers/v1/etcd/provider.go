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

// Package etcd implements a secret store provider for etcd.
package etcd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	errMissingEtcdSpec    = "missing etcd spec"
	errMissingEndpoints   = "etcd endpoints must be specified"
	errMissingUsername    = "missing username in secretRef"
	errMissingPassword    = "missing password in secretRef"
	errMissingClientCert  = "missing clientCert in tls auth"
	errMissingClientKey   = "missing clientKey in tls auth"
	errNilStore           = "secret store is nil"
	errGetCredentials     = "failed to get credentials: %w"
	errLoadTLSCert        = "failed to load TLS certificate: %w"
	errLoadCACert         = "failed to load CA certificate: %w"
	errCreateClient       = "failed to create etcd client: %w"
	defaultPrefix         = "/external-secrets/"
	defaultDialTimeout    = 5 * time.Second
	defaultRequestTimeout = 10 * time.Second
)

var _ esv1.Provider = &Provider{}

// Provider implements the esv1.Provider interface for etcd.
type Provider struct{}

// Client implements the esv1.SecretsClient interface for etcd.
type Client struct {
	kv        clientv3.KV
	client    *clientv3.Client
	prefix    string
	storeKind string
}

// Capabilities returns the capabilities of the etcd provider.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewClient creates a new etcd client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	if store == nil {
		return nil, errors.New(errNilStore)
	}

	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Etcd == nil {
		return nil, errors.New(errMissingEtcdSpec)
	}

	etcdSpec := storeSpec.Provider.Etcd
	storeKind := store.GetKind()

	// Build client config
	cfg := clientv3.Config{
		Endpoints:   etcdSpec.Endpoints,
		DialTimeout: defaultDialTimeout,
	}

	// Configure TLS first (for CA certificate)
	var tlsCfg *tls.Config
	if etcdSpec.CAProvider != nil {
		caCert, err := esutils.FetchCACertFromSource(ctx, esutils.CreateCertOpts{
			CAProvider: etcdSpec.CAProvider,
			StoreKind:  storeKind,
			Namespace:  namespace,
			Client:     kube,
		})
		if err != nil {
			return nil, fmt.Errorf(errLoadCACert, err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("failed to parse CA certificate")
		}

		tlsCfg = &tls.Config{
			RootCAs:    caCertPool,
			MinVersion: tls.VersionTLS12,
		}
	}

	// Configure authentication
	if etcdSpec.Auth != nil {
		if err := p.configureAuth(ctx, kube, storeKind, namespace, etcdSpec, &cfg, &tlsCfg); err != nil {
			return nil, fmt.Errorf(errGetCredentials, err)
		}
	}

	if tlsCfg != nil {
		cfg.TLS = tlsCfg
	}

	// Create etcd client
	etcdClient, err := clientv3.New(cfg)
	if err != nil {
		return nil, fmt.Errorf(errCreateClient, err)
	}

	prefix := etcdSpec.Prefix
	if prefix == "" {
		prefix = defaultPrefix
	}

	return &Client{
		kv:        etcdClient.KV,
		client:    etcdClient,
		prefix:    prefix,
		storeKind: storeKind,
	}, nil
}

func (p *Provider) configureAuth(ctx context.Context, kube client.Client, storeKind, namespace string, etcdSpec *esv1.EtcdProvider, cfg *clientv3.Config, tlsCfg **tls.Config) error {
	if etcdSpec.Auth.SecretRef != nil {
		// Username/password authentication
		username, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &etcdSpec.Auth.SecretRef.Username)
		if err != nil {
			return fmt.Errorf("failed to get username: %w", err)
		}

		password, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &etcdSpec.Auth.SecretRef.Password)
		if err != nil {
			return fmt.Errorf("failed to get password: %w", err)
		}

		cfg.Username = username
		cfg.Password = password
	}

	if etcdSpec.Auth.TLS != nil {
		// TLS client certificate authentication
		clientCert, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &etcdSpec.Auth.TLS.ClientCert)
		if err != nil {
			return fmt.Errorf("failed to get client certificate: %w", err)
		}

		clientKey, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &etcdSpec.Auth.TLS.ClientKey)
		if err != nil {
			return fmt.Errorf("failed to get client key: %w", err)
		}

		cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
		if err != nil {
			return fmt.Errorf(errLoadTLSCert, err)
		}

		if *tlsCfg == nil {
			*tlsCfg = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}
		(*tlsCfg).Certificates = []tls.Certificate{cert}
	}

	return nil
}

// ValidateStore validates the etcd store configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil {
		return nil, errors.New(errMissingEtcdSpec)
	}

	etcdSpec := storeSpec.Provider.Etcd
	if etcdSpec == nil {
		return nil, errors.New(errMissingEtcdSpec)
	}

	if len(etcdSpec.Endpoints) == 0 {
		return nil, errors.New(errMissingEndpoints)
	}

	if etcdSpec.Auth != nil {
		// Validate that both auth methods aren't specified simultaneously (though they can be combined)
		if etcdSpec.Auth.SecretRef != nil {
			if etcdSpec.Auth.SecretRef.Username.Name == "" || etcdSpec.Auth.SecretRef.Username.Key == "" {
				return nil, errors.New(errMissingUsername)
			}
			if etcdSpec.Auth.SecretRef.Password.Name == "" || etcdSpec.Auth.SecretRef.Password.Key == "" {
				return nil, errors.New(errMissingPassword)
			}
		}

		if etcdSpec.Auth.TLS != nil {
			if etcdSpec.Auth.TLS.ClientCert.Name == "" || etcdSpec.Auth.TLS.ClientCert.Key == "" {
				return nil, errors.New(errMissingClientCert)
			}
			if etcdSpec.Auth.TLS.ClientKey.Name == "" || etcdSpec.Auth.TLS.ClientKey.Key == "" {
				return nil, errors.New(errMissingClientKey)
			}
		}
	}

	return nil, nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Etcd: &esv1.EtcdProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
