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
package gitlab

// TODO - GitLab only accepts variable names with alphanumeric and '_'
// whereas ESO only accepts names with alphanumeric and '-'.
// Current workaround is to remove all hyphens and underscores set in e2e/framework/util/util.go
// and in e2e/suite/common/common.go, but this breaks Azure provider.

import (
	"fmt"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var _ = Describe("[gitlab]", Label("gitlab"), func() {
	f := framework.New("eso-gitlab")
	prov := newFromEnv(f)

	DescribeTable("sync secrets", framework.TableFuncWithExternalSecret(f, prov),
		Entry(common.SimpleDataSync(f)),
		Entry(common.JSONDataWithProperty(f)),
		Entry(common.JSONDataFromSync(f)),
		Entry(common.JSONDataFromRewrite(f)),
		Entry(common.NestedJSONWithGJSON(f)),
		Entry(common.JSONDataWithTemplate(f)),
		Entry(common.SyncWithoutTargetName(f)),
		Entry(common.JSONDataWithoutTargetName(f)),
		Entry(emptyVariableValue(f)),
		Entry(missingVariableWithDeletionPolicy(f)),
	)
})

// emptyVariableValue syncs an existing variable whose value is an empty string.
func emptyVariableValue(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[gitlab] should sync an existing variable with an empty value", func(tc *framework.TestCase) {
		secretKey := fmt.Sprintf("%s-%s", f.Namespace.Name, "empty")
		remoteRefKey := f.MakeRemoteRefKey(secretKey)
		tc.Secrets = map[string]framework.SecretEntry{
			remoteRefKey: {Value: ""},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey: []byte(""),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: secretKey,
				RemoteRef: esv1.ExternalSecretDataRemoteRef{
					Key: remoteRefKey,
				},
			},
		}
	}
}

// missingVariableWithDeletionPolicy skips a variable missing from GitLab and syncs the rest when deletionPolicy is Delete.
func missingVariableWithDeletionPolicy(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[gitlab] should skip a missing variable and sync the rest with deletionPolicy=Delete", func(tc *framework.TestCase) {
		presentKey := fmt.Sprintf("%s-%s", f.Namespace.Name, "present")
		missingKey := fmt.Sprintf("%s-%s", f.Namespace.Name, "missing")
		remoteRefPresent := f.MakeRemoteRefKey(presentKey)
		remoteRefMissing := f.MakeRemoteRefKey(missingKey)
		secretValue := "bar"
		tc.Secrets = map[string]framework.SecretEntry{
			remoteRefPresent: {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				presentKey: []byte(secretValue),
			},
		}
		tc.ExternalSecret.Spec.Target.DeletionPolicy = esv1.DeletionPolicyDelete
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: presentKey,
				RemoteRef: esv1.ExternalSecretDataRemoteRef{
					Key: remoteRefPresent,
				},
			},
			{
				SecretKey: missingKey,
				RemoteRef: esv1.ExternalSecretDataRemoteRef{
					Key: remoteRefMissing,
				},
			},
		}
	}
}