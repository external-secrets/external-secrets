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
package vault

import (

	// nolint
	. "github.com/onsi/ginkgo"
	// nolint
	. "github.com/onsi/ginkgo/extensions/table"

	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/suite/common"
)

var _ = Describe("[vault] ", func() {
	f := framework.New("eso-vault")

	DescribeTable("sync secrets",
		framework.TableFunc(f,
			newVaultProvider(f)),
		// uses token auth
		compose("with token auth", f, common.JSONDataFromSync, useTokenAuth),
		compose("with token auth", f, common.JSONDataWithProperty, useTokenAuth),
		compose("with token auth", f, common.JSONDataWithTemplate, useTokenAuth),
		compose("with token auth", f, common.DataPropertyDockerconfigJSON, useTokenAuth),
		compose("with token auth", f, common.SyncWithoutTargetName, useTokenAuth),
		// use cert auth
		compose("with cert auth", f, common.JSONDataFromSync, useCertAuth),
		compose("with cert auth", f, common.JSONDataWithProperty, useCertAuth),
		compose("with cert auth", f, common.JSONDataWithTemplate, useCertAuth),
		compose("with cert auth", f, common.DataPropertyDockerconfigJSON, useCertAuth),
		compose("with cert auth", f, common.SyncWithoutTargetName, useTokenAuth),
		// use approle auth
		compose("with appRole auth", f, common.JSONDataFromSync, useApproleAuth),
		compose("with appRole auth", f, common.JSONDataWithProperty, useApproleAuth),
		compose("with appRole auth", f, common.JSONDataWithTemplate, useApproleAuth),
		compose("with appRole auth", f, common.DataPropertyDockerconfigJSON, useApproleAuth),
		compose("with appRole auth", f, common.SyncWithoutTargetName, useTokenAuth),
		// use v1 provider
		compose("with v1 kv provider", f, common.JSONDataFromSync, useV1Provider),
		compose("with v1 kv provider", f, common.JSONDataWithProperty, useV1Provider),
		compose("with v1 kv provider", f, common.JSONDataWithTemplate, useV1Provider),
		compose("with v1 kv provider", f, common.DataPropertyDockerconfigJSON, useV1Provider),
		compose("with v1 kv provider", f, common.SyncWithoutTargetName, useTokenAuth),
		// use jwt provider
		compose("with jwt provider", f, common.JSONDataFromSync, useJWTProvider),
		compose("with jwt provider", f, common.JSONDataWithProperty, useJWTProvider),
		compose("with jwt provider", f, common.JSONDataWithTemplate, useJWTProvider),
		compose("with jwt provider", f, common.DataPropertyDockerconfigJSON, useJWTProvider),
		compose("with jwt provider", f, common.SyncWithoutTargetName, useTokenAuth),
		// use kubernetes provider
		compose("with kubernetes provider", f, common.JSONDataFromSync, useKubernetesProvider),
		compose("with kubernetes provider", f, common.JSONDataWithProperty, useKubernetesProvider),
		compose("with kubernetes provider", f, common.JSONDataWithTemplate, useKubernetesProvider),
		compose("with kubernetes provider", f, common.DataPropertyDockerconfigJSON, useKubernetesProvider),
		compose("with kubernetes provider", f, common.SyncWithoutTargetName, useTokenAuth),
	)
})

func compose(descAppend string, f *framework.Framework, fn func(f *framework.Framework) (string, func(*framework.TestCase)), tweaks ...func(*framework.TestCase)) TableEntry {
	desc, tfn := fn(f)
	tweaks = append(tweaks, tfn)
	te := Entry(desc + " " + descAppend)

	// need to convert []func to []interface{}
	ifs := make([]interface{}, len(tweaks))
	for i := 0; i < len(tweaks); i++ {
		ifs[i] = tweaks[i]
	}
	te.Parameters = ifs
	return te
}

func useTokenAuth(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = tc.Framework.Namespace.Name
}

func useCertAuth(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = certAuthProviderName
}

func useApproleAuth(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = appRoleAuthProviderName
}

func useV1Provider(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = kvv1ProviderName
}

func useJWTProvider(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = jwtProviderName
}

func useKubernetesProvider(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = kubernetesProviderName
}
