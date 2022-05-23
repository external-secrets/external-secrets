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
	"fmt"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/suites/provider/cases/common"
)

const (
	withTokenAuth = "with token auth"
	withCertAuth  = "with cert auth"
	withApprole   = "with approle auth"
	withV1        = "with v1 provider"
	withJWT       = "with jwt provider"
	withJWTK8s    = "with jwt k8s provider"
	withK8s       = "with kubernetes provider"
)

var _ = Describe("[vault]", Label("vault"), func() {
	f := framework.New("eso-vault")
	prov := newVaultProvider(f)

	DescribeTable("sync secrets",
		framework.TableFunc(f, prov),
		// uses token auth
		framework.Compose(withTokenAuth, f, common.FindByName, useTokenAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataFromSync, useTokenAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataWithProperty, useTokenAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataWithTemplate, useTokenAuth),
		framework.Compose(withTokenAuth, f, common.DataPropertyDockerconfigJSON, useTokenAuth),
		framework.Compose(withTokenAuth, f, common.JSONDataWithoutTargetName, useTokenAuth),
		framework.Compose(withTokenAuth, f, common.SyncV1Alpha1, useTokenAuth),
		// use cert auth
		framework.Compose(withCertAuth, f, common.FindByName, useCertAuth),
		framework.Compose(withCertAuth, f, common.JSONDataFromSync, useCertAuth),
		framework.Compose(withCertAuth, f, common.JSONDataWithProperty, useCertAuth),
		framework.Compose(withCertAuth, f, common.JSONDataWithTemplate, useCertAuth),
		framework.Compose(withCertAuth, f, common.DataPropertyDockerconfigJSON, useCertAuth),
		framework.Compose(withCertAuth, f, common.JSONDataWithoutTargetName, useCertAuth),
		// use approle auth
		framework.Compose(withApprole, f, common.FindByName, useApproleAuth),
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
		framework.Compose(withJWT, f, common.FindByName, useJWTProvider),
		framework.Compose(withJWT, f, common.JSONDataFromSync, useJWTProvider),
		framework.Compose(withJWT, f, common.JSONDataWithProperty, useJWTProvider),
		framework.Compose(withJWT, f, common.JSONDataWithTemplate, useJWTProvider),
		framework.Compose(withJWT, f, common.DataPropertyDockerconfigJSON, useJWTProvider),
		framework.Compose(withJWT, f, common.JSONDataWithoutTargetName, useJWTProvider),
		// use jwt k8s provider
		framework.Compose(withJWTK8s, f, common.JSONDataFromSync, useJWTK8sProvider),
		framework.Compose(withJWTK8s, f, common.JSONDataWithProperty, useJWTK8sProvider),
		framework.Compose(withJWTK8s, f, common.JSONDataWithTemplate, useJWTK8sProvider),
		framework.Compose(withJWTK8s, f, common.DataPropertyDockerconfigJSON, useJWTK8sProvider),
		framework.Compose(withJWTK8s, f, common.JSONDataWithoutTargetName, useJWTK8sProvider),
		// use kubernetes provider
		framework.Compose(withK8s, f, common.FindByName, useKubernetesProvider),
		framework.Compose(withK8s, f, common.JSONDataFromSync, useKubernetesProvider),
		framework.Compose(withK8s, f, common.JSONDataWithProperty, useKubernetesProvider),
		framework.Compose(withK8s, f, common.JSONDataWithTemplate, useKubernetesProvider),
		framework.Compose(withK8s, f, common.DataPropertyDockerconfigJSON, useKubernetesProvider),
		framework.Compose(withK8s, f, common.JSONDataWithoutTargetName, useKubernetesProvider),
		// vault-specific test cases
		Entry("secret value via data without property should return json-encoded string", Label("json"), testJSONWithoutProperty),
		Entry("secret value via data with property should return json-encoded string", Label("json"), testJSONWithProperty),
		Entry("dataFrom without property should extract key/value pairs", Label("json"), testDataFromJSONWithoutProperty),
		Entry("dataFrom with property should extract key/value pairs", Label("json"), testDataFromJSONWithProperty),
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

func useJWTK8sProvider(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = jwtK8sProviderName
}

func useKubernetesProvider(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = kubernetesProviderName
}

const jsonVal = `{"foo":{"nested":{"bar":"mysecret","baz":"bang"}}}`

// when no property is set it should return the json-encoded at path.
func testJSONWithoutProperty(tc *framework.TestCase) {
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

// when property is set it should return the json-encoded at path.
func testJSONWithProperty(tc *framework.TestCase) {
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

// when no property is set it should extract the key/value pairs at the given path
// note: it should json-encode if a value contains nested data
func testDataFromJSONWithoutProperty(tc *framework.TestCase) {
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

// when property is set it should extract values with dataFrom at the given path.
func testDataFromJSONWithProperty(tc *framework.TestCase) {
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
