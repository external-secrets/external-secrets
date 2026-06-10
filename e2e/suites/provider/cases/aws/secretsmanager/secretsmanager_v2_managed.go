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
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("[awsmanaged] v2 IRSA via referenced service account", Label("aws", "secretsmanager", "managed", "v2"), Ordered, func() {
	f := framework.New("eso-aws-managed-v2-ref")
	prov := NewProviderV2(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
		skipIfAWSManagedIRSAEnvMissing(prov.access)
	})

	DescribeTable("sync secretsmanager secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.SimpleDataSync, useV2ReferencedIRSA(prov)),
		framework.Compose(awscommon.WithReferencedIRSA, f, common.FindByName, useV2ReferencedIRSA(prov)),
	)
})

var _ = Describe("[awsmanaged] v2 with mounted IRSA", Label("aws", "secretsmanager", "managed", "v2"), Ordered, func() {
	f := framework.New("eso-aws-managed-v2-mounted")
	prov := NewProviderV2(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
		skipIfAWSManagedIRSAEnvMissing(prov.access)

		f.Install(addon.NewESO(
			addon.WithControllerClass(f.BaseName+"-mounted"),
			addon.WithReleaseName(f.Namespace.Name),
			addon.WithNamespace(prov.access.SANamespace),
			addon.WithoutWebhook(),
			addon.WithoutCertController(),
			addon.WithV2AWSProvider(),
			addon.WithV2ProviderServiceAccount("aws", prov.access.SAName),
		))
	})

	DescribeTable("sync secretsmanager secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		framework.Compose(awscommon.WithMountedIRSA, f, common.SimpleDataSync, useV2MountedIRSA(prov)),
		framework.Compose(awscommon.WithMountedIRSA, f, common.FindByName, useV2MountedIRSA(prov)),
	)
})

func useV2ReferencedIRSA(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareNamespacedProvider(awsAuthProfileReferencedIRSA)
	}
}

func useV2MountedIRSA(prov *ProviderV2) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.Prepare = prov.prepareNamespacedProviderAtAddress(
			awsAuthProfileMountedIRSA,
			frameworkv2.ProviderAddressInNamespace("aws", prov.access.SANamespace),
		)
	}
}
