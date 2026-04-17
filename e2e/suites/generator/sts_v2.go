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

package generator

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

var _ = Describe("sts generator v2", Label("aws", "sts", "v2"), func() {
	f := framework.New("sts-v2")

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
		skipIfAWSGeneratorCredentialsMissing()
	})

	injectGenerator := func(tc *testCase) {
		createAWSGeneratorCredentialsSecret(f)
		tc.Generator = &genv1alpha1.STSSessionToken{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version,
				Kind:       genv1alpha1.STSSessionTokenKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      generatorName,
				Namespace: f.Namespace.Name,
			},
			Spec: genv1alpha1.STSSessionTokenSpec{
				Region: os.Getenv("AWS_REGION"),
				Auth:   awsGeneratorAuth(),
			},
		}
	}

	customResourceGenerator := func(tc *testCase) {
		tc.ExternalSecret.Spec.DataFrom = []esv1.ExternalSecretDataFromRemoteRef{{
			SourceRef: &esv1.StoreGeneratorSourceRef{
				GeneratorRef: &esv1.GeneratorRef{
					Kind: "STSSessionToken",
					Name: generatorName,
				},
			},
		}}
		tc.AfterSync = func(secret *v1.Secret) {
			Expect(string(secret.Data["access_key_id"])).ToNot(BeEmpty())
			Expect(string(secret.Data["secret_access_key"])).ToNot(BeEmpty())
			Expect(string(secret.Data["session_token"])).ToNot(BeEmpty())
			Expect(string(secret.Data["expiration"])).ToNot(BeEmpty())
		}
	}

	DescribeTable("generate sts session tokens through the v2 aws provider", generatorTableFunc,
		Entry("using custom resource generator", f, injectGenerator, customResourceGenerator),
	)
})
