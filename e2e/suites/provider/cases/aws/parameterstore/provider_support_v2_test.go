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

package aws

import (
	"strings"
	"testing"
)

func TestNewParameterStoreV2StoreProviderUsesStaticSessionTokenSelector(t *testing.T) {
	t.Parallel()

	provider := newParameterStoreV2StoreProvider("ps-static-credentials", awsV2AccessConfig{
		Region: "eu-central-1",
	}, nil)
	if provider.AWS == nil {
		t.Fatal("expected AWS provider config")
	}
	if provider.AWS.Service != "ParameterStore" {
		t.Fatalf("expected ParameterStore service, got %q", provider.AWS.Service)
	}
	if provider.AWS.Auth.SecretRef == nil || provider.AWS.Auth.SecretRef.SessionToken == nil {
		t.Fatal("expected session token selector to be configured for static auth")
	}
	if provider.AWS.Auth.SecretRef.SessionToken.Name != "ps-static-credentials" || provider.AWS.Auth.SecretRef.SessionToken.Key != "st" {
		t.Fatalf("unexpected session token selector: %+v", provider.AWS.Auth.SecretRef.SessionToken)
	}
}

func TestParameterStoreRemoteRefKeyAvoidsReservedPrefixes(t *testing.T) {
	t.Parallel()

	got := parameterStoreRemoteRefKey("aws-v2-ps-refresh-remote", "e2e-tests-eso-aws-ps-v2-6s27x")
	if !strings.HasPrefix(got, "/e2e/") {
		t.Fatalf("expected /e2e/ prefix, got %q", got)
	}
	if strings.HasPrefix(strings.TrimPrefix(got, "/"), "aws") || strings.HasPrefix(strings.TrimPrefix(got, "/"), "ssm") {
		t.Fatalf("expected non-reserved parameter prefix, got %q", got)
	}
	if !strings.Contains(got, "aws-v2-ps-refresh-remote") {
		t.Fatalf("expected remote key to retain base name, got %q", got)
	}
}
