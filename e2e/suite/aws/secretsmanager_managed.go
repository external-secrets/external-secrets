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

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/framework/addon"
	"github.com/external-secrets/external-secrets/e2e/suite/common"
)

const (
	withReferencedIRSA = "with referenced IRSA"
	withMountedIRSA    = "with mounted IRSA"
)

// here we use the global eso instance
// that uses the service account in the default namespace
// which was created by terraform.
var _ = Describe("[awsmanaged] IRSA via referenced service account", Label("aws", "secretsmanager", "managed"), func() {
	f := framework.New("eso-aws-managed")
	prov := NewFromEnv(f)

	DescribeTable("sync secrets",
		framework.TableFunc(f,
			prov),
		framework.Compose(withReferencedIRSA, f, common.SimpleDataSync, useClusterSecretStore(prov)),
		framework.Compose(withReferencedIRSA, f, common.NestedJSONWithGJSON, useClusterSecretStore(prov)),
		framework.Compose(withReferencedIRSA, f, common.JSONDataFromSync, useClusterSecretStore(prov)),
		framework.Compose(withReferencedIRSA, f, common.JSONDataWithProperty, useClusterSecretStore(prov)),
		framework.Compose(withReferencedIRSA, f, common.JSONDataWithTemplate, useClusterSecretStore(prov)),
		framework.Compose(withReferencedIRSA, f, common.DockerJSONConfig, useClusterSecretStore(prov)),
		framework.Compose(withReferencedIRSA, f, common.DataPropertyDockerconfigJSON, useClusterSecretStore(prov)),
		framework.Compose(withReferencedIRSA, f, common.SSHKeySync, useClusterSecretStore(prov)),
		framework.Compose(withReferencedIRSA, f, common.SSHKeySyncDataProperty, useClusterSecretStore(prov)),
		framework.Compose(withReferencedIRSA, f, common.SyncWithoutTargetName, useClusterSecretStore(prov)),
		framework.Compose(withReferencedIRSA, f, common.JSONDataWithoutTargetName, useClusterSecretStore(prov)),
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
		))
	})

	DescribeTable("sync secrets",
		framework.TableFunc(f,
			prov),
		framework.Compose(withMountedIRSA, f, common.SimpleDataSync, useMountedIRSAStore(prov)),
		framework.Compose(withMountedIRSA, f, common.NestedJSONWithGJSON, useMountedIRSAStore(prov)),
		framework.Compose(withMountedIRSA, f, common.JSONDataFromSync, useMountedIRSAStore(prov)),
		framework.Compose(withMountedIRSA, f, common.JSONDataWithProperty, useMountedIRSAStore(prov)),
		framework.Compose(withMountedIRSA, f, common.JSONDataWithTemplate, useMountedIRSAStore(prov)),
		framework.Compose(withMountedIRSA, f, common.DockerJSONConfig, useMountedIRSAStore(prov)),
		framework.Compose(withMountedIRSA, f, common.DataPropertyDockerconfigJSON, useMountedIRSAStore(prov)),
		framework.Compose(withMountedIRSA, f, common.SSHKeySync, useMountedIRSAStore(prov)),
		framework.Compose(withMountedIRSA, f, common.SSHKeySyncDataProperty, useMountedIRSAStore(prov)),
		framework.Compose(withMountedIRSA, f, common.SyncWithoutTargetName, useMountedIRSAStore(prov)),
		framework.Compose(withMountedIRSA, f, common.JSONDataWithoutTargetName, useMountedIRSAStore(prov)),
	)
})

func useClusterSecretStore(prov *SMProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1alpha1.ClusterSecretStoreKind
		tc.ExternalSecret.Spec.SecretStoreRef.Name = prov.ReferencedIRSAStoreName()
	}
}

func useMountedIRSAStore(prov *SMProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1alpha1.SecretStoreKind
		tc.ExternalSecret.Spec.SecretStoreRef.Name = prov.MountedIRSAStoreName()
	}
}
