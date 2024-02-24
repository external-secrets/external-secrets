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

package oracle

import (
	"context"

	"github.com/oracle/oci-go-sdk/v65/common/auth"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// TokenProvider implements the ServiceAccountTokenProvider interface to create service account tokens for OCI authentication.
type TokenProvider struct {
	Name      string
	Namespace string
	Audiences []string
	Clientset kubernetes.Interface
}

var _ auth.ServiceAccountTokenProvider = &TokenProvider{}

// NewTokenProvider creates a new TokenProvider for a given service account.
func NewTokenProvider(clientset kubernetes.Interface, serviceAccountRef *esmeta.ServiceAccountSelector, namespace string) *TokenProvider {
	// "api" is the default OCI workload identity audience.
	audiences := []string{"api"}
	if len(serviceAccountRef.Audiences) > 0 {
		audiences = append(audiences, serviceAccountRef.Audiences...)
	}
	if serviceAccountRef.Namespace != nil {
		namespace = *serviceAccountRef.Namespace
	}
	return &TokenProvider{
		Name:      serviceAccountRef.Name,
		Namespace: namespace,
		Audiences: audiences,
		Clientset: clientset,
	}
}

// ServiceAccountToken creates a new service account token for OCI authentication.
func (t *TokenProvider) ServiceAccountToken() (string, error) {
	tok, err := t.Clientset.CoreV1().ServiceAccounts(t.Namespace).CreateToken(context.Background(), t.Name, &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences: t.Audiences,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	return tok.Status.Token, nil
}
