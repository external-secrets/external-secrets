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

// Package e2e contains end-to-end tests for the CRD provider.
//
// Tests are guarded by the "e2e" build tag and require a reachable Kubernetes
// cluster. The suite creates all required fixtures (CRDs, Namespaces,
// ServiceAccounts, RBAC, custom resource objects) at startup and removes them
// on exit. No Helm chart or ESO controller installation is needed.
//
// # Running
//
// providers/v1/crd has its own go.mod. Run from inside that directory with
// GOWORK=off so Go uses the local module rather than the workspace root:
//
//	cd providers/v1/crd
//	GOWORK=off KUBECONFIG=~/.kube/config go test -tags e2e ./test/... -v
//
// # Coverage
//
// The suite covers SecretStore and ClusterSecretStore scenarios for both
// namespaced and cluster-scoped CRD kinds, including key-format validation,
// value extraction, cross-namespace listing, and access-control verification
// (legacy SA token mode and explicit-mode impersonation).
package e2e
