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

package esutils

import (
	"context"
	"errors"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/feature"
)

const errUnableCreateK8sSAToken = "cannot create service account token: %q"

// ErrMultipleAuthMethods is returned when more than one inline auth method is configured.
var ErrMultipleAuthMethods = errors.New("multiple auth methods provided: set exactly one of Token, ServiceAccount, or Cert")

// ErrAuthRefWithInlineAuth is returned when both authRef and inline auth are set.
var ErrAuthRefWithInlineAuth = errors.New("authRef and inline auth cannot both be set")

// BuildRESTConfigFromKubernetesConnection builds a *rest.Config from the same
// server/auth fields used by the Kubernetes SecretStore provider. It is shared
// by the kubernetes and CRD providers.
func BuildRESTConfigFromKubernetesConnection(
	ctx context.Context,
	ctrlClient kclient.Client,
	coreV1 typedcorev1.CoreV1Interface,
	storeKind, esNamespace string,
	server esv1.KubernetesServer,
	auth *esv1.KubernetesAuth,
	authRef *esmeta.SecretKeySelector,
) (*rest.Config, error) {
	if authRef != nil {
		if auth != nil {
			return nil, ErrAuthRefWithInlineAuth
		}
		cfg, err := fetchKubernetesSecretKey(ctx, ctrlClient, storeKind, esNamespace, *authRef)
		if err != nil {
			return nil, err
		}
		return restConfigFromKubeConfig(cfg, feature.HostAccessAllowed(storeKind))
	}

	if auth == nil {
		return nil, errors.New("no auth provider given")
	}

	if server.URL == "" {
		return nil, errors.New("no server URL provided")
	}

	cfg := &rest.Config{
		Host: server.URL,
	}

	ca, err := FetchCACertFromSource(ctx, CreateCertOpts{
		CABundle:   server.CABundle,
		CAProvider: server.CAProvider,
		StoreKind:  storeKind,
		Namespace:  esNamespace,
		Client:     ctrlClient,
	})
	if err != nil {
		return nil, err
	}

	cfg.TLSClientConfig = rest.TLSClientConfig{
		Insecure: false,
		CAData:   ca,
	}

	if err := applyInlineAuth(ctx, ctrlClient, coreV1, storeKind, esNamespace, auth, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func restConfigFromKubeConfig(data []byte, allowHostFiles bool) (*rest.Config, error) {
	cfg, err := clientcmd.Load(data)
	if err != nil {
		return nil, err
	}
	if err := validateKubeConfigHostAccess(cfg, allowHostFiles); err != nil {
		return nil, err
	}

	restCfg, err := clientcmd.NewNonInteractiveClientConfig(*cfg, cfg.CurrentContext, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		return nil, err
	}
	if !allowHostFiles && (restCfg.BearerTokenFile != "" || restCfg.CAFile != "" || restCfg.CertFile != "" || restCfg.KeyFile != "" || restCfg.ExecProvider != nil || restCfg.AuthProvider != nil) {
		return nil, errors.New("kubeconfig produced a forbidden controller-host credential source")
	}
	return restCfg, nil
}

func validateKubeConfigHostAccess(cfg *clientcmdapi.Config, allowHostFiles bool) error {
	for name, cluster := range cfg.Clusters {
		if cluster != nil && cluster.CertificateAuthority != "" && !allowHostFiles {
			return fmt.Errorf(
				"kubeconfig cluster %q uses a certificate-authority file; controller-host files are only supported by ClusterSecretStore or when explicitly enabled by the operator",
				name,
			)
		}
	}
	for name, authInfo := range cfg.AuthInfos {
		if authInfo == nil {
			continue
		}
		if authInfo.Exec != nil {
			return fmt.Errorf("kubeconfig user %q uses forbidden exec authentication", name)
		}
		if authInfo.AuthProvider != nil {
			return fmt.Errorf("kubeconfig user %q uses forbidden auth-provider authentication", name)
		}
		if !allowHostFiles {
			switch {
			case authInfo.TokenFile != "":
				return fmt.Errorf(
					"kubeconfig user %q uses a token file; controller-host files are only supported by ClusterSecretStore or when explicitly enabled by the operator",
					name,
				)
			case authInfo.ClientCertificate != "":
				return fmt.Errorf(
					"kubeconfig user %q uses a client-certificate file; controller-host files are only supported by ClusterSecretStore or when explicitly enabled by the operator",
					name,
				)
			case authInfo.ClientKey != "":
				return fmt.Errorf(
					"kubeconfig user %q uses a client-key file; controller-host files are only supported by ClusterSecretStore or when explicitly enabled by the operator",
					name,
				)
			}
		}
	}
	return nil
}

// applyInlineAuth validates that exactly one auth method is set and populates cfg accordingly.
func applyInlineAuth(
	ctx context.Context,
	ctrlClient kclient.Client,
	coreV1 typedcorev1.CoreV1Interface,
	storeKind, esNamespace string,
	auth *esv1.KubernetesAuth,
	cfg *rest.Config,
) error {
	set := 0
	if auth.Token != nil {
		set++
	}
	if auth.ServiceAccount != nil {
		set++
	}
	if auth.Cert != nil {
		set++
	}
	if set > 1 {
		return ErrMultipleAuthMethods
	}

	switch {
	case auth.Token != nil:
		token, err := fetchKubernetesSecretKey(ctx, ctrlClient, storeKind, esNamespace, auth.Token.BearerToken)
		if err != nil {
			return fmt.Errorf("could not fetch Auth.Token.BearerToken: %w", err)
		}
		cfg.BearerToken = string(token)
	case auth.ServiceAccount != nil:
		token, err := serviceAccountToken(ctx, coreV1, storeKind, esNamespace, auth.ServiceAccount)
		if err != nil {
			return fmt.Errorf("could not fetch Auth.ServiceAccount: %w", err)
		}
		cfg.BearerToken = string(token)
	case auth.Cert != nil:
		key, cert, err := clientCertKeyFromSecrets(ctx, ctrlClient, storeKind, esNamespace, auth.Cert)
		if err != nil {
			return fmt.Errorf("could not fetch client key and cert: %w", err)
		}
		cfg.TLSClientConfig.KeyData = key
		cfg.TLSClientConfig.CertData = cert
	default:
		return errors.New("no auth provider given")
	}
	return nil
}

// fetchKubernetesSecretKey resolves a SecretKeySelector and returns its value as bytes.
func fetchKubernetesSecretKey(ctx context.Context, ctrlClient kclient.Client, storeKind, esNamespace string, ref esmeta.SecretKeySelector) ([]byte, error) {
	secret, err := resolvers.SecretKeyRef(
		ctx,
		ctrlClient,
		storeKind,
		esNamespace,
		&ref,
	)
	if err != nil {
		return nil, err
	}
	return []byte(secret), nil
}

// serviceAccountToken creates a short-lived token for the referenced service account.
// For ClusterSecretStore, serviceAccountRef.namespace overrides the provided namespace.
func serviceAccountToken(ctx context.Context, coreV1 typedcorev1.CoreV1Interface, storeKind, namespace string, serviceAccountRef *esmeta.ServiceAccountSelector) ([]byte, error) {
	if storeKind == esv1.ClusterSecretStoreKind && serviceAccountRef.Namespace != nil {
		namespace = *serviceAccountRef.Namespace
	}
	expirationSeconds := int64(3600)
	tr, err := coreV1.ServiceAccounts(namespace).CreateToken(ctx, serviceAccountRef.Name, &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         serviceAccountRef.Audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateK8sSAToken, err)
	}
	return []byte(tr.Status.Token), nil
}

// clientCertKeyFromSecrets fetches client certificate and key material from Secrets.
// It returns key bytes first, then certificate bytes to match rest.TLSClientConfig fields.
func clientCertKeyFromSecrets(ctx context.Context, ctrlClient kclient.Client, storeKind, esNamespace string, cert *esv1.CertAuth) ([]byte, []byte, error) {
	certPEM, err := fetchKubernetesSecretKey(ctx, ctrlClient, storeKind, esNamespace, cert.ClientCert)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to fetch client certificate: %w", err)
	}
	keyPEM, err := fetchKubernetesSecretKey(ctx, ctrlClient, storeKind, esNamespace, cert.ClientKey)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to fetch client key: %w", err)
	}
	return keyPEM, certPEM, nil
}
