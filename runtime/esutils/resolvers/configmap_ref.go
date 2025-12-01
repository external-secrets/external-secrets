// /*
// Copyright © 2025 ESO Maintainer Team
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

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	errGetKubeConfigMap = "cannot get Kubernetes configmap %q: %w"
	errConfigMapKeyFmt  = "cannot find configmap data for key: %q"
)

// ConfigMapKeyRef resolves a metav1.SecretKeySelector and returns the value of the secret it points to.
// A user must pass the namespace of the originating ExternalSecret, as this may differ
// from the namespace defined in the SecretKeySelector.
// This func ensures that only a ClusterSecretStore is able to request secrets across namespaces.
func ConfigMapKeyRef(
	ctx context.Context,
	c client.Client,
	esNamespace string,
	ref *esmeta.SecretKeySelector) (string, error) {
	key := types.NamespacedName{
		Namespace: esNamespace,
		Name:      ref.Name,
	}
	if ref.Namespace != nil {
		key.Namespace = *ref.Namespace
	}
	configMap := &corev1.ConfigMap{}
	err := c.Get(ctx, key, configMap)
	if err != nil {
		return "", fmt.Errorf(errGetKubeConfigMap, ref.Name, err)
	}
	val, ok := configMap.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf(errConfigMapKeyFmt, ref.Key)
	}
	return val, nil
}
