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
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const awsCredsSecretName = "aws-creds"

func skipIfAWSGeneratorCredentialsMissing() {
	if os.Getenv("AWS_REGION") == "" || os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		Skip("AWS static generator credentials are required")
	}
}

func createAWSGeneratorCredentialsSecret(f *framework.Framework) {
	err := f.CRClient.Create(GinkgoT().Context(), &v1.Secret{
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
}

func awsGeneratorAuth() genv1alpha1.AWSAuth {
	auth := genv1alpha1.AWSAuth{
		SecretRef: &genv1alpha1.AWSAuthSecretRef{
			AccessKeyID: esmeta.SecretKeySelector{
				Name: awsCredsSecretName,
				Key:  "akid",
			},
			SecretAccessKey: esmeta.SecretKeySelector{
				Name: awsCredsSecretName,
				Key:  "sak",
			},
		},
	}
	if os.Getenv("AWS_SESSION_TOKEN") != "" {
		auth.SecretRef.SessionToken = &esmeta.SecretKeySelector{
			Name: awsCredsSecretName,
			Key:  "st",
		}
	}
	return auth
}
