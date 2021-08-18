package alibaba

import (

	// nolint
	. "github.com/onsi/ginkgo"
	// nolint
	. "github.com/onsi/ginkgo/extensions/table"

	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/suite/common"
)

var _ = Describe("[alibaba] ", func() {
	f := framework.New("eso-alibaba")

	DescribeTable("sync secrets",
		framework.TableFunc(f,
			newVaultProvider(f)),
		// uses token auth
		compose("with token auth", f, common.JSONDataFromSync, useTokenAuth),
		compose("with token auth", f, common.JSONDataWithProperty, useTokenAuth),
		compose("with token auth", f, common.JSONDataWithTemplate, useTokenAuth),
		compose("with token auth", f, common.DataPropertyDockerconfigJSON, useTokenAuth),
		// use cert auth
		compose("with cert auth", f, common.JSONDataFromSync, useCertAuth),
		compose("with cert auth", f, common.JSONDataWithProperty, useCertAuth),
		compose("with cert auth", f, common.JSONDataWithTemplate, useCertAuth),
		compose("with cert auth", f, common.DataPropertyDockerconfigJSON, useCertAuth),
		// use approle auth
		compose("with appRole auth", f, common.JSONDataFromSync, useApproleAuth),
		compose("with appRole auth", f, common.JSONDataWithProperty, useApproleAuth),
		compose("with appRole auth", f, common.JSONDataWithTemplate, useApproleAuth),
		compose("with appRole auth", f, common.DataPropertyDockerconfigJSON, useApproleAuth),
		// use v1 provider
		compose("with v1 kv provider", f, common.JSONDataFromSync, useV1Provider),
		compose("with v1 kv provider", f, common.JSONDataWithProperty, useV1Provider),
		compose("with v1 kv provider", f, common.JSONDataWithTemplate, useV1Provider),
		compose("with v1 kv provider", f, common.DataPropertyDockerconfigJSON, useV1Provider),
		// use jwt provider
		compose("with jwt provider", f, common.JSONDataFromSync, useJWTProvider),
		compose("with jwt provider", f, common.JSONDataWithProperty, useJWTProvider),
		compose("with jwt provider", f, common.JSONDataWithTemplate, useJWTProvider),
		compose("with jwt provider", f, common.DataPropertyDockerconfigJSON, useJWTProvider),
		// use kubernetes provider
		compose("with kubernetes provider", f, common.JSONDataFromSync, useKubernetesProvider),
		compose("with kubernetes provider", f, common.JSONDataWithProperty, useKubernetesProvider),
		compose("with kubernetes provider", f, common.JSONDataWithTemplate, useKubernetesProvider),
		compose("with kubernetes provider", f, common.DataPropertyDockerconfigJSON, useKubernetesProvider),
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
