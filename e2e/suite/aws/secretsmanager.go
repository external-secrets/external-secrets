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
	"fmt"

	// nolint
	. "github.com/onsi/ginkgo"

	// nolint
	. "github.com/onsi/ginkgo/extensions/table"

	// nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/suite/common"
)

var _ = Describe("[aws] ", func() {
	f := framework.New("eso-aws")
	prov := newSMProvider(f, "http://localstack.default")

	jwt := func(tc *framework.TestCase) {
		saName := "my-sa"
		err := f.CRClient.Create(context.Background(), &v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      saName,
				Namespace: f.Namespace.Name,
				Annotations: map[string]string{
					"eks.amazonaws.com/role-arn": "arn:aws:iam::account:role/my-example-role",
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		// create secret store
		secretStore := &esv1alpha1.SecretStore{
			TypeMeta: metav1.TypeMeta{
				Kind:       esv1alpha1.SecretStoreKind,
				APIVersion: esv1alpha1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Namespace.Name,
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.SecretStoreSpec{
				Provider: &esv1alpha1.SecretStoreProvider{
					AWS: &esv1alpha1.AWSProvider{
						Service: esv1alpha1.AWSServiceSecretsManager,
						Region:  "us-east-1",
						Auth: esv1alpha1.AWSAuth{
							JWTAuth: &esv1alpha1.AWSJWTAuth{
								ServiceAccountRef: &esmeta.ServiceAccountSelector{
									Name:      saName,
									Namespace: &f.Namespace.Name,
								},
							},
						},
					},
				},
			},
		}
		err = f.CRClient.Patch(context.Background(), secretStore, client.Apply, client.FieldOwner("e2e-case"), client.ForceOwnership)
		Expect(err).ToNot(HaveOccurred())

		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretKey2 := fmt.Sprintf("%s-%s", f.Namespace.Name, "other")
		secretValue := "bar"
		tc.Secrets = map[string]string{
			secretKey1: secretValue,
			secretKey2: secretValue,
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey1: []byte(secretValue),
				secretKey2: []byte(secretValue),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: secretKey1,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: secretKey1,
				},
			},
			{
				SecretKey: secretKey2,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: secretKey2,
				},
			},
		}
	}

	DescribeTable("sync secrets",
		framework.TableFunc(f,
			prov),
		Entry(common.SimpleDataSync(f)),
		Entry(common.NestedJSONWithGJSON(f)),
		Entry(common.JSONDataFromSync(f)),
		Entry(common.JSONDataWithProperty(f)),
		Entry(common.JSONDataWithTemplate(f)),
		Entry("should sync secrets with jwt auth", jwt),
		Entry(common.DockerJSONConfig(f)),
		Entry(common.DataPropertyDockerconfigJSON(f)),
		Entry(common.SSHKeySync(f)),
		Entry(common.SSHKeySyncDataProperty(f)),
	)
})
