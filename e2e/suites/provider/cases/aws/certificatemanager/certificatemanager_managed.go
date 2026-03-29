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

	// nolint
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

// here we use the global eso instance
// that uses the service account in the default namespace
// which was created by terraform.
var _ = Describe("[awsmanaged] IRSA via referenced service account", Label("aws", "certificatemanager", "managed"), Ordered, func() {
	f := framework.New("eso-aws-acm-managed")
	prov := NewFromEnv(f)

	DescribeTable("push ACM secrets",
		framework.TableFuncWithPushSecret(f, prov, nil),
		framework.Compose(withStaticAuth, f, PushSecretImport(prov, types.KeyAlgorithmRsa2048), usePushStaticAuth),
		framework.Compose(withStaticAuth, f, PushSecretImport(prov, types.KeyAlgorithmEcPrime256v1), usePushStaticAuth),
		framework.Compose(awscommon.WithReferencedIRSA, f, PushSecretWithTags(prov), usePushClusterSecretStore),
		framework.Compose(awscommon.WithReferencedIRSA, f, PushSecretDelete(prov), usePushClusterSecretStore),
	)
})

// here we create a central eso instance in the default namespace
// that mounts the service account which was created by terraform.
var _ = Describe("[awsmanaged] with mounted IRSA", Label("aws", "certificatemanager", "managed"), Ordered, func() {
	f := framework.New("eso-aws-acm-managed")
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

	DescribeTable("push ACM secrets",
		framework.TableFuncWithPushSecret(f, prov, nil),
		framework.Compose(withStaticAuth, f, PushSecretImport(prov, types.KeyAlgorithmRsa2048), usePushStaticAuth),
		framework.Compose(withStaticAuth, f, PushSecretImport(prov, types.KeyAlgorithmEcPrime256v1), usePushStaticAuth),
		framework.Compose(awscommon.WithMountedIRSA, f, PushSecretWithTags(prov), usePushMountedIRSAStore),
		framework.Compose(awscommon.WithMountedIRSA, f, PushSecretDelete(prov), usePushMountedIRSAStore),
	)
})

func usePushClusterSecretStore(tc *framework.TestCase) {
	if tc.PushSecret != nil {
		tc.PushSecret.Spec.SecretStoreRefs = []esv1alpha1.PushSecretStoreRef{
			{
				Name: awscommon.ReferencedIRSAStoreName(tc.Framework),
				Kind: esv1.ClusterSecretStoreKind,
			},
		}
	}
}

func usePushMountedIRSAStore(tc *framework.TestCase) {
	if tc.PushSecret != nil {
		tc.PushSecret.Spec.SecretStoreRefs = []esv1alpha1.PushSecretStoreRef{
			{
				Name: awscommon.MountedIRSAStoreName(tc.Framework),
				Kind: esv1.SecretStoreKind,
			},
		}
	}
}
