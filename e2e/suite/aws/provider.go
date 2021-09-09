/*
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

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	//nolint
	. "github.com/onsi/ginkgo"

	// nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/auth"
)

type SMProvider struct {
	url       string
	client    *secretsmanager.SecretsManager
	framework *framework.Framework
}

func newSMProvider(f *framework.Framework, url string) *SMProvider {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Credentials: credentials.NewStaticCredentials("foobar", "foobar", "secret-manager"),
			EndpointResolver: auth.ResolveEndpointWithServiceMap(map[string]string{
				"secretsmanager": url,
			}),
			Region: aws.String("eu-east-1"),
		},
	})
	Expect(err).ToNot(HaveOccurred())
	sm := secretsmanager.New(sess)
	prov := &SMProvider{
		url:       url,
		client:    sm,
		framework: f,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func (s *SMProvider) CreateSecret(key, val string) {
	_, err := s.client.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         aws.String(key),
		SecretString: aws.String(val),
	})
	Expect(err).ToNot(HaveOccurred())
}

func (s *SMProvider) DeleteSecret(key string) {
	_, err := s.client.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId: aws.String(key),
	})
	Expect(err).ToNot(HaveOccurred())
}

func (s *SMProvider) BeforeEach() {
	By("creating a AWS SM credentials secret")
	awsCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider-secret",
			Namespace: s.framework.Namespace.Name,
		},
		StringData: map[string]string{
			"kid": "foobar",
			"sak": "foobar",
		},
	}
	err := s.framework.CRClient.Create(context.Background(), awsCreds)
	Expect(err).ToNot(HaveOccurred())

	By("creating a AWS SM secret store")
	secretStore := &esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				AWS: &esv1alpha1.AWSProvider{
					Service: esv1alpha1.AWSServiceSecretsManager,
					Region:  "us-east-1",
					Auth: esv1alpha1.AWSAuth{
						SecretRef: &esv1alpha1.AWSAuthSecretRef{
							AccessKeyID: esmeta.SecretKeySelector{
								Name: "provider-secret",
								Key:  "kid",
							},
							SecretAccessKey: esmeta.SecretKeySelector{
								Name: "provider-secret",
								Key:  "sak",
							},
						},
					},
				},
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
