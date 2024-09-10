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
	"errors"
	"fmt"

	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errFetchAccessIDSecret        = "could not fetch accessID secret: %w"
	errFetchAccessTypeSecret      = "could not fetch AccessType secret: %w"
	errFetchAccessTypeParamSecret = "could not fetch AccessTypeParam secret: %w"
	errMissingSAK                 = "missing SecretAccessKey"
	errMissingAKID                = "missing AccessKeyID"
)

func (a *akeylessBase) TokenFromSecretRef(ctx context.Context) (string, error) {
	prov := a.store
	if prov.Auth.KubernetesAuth != nil {
		auth := prov.Auth.KubernetesAuth
		return a.GetToken(auth.AccessID, "k8s", auth.K8sConfName, auth)
	}

	accessID, err := resolvers.SecretKeyRef(
		ctx,
		a.kube,
		a.storeKind,
		a.namespace,
		&prov.Auth.SecretRef.AccessID,
	)
	if err != nil {
		return "", fmt.Errorf(errFetchAccessIDSecret, err)
	}
	accessType, err := resolvers.SecretKeyRef(
		ctx,
		a.kube,
		a.storeKind,
		a.namespace,
		&prov.Auth.SecretRef.AccessType,
	)
	if err != nil {
		return "", fmt.Errorf(errFetchAccessTypeSecret, err)
	}
	accessTypeParam, err := resolvers.SecretKeyRef(
		ctx,
		a.kube,
		a.storeKind,
		a.namespace,
		&prov.Auth.SecretRef.AccessTypeParam,
	)
	if err != nil {
		return "", fmt.Errorf(errFetchAccessTypeParamSecret, err)
	}

	if accessID == "" {
		return "", errors.New(errMissingSAK)
	}
	if accessType == "" {
		return "", errors.New(errMissingAKID)
	}

	return a.GetToken(accessID, accessType, accessTypeParam, prov.Auth.KubernetesAuth)
}
