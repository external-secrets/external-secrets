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

package vault

import (
	"context"
	"fmt"
	"os"

	authkubernetes "github.com/hashicorp/vault/api/auth/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	serviceAccTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	errServiceAccount   = "cannot read Kubernetes service account token from file system: %w"
	errGetKubeSA        = "cannot get Kubernetes service account %q: %w"
	errGetKubeSASecrets = "cannot find secrets bound to service account: %q"
	errGetKubeSANoToken = "cannot find token in secrets bound to service account: %q"
)

func setKubernetesAuthToken(ctx context.Context, v *client) (bool, error) {
	kubernetesAuth := v.store.Auth.Kubernetes
	if kubernetesAuth != nil {
		err := v.requestTokenWithKubernetesAuth(ctx, kubernetesAuth)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func (c *client) requestTokenWithKubernetesAuth(ctx context.Context, kubernetesAuth *esv1beta1.VaultKubernetesAuth) error {
	jwtString, err := getJwtString(ctx, c, kubernetesAuth)
	if err != nil {
		return err
	}
	k, err := authkubernetes.NewKubernetesAuth(kubernetesAuth.Role, authkubernetes.WithServiceAccountToken(jwtString), authkubernetes.WithMountPath(kubernetesAuth.Path))
	if err != nil {
		return err
	}
	_, err = c.auth.Login(ctx, k)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultLogin, err)
	if err != nil {
		return err
	}
	return nil
}

func getJwtString(ctx context.Context, v *client, kubernetesAuth *esv1beta1.VaultKubernetesAuth) (string, error) {
	if kubernetesAuth.ServiceAccountRef != nil {
		// Kubernetes <v1.24 fetch token via ServiceAccount.Secrets[]
		// this behavior was removed in v1.24 and we must use TokenRequest API (see below)
		jwt, err := v.secretKeyRefForServiceAccount(ctx, kubernetesAuth.ServiceAccountRef)
		if jwt != "" {
			return jwt, err
		}
		if err != nil {
			v.log.V(1).Info("unable to fetch jwt from service account secret, trying service account token next")
		}
		// Kubernetes >=v1.24: fetch token via TokenRequest API
		// note: this is a massive change from vault perspective: the `iss` claim will very likely change.
		// Vault 1.9 deprecated issuer validation by default, and authentication with Vault clusters <1.9 will likely fail.
		jwt, err = createServiceAccountToken(
			ctx,
			v.corev1,
			v.storeKind,
			v.namespace,
			*kubernetesAuth.ServiceAccountRef,
			nil,
			600)
		if err != nil {
			return "", err
		}
		return jwt, nil
	} else if kubernetesAuth.SecretRef != nil {
		tokenRef := kubernetesAuth.SecretRef
		if tokenRef.Key == "" {
			tokenRef = kubernetesAuth.SecretRef.DeepCopy()
			tokenRef.Key = "token"
		}
		jwt, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, tokenRef)
		if err != nil {
			return "", err
		}
		return jwt, nil
	} else {
		// Kubernetes authentication is specified, but without a referenced
		// Kubernetes secret. We check if the file path for in-cluster service account
		// exists and attempt to use the token for Vault Kubernetes auth.
		if _, err := os.Stat(serviceAccTokenPath); err != nil {
			return "", fmt.Errorf(errServiceAccount, err)
		}
		jwtByte, err := os.ReadFile(serviceAccTokenPath)
		if err != nil {
			return "", fmt.Errorf(errServiceAccount, err)
		}
		return string(jwtByte), nil
	}
}

func (c *client) secretKeyRefForServiceAccount(ctx context.Context, serviceAccountRef *esmeta.ServiceAccountSelector) (string, error) {
	serviceAccount := &corev1.ServiceAccount{}
	ref := types.NamespacedName{
		Namespace: c.namespace,
		Name:      serviceAccountRef.Name,
	}
	if (c.storeKind == esv1beta1.ClusterSecretStoreKind) &&
		(serviceAccountRef.Namespace != nil) {
		ref.Namespace = *serviceAccountRef.Namespace
	}
	err := c.kube.Get(ctx, ref, serviceAccount)
	if err != nil {
		return "", fmt.Errorf(errGetKubeSA, ref.Name, err)
	}
	if len(serviceAccount.Secrets) == 0 {
		return "", fmt.Errorf(errGetKubeSASecrets, ref.Name)
	}
	for _, tokenRef := range serviceAccount.Secrets {
		token, err := resolvers.SecretKeyRef(ctx, c.kube, c.storeKind, c.namespace, &esmeta.SecretKeySelector{
			Name:      tokenRef.Name,
			Namespace: &ref.Namespace,
			Key:       "token",
		})
		if err != nil {
			continue
		}
		return token, nil
	}
	return "", fmt.Errorf(errGetKubeSANoToken, ref.Name)
}
