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

package vault

import (
	"context"
	"fmt"
	"os"

	authkubernetes "github.com/hashicorp/vault/api/auth/kubernetes"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	serviceAccTokenPath       = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	errServiceAccount         = "cannot read Kubernetes service account token from file system: %w"
	errGetKubeSA              = "cannot get Kubernetes service account %q: %w"
	errGetKubeSASecrets       = "cannot find secrets bound to service account: %q"
	errGetKubeSANoToken       = "cannot find token in secrets bound to service account: %q"
	errServiceAccountNotFound = "serviceaccounts %q not found"
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

func (c *client) requestTokenWithKubernetesAuth(ctx context.Context, kubernetesAuth *esv1.VaultKubernetesAuth) error {
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

func getJwtString(ctx context.Context, v *client, kubernetesAuth *esv1.VaultKubernetesAuth) (string, error) {
	if kubernetesAuth.ServiceAccountRef != nil {
		return createServiceAccountToken(ctx, v.corev1, v.storeKind, v.namespace, *kubernetesAuth.ServiceAccountRef)
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
	}

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
