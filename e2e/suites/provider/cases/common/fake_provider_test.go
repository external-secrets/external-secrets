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
	"testing"

	"github.com/external-secrets/external-secrets-e2e/framework"
)

func TestFakeProviderHelpersDoNotRequireNamespaceAtConstruction(t *testing.T) {
	t.Parallel()

	f := &framework.Framework{}

	assertDoesNotPanic(t, func() { FakeProviderSync(f) })
	assertDoesNotPanic(t, func() { FakeProviderRefresh(f) })
	assertDoesNotPanic(t, func() { FakeProviderFind(f) })
}

func assertDoesNotPanic(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	fn()
}
