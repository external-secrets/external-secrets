/*
Copyright © 2025 ESO Maintainer Team

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

/*
Package secretmanager implements the GCP Secret Manager provider for External Secrets.
It provides functionality to interact with GCP Secret Manager, handle workload identity,
and manage secret operations.
*/
package secretmanager

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// NewTokenSource creates a new OAuth2 token source for GCP Secret Manager authentication.
// It attempts to create a token source using service account credentials, workload identity,
// or workload identity federation in that order.
func NewTokenSource(ctx context.Context, auth esv1.GCPSMAuth, projectID, storeKind string, kube kclient.Client, namespace string) (oauth2.TokenSource, error) {
	ts, err := serviceAccountTokenSource(ctx, auth, storeKind, kube, namespace)
	if ts != nil || err != nil {
		return ts, err
	}
	wi, err := newWorkloadIdentity(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize workload identity: %w", err)
	}
	defer func() {
		_ = wi.Close()
	}()
	isClusterKind := storeKind == esv1.ClusterSecretStoreKind
	ts, err = wi.TokenSource(ctx, auth, isClusterKind, kube, namespace)
	if ts != nil || err != nil {
		return ts, err
	}
	wif, err := newWorkloadIdentityFederation(kube, auth.WorkloadIdentityFederation, isClusterKind, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize workload identity federation: %w", err)
	}
	ts, err = wif.TokenSource(ctx)
	if ts != nil || err != nil {
		return ts, err
	}
	return google.DefaultTokenSource(ctx, CloudPlatformRole)
}

func serviceAccountTokenSource(ctx context.Context, auth esv1.GCPSMAuth, storeKind string, kube kclient.Client, namespace string) (oauth2.TokenSource, error) {
	sr := auth.SecretRef
	if sr == nil {
		return nil, nil
	}
	credentials, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		storeKind,
		namespace,
		&auth.SecretRef.SecretAccessKey)
	if err != nil {
		return nil, err
	}
	config, err := google.JWTConfigFromJSON([]byte(credentials), CloudPlatformRole)
	if err != nil {
		return nil, fmt.Errorf(errUnableProcessJSONCredentials, err)
	}
	return config.TokenSource(ctx), nil
}
