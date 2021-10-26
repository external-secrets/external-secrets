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

const (
	withTokenAuth = "with token auth"
	withCertAuth  = "with cert auth"
	withApprole   = "with approle auth"
	withV1        = "with v1 provider"
	withJWT       = "with jwt provider"
	withK8s       = "with kubernetes provider"
)

var _ = Describe("[vault] ", func() {
	f := framework.New("eso-vault")

	DescribeTable("sync secrets",
		framework.TableFunc(f,
			newVaultProvider(f)),
		// uses token auth
		compose(withTokenAuth, f, common.JSONDataFromSync, useTokenAuth),
		compose(withTokenAuth, f, common.JSONDataWithProperty, useTokenAuth),
		compose(withTokenAuth, f, common.JSONDataWithTemplate, useTokenAuth),
		compose(withTokenAuth, f, common.DataPropertyDockerconfigJSON, useTokenAuth),
		compose(withTokenAuth, f, common.JSONDataWithoutTargetName, useTokenAuth),
		// use cert auth
		compose(withCertAuth, f, common.JSONDataFromSync, useCertAuth),
		compose(withCertAuth, f, common.JSONDataWithProperty, useCertAuth),
		compose(withCertAuth, f, common.JSONDataWithTemplate, useCertAuth),
		compose(withCertAuth, f, common.DataPropertyDockerconfigJSON, useCertAuth),
		compose(withCertAuth, f, common.JSONDataWithoutTargetName, useTokenAuth),
		// use approle auth
		compose(withApprole, f, common.JSONDataFromSync, useApproleAuth),
		compose(withApprole, f, common.JSONDataWithProperty, useApproleAuth),
		compose(withApprole, f, common.JSONDataWithTemplate, useApproleAuth),
		compose(withApprole, f, common.DataPropertyDockerconfigJSON, useApproleAuth),
		compose(withApprole, f, common.JSONDataWithoutTargetName, useTokenAuth),
		// use v1 provider
		compose(withV1, f, common.JSONDataFromSync, useV1Provider),
		compose(withV1, f, common.JSONDataWithProperty, useV1Provider),
		compose(withV1, f, common.JSONDataWithTemplate, useV1Provider),
		compose(withV1, f, common.DataPropertyDockerconfigJSON, useV1Provider),
		compose(withV1, f, common.JSONDataWithoutTargetName, useTokenAuth),
		// use jwt provider
		compose(withJWT, f, common.JSONDataFromSync, useJWTProvider),
		compose(withJWT, f, common.JSONDataWithProperty, useJWTProvider),
		compose(withJWT, f, common.JSONDataWithTemplate, useJWTProvider),
		compose(withJWT, f, common.DataPropertyDockerconfigJSON, useJWTProvider),
		compose(withJWT, f, common.JSONDataWithoutTargetName, useTokenAuth),
		// use kubernetes provider
		compose(withK8s, f, common.JSONDataFromSync, useKubernetesProvider),
		compose(withK8s, f, common.JSONDataWithProperty, useKubernetesProvider),
		compose(withK8s, f, common.JSONDataWithTemplate, useKubernetesProvider),
		compose(withK8s, f, common.DataPropertyDockerconfigJSON, useKubernetesProvider),
		compose(withK8s, f, common.JSONDataWithoutTargetName, useTokenAuth),
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
