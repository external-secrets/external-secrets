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

package auth

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/credentials"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
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

// FetchToken satisfies the stscreds.TokenFetcher interface
// it is used to generate service account tokens which are consumed by the aws sdk.
func (p authTokenFetcher) FetchToken(ctx credentials.Context) ([]byte, error) {
	log.V(1).Info("fetching token", "ns", p.Namespace, "sa", p.ServiceAccount)
	tokRsp, err := p.k8sClient.ServiceAccounts(p.Namespace).CreateToken(ctx, p.ServiceAccount, &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences: p.Audiences,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating service account token: %w", err)
	}
	return []byte(tokRsp.Status.Token), nil
}
