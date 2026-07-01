package openbao

import (
	"context"
	"fmt"
	"os"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	serviceAccTokenPath       = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	errServiceAccount         = "cannot read Kubernetes service account token from file system: %w"
	errGetKubeSA              = "cannot get Kubernetes service account %q: %w"
	errGetKubeSASecrets       = "cannot find secrets bound to service account: %q"
	errGetKubeSANoToken       = "cannot find token in secrets bound to service account: %q"
	errServiceAccountNotFound = "serviceaccounts %q not found"
	errGetKubeSATokenRequest  = "cannot request Kubernetes service account token for service account %q: %w"
)

func getJwtString(ctx context.Context, c *client, kubernetesAuth *esv1.OpenBaoKubernetesAuth, namespace string) (string, error) {
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
		jwt, err := resolvers.SecretKeyRef(ctx, c.kubernetesClient, c.storeKind, "", tokenRef)
		if err != nil {
			return "", err
		}
		return jwt, nil
	}

	// Kubernetes authentication is specified, but without a referenced
	// Kubernetes secret. We check if the file path for in-cluster service account
	// exists and attempt to use the token for Kubernetes auth.
	if _, err := os.Stat(serviceAccTokenPath); err != nil {
		return "", fmt.Errorf(errServiceAccount, err)
	}
	jwtByte, err := os.ReadFile(serviceAccTokenPath)
	if err != nil {
		return "", fmt.Errorf(errServiceAccount, err)
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
		return "", fmt.Errorf(errGetKubeSATokenRequest, serviceAccountRef.Name, err)
	}
	return tokenResponse.Status.Token, nil
}
