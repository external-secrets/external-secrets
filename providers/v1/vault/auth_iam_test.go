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

package vault

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestSetAWSCredentialEnvVars(t *testing.T) {
	creds := aws.Credentials{
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		SessionToken:    "test-session-token",
	}

	t.Run("sets the credential environment variables", func(t *testing.T) {
		restore, err := setAWSCredentialEnvVars(creds)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer restore()

		for key, want := range map[string]string{
			"AWS_ACCESS_KEY_ID":     creds.AccessKeyID,
			"AWS_SECRET_ACCESS_KEY": creds.SecretAccessKey,
			"AWS_SESSION_TOKEN":     creds.SessionToken,
		} {
			if got := os.Getenv(key); got != want {
				t.Errorf("expected %s to be %q, got %q", key, want, got)
			}
		}
	})

	t.Run("restores previously set values", func(t *testing.T) {
		t.Setenv("AWS_ACCESS_KEY_ID", "previous-access")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "previous-secret")
		t.Setenv("AWS_SESSION_TOKEN", "previous-token")

		restore, err := setAWSCredentialEnvVars(creds)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		restore()

		for key, want := range map[string]string{
			"AWS_ACCESS_KEY_ID":     "previous-access",
			"AWS_SECRET_ACCESS_KEY": "previous-secret",
			"AWS_SESSION_TOKEN":     "previous-token",
		} {
			if got := os.Getenv(key); got != want {
				t.Errorf("expected %s to be restored to %q, got %q", key, want, got)
			}
		}
	})

	t.Run("unsets previously unset values on restore", func(t *testing.T) {
		restore, err := setAWSCredentialEnvVars(creds)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, ok := os.LookupEnv("AWS_SESSION_TOKEN"); !ok {
			t.Errorf("expected AWS_SESSION_TOKEN to be set before restore")
		}

		restore()

		if _, ok := os.LookupEnv("AWS_SESSION_TOKEN"); ok {
			t.Errorf("expected AWS_SESSION_TOKEN to be unset after restore")
		}
	})
}
