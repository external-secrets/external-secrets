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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var _ = Describe("[aws] v2 namespaced provider", Label("aws", "parameterstore", "v2", "namespaced-provider"), func() {
	f := framework.New("eso-aws-ps-v2")
	prov := NewProviderV2(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
	})

	DescribeTable("namespaced provider",
		framework.TableFuncWithExternalSecret(f, prov),
		framework.Compose(withStaticAuth, f, func(_ *framework.Framework) (string, func(*framework.TestCase)) {
			return common.NamespacedProviderSync(f, common.NamespacedProviderSyncConfig{
				Description:        "[aws] should sync an ExternalSecret through a namespaced ParameterStore Provider using static credentials",
				ExternalSecretName: "aws-v2-ps-static-es",
				TargetSecretName:   "aws-v2-ps-static-target",
				RemoteKey:          f.MakeRemoteRefKey("aws-v2-ps-static-remote"),
				RemoteSecretValue:  "aws-v2-ps-static-value",
				SecretKey:          "value",
				ExpectedValue:      "aws-v2-ps-static-value",
			})
		}, useV2StaticAuth(prov)),
		framework.Compose(withStaticAuth, f, func(_ *framework.Framework) (string, func(*framework.TestCase)) {
			return common.NamespacedProviderRefresh(f, common.NamespacedProviderRefreshConfig{
				Description:         "[aws] should refresh synced ParameterStore secrets after the remote parameter changes",
				ExternalSecretName:  "aws-v2-ps-refresh-es",
				TargetSecretName:    "aws-v2-ps-refresh-target",
				RemoteKey:           f.MakeRemoteRefKey("aws-v2-ps-refresh-remote"),
				InitialSecretValue:  "aws-v2-ps-initial",
				UpdatedSecretValue:  "aws-v2-ps-updated",
				SecretKey:           "value",
				InitialExpectedData: "aws-v2-ps-initial",
				UpdatedExpectedData: "aws-v2-ps-updated",
				RefreshInterval:     10 * time.Second,
				WaitTimeout:         30 * time.Second,
			})
		}, useV2StaticAuth(prov)),
		framework.Compose(withStaticAuth, f, FindByName, useV2StaticAuth(prov)),
		framework.Compose(withStaticAuth, f, FindByTag, useV2StaticAuth(prov)),
		framework.Compose(withStaticAuth, f, versionedParameterV2(prov), useV2StaticAuth(prov)),
		framework.Compose(withStaticAuth, f, common.StatusNotUpdatedAfterSuccessfulSync, useV2StaticAuth(prov)),
	)
})

func versionedParameterV2(prov framework.SecretStoreProvider) func(*framework.Framework) (string, func(*framework.TestCase)) {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "[common] should read versioned secrets", func(tc *framework.TestCase) {
			secretKey := fmt.Sprintf("/e2e/versioned/%s/%s", f.Namespace.Name, "one")
			versions := []int{1, 2, 3, 4, 5}

			tc.ExpectedSecret = commonVersionedExpectedSecret(versions)
			tc.ExternalSecret.Spec.Data = commonVersionedExternalSecretData(secretKey, versions)
			tc.Cleanup = func() {
				prov.DeleteSecret(secretKey)
			}

			for _, version := range versions {
				prov.CreateSecret(secretKey, framework.SecretEntry{
					Value: fmt.Sprintf("value%d", version),
				})
			}
		}
	}
}

func commonVersionedExpectedSecret(versions []int) *corev1.Secret {
	data := make(map[string][]byte, len(versions))
	for _, version := range versions {
		data[fmt.Sprintf("v%d", version)] = []byte(fmt.Sprintf("value%d", version))
	}
	return &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}
}

func commonVersionedExternalSecretData(secretKey string, versions []int) []esapi.ExternalSecretData {
	data := make([]esapi.ExternalSecretData, 0, len(versions))
	for _, version := range versions {
		data = append(data, esapi.ExternalSecretData{
			SecretKey: fmt.Sprintf("v%d", version),
			RemoteRef: esapi.ExternalSecretDataRemoteRef{
				Key:     secretKey,
				Version: fmt.Sprintf("%d", version),
			},
		})
	}
	return data
}
