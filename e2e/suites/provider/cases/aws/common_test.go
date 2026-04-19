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

package common

import (
	"strings"
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestCredentialsSecretName(t *testing.T) {
	t.Parallel()

	if got := CredentialsSecretName("aws-config"); got != "aws-config-credentials" {
		t.Fatalf("unexpected credentials secret name: %q", got)
	}
}

func TestStaticCredentialsSecretDataPreservesSessionToken(t *testing.T) {
	t.Parallel()

	got := StaticCredentialsSecretData("kid", "sak", "st")
	if got[StaticAccessKeyIDKey] != "kid" {
		t.Fatalf("unexpected access key id: %q", got[StaticAccessKeyIDKey])
	}
	if got[StaticSecretAccessKeyKey] != "sak" {
		t.Fatalf("unexpected secret access key: %q", got[StaticSecretAccessKeyKey])
	}
	if got[StaticSessionTokenKey] != "st" {
		t.Fatalf("unexpected session token: %q", got[StaticSessionTokenKey])
	}
}

func TestProviderConfigNamespaceForManifestScope(t *testing.T) {
	t.Parallel()

	if got := ProviderConfigNamespace(esv1.AuthenticationScopeManifestNamespace, "provider-ns", "workload-ns"); got != "workload-ns" {
		t.Fatalf("expected workload namespace, got %q", got)
	}
}

func TestProviderConfigNamespaceForProviderScope(t *testing.T) {
	t.Parallel()

	if got := ProviderConfigNamespace(esv1.AuthenticationScopeProviderNamespace, "provider-ns", "workload-ns"); got != "provider-ns" {
		t.Fatalf("expected provider namespace, got %q", got)
	}
}

func TestProviderReferenceNamespaceForManifestScope(t *testing.T) {
	t.Parallel()

	if got := ProviderReferenceNamespace(esv1.AuthenticationScopeManifestNamespace, "provider-ns"); got != "" {
		t.Fatalf("expected empty provider reference namespace, got %q", got)
	}
}

func TestProviderReferenceNamespaceForProviderScope(t *testing.T) {
	t.Parallel()

	if got := ProviderReferenceNamespace(esv1.AuthenticationScopeProviderNamespace, "provider-ns"); got != "provider-ns" {
		t.Fatalf("expected provider namespace, got %q", got)
	}
}

func TestNewV2ClusterProviderScenarioManifestScope(t *testing.T) {
	t.Parallel()

	called := false
	got := NewV2ClusterProviderScenario("workload-ns", "case", esv1.AuthenticationScopeManifestNamespace, func(prefix string) string {
		called = true
		return prefix + "-provider"
	})
	if called {
		t.Fatal("expected provider namespace factory to be unused for manifest scope")
	}
	if got.ConfigName != "case-config" {
		t.Fatalf("unexpected config name: %q", got.ConfigName)
	}
	if got.ConfigNamespace != "workload-ns" {
		t.Fatalf("unexpected config namespace: %q", got.ConfigNamespace)
	}
	if got.ProviderNamespace != "workload-ns" {
		t.Fatalf("unexpected provider namespace: %q", got.ProviderNamespace)
	}
	if got.ProviderRefNamespace != "" {
		t.Fatalf("expected empty provider reference namespace, got %q", got.ProviderRefNamespace)
	}
	if got.WorkloadNamespace != "workload-ns" {
		t.Fatalf("unexpected workload namespace: %q", got.WorkloadNamespace)
	}
	if got.NamePrefix != "workload-ns-case" {
		t.Fatalf("unexpected name prefix: %q", got.NamePrefix)
	}
}

func TestNewV2ClusterProviderScenarioProviderScope(t *testing.T) {
	t.Parallel()

	var gotPrefix string
	got := NewV2ClusterProviderScenario("workload-ns", "case", esv1.AuthenticationScopeProviderNamespace, func(prefix string) string {
		gotPrefix = prefix
		return "provider-ns"
	})
	if gotPrefix != "case-provider" {
		t.Fatalf("unexpected provider namespace prefix: %q", gotPrefix)
	}
	if got.ConfigNamespace != "provider-ns" {
		t.Fatalf("unexpected config namespace: %q", got.ConfigNamespace)
	}
	if got.ProviderNamespace != "provider-ns" {
		t.Fatalf("unexpected provider namespace: %q", got.ProviderNamespace)
	}
	if got.ProviderRefNamespace != "provider-ns" {
		t.Fatalf("unexpected provider reference namespace: %q", got.ProviderRefNamespace)
	}
}

func TestPushSecretMetadataWithRemoteNamespace(t *testing.T) {
	t.Parallel()

	got := PushSecretMetadataWithRemoteNamespace("target-ns")
	if got == nil {
		t.Fatal("expected metadata payload")
	}
	raw := string(got.Raw)
	if !strings.Contains(raw, `"remoteNamespace":"target-ns"`) {
		t.Fatalf("expected remote namespace in metadata, got %q", raw)
	}
}
