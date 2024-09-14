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
	awscommon "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	withStaticAuth         = "with static auth"
	withExtID              = "with externalID"
	withSessionTags        = "with session tags"
	withReferentStaticAuth = "with static referent auth"
)

var _ = Describe("[aws] ", Label("aws", "secretsmanager"), func() {
	f := framework.New("eso-aws-sm")
	prov := NewFromEnv(f)

	DescribeTable("sync secrets",
		framework.TableFuncWithExternalSecret(f,
			prov),
		framework.Compose(withStaticAuth, f, common.SimpleDataSync, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.NestedJSONWithGJSON, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.JSONDataFromSync, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.JSONDataFromRewrite, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.JSONDataWithProperty, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.JSONDataWithTemplate, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.DockerJSONConfig, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.DataPropertyDockerconfigJSON, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.SSHKeySync, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.SSHKeySyncDataProperty, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.SyncWithoutTargetName, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.JSONDataWithoutTargetName, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.FindByName, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.FindByNameWithPath, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.FindByTag, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.FindByTagWithPath, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.SyncV1Alpha1, useStaticAuth),
		framework.Compose(withStaticAuth, f, common.DeletionPolicyDelete, useStaticAuth),

		// referent auth
		framework.Compose(withStaticAuth, f, common.SimpleDataSync, useReferentStaticAuth),

		// test assume role with external-id and session tags
		framework.Compose(withExtID, f, SimpleSyncWithNamespaceTags(prov), useExtIDAuth),
		framework.Compose(withSessionTags, f, SimpleSyncWithNamespaceTags(prov), useSessionTagsAuth),
	)
})

func useStaticAuth(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = awscommon.StaticStoreName
	if tc.ExternalSecretV1Alpha1 != nil {
		tc.ExternalSecretV1Alpha1.Spec.SecretStoreRef.Name = awscommon.StaticStoreName
	}
}

func useExtIDAuth(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = awscommon.ExternalIDStoreName
	if tc.ExternalSecretV1Alpha1 != nil {
		tc.ExternalSecretV1Alpha1.Spec.SecretStoreRef.Name = awscommon.ExternalIDStoreName
	}
}

func useSessionTagsAuth(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = awscommon.SessionTagsStoreName
	if tc.ExternalSecretV1Alpha1 != nil {
		tc.ExternalSecretV1Alpha1.Spec.SecretStoreRef.Name = awscommon.SessionTagsStoreName
	}
}

func useReferentStaticAuth(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = awscommon.ReferentSecretStoreName(tc.Framework)
	tc.ExternalSecret.Spec.SecretStoreRef.Kind = esapi.ClusterSecretStoreKind
}
