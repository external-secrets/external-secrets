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
