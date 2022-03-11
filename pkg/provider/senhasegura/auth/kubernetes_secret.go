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
	"context"
	"fmt"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errRequiredNamespaceNotFound   = "invalid ClusterSecretStore: missing namespace in %s"
	errCannotFetchKubernetesSecret = "could not fetch Kubernetes secret %s"
)

/*
	getKubernetesSecret get Kubernetes Secret based on object parameter in namespace where ESO is installed or another, if ClusterSecretStore is used
*/
func getKubernetesSecret(ctx context.Context, object esmeta.SecretKeySelector, store esv1beta1.GenericStore, kube client.Client, namespace string) (string, error) {
	ke := client.ObjectKey{
		Name:      object.Name,
		Namespace: namespace, // Default to ExternalSecret namespace
	}

	// Only ClusterStore is allowed to set namespace (and then it's required)
	if store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind {
		if object.Namespace == nil {
			return "", fmt.Errorf(errRequiredNamespaceNotFound, object.Key)
		}
		ke.Namespace = *object.Namespace
	}

	secret := v1.Secret{}
	err := kube.Get(ctx, ke, &secret)
	if err != nil {
		return "", fmt.Errorf(errCannotFetchKubernetesSecret, object.Name)
	}

	return string(secret.Data[object.Key]), nil
}
