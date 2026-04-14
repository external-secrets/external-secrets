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

package fake

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestUpsertFakeProviderDataReplacesMatchingEntry(t *testing.T) {
	input := []esv1.FakeProviderData{
		{Key: "other", Value: "untouched"},
		{Key: "remote", Value: "old", Version: "v1"},
	}

	got := upsertFakeProviderData(input, esv1.FakeProviderData{
		Key:     "remote",
		Value:   "new",
		Version: "v1",
	})

	want := []esv1.FakeProviderData{
		{Key: "other", Value: "untouched"},
		{Key: "remote", Value: "new", Version: "v1"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("upsertFakeProviderData mismatch (-want +got):\n%s", diff)
	}
}

func TestRemoveFakeProviderDataRemovesOnlyExactMatch(t *testing.T) {
	input := []esv1.FakeProviderData{
		{Key: "remote", Value: "keep-version", Version: "v2"},
		{Key: "remote", Value: "drop-version", Version: "v1"},
		{Key: "other", Value: "keep"},
	}

	got := removeFakeProviderData(input, "remote", "v1")

	want := []esv1.FakeProviderData{
		{Key: "remote", Value: "keep-version", Version: "v2"},
		{Key: "other", Value: "keep"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("removeFakeProviderData mismatch (-want +got):\n%s", diff)
	}
}

func TestProviderReferenceNamespace(t *testing.T) {
	if got := providerReferenceNamespace(esv1.AuthenticationScopeManifestNamespace, "provider-ns"); got != "" {
		t.Fatalf("expected empty providerRef namespace for manifest scope, got %q", got)
	}
	if got := providerReferenceNamespace(esv1.AuthenticationScopeProviderNamespace, "provider-ns"); got != "provider-ns" {
		t.Fatalf("expected providerRef namespace for provider scope, got %q", got)
	}
}

func TestFakeConfigNamespaceForAuthScope(t *testing.T) {
	if got := fakeConfigNamespaceForAuthScope(esv1.AuthenticationScopeManifestNamespace, "manifest-ns", "provider-ns"); got != "manifest-ns" {
		t.Fatalf("expected manifest namespace for manifest scope, got %q", got)
	}
	if got := fakeConfigNamespaceForAuthScope(esv1.AuthenticationScopeProviderNamespace, "manifest-ns", "provider-ns"); got != "provider-ns" {
		t.Fatalf("expected provider namespace for provider scope, got %q", got)
	}
}
