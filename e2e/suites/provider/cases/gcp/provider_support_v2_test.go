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

package gcp

import (
	"testing"

	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	gcpsmv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/gcp/v2alpha1"
)

func TestNewSecretManagerV2StaticConfig(t *testing.T) {
	t.Parallel()

	cfg := newSecretManagerV2StaticConfig("workload-ns", "gcp-config", gcpAccessConfig{
		Credentials: "service-account-json",
		ProjectID:   "project-1",
	})

	if cfg.APIVersion != gcpsmv2alpha1.GroupVersion.String() {
		t.Fatalf("unexpected apiVersion: %q", cfg.APIVersion)
	}
	if cfg.Kind != gcpsmv2alpha1.SecretManagerKind {
		t.Fatalf("unexpected kind: %q", cfg.Kind)
	}
	if cfg.Namespace != "workload-ns" || cfg.Name != "gcp-config" {
		t.Fatalf("unexpected object metadata: %s/%s", cfg.Namespace, cfg.Name)
	}
	if cfg.Spec.ProjectID != "project-1" {
		t.Fatalf("unexpected project ID: %q", cfg.Spec.ProjectID)
	}
	if cfg.Spec.Auth.SecretRef == nil {
		t.Fatal("expected static auth secretRef")
	}
	if got := cfg.Spec.Auth.SecretRef.SecretAccessKey.Name; got != staticCredentialsSecretName {
		t.Fatalf("unexpected secret ref name: %q", got)
	}
	if got := cfg.Spec.Auth.SecretRef.SecretAccessKey.Key; got != serviceAccountKey {
		t.Fatalf("unexpected secret ref key: %q", got)
	}
}

func TestNewSecretManagerV2WorkloadIdentityConfig(t *testing.T) {
	t.Parallel()

	cfg := newSecretManagerV2WorkloadIdentityConfig("provider-ns", "gcp-config", gcpAccessConfig{
		ProjectID:          "project-1",
		ServiceAccountName: "provider-sa",
		ClusterLocation:    "europe-west3",
		ClusterName:        "cluster-1",
	}, "external-secrets-system")

	if cfg.Spec.Auth.WorkloadIdentity == nil {
		t.Fatal("expected workload identity auth")
	}
	if got := cfg.Spec.Auth.WorkloadIdentity.ServiceAccountRef.Name; got != "provider-sa" {
		t.Fatalf("unexpected service account name: %q", got)
	}
	if cfg.Spec.Auth.WorkloadIdentity.ServiceAccountRef.Namespace == nil || *cfg.Spec.Auth.WorkloadIdentity.ServiceAccountRef.Namespace != "external-secrets-system" {
		t.Fatalf("unexpected service account namespace: %#v", cfg.Spec.Auth.WorkloadIdentity.ServiceAccountRef.Namespace)
	}
	if got := cfg.Spec.Auth.WorkloadIdentity.ClusterLocation; got != "europe-west3" {
		t.Fatalf("unexpected cluster location: %q", got)
	}
	if got := cfg.Spec.Auth.WorkloadIdentity.ClusterName; got != "cluster-1" {
		t.Fatalf("unexpected cluster name: %q", got)
	}
}

func TestProviderAddressInNamespace(t *testing.T) {
	t.Parallel()

	got := frameworkv2.ProviderAddressInNamespace("gcp", "gcp-provider-system")
	if got != "provider-gcp.gcp-provider-system.svc:8080" {
		t.Fatalf("unexpected address: %s", got)
	}
}

func TestNewV2ClusterProviderScenarioManifestScope(t *testing.T) {
	t.Parallel()

	got := newV2ClusterProviderScenario("workload-ns", "case", esv1.AuthenticationScopeManifestNamespace, func(string) string {
		t.Fatal("createProviderNamespace should not be called for manifest scope")
		return ""
	})

	if got.ConfigNamespace != "workload-ns" {
		t.Fatalf("unexpected config namespace: %q", got.ConfigNamespace)
	}
	if got.ProviderNamespace != "workload-ns" {
		t.Fatalf("unexpected provider namespace: %q", got.ProviderNamespace)
	}
	if got.ProviderRefNamespace != "" {
		t.Fatalf("expected empty provider ref namespace, got %q", got.ProviderRefNamespace)
	}
	if got.NamePrefix != "workload-ns-case" {
		t.Fatalf("unexpected name prefix: %q", got.NamePrefix)
	}
}

func TestNewV2ClusterProviderScenarioProviderScope(t *testing.T) {
	t.Parallel()

	var createArg string
	got := newV2ClusterProviderScenario("workload-ns", "case", esv1.AuthenticationScopeProviderNamespace, func(prefix string) string {
		createArg = prefix
		return "provider-ns"
	})

	if createArg != "case-provider" {
		t.Fatalf("unexpected provider namespace prefix: %q", createArg)
	}
	if got.ConfigNamespace != "provider-ns" {
		t.Fatalf("unexpected config namespace: %q", got.ConfigNamespace)
	}
	if got.ProviderNamespace != "provider-ns" {
		t.Fatalf("unexpected provider namespace: %q", got.ProviderNamespace)
	}
	if got.ProviderRefNamespace != "provider-ns" {
		t.Fatalf("unexpected provider ref namespace: %q", got.ProviderRefNamespace)
	}
	if got.ClusterProviderName() != "workload-ns-case-cluster-provider" {
		t.Fatalf("unexpected cluster provider name: %q", got.ClusterProviderName())
	}
}
