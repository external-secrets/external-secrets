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

package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// mostly taken from:
// https://github.com/aws/secrets-store-csi-driver-provider-aws/blob/main/auth/auth.go#L140-L145

type authTokenFetcher struct {
	Namespace string
	// Audience is the token aud claim
	// which is verified by the aws oidc provider
	// see: https://github.com/external-secrets/external-secrets/issues/1251#issuecomment-1161745849
	Audiences      []string
	ServiceAccount string
	k8sClient      corev1.CoreV1Interface
}

// GetIdentityToken satisfies the stscreds.IdentityTokenRetriever interface
// it is used to generate service account tokens which are consumed by the aws sdk.
func (p authTokenFetcher) GetIdentityToken() ([]byte, error) {
	log.V(1).Info("fetching token", "ns", p.Namespace, "sa", p.ServiceAccount)
	tokRsp, err := p.k8sClient.ServiceAccounts(p.Namespace).CreateToken(context.Background(), p.ServiceAccount, &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences: p.Audiences,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating service account token: %w", err)
	}
	return []byte(tokRsp.Status.Token), nil
}

// secretTokenFetcher reads a JWT from a key in a Kubernetes Secret.
// The Secret is read on every call so that tokens rotated by an external
// identity provider are picked up when the AWS credentials are refreshed.
type secretTokenFetcher struct {
	kube      client.Client
	storeKind string
	namespace string
	ref       esmeta.SecretKeySelector
}

// GetIdentityToken satisfies the stscreds.IdentityTokenRetriever interface.
func (p secretTokenFetcher) GetIdentityToken() ([]byte, error) {
	log.V(1).Info("fetching token from secret", "ns", p.namespace, "secret", p.ref.Name, "key", p.ref.Key)
	token, err := resolvers.SecretKeyRef(context.Background(), p.kube, p.storeKind, p.namespace, &p.ref)
	if err != nil {
		return nil, fmt.Errorf(errFetchJWTSecret, err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf(errFetchJWTSecret, errors.New("token is empty"))
	}
	return []byte(token), nil
}
