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

package kubernetes

import (
	"context"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	errInvalidClusterStoreMissingNamespace = "missing namespace"
	errFetchCredentials                    = "could not fetch credentials: %w"
	errMissingCredentials                  = "missing credentials: \"%s\""
	errEmptyKey                            = "key %s found but empty"
	errUnableCreateToken                   = "cannot create service account token: %q"
)

func (k *BaseClient) setAuth(ctx context.Context) error {
	err := k.setCA(ctx)
	if err != nil {
		return err
	}
	if k.store.Auth.Token != nil {
		k.BearerToken, err = k.fetchSecretKey(ctx, k.store.Auth.Token.BearerToken)
		if err != nil {
			return fmt.Errorf("could not fetch Auth.Token.BearerToken: %w", err)
		}
		return nil
	}
	if k.store.Auth.ServiceAccount != nil {
		k.BearerToken, err = k.serviceAccountToken(ctx, k.store.Auth.ServiceAccount)
		if err != nil {
			return fmt.Errorf("could not fetch Auth.ServiceAccount: %w", err)
		}
		return nil
	}
	if k.store.Auth.Cert != nil {
		return k.setClientCert(ctx)
	}
	return fmt.Errorf("no credentials provided")
}

func (k *BaseClient) setCA(ctx context.Context) error {
	if k.store.Server.CABundle != nil {
		k.CA = k.store.Server.CABundle
		return nil
	}
	if k.store.Server.CAProvider != nil {
		var ca []byte
		var err error
		switch k.store.Server.CAProvider.Type {
		case esv1beta1.CAProviderTypeConfigMap:
			keySelector := esmeta.SecretKeySelector{
				Name:      k.store.Server.CAProvider.Name,
				Namespace: k.store.Server.CAProvider.Namespace,
				Key:       k.store.Server.CAProvider.Key,
			}
			ca, err = k.fetchConfigMapKey(ctx, keySelector)
			if err != nil {
				return fmt.Errorf("unable to fetch Server.CAProvider ConfigMap: %w", err)
			}
		case esv1beta1.CAProviderTypeSecret:
			keySelector := esmeta.SecretKeySelector{
				Name:      k.store.Server.CAProvider.Name,
				Namespace: k.store.Server.CAProvider.Namespace,
				Key:       k.store.Server.CAProvider.Key,
			}
			ca, err = k.fetchSecretKey(ctx, keySelector)
			if err != nil {
				return fmt.Errorf("unable to fetch Server.CAProvider Secret: %w", err)
			}
		}
		k.CA = ca
		return nil
	}
	return fmt.Errorf("no Certificate Authority provided")
}

func (k *BaseClient) setClientCert(ctx context.Context) error {
	var err error
	k.Certificate, err = k.fetchSecretKey(ctx, k.store.Auth.Cert.ClientCert)
	if err != nil {
		return fmt.Errorf("unable to fetch client certificate: %w", err)
	}
	k.Key, err = k.fetchSecretKey(ctx, k.store.Auth.Cert.ClientKey)
	if err != nil {
		return fmt.Errorf("unable to fetch client key: %w", err)
	}
	return nil
}

func (k *BaseClient) serviceAccountToken(ctx context.Context, serviceAccountRef *esmeta.ServiceAccountSelector) ([]byte, error) {
	namespace := k.namespace
	if (k.storeKind == esv1beta1.ClusterSecretStoreKind) &&
		(serviceAccountRef.Namespace != nil) {
		namespace = *serviceAccountRef.Namespace
	}
	expirationSeconds := int64(3600)
	tr, err := k.kubeClientset.ServiceAccounts(namespace).CreateToken(ctx, serviceAccountRef.Name, &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         serviceAccountRef.Audiences,
			ExpirationSeconds: &expirationSeconds,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateToken, err)
	}
	return []byte(tr.Status.Token), nil
}

func (k *BaseClient) fetchSecretKey(ctx context.Context, key esmeta.SecretKeySelector) ([]byte, error) {
	keySecret := &corev1.Secret{}
	objectKey := types.NamespacedName{
		Name:      key.Name,
		Namespace: k.namespace,
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if k.storeKind == esv1beta1.ClusterSecretStoreKind {
		if key.Namespace == nil {
			return nil, fmt.Errorf(errInvalidClusterStoreMissingNamespace)
		}
		objectKey.Namespace = *key.Namespace
	}
	err := k.kube.Get(ctx, objectKey, keySecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchCredentials, err)
	}
	val, ok := keySecret.Data[key.Key]
	if !ok {
		return nil, fmt.Errorf(errMissingCredentials, key.Key)
	}
	if len(val) == 0 {
		return nil, fmt.Errorf(errEmptyKey, key.Key)
	}
	return val, nil
}

func (k *BaseClient) fetchConfigMapKey(ctx context.Context, key esmeta.SecretKeySelector) ([]byte, error) {
	configMap := &corev1.ConfigMap{}
	objectKey := types.NamespacedName{
		Name:      key.Name,
		Namespace: k.namespace,
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if k.storeKind == esv1beta1.ClusterSecretStoreKind {
		if key.Namespace == nil {
			return nil, fmt.Errorf(errInvalidClusterStoreMissingNamespace)
		}
		objectKey.Namespace = *key.Namespace
	}
	err := k.kube.Get(ctx, objectKey, configMap)
	if err != nil {
		return nil, fmt.Errorf(errFetchCredentials, err)
	}
	val, ok := configMap.Data[key.Key]
	if !ok {
		return nil, fmt.Errorf(errMissingCredentials, key.Key)
	}
	if val == "" {
		return nil, fmt.Errorf(errEmptyKey, key.Key)
	}
	return []byte(val), nil
}
