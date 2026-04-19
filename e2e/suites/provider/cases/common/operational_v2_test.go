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

import "testing"

func TestOperationalRuntimeSupportsDisruptionLifecycle(t *testing.T) {
	runtimeWithoutHooks := &OperationalRuntime{}
	if runtimeWithoutHooks.SupportsDisruptionLifecycle() {
		t.Fatalf("expected false when all hooks are nil")
	}

	runtimeWithBreakOnly := &OperationalRuntime{
		MakeUnavailable: func() {},
	}
	if runtimeWithBreakOnly.SupportsDisruptionLifecycle() {
		t.Fatalf("expected false when RestoreAvailability is nil")
	}

	runtimeWithRestoreOnly := &OperationalRuntime{
		RestoreAvailability: func() {},
	}
	if runtimeWithRestoreOnly.SupportsDisruptionLifecycle() {
		t.Fatalf("expected false when MakeUnavailable is nil")
	}

	runtimeWithBoth := &OperationalRuntime{
		MakeUnavailable:     func() {},
		RestoreAvailability: func() {},
	}
	if !runtimeWithBoth.SupportsDisruptionLifecycle() {
		t.Fatalf("expected true when both hooks exist")
	}
}

func TestOperationalRuntimeSupportsRestart(t *testing.T) {
	runtime := &OperationalRuntime{}
	if runtime.SupportsRestart() {
		t.Fatalf("expected false when RestartBackend is nil")
	}

	runtime.RestartBackend = func() {}
	if !runtime.SupportsRestart() {
		t.Fatalf("expected true when RestartBackend is present")
	}
}
