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

package openbao

import (
	"context"
	"fmt"
	"os"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

func getJwt(ctx context.Context, c *client, kubernetesAuth *esv1.OpenBaoKubernetesAuth, kube k8sClient.Client, namespace string) (string, error) {
	if kubernetesAuth.ServiceAccountRef != nil {
		return createServiceAccountToken(
			ctx,
			c.corev1,
			c.storeKind,
			namespace,
			*kubernetesAuth.ServiceAccountRef,
			nil,
			600)
	}

	if kubernetesAuth.SecretRef != nil {
		tokenRef := kubernetesAuth.SecretRef
		if tokenRef.Key == "" {
			tokenRef = kubernetesAuth.SecretRef.DeepCopy()
			tokenRef.Key = "token"
		}
		jwt, err := resolvers.SecretKeyRef(ctx, kube, c.storeKind, namespace, tokenRef)
		if err != nil {
			return "", err
		}
		return jwt, nil
	}

	// Kubernetes authentication is specified, but without a referenced
	// Kubernetes secret. We check if the file path for in-cluster service account
	// exists and attempt to use the token for Kubernetes auth.
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err != nil {
		return "", fmt.Errorf("cannot read Kubernetes service account token from file system: %w", err)
	}
	jwtByte, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", fmt.Errorf("cannot read Kubernetes service account token from file system: %w", err)
	}
	return string(jwtByte), nil
}

func createServiceAccountToken(
	ctx context.Context,
	corev1Client typedcorev1.CoreV1Interface,
	storeKind string,
	namespace string,
	serviceAccountRef esmeta.ServiceAccountSelector,
	additionalAud []string,
	expirationSeconds int64) (string, error) {
	audiences := serviceAccountRef.Audiences
	if len(additionalAud) > 0 {
		audiences = append(audiences, additionalAud...)
	}
	tokenRequest := &authv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: authv1.TokenRequestSpec{
			Audiences:         audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}
	if (storeKind == esv1.ClusterSecretStoreKind) &&
		(serviceAccountRef.Namespace != nil) {
		tokenRequest.Namespace = *serviceAccountRef.Namespace
	}
	tokenResponse, err := corev1Client.ServiceAccounts(tokenRequest.Namespace).
		CreateToken(ctx, serviceAccountRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("cannot request Kubernetes service account token for service account %q: %w", serviceAccountRef.Name, err)
	}
	return tokenResponse.Status.Token, nil
}
