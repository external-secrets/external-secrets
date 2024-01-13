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

package utils

import (
	"context"
	"fmt"

	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	errGetKubeSecret         = "cannot get Kubernetes secret %q: %w"
	errSecretKeyFmt          = "cannot find secret data for key: %q"
	errGetKubeSATokenRequest = "cannot request Kubernetes service account token for service account %q: %w"
)

// Resolves a metav1.SecretKeySelector and returns the value of the secret it points to.
// A user must pass the namespace of the originating ExternalSecret, as this may differ
// from the namespace defined in the SecretKeySelector.
// This func ensures that only a ClusterSecretStore is able to request secrets across namespaces.
func ResolveSecretKeyRef(
	ctx context.Context,
	c client.Client,
	storeKind string,
	esNamespace string,
	ref *esmeta.SecretKeySelector) (string, error) {
	key := types.NamespacedName{
		Namespace: esNamespace,
		Name:      ref.Name,
	}
	if (storeKind == esv1beta1.ClusterSecretStoreKind) &&
		(ref.Namespace != nil) {
		key.Namespace = *ref.Namespace
	}
	secret := &corev1.Secret{}
	err := c.Get(ctx, key, secret)
	if err != nil {
		return "", fmt.Errorf(errGetKubeSecret, ref.Name, err)
	}
	val, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf(errSecretKeyFmt, ref.Key)
	}
	return string(val), nil
}

func CreateServiceAccountToken(
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
	if (storeKind == esv1beta1.ClusterSecretStoreKind) &&
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

func NewKubeClient() (*kubernetes.Clientset, error) {
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	c, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	return c, err
}
