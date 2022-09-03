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

package akeyless

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errInvalidClusterStoreMissingAKIDNamespace = "invalid ClusterSecretStore: missing Akeyless AccessID Namespace"
	errInvalidClusterStoreMissingSAKNamespace  = "invalid ClusterSecretStore: missing Akeyless AccessType Namespace"
	errFetchAKIDSecret                         = "could not fetch accessID secret: %w"
	errFetchSAKSecret                          = "could not fetch AccessType secret: %w"
	errMissingSAK                              = "missing SecretAccessKey"
	errMissingAKID                             = "missing AccessKeyID"
)

func (a *akeylessBase) TokenFromSecretRef(ctx context.Context) (string, error) {
	prov, err := GetAKeylessProvider(a.store)
	if err != nil {
		return "", err
	}

	if prov.Auth.KubernetesAuth != nil {
		kAuth := prov.Auth.KubernetesAuth
		return a.GetToken(kAuth.AccessID, "k8s", kAuth.K8sConfName, kAuth)
	}

	ke := client.ObjectKey{
		Name:      prov.Auth.SecretRef.AccessID.Name,
		Namespace: a.namespace, // default to ExternalSecret namespace
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if a.store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind {
		if prov.Auth.SecretRef.AccessID.Namespace == nil {
			return "", fmt.Errorf(errInvalidClusterStoreMissingAKIDNamespace)
		}
		ke.Namespace = *prov.Auth.SecretRef.AccessID.Namespace
	}
	accessIDSecret := v1.Secret{}
	err = a.kube.Get(ctx, ke, &accessIDSecret)
	if err != nil {
		return "", fmt.Errorf(errFetchAKIDSecret, err)
	}
	ke = client.ObjectKey{
		Name:      prov.Auth.SecretRef.AccessType.Name,
		Namespace: a.namespace, // default to ExternalSecret namespace
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if a.store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind {
		if prov.Auth.SecretRef.AccessType.Namespace == nil {
			return "", fmt.Errorf(errInvalidClusterStoreMissingSAKNamespace)
		}
		ke.Namespace = *prov.Auth.SecretRef.AccessType.Namespace
	}
	accessTypeSecret := v1.Secret{}
	err = a.kube.Get(ctx, ke, &accessTypeSecret)
	if err != nil {
		return "", fmt.Errorf(errFetchSAKSecret, err)
	}

	ke = client.ObjectKey{
		Name:      prov.Auth.SecretRef.AccessTypeParam.Name,
		Namespace: a.namespace, // default to ExternalSecret namespace
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if a.store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind {
		if prov.Auth.SecretRef.AccessType.Namespace == nil {
			return "", fmt.Errorf(errInvalidClusterStoreMissingSAKNamespace)
		}
		ke.Namespace = *prov.Auth.SecretRef.AccessType.Namespace
	}
	accessTypeParamSecret := v1.Secret{}
	err = a.kube.Get(ctx, ke, &accessTypeParamSecret)
	if err != nil {
		return "", fmt.Errorf(errFetchSAKSecret, err)
	}
	accessID := string(accessIDSecret.Data[prov.Auth.SecretRef.AccessID.Key])
	accessType := string(accessTypeSecret.Data[prov.Auth.SecretRef.AccessType.Key])
	accessTypeParam := string(accessTypeSecret.Data[prov.Auth.SecretRef.AccessTypeParam.Key])

	if accessID == "" {
		return "", fmt.Errorf(errMissingSAK)
	}
	if accessType == "" {
		return "", fmt.Errorf(errMissingAKID)
	}

	return a.GetToken(accessID, accessType, accessTypeParam, prov.Auth.KubernetesAuth)
}
