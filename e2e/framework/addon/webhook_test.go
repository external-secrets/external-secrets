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

package addon

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestExternalSecretWebhookURLUsesReleaseName(t *testing.T) {
	t.Helper()

	got := externalSecretWebhookURL("default", "eso")
	want := "https://eso-external-secrets-webhook.default.svc.cluster.local/validate-external-secrets-io-v1-externalsecret"
	if got != want {
		t.Fatalf("unexpected webhook URL: got %q want %q", got, want)
	}
}

func TestExternalSecretWebhookURLUsesHelmFullnameWhenReleaseContainsChartName(t *testing.T) {
	t.Helper()

	got := externalSecretWebhookURL("external-secrets-system", "external-secrets")
	want := "https://external-secrets-webhook.external-secrets-system.svc.cluster.local/validate-external-secrets-io-v1-externalsecret"
	if got != want {
		t.Fatalf("unexpected webhook URL: got %q want %q", got, want)
	}
}

func TestWaitForExternalSecretWebhookReadyRetriesUntilOK(t *testing.T) {
	t.Helper()

	originalURL := externalSecretWebhookURL
	originalPollInterval := webhookReadyPollInterval
	originalTimeout := webhookReadyTimeout
	originalContext := webhookReadyContext
	t.Cleanup(func() {
		externalSecretWebhookURL = originalURL
		webhookReadyPollInterval = originalPollInterval
		webhookReadyTimeout = originalTimeout
		webhookReadyContext = originalContext
	})

	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	externalSecretWebhookURL = func(string, string) string { return server.URL }
	webhookReadyPollInterval = 10 * time.Millisecond
	webhookReadyTimeout = time.Second
	webhookReadyContext = context.Background

	if err := waitForExternalSecretWebhookReady("external-secrets-system", "external-secrets"); err != nil {
		t.Fatalf("waitForExternalSecretWebhookReady returned error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 webhook attempts, got %d", attempts)
	}
}

func TestWaitForExternalSecretWebhookReadyTimesOut(t *testing.T) {
	t.Helper()

	originalURL := externalSecretWebhookURL
	originalPollInterval := webhookReadyPollInterval
	originalTimeout := webhookReadyTimeout
	originalContext := webhookReadyContext
	t.Cleanup(func() {
		externalSecretWebhookURL = originalURL
		webhookReadyPollInterval = originalPollInterval
		webhookReadyTimeout = originalTimeout
		webhookReadyContext = originalContext
	})

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	externalSecretWebhookURL = func(string, string) string { return server.URL }
	webhookReadyPollInterval = 10 * time.Millisecond
	webhookReadyTimeout = 50 * time.Millisecond
	webhookReadyContext = context.Background

	if err := waitForExternalSecretWebhookReady("external-secrets-system", "external-secrets"); err == nil {
		t.Fatal("expected waitForExternalSecretWebhookReady to time out")
	}
}
