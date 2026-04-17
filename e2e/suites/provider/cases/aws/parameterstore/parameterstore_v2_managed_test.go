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
	"errors"
	"testing"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestConfigureV2ReferencedIRSAStoreRefUsesClusterProvider(t *testing.T) {
	t.Parallel()

	tc := &framework.TestCase{
		ExternalSecret: &esv1.ExternalSecret{
			Spec: esv1.ExternalSecretSpec{
				SecretStoreRef: esv1.SecretStoreRef{
					Name: "placeholder",
					Kind: esv1.ProviderKindStr,
				},
			},
		},
	}

	configureV2ReferencedIRSAStoreRef(tc, "aws-irsa-cluster-provider")

	if got := tc.ExternalSecret.Spec.SecretStoreRef.Kind; got != esv1.ClusterProviderKindStr {
		t.Fatalf("expected cluster provider kind %q, got %q", esv1.ClusterProviderKindStr, got)
	}
	if got := tc.ExternalSecret.Spec.SecretStoreRef.Name; got != "aws-irsa-cluster-provider" {
		t.Fatalf("expected cluster provider ref %q, got %q", "aws-irsa-cluster-provider", got)
	}
}

func TestWrapCleanupWithClusterProviderDeleteRunsPreviousCleanupFirst(t *testing.T) {
	t.Parallel()

	var calls []string
	cleanup := wrapCleanupWithClusterProviderDelete(
		func() { calls = append(calls, "previous") },
		func() error {
			calls = append(calls, "delete")
			return nil
		},
	)

	cleanup()

	if len(calls) != 2 {
		t.Fatalf("expected 2 cleanup calls, got %d", len(calls))
	}
	if calls[0] != "previous" || calls[1] != "delete" {
		t.Fatalf("expected call order [previous delete], got %v", calls)
	}
}

func TestWrapCleanupWithClusterProviderDeleteRunsDeleteWithoutPreviousCleanup(t *testing.T) {
	t.Parallel()

	var deleteCalled bool
	cleanup := wrapCleanupWithClusterProviderDelete(nil, func() error {
		deleteCalled = true
		return nil
	})

	cleanup()

	if !deleteCalled {
		t.Fatal("expected delete callback to run")
	}
}

func TestWrapCleanupWithClusterProviderDeleteIgnoresNotFoundDeleteError(t *testing.T) {
	t.Parallel()

	cleanup := wrapCleanupWithClusterProviderDelete(nil, func() error {
		return errClusterProviderNotFoundForCleanup
	})

	cleanup()
}

func TestWrapCleanupWithClusterProviderDeletePanicsOnUnexpectedDeleteError(t *testing.T) {
	t.Parallel()

	cleanup := wrapCleanupWithClusterProviderDelete(nil, func() error {
		return errors.New("boom")
	})

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unexpected delete error")
		}
	}()

	cleanup()
}
