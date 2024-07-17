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

package kubernetes

import (
	"context"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errUnableCreateToken = "cannot create service account token: %q"
)

func (c *Client) getAuth(ctx context.Context) (*rest.Config, error) {
	if c.store.AuthRef != nil {
		cfg, err := c.fetchSecretKey(ctx, *c.store.AuthRef)
		if err != nil {
			return nil, err
		}

		return clientcmd.RESTConfigFromKubeConfig(cfg)
	}

	ca, err := utils.FetchCACertFromSource(ctx, utils.CreateCertOpts{
		CABundle:   c.store.Server.CABundle,
		CAProvider: c.store.Server.CAProvider,
		StoreKind:  c.storeKind,
		Namespace:  c.namespace,
		Client:     c.ctrlClient,
	})
	if err != nil {
		return nil, err
	}

	var token []byte
	if c.store.Auth.Token != nil {
		token, err = c.fetchSecretKey(ctx, c.store.Auth.Token.BearerToken)
		if err != nil {
			return nil, fmt.Errorf("could not fetch Auth.Token.BearerToken: %w", err)
		}
	} else if c.store.Auth.ServiceAccount != nil {
		token, err = c.serviceAccountToken(ctx, c.store.Auth.ServiceAccount)
		if err != nil {
			return nil, fmt.Errorf("could not fetch Auth.ServiceAccount: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no auth provider given")
	}

	var key, cert []byte
	if c.store.Auth.Cert != nil {
		key, cert, err = c.getClientKeyAndCert(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not fetch client key and cert: %w", err)
		}
	}

	if c.store.Server.URL == "" {
		return nil, fmt.Errorf("no server URL provided")
	}

	return &rest.Config{
		Host:        c.store.Server.URL,
		BearerToken: string(token),
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: false,
			CertData: cert,
			KeyData:  key,
			CAData:   ca,
		},
	}, nil
}

func (c *Client) getClientKeyAndCert(ctx context.Context) ([]byte, []byte, error) {
	var err error
	cert, err := c.fetchSecretKey(ctx, c.store.Auth.Cert.ClientCert)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to fetch client certificate: %w", err)
	}
	key, err := c.fetchSecretKey(ctx, c.store.Auth.Cert.ClientKey)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to fetch client key: %w", err)
	}
	return key, cert, nil
}

func (c *Client) serviceAccountToken(ctx context.Context, serviceAccountRef *esmeta.ServiceAccountSelector) ([]byte, error) {
	namespace := c.namespace
	if (c.storeKind == esv1beta1.ClusterSecretStoreKind) &&
		(serviceAccountRef.Namespace != nil) {
		namespace = *serviceAccountRef.Namespace
	}
	expirationSeconds := int64(3600)
	tr, err := c.ctrlClientset.ServiceAccounts(namespace).CreateToken(ctx, serviceAccountRef.Name, &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         serviceAccountRef.Audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateToken, err)
	}
	return []byte(tr.Status.Token), nil
}

func (c *Client) fetchSecretKey(ctx context.Context, ref esmeta.SecretKeySelector) ([]byte, error) {
	secret, err := resolvers.SecretKeyRef(
		ctx,
		c.ctrlClient,
		c.storeKind,
		c.namespace,
		&ref,
	)
	if err != nil {
		return nil, err
	}
	return []byte(secret), nil
}
