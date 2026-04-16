/*
Copyright © The ESO Authors

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

package kubernetes

import (
	"testing"

	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
)

func TestKubernetesBackendTargetUsesProviderNamespaceAndSelector(t *testing.T) {
	target := kubernetesBackendTarget()
	if target.Namespace != frameworkv2.ProviderNamespace {
		t.Fatalf("expected provider namespace %q, got %q", frameworkv2.ProviderNamespace, target.Namespace)
	}
	if target.PodLabelSelector != "app.kubernetes.io/name=external-secrets-provider-kubernetes" {
		t.Fatalf("unexpected selector %q", target.PodLabelSelector)
	}
}
