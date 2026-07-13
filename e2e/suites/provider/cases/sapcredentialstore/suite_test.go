//go:build e2e_sapcredentialstore

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

package sapcredentialstore

import (
	"testing"

	//nolint
	. "github.com/onsi/ginkgo/v2"
	//nolint
	. "github.com/onsi/gomega"
)

// TestSAPCredentialStore is the Ginkgo test runner entry point.
func TestSAPCredentialStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SAP Credential Store E2E Suite")
}
