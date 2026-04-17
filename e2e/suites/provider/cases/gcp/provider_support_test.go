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
	"os"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
)

func TestGCPAccessConfigMissingStaticEnv(t *testing.T) {
	t.Parallel()

	cfg := gcpAccessConfig{}
	want := []string{
		"GCP_SERVICE_ACCOUNT_KEY",
		"GCP_FED_PROJECT_ID",
	}

	if got := cfg.missingStaticEnv(); !reflect.DeepEqual(got, want) {
		t.Fatalf("missingStaticEnv() = %v, want %v", got, want)
	}
}

func TestGCPAccessConfigMissingManagedEnv(t *testing.T) {
	t.Parallel()

	cfg := gcpAccessConfig{}
	want := []string{
		"GCP_KSA_NAME",
		"GCP_FED_REGION",
		"GCP_GKE_CLUSTER",
	}

	if got := cfg.missingManagedEnv(); !reflect.DeepEqual(got, want) {
		t.Fatalf("missingManagedEnv() = %v, want %v", got, want)
	}
}

func TestGCPAccessConfigMissingEnvComplete(t *testing.T) {
	t.Parallel()

	cfg := gcpAccessConfig{
		Credentials:        "creds",
		ProjectID:          "project",
		ServiceAccountName: "ksa",
		ClusterLocation:    "region",
		ClusterName:        "cluster",
	}

	if got := cfg.missingStaticEnv(); len(got) != 0 {
		t.Fatalf("missingStaticEnv() = %v, want none", got)
	}
	if got := cfg.missingManagedEnv(); len(got) != 0 {
		t.Fatalf("missingManagedEnv() = %v, want none", got)
	}
}

func TestProviderV2NamespacedSuiteDoesNotIncludeWorkloadIdentity(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("provider_v2.go")
	if err != nil {
		t.Fatalf("read provider_v2.go: %v", err)
	}

	for _, forbidden := range []string{
		"withWorkloadIdentity",
		"useV2WorkloadIdentity(",
		"gcp-v2-wi-",
	} {
		if strings.Contains(string(content), forbidden) {
			t.Fatalf("non-managed v2 suite must not include workload identity coverage, found %q", forbidden)
		}
	}
}

func TestProviderV2RefreshSuiteOverridesDefaultRemoteMutation(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("provider_v2.go")
	if err != nil {
		t.Fatalf("read provider_v2.go: %v", err)
	}

	for _, required := range []string{
		"UpdateRemoteSecret:",
		"prov.UpdateSecret(",
	} {
		if !strings.Contains(string(content), required) {
			t.Fatalf("expected GCP v2 refresh suite to include %q", required)
		}
	}
}

func TestProviderV2FindSuiteUsesScopedRemoteSecretNames(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("provider_v2.go")
	if err != nil {
		t.Fatalf("read provider_v2.go: %v", err)
	}

	for _, required := range []string{
		`f.MakeRemoteRefKey("gcp-v2-find-one")`,
		`f.MakeRemoteRefKey("gcp-v2-find-two")`,
		`f.MakeRemoteRefKey("gcp-v2-ignore")`,
	} {
		if !strings.Contains(string(content), required) {
			t.Fatalf("expected GCP v2 find suite to include %q", required)
		}
	}
}

func TestConfigureGCPRemoteRefKeyKeepsBaseWithoutNamespace(t *testing.T) {
	t.Parallel()

	f := &framework.Framework{
		MakeRemoteRefKey: func(base string) string { return base },
	}

	configureGCPRemoteRefKey(f)

	if got := f.MakeRemoteRefKey("remote-key"); got != "remote-key" {
		t.Fatalf("MakeRemoteRefKey() = %q, want %q", got, "remote-key")
	}
}

func TestConfigureGCPRemoteRefKeyAppendsNamespaceSuffix(t *testing.T) {
	t.Parallel()

	f := &framework.Framework{
		Namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ns-123456789",
			},
		},
	}

	configureGCPRemoteRefKey(f)

	if got := f.MakeRemoteRefKey("remote-key"); got != "remote-key-23456789" {
		t.Fatalf("MakeRemoteRefKey() = %q, want %q", got, "remote-key-23456789")
	}
}
