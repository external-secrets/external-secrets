package gcp

import (
	"reflect"
	"testing"
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
