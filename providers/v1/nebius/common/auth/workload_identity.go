// /*
// Copyright © The ESO Authors
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

package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	workloadIdentityTokenTTL = 10 * time.Minute
	// NebiusIamAudience is the audience required for Nebius IAM token exchange.
	NebiusIamAudience = "sts.nebius.com"
)

// ResolvedFederatedCredentials contains resolved workload identity credentials.
type ResolvedFederatedCredentials struct {
	ServiceAccountID string
	SubjectToken     string
}

func (r ResolvedFederatedCredentials) isResolvedCredentials() {
}

// NewFederatedAccountCredentialsRequest creates a request for federated account credentials.
func NewFederatedAccountCredentialsRequest(
	serviceAccountRef esmeta.ServiceAccountSelector,
	serviceAccountID string,
	store esv1.GenericStore,
	corev1 typedcorev1.CoreV1Interface,
	namespace string,
) (*CredentialRequest, error) {
	return &CredentialRequest{
		cacheKey: getWICacheKey(namespace, store),
		resolve: func(ctx context.Context) (ResolvedCredentials, error) {
			tokenNamespace := namespace
			if store.GetKind() == esv1.ClusterSecretStoreKind && serviceAccountRef.Namespace != nil {
				tokenNamespace = *serviceAccountRef.Namespace
			}

			audiences := []string{NebiusIamAudience}

			tokenRequest := &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences:         audiences,
					ExpirationSeconds: new(int64(workloadIdentityTokenTTL.Seconds())),
				},
			}
			tokenResponse, err := corev1.ServiceAccounts(tokenNamespace).CreateToken(
				ctx,
				serviceAccountRef.Name,
				tokenRequest,
				metav1.CreateOptions{},
			)
			if err != nil {
				return nil, fmt.Errorf("get workload identity token for service account %s/%s: %w", tokenNamespace, serviceAccountRef.Name, err)
			}
			return &ResolvedFederatedCredentials{
				ServiceAccountID: serviceAccountID,
				SubjectToken:     strings.TrimSpace(tokenResponse.Status.Token),
			}, nil
		},
	}, nil
}

func getWICacheKey(effectiveNamespace string, store esv1.GenericStore) string {
	cacheKey := fmt.Sprintf("workload-identity|%s|%d|%s",
		store.GetUID(),
		store.GetGeneration(),
		effectiveNamespace,
	)
	return cacheKey
}
