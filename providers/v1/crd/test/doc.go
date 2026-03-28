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
// cluster. Point KUBECONFIG at the desired cluster before running:
//
//	go test -tags e2e ./providers/v1/crd/test/... -v
//
// The suite creates all required fixtures (CRDs, Namespace, ServiceAccount,
// RBAC, custom resource objects) at startup and removes them on exit.
// A pre-existing cluster is the only prerequisite – no Helm chart or ESO
// controller installation is needed.
package e2e
