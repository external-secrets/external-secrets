/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aws

import (

	// nolint
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
)

// here we use the global eso instance
// that uses the service account in the default namespace
// which was created by terraform.
var _ = Describe("[awsmanaged] IRSA via referenced service account", Label("aws", "secretsmanager", "managed"), func() {
	f := framework.New("eso-aws-managed")
	prov := NewFromEnv(f)

	// nolint
	DescribeTable("sync secretsmanager secrets",
		framework.TableFuncWithExternalSecret(f,
			prov),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.SimpleDataSync, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.NestedJSONWithGJSON, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.JSONDataFromSync, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.JSONDataWithProperty, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.JSONDataWithTemplate, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.DockerJSONConfig, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.DataPropertyDockerconfigJSON, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.SSHKeySync, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.SSHKeySyncDataProperty, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.SyncWithoutTargetName, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.JSONDataWithoutTargetName, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.FindByName, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.FindByNameWithPath, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.FindByTag, awscommon.UseClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.FindByTagWithPath, awscommon.UseClusterSecretStore),
	)
})

// here we create a central eso instance in the default namespace
// that mounts the service account which was created by terraform.
var _ = Describe("[awsmanaged] with mounted IRSA", Label("aws", "secretsmanager", "managed"), func() {
	f := framework.New("eso-aws-managed")
	prov := NewFromEnv(f)

	// each test case gets its own ESO instance
	BeforeEach(func() {
		f.Install(addon.NewESO(
			addon.WithControllerClass(f.BaseName),
			addon.WithServiceAccount(prov.ServiceAccountName),
			addon.WithReleaseName(f.Namespace.Name),
			addon.WithNamespace("default"),
			addon.WithoutWebhook(),
			addon.WithoutCertController(),
		))
	})

	// nolint
	DescribeTable("sync secretsmanager secrets",
		framework.TableFuncWithExternalSecret(f,
			prov),
		framework.Compose(awscommon.WithMountedIRSA, f, common.SimpleDataSync, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.NestedJSONWithGJSON, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.JSONDataFromSync, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.JSONDataWithProperty, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.JSONDataWithTemplate, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.DockerJSONConfig, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.DataPropertyDockerconfigJSON, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.SSHKeySync, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.SSHKeySyncDataProperty, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.SyncWithoutTargetName, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.JSONDataWithoutTargetName, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.FindByName, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.FindByNameWithPath, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.FindByTag, awscommon.UseMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, common.FindByTagWithPath, awscommon.UseMountedIRSAStore),
	)
})
