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

package infisical

import (
	//nolint
	. "github.com/onsi/ginkgo/v2"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/runtime/testing/fake"
)

const (
	withUniversalAuth        = "with universal auth"
	withUniversalAuthCluster = "with universal auth and cluster store"
)

// The suite seeds read-path secrets through the SDK and exercises the push
// path through the provider's PushSecret implementation.
// FindByTag is excluded because the provider does not implement tag lookup
// (it returns "find by tags not supported"), and FindByNameWithPath is
// excluded because the provider matches ref.Path as a prefix of the absolute
// Infisical secret path, which a bare namespace name never satisfies.
var _ = Describe("[infisical]", Label("infisical"), Ordered, func() {
	f := framework.New("infisical")
	infisical := addon.NewInfisical()
	prov := newInfisicalProvider(f, infisical)
	fakeSecretClient := fake.New()

	BeforeAll(func() {
		addon.InstallGlobalAddon(infisical)
	})

	DescribeTable("sync secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		framework.Compose(withUniversalAuth, f, common.SimpleDataSync, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.SyncWithoutTargetName, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.JSONDataWithProperty, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.JSONDataWithoutTargetName, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.JSONDataWithTemplate, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.JSONDataWithTemplateFromLiteral, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.TemplateFromConfigmaps, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.JSONDataFromSync, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.JSONDataFromRewrite, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.NestedJSONWithGJSON, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.FindByName, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.FindByNameAndRewrite, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.DataPropertyDockerconfigJSON, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.SSHKeySync, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.SSHKeySyncDataProperty, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.DecodingPolicySync, useUniversalAuth(prov)),
		framework.Compose(withUniversalAuth, f, common.StatusNotUpdatedAfterSuccessfulSync, useUniversalAuth(prov)),
		// DeletionPolicyDelete depends on GetSecret returning NoSecretErr for a
		// missing key, which the provider now does (see issue #6413).
		framework.Compose(withUniversalAuth, f, common.DeletionPolicyDelete, useUniversalAuth(prov)),
		// one case through a ClusterSecretStore to cover the cluster-scoped path
		framework.Compose(withUniversalAuthCluster, f, common.JSONDataFromSync, useUniversalAuthClusterStore(prov)),
	)

	DescribeTable("push secrets",
		framework.TableFuncWithPushSecret(f, prov, fakeSecretClient),
		framework.Compose(withUniversalAuth, f, pushSecretValue(prov), useUniversalAuthForPush(prov)),
		framework.Compose(withUniversalAuth, f, pushSecretDeletesOnPolicy(prov), useUniversalAuthForPush(prov)),
	)
})

func useUniversalAuth(prov *infisicalProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateUniversalAuthStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = tc.Framework.Namespace.Name
	}
}

func useUniversalAuthClusterStore(prov *infisicalProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateUniversalAuthClusterStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = clusterStoreName(tc.Framework)
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1.ClusterSecretStoreKind
	}
}

// useUniversalAuthForPush creates the namespaced Universal Auth store and points
// the test's PushSecret at it.
func useUniversalAuthForPush(prov *infisicalProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateUniversalAuthStore()
		tc.PushSecret.Spec.SecretStoreRefs = []esv1alpha1.PushSecretStoreRef{
			{Name: tc.Framework.Namespace.Name},
		}
	}
}
