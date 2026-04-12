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

package kubernetes

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var _ = Describe("[kubernetes] v2 capabilities", Label("kubernetes", "v2", "capabilities"), func() {
	f := framework.New("eso-kubernetes-v2-capabilities")
	NewProvider(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
	})

	It("reports READ_WRITE capabilities for the Kubernetes provider", func() {
		frameworkv2.WaitForProviderConnectionReady(f, f.Namespace.Name, f.Namespace.Name, defaultV2WaitTimeout)
		frameworkv2.VerifyProviderConnectionCapabilities(f, f.Namespace.Name, f.Namespace.Name, esv1.ProviderReadWrite)
	})
})
