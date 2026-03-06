/*
Copyright © 2025 ESO Maintainer Team

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

package auth

import (
	"context"
	"os"
	"testing"

	authenticationv1 "k8s.io/api/authentication/v1"
)

// testSATokenGenerator is a mock saTokenGenerator for testing.
type testSATokenGenerator struct{}

func (m *testSATokenGenerator) Generate(_ context.Context, _ []string, _, _ string) (*authenticationv1.TokenRequest, error) {
	return &authenticationv1.TokenRequest{
		Status: authenticationv1.TokenRequestStatus{
			Token: "mock-k8s-token",
		},
	}, nil
}

// TestMain sets up the test environment by replacing the saTokenGenerator factory
// with a mock implementation to avoid requiring a real Kubernetes cluster.
//
// Note: We don't restore the original newSATokenGeneratorFunc because:
// 1. TestMain runs once for the entire package
// 2. os.Exit terminates the process after tests complete
// 3. No other code runs after this that would need the original function
//
// For individual tests that need custom mocks, use t.Cleanup to restore:
//
//	original := newSATokenGeneratorFunc
//	t.Cleanup(func() { newSATokenGeneratorFunc = original })
//	newSATokenGeneratorFunc = customMock
func TestMain(m *testing.M) {
	// Replace the factory function with a mock for all tests
	newSATokenGeneratorFunc = func() (saTokenGenerator, error) {
		return &testSATokenGenerator{}, nil
	}

	os.Exit(m.Run())
}
