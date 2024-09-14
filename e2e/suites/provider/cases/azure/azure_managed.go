/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/
package azure

import (

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	// . "github.com/onsi/gomega"
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	withPodID = "sync secrets with workload identity"
)

// Deploys eso to the default namespace
// that uses the service account provisioned by terraform
// to test workload-identity authentication.
var _ = Describe("[azuremanaged] with pod identity", Label("azure", "keyvault", "managed", "workload-identity"), func() {
	f := framework.New("eso-azuremanaged")
	prov := newFromWorkloadIdentity(f)

	// each test case gets its own ESO instance
	BeforeEach(func() {
		f.Install(addon.NewESO(
			addon.WithControllerClass(f.BaseName),
			addon.WithReleaseName(f.Namespace.Name),
			addon.WithNamespace("external-secrets-operator"),
			addon.WithServiceAccount("external-secrets-operator"),
			addon.WithoutWebhook(),
			addon.WithoutCertController(),
		))
	})

	DescribeTable("sync secrets",
		framework.TableFuncWithExternalSecret(f,
			prov),
		// uses pod id
		framework.Compose(withPodID, f, common.SimpleDataSync, usePodIDESReference),
		framework.Compose(withPodID, f, common.JSONDataWithProperty, usePodIDESReference),
		framework.Compose(withPodID, f, common.JSONDataFromSync, usePodIDESReference),
		framework.Compose(withPodID, f, common.NestedJSONWithGJSON, usePodIDESReference),
		framework.Compose(withPodID, f, common.JSONDataWithTemplate, usePodIDESReference),
		framework.Compose(withPodID, f, common.DockerJSONConfig, usePodIDESReference),
		framework.Compose(withPodID, f, common.DataPropertyDockerconfigJSON, usePodIDESReference),
		framework.Compose(withPodID, f, common.SSHKeySync, usePodIDESReference),
		framework.Compose(withPodID, f, common.SSHKeySyncDataProperty, usePodIDESReference),
		framework.Compose(withPodID, f, common.SyncWithoutTargetName, usePodIDESReference),
		framework.Compose(withPodID, f, common.JSONDataWithoutTargetName, usePodIDESReference),
	)
})

func usePodIDESReference(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1beta1.ClusterSecretStoreKind
}
