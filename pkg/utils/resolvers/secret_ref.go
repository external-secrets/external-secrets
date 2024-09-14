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

package resolvers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (

	// This is used to determine if a store is cluster-scoped or not.
	// The EmptyStoreKind is not cluster-scoped, hence resources
	// cannot be resolved across namespaces.
	// TODO: when we implement cluster-scoped generators
	// we can remove this and replace it with a interface.
	EmptyStoreKind = "EmptyStoreKind"

	errGetKubeSecret         = "cannot get Kubernetes secret %q: %w"
	errSecretKeyFmt          = "cannot find secret data for key: %q"
	errGetKubeSATokenRequest = "cannot request Kubernetes service account token for service account %q: %w"
)

// SecretKeyRef resolves a metav1.SecretKeySelector and returns the value of the secret it points to.
// A user must pass the namespace of the originating ExternalSecret, as this may differ
// from the namespace defined in the SecretKeySelector.
// This func ensures that only a ClusterSecretStore is able to request secrets across namespaces.
func SecretKeyRef(
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
