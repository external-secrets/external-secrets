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

	"golang.org/x/oauth2"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	gcpauth "github.com/external-secrets/external-secrets/runtime/gcp/auth"
)

// NewTokenSource creates a new OAuth2 token source for GCP Secret Manager authentication.
// This is a wrapper around the shared runtime/gcp/auth implementation.
func NewTokenSource(ctx context.Context, auth esv1.GCPSMAuth, projectID, storeKind string, kube kclient.Client, namespace string) (oauth2.TokenSource, error) {
	return gcpauth.NewTokenSource(ctx, auth, projectID, storeKind, kube, namespace)
}
