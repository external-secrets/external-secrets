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
	"context"
	"os"

	//nolint
	. "github.com/onsi/gomega"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ecr generator", Label("ecr"), func() {
	f := framework.New("ecr")

	const awsCredsSecretName = "aws-creds"

	injectGenerator := func(tc *testCase) {
		err := f.CRClient.Create(context.Background(), &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      awsCredsSecretName,
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"akid": []byte(os.Getenv("AWS_ACCESS_KEY_ID")),
				"sak":  []byte(os.Getenv("AWS_SECRET_ACCESS_KEY")),
				"st":   []byte(os.Getenv("AWS_SESSION_TOKEN")),
			},
		})
		Expect(err).ToNot(HaveOccurred())

		tc.Generator = &genv1alpha1.ECRAuthorizationToken{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version,
				Kind:       genv1alpha1.ECRAuthorizationTokenKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      generatorName,
				Namespace: f.Namespace.Name,
			},
			Spec: genv1alpha1.ECRAuthorizationTokenSpec{
				Region: os.Getenv("AWS_REGION"),
				Auth: genv1alpha1.AWSAuth{
					SecretRef: &genv1alpha1.AWSAuthSecretRef{
						AccessKeyID: esmeta.SecretKeySelector{
							Name: awsCredsSecretName,
							Key:  "akid",
						},
						SecretAccessKey: esmeta.SecretKeySelector{
							Name: awsCredsSecretName,
							Key:  "sak",
						},
						SessionToken: &esmeta.SecretKeySelector{
							Name: awsCredsSecretName,
							Key:  "st",
						},
					},
				},
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
						Kind: "ECRAuthorizationToken",
						Name: generatorName,
					},
				},
			},
		}
		tc.AfterSync = func(secret *v1.Secret) {
			Expect(string(secret.Data["username"])).To(Equal("AWS"))
			Expect(string(secret.Data["password"])).ToNot(BeEmpty())
		}
	}

	DescribeTable("generate secrets with password generator", generatorTableFunc,
		Entry("using custom resource generator", f, injectGenerator, customResourceGenerator),
	)
})
