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
package secretmanager

import (
	"context"
	"fmt"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewTokenSource(ctx context.Context, auth esv1beta1.GCPSMAuth, projectID string, isClusterKind bool, kube kclient.Client, namespace string) (oauth2.TokenSource, error) {
	ts, err := serviceAccountTokenSource(ctx, auth, isClusterKind, kube, namespace)
	if ts != nil || err != nil {
		return ts, err
	}
	wi, err := newWorkloadIdentity(ctx, projectID)
	if err != nil {
		useMu.Unlock()
		return nil, fmt.Errorf("unable to initialize workload identity")
	}
	ts, err = wi.TokenSource(ctx, auth, isClusterKind, kube, namespace)
	if ts != nil || err != nil {
		return ts, err
	}
	return google.DefaultTokenSource(ctx, CloudPlatformRole)
}

func serviceAccountTokenSource(ctx context.Context, auth esv1beta1.GCPSMAuth, isClusterKind bool, kube kclient.Client, namespace string) (oauth2.TokenSource, error) {
	sr := auth.SecretRef
	if sr == nil {
		return nil, nil
	}
	credentialsSecret := &v1.Secret{}
	credentialsSecretName := sr.SecretAccessKey.Name
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if isClusterKind {
		if credentialsSecretName != "" && sr.SecretAccessKey.Namespace == nil {
			return nil, fmt.Errorf(errInvalidClusterStoreMissingSAKNamespace)
		} else if credentialsSecretName != "" {
			objectKey.Namespace = *sr.SecretAccessKey.Namespace
		}
	}
	err := kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchSAKSecret, err)
	}
	credentials := credentialsSecret.Data[sr.SecretAccessKey.Key]
	if (credentials == nil) || (len(credentials) == 0) {
		return nil, fmt.Errorf(errMissingSAK)
	}
	config, err := google.JWTConfigFromJSON(credentials, CloudPlatformRole)
	if err != nil {
		return nil, fmt.Errorf(errUnableProcessJSONCredentials, err)
	}
	return config.TokenSource(ctx), nil
}
