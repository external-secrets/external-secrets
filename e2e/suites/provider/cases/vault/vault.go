/*
Copyright © 2025 ESO Maintainer Team

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
package vault

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	withTokenAuth           = "with token auth"
	withTokenAuthAndMTLS    = "with token auth and mTLS"
	withCertAuth            = "with cert auth"
	withApprole             = "with approle auth"
	withV1                  = "with v1 provider"
	withJWT                 = "with jwt provider"
	withJWTK8s              = "with jwt k8s provider"
	withK8s                 = "with kubernetes provider"
	withReferentAuth        = "with referent provider"
	withReferentAuthAndMTLS = "with referent provider and mTLS"
)

var _ = Describe("[vault]", Label("vault"), Ordered, func() {
	f := framework.New("vault")
	vault := addon.NewVault()
	prov := newVaultProvider(f, vault)

	BeforeAll(func() {
		addon.InstallGlobalAddon(vault)
	})

	DescribeTable("sync secrets",
		framework.TableFuncWithExternalSecret(f, prov),
		// uses token auth
		framework.Compose(withTokenAuth, f, common.FindByName, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.FindByNameAndRewrite, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataFromSync, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataFromRewrite, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataWithProperty, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataWithTemplate, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.DataPropertyDockerconfigJSON, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataWithoutTargetName, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.DecodingPolicySync, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.JSONDataWithTemplateFromLiteral, useTokenAuth(prov)),
		framework.Compose(withTokenAuth, f, common.TemplateFromConfigmaps, useTokenAuth(prov)),
		// use cert auth
		framework.Compose(withCertAuth, f, common.FindByName, useCertAuth(prov)),
		framework.Compose(withCertAuth, f, common.FindByNameAndRewrite, useCertAuth(prov)),
		framework.Compose(withCertAuth, f, common.JSONDataFromSync, useCertAuth(prov)),
		framework.Compose(withCertAuth, f, common.JSONDataFromRewrite, useCertAuth(prov)),
		framework.Compose(withCertAuth, f, common.JSONDataWithProperty, useCertAuth(prov)),
		framework.Compose(withCertAuth, f, common.JSONDataWithTemplate, useCertAuth(prov)),
		framework.Compose(withCertAuth, f, common.DataPropertyDockerconfigJSON, useCertAuth(prov)),
		framework.Compose(withCertAuth, f, common.JSONDataWithoutTargetName, useCertAuth(prov)),
		// use approle auth
		framework.Compose(withApprole, f, common.FindByName, useApproleAuth(prov)),
		framework.Compose(withApprole, f, common.FindByNameAndRewrite, useApproleAuth(prov)),
		framework.Compose(withApprole, f, common.JSONDataFromSync, useApproleAuth(prov)),
		framework.Compose(withApprole, f, common.JSONDataFromRewrite, useApproleAuth(prov)),
		framework.Compose(withApprole, f, common.JSONDataWithProperty, useApproleAuth(prov)),
		framework.Compose(withApprole, f, common.JSONDataWithTemplate, useApproleAuth(prov)),
		framework.Compose(withApprole, f, common.DataPropertyDockerconfigJSON, useApproleAuth(prov)),
		framework.Compose(withApprole, f, common.JSONDataWithoutTargetName, useApproleAuth(prov)),
		// use v1 provider
		framework.Compose(withV1, f, common.FindByName, useV1Provider(prov)),
		framework.Compose(withV1, f, common.FindByNameAndRewrite, useV1Provider(prov)),
		framework.Compose(withV1, f, common.JSONDataFromSync, useV1Provider(prov)),
		framework.Compose(withV1, f, common.JSONDataFromRewrite, useV1Provider(prov)),
		framework.Compose(withV1, f, common.JSONDataWithProperty, useV1Provider(prov)),
		framework.Compose(withV1, f, common.JSONDataWithTemplate, useV1Provider(prov)),
		framework.Compose(withV1, f, common.DataPropertyDockerconfigJSON, useV1Provider(prov)),
		framework.Compose(withV1, f, common.JSONDataWithoutTargetName, useV1Provider(prov)),
		// use jwt provider
		framework.Compose(withJWT, f, common.FindByName, useJWTProvider(prov)),
		framework.Compose(withJWT, f, common.FindByNameAndRewrite, useJWTProvider(prov)),
		framework.Compose(withJWT, f, common.JSONDataFromSync, useJWTProvider(prov)),
		framework.Compose(withJWT, f, common.JSONDataFromRewrite, useJWTProvider(prov)),
		framework.Compose(withJWT, f, common.JSONDataWithProperty, useJWTProvider(prov)),
		framework.Compose(withJWT, f, common.JSONDataWithTemplate, useJWTProvider(prov)),
		framework.Compose(withJWT, f, common.DataPropertyDockerconfigJSON, useJWTProvider(prov)),
		framework.Compose(withJWT, f, common.JSONDataWithoutTargetName, useJWTProvider(prov)),
		// use jwt k8s provider
		framework.Compose(withJWTK8s, f, common.JSONDataFromSync, useJWTK8sProvider(prov)),
		framework.Compose(withJWTK8s, f, common.JSONDataFromRewrite, useJWTK8sProvider(prov)),
		framework.Compose(withJWTK8s, f, common.JSONDataWithProperty, useJWTK8sProvider(prov)),
		framework.Compose(withJWTK8s, f, common.JSONDataWithTemplate, useJWTK8sProvider(prov)),
		framework.Compose(withJWTK8s, f, common.DataPropertyDockerconfigJSON, useJWTK8sProvider(prov)),
		framework.Compose(withJWTK8s, f, common.JSONDataWithoutTargetName, useJWTK8sProvider(prov)),
		// use kubernetes provider
		framework.Compose(withK8s, f, common.FindByName, useKubernetesProvider(prov)),
		framework.Compose(withK8s, f, common.FindByNameAndRewrite, useKubernetesProvider(prov)),
		framework.Compose(withK8s, f, common.JSONDataFromSync, useKubernetesProvider(prov)),
		framework.Compose(withK8s, f, common.JSONDataFromRewrite, useKubernetesProvider(prov)),
		framework.Compose(withK8s, f, common.JSONDataWithProperty, useKubernetesProvider(prov)),
		framework.Compose(withK8s, f, common.JSONDataWithTemplate, useKubernetesProvider(prov)),
		framework.Compose(withK8s, f, common.DataPropertyDockerconfigJSON, useKubernetesProvider(prov)),
		framework.Compose(withK8s, f, common.JSONDataWithoutTargetName, useKubernetesProvider(prov)),
		// use referent auth
		framework.Compose(withReferentAuth, f, common.JSONDataFromSync, useReferentAuth(prov)),
		// vault-specific test cases
		Entry("secret value via data without property should return json-encoded string", Label("json"), testJSONWithoutProperty(prov)),
		Entry("secret value via data with property should return json-encoded string", Label("json"), testJSONWithProperty(prov)),
		Entry("dataFrom without property should extract key/value pairs", Label("json"), testDataFromJSONWithoutProperty(prov)),
		Entry("dataFrom with property should extract key/value pairs", Label("json"), testDataFromJSONWithProperty(prov)),
		// mTLS
		framework.Compose(withTokenAuthAndMTLS, f, common.FindByName, useMTLSAndTokenAuth(prov)),
		framework.Compose(withReferentAuthAndMTLS, f, common.JSONDataFromSync, useMTLSAndReferentAuth(prov)),
		Entry("store without clientTLS configuration should not be valid", Label("vault-invalid-store"), testInvalidMtlsStore(prov)),
	)
})

func useTokenAuth(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateTokenStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = tc.Framework.Namespace.Name
	}
}

func useMTLSAndTokenAuth(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateTokenStore(WithMTLS)
		tc.ExternalSecret.Spec.SecretStoreRef.Name = tc.Framework.Namespace.Name + mtlsSuffix
	}
}

func useCertAuth(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateCertStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = certAuthProviderName
	}
}

func useApproleAuth(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateAppRoleStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = appRoleAuthProviderName
	}
}

func useV1Provider(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateV1Store()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = kvv1ProviderName
	}
}

func useJWTProvider(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateJWTStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = jwtProviderName
	}
}

func useJWTK8sProvider(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateJWTK8sStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = jwtK8sProviderName
	}
}

func useKubernetesProvider(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateKubernetesAuthStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = kubernetesProviderName
	}
}

func useReferentAuth(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateReferentTokenStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = referentSecretStoreName(tc.Framework)
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esapi.ClusterSecretStoreKind
	}
}

func useMTLSAndReferentAuth(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateReferentTokenStore(WithMTLS)
		tc.ExternalSecret.Spec.SecretStoreRef.Name = referentSecretStoreName(tc.Framework) + mtlsSuffix
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esapi.ClusterSecretStoreKind
	}
}

const jsonVal = `{"foo":{"nested":{"bar":"mysecret","baz":"bang"}}}`

// when no property is set it should return the json-encoded at path.
func testJSONWithoutProperty(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateTokenStore()
		secretKey := fmt.Sprintf("%s-%s", tc.Framework.Namespace.Name, "json")
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey: {Value: jsonVal},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey: []byte(jsonVal),
			},
		}
		tc.ExternalSecret.Spec.Data = []esapi.ExternalSecretData{
			{
				SecretKey: secretKey,
				RemoteRef: esapi.ExternalSecretDataRemoteRef{
					Key: secretKey,
				},
			},
		}
	}
}

// when property is set it should return the json-encoded at path.
func testJSONWithProperty(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateTokenStore()
		secretKey := fmt.Sprintf("%s-%s", tc.Framework.Namespace.Name, "json")
		expectedVal := `{"bar":"mysecret","baz":"bang"}`
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey: {Value: jsonVal},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey: []byte(expectedVal),
			},
		}
		tc.ExternalSecret.Spec.Data = []esapi.ExternalSecretData{
			{
				SecretKey: secretKey,
				RemoteRef: esapi.ExternalSecretDataRemoteRef{
					Key:      secretKey,
					Property: "foo.nested",
				},
			},
		}
	}
}

// when no property is set it should extract the key/value pairs at the given path
// note: it should json-encode if a value contains nested data
func testDataFromJSONWithoutProperty(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateTokenStore()
		secretKey := fmt.Sprintf("%s-%s", tc.Framework.Namespace.Name, "json")
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey: {Value: jsonVal},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				"foo": []byte(`{"nested":{"bar":"mysecret","baz":"bang"}}`),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esapi.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esapi.ExternalSecretDataRemoteRef{
					Key: secretKey,
				},
			},
		}
	}
}

// when property is set it should extract values with dataFrom at the given path.
func testDataFromJSONWithProperty(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateTokenStore()
		secretKey := fmt.Sprintf("%s-%s", tc.Framework.Namespace.Name, "json")
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey: {Value: jsonVal},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				"bar": []byte(`mysecret`),
				"baz": []byte(`bang`),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esapi.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esapi.ExternalSecretDataRemoteRef{
					Key:      secretKey,
					Property: "foo.nested",
				},
			},
		}
	}
}

func testInvalidMtlsStore(prov *vaultProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateTokenStore(WithInvalidMTLS)
		tc.ExternalSecret = nil
		tc.ExpectedSecret = nil

		err := wait.PollUntilContextTimeout(GinkgoT().Context(), time.Second*10, time.Minute, true, func(ctx context.Context) (bool, error) {
			var ss esapi.SecretStore
			err := tc.Framework.CRClient.Get(ctx, types.NamespacedName{
				Namespace: tc.Framework.Namespace.Name,
				Name:      tc.Framework.Namespace.Name + invalidMtlSuffix,
			}, &ss)
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			if len(ss.Status.Conditions) == 0 {
				return false, nil
			}
			Expect(string(ss.Status.Conditions[0].Type)).Should(Equal("Ready"))
			Expect(string(ss.Status.Conditions[0].Status)).Should(Equal("False"))
			Expect(ss.Status.Conditions[0].Reason).Should(Equal("InvalidProviderConfig"))
			Expect(ss.Status.Conditions[0].Message).Should(ContainSubstring("unable to validate store"))
			return true, nil
		})
		Expect(err).ToNot(HaveOccurred())
	}
}
