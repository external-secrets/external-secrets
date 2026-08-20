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

package safeguard

import (
	"context"
	"fmt"

	sg "github.com/OneIdentity/safeguard-go"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

func newA2AContext(ctx context.Context, store esv1.GenericStore, cfg *esv1.SafeguardProvider, kube client.Client, namespace string) (*sg.A2AContext, error) {
	certPEM, err := loadSecretRef(ctx, store, kube, namespace, &cfg.Auth.A2A.Certificate)
	if err != nil {
		return nil, fmt.Errorf("certificate: %w", err)
	}

	var keyPEM []byte
	if cfg.Auth.A2A.CertificateKey != nil {
		keyPEM, err = loadSecretRef(ctx, store, kube, namespace, cfg.Auth.A2A.CertificateKey)
		if err != nil {
			return nil, fmt.Errorf("certificateKey: %w", err)
		}
	}

	certPassword := sg.NewSecretString("")
	defer certPassword.Zero()
	if cfg.Auth.A2A.CertificatePassword != nil {
		password, err := loadSecretRef(ctx, store, kube, namespace, cfg.Auth.A2A.CertificatePassword)
		if err != nil {
			return nil, fmt.Errorf("certificatePassword: %w", err)
		}
		certPassword = sg.NewSecretString(string(password))
	}

	connOpts, err := connectionOptions(ctx, store, cfg, kube, namespace)
	if err != nil {
		return nil, err
	}

	var a2aOpts []sg.A2AOption
	if len(keyPEM) > 0 {
		a2aOpts = append(a2aOpts, sg.WithA2APrivateKeyPEM(keyPEM))
	}
	if len(connOpts) > 0 {
		a2aOpts = append(a2aOpts, sg.WithA2AConnectionOptions(connOpts...))
	}

	return sg.NewA2AContext(cfg.Appliance, certPEM, certPassword, a2aOpts...)
}

func connectionOptions(ctx context.Context, store esv1.GenericStore, cfg *esv1.SafeguardProvider, kube client.Client, namespace string) ([]sg.Option, error) {
	var opts []sg.Option

	if len(cfg.CABundle) > 0 || cfg.CAProvider != nil {
		cert, err := esutils.FetchCACertFromSource(ctx, esutils.CreateCertOpts{
			StoreKind:  store.GetKind(),
			Client:     kube,
			Namespace:  namespace,
			CABundle:   cfg.CABundle,
			CAProvider: cfg.CAProvider,
		})
		if err != nil {
			return nil, err
		}
		opts = append(opts, sg.WithCABundle(cert))
	}

	if cfg.APIVersion != "" {
		opts = append(opts, sg.WithAPIVersion(cfg.APIVersion))
	}

	return opts, nil
}

func loadSecretRef(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string, ref *esv1.SafeguardProviderSecretRef) ([]byte, error) {
	if ref.SecretRef == nil {
		return []byte(ref.Value), nil
	}
	value, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, ref.SecretRef)
	if err != nil {
		return nil, err
	}
	return []byte(value), nil
}
