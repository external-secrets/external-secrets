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
		framework.Compose(withTokenAuth, f, common.JSONDataFromSync, useTokenAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataWithProperty, useTokenAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataWithTemplate, useTokenAuth),
		framework.Compose(withTokenAuth, f, common.DataPropertyDockerconfigJSON, useTokenAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataWithoutTargetName, useTokenAuth),
		// use cert auth
		framework.Compose(withCertAuth, f, common.JSONDataFromSync, useCertAuth),
		framework.Compose(withCertAuth, f, common.JSONDataWithProperty, useCertAuth),
		framework.Compose(withCertAuth, f, common.JSONDataWithTemplate, useCertAuth),
		framework.Compose(withCertAuth, f, common.DataPropertyDockerconfigJSON, useCertAuth),
		framework.Compose(withCertAuth, f, common.JSONDataWithoutTargetName, useCertAuth),
		// use approle auth
		framework.Compose(withApprole, f, common.JSONDataFromSync, useApproleAuth),
		framework.Compose(withApprole, f, common.JSONDataWithProperty, useApproleAuth),
		framework.Compose(withApprole, f, common.JSONDataWithTemplate, useApproleAuth),
		framework.Compose(withApprole, f, common.DataPropertyDockerconfigJSON, useApproleAuth),
		framework.Compose(withApprole, f, common.JSONDataWithoutTargetName, useApproleAuth),
		// use v1 provider
		framework.Compose(withV1, f, common.JSONDataFromSync, useV1Provider),
		framework.Compose(withV1, f, common.JSONDataWithProperty, useV1Provider),
		framework.Compose(withV1, f, common.JSONDataWithTemplate, useV1Provider),
		framework.Compose(withV1, f, common.DataPropertyDockerconfigJSON, useV1Provider),
		framework.Compose(withV1, f, common.JSONDataWithoutTargetName, useV1Provider),
		// use jwt provider
		framework.Compose(withJWT, f, common.JSONDataFromSync, useJWTProvider),
		framework.Compose(withJWT, f, common.JSONDataWithProperty, useJWTProvider),
		framework.Compose(withJWT, f, common.JSONDataWithTemplate, useJWTProvider),
		framework.Compose(withJWT, f, common.DataPropertyDockerconfigJSON, useJWTProvider),
		framework.Compose(withJWT, f, common.JSONDataWithoutTargetName, useJWTProvider),
		// use kubernetes provider
		framework.Compose(withK8s, f, common.JSONDataFromSync, useKubernetesProvider),
		framework.Compose(withK8s, f, common.JSONDataWithProperty, useKubernetesProvider),
		framework.Compose(withK8s, f, common.JSONDataWithTemplate, useKubernetesProvider),
		framework.Compose(withK8s, f, common.DataPropertyDockerconfigJSON, useKubernetesProvider),
		framework.Compose(withK8s, f, common.JSONDataWithoutTargetName, useKubernetesProvider),
	)
})

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
