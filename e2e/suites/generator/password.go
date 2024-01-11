/*
Copyright 2020 The cert-manager Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package generator

import (

	//nolint
	. "github.com/onsi/gomega"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("password generator", Label("password"), func() {
	f := framework.New("password")

	injectGenerator := func(tc *testCase) {
		tc.Generator = &genv1alpha1.Password{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version,
				Kind:       genv1alpha1.PasswordKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      generatorName,
				Namespace: f.Namespace.Name,
			},
			Spec: genv1alpha1.PasswordSpec{
				Length: 24,
			},
		}
	}

	customResourceGenerator := func(tc *testCase) {
		tc.ExternalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				SourceRef: &esv1beta1.StoreGeneratorSourceRef{
					GeneratorRef: &esv1beta1.GeneratorRef{
						// we don't need to specify the apiVersion,
						// this should be inferred by the controller.
						Kind: "Password",
						Name: generatorName,
					},
				},
			},
		}
		tc.AfterSync = func(secret *v1.Secret) {
			Expect(len(string(secret.Data["password"]))).To(Equal(24))
		}
	}

	DescribeTable("generate secrets with password generator", generatorTableFunc,
		Entry("using custom resource generator", f, injectGenerator, customResourceGenerator),
	)
})
