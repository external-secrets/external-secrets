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
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const errUnableCreateK8sSAToken = "cannot create service account token: %q"

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
		cfg, err := fetchKubernetesSecretKey(ctx, ctrlClient, storeKind, esNamespace, *authRef)
		if err != nil {
			return nil, err
		}
		return clientcmd.RESTConfigFromKubeConfig(cfg)
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

	switch {
	case auth.Token != nil:
		token, err := fetchKubernetesSecretKey(ctx, ctrlClient, storeKind, esNamespace, auth.Token.BearerToken)
		if err != nil {
			return nil, fmt.Errorf("could not fetch Auth.Token.BearerToken: %w", err)
		}
		cfg.BearerToken = string(token)
	case auth.ServiceAccount != nil:
		token, err := serviceAccountTokenFromCoreV1(ctx, coreV1, storeKind, esNamespace, auth.ServiceAccount)
		if err != nil {
			return nil, fmt.Errorf("could not fetch Auth.ServiceAccount: %w", err)
		}
		cfg.BearerToken = string(token)
	case auth.Cert != nil:
		key, cert, err := clientCertKeyFromSecrets(ctx, ctrlClient, storeKind, esNamespace, auth.Cert)
		if err != nil {
			return nil, fmt.Errorf("could not fetch client key and cert: %w", err)
		}
		cfg.TLSClientConfig.KeyData = key
		cfg.TLSClientConfig.CertData = cert
	default:
		return nil, errors.New("no auth provider given")
	}

	return cfg, nil
}

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

func serviceAccountTokenFromCoreV1(ctx context.Context, coreV1 typedcorev1.CoreV1Interface, storeKind, esNamespace string, serviceAccountRef *esmeta.ServiceAccountSelector) ([]byte, error) {
	namespace := esNamespace
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
