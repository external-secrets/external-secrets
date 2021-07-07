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
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

const (
	targetSecret = "target-secret"
)

var _ = Describe("[aws] ", func() {
	f := framework.New("eso-aws")
	var secretStore *esv1alpha1.SecretStore
	localstackURL := "http://localstack.default"

	BeforeEach(func() {
		By("creating an secret store for localstack")
		awsCreds := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Namespace.Name,
				Namespace: f.Namespace.Name,
			},
			StringData: map[string]string{
				"kid": "foobar",
				"sak": "foobar",
			},
		}
		err := f.CRClient.Create(context.Background(), awsCreds)
		Expect(err).ToNot(HaveOccurred())
		secretStore = &esv1alpha1.SecretStore{
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
							SecretRef: &esv1alpha1.AWSAuthSecretRef{
								AccessKeyID: esmeta.SecretKeySelector{
									Name: f.Namespace.Name,
									Key:  "kid",
								},
								SecretAccessKey: esmeta.SecretKeySelector{
									Name: f.Namespace.Name,
									Key:  "sak",
								},
							},
						},
					},
				},
			},
		}
		err = f.CRClient.Create(context.Background(), secretStore)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should sync multiple secrets", func() {
		By("creating a AWS SM Secret")
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretKey2 := fmt.Sprintf("%s-%s", f.Namespace.Name, "other")
		secretValue := "bar"
		err := CreateAWSSecretsManagerSecret(
			localstackURL,
			secretKey1, secretValue)
		Expect(err).ToNot(HaveOccurred())
		err = CreateAWSSecretsManagerSecret(
			localstackURL,
			secretKey2, secretValue)
		Expect(err).ToNot(HaveOccurred())

		err = f.CRClient.Create(context.Background(), &esv1alpha1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "simple-sync",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.ExternalSecretSpec{
				SecretStoreRef: esv1alpha1.SecretStoreRef{
					Name: f.Namespace.Name,
				},
				Target: esv1alpha1.ExternalSecretTarget{
					Name: targetSecret,
				},
				Data: []esv1alpha1.ExternalSecretData{
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
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = f.WaitForSecretValue(f.Namespace.Name, targetSecret, map[string][]byte{
			secretKey1: []byte(secretValue),
			secretKey2: []byte(secretValue),
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should sync secrets with dataFrom", func() {
		By("creating a AWS SM Secret")
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		targetSecretKey1 := "name"
		targetSecretValue1 := "great-name"
		targetSecretKey2 := "surname"
		targetSecretValue2 := "great-surname"
		secretValue := fmt.Sprintf("{ \"%s\": \"%s\", \"%s\": \"%s\" }", targetSecretKey1, targetSecretValue1, targetSecretKey2, targetSecretValue2)
		err := CreateAWSSecretsManagerSecret(
			localstackURL,
			secretKey1, secretValue)
		Expect(err).ToNot(HaveOccurred())
		err = f.CRClient.Create(context.Background(), &esv1alpha1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "datafrom-sync",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.ExternalSecretSpec{
				SecretStoreRef: esv1alpha1.SecretStoreRef{
					Name: f.Namespace.Name,
				},
				Target: esv1alpha1.ExternalSecretTarget{
					Name: targetSecret,
				},
				DataFrom: []esv1alpha1.ExternalSecretDataRemoteRef{
					{
						Key: secretKey1,
					},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = f.WaitForSecretValue(f.Namespace.Name, targetSecret, map[string][]byte{
			targetSecretKey1: []byte(targetSecretValue1),
			targetSecretKey2: []byte(targetSecretValue2),
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should sync secrets and get inner keys", func() {
		By("creating a AWS SM Secret")
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		targetSecretKey1 := "firstname"
		targetSecretValue1 := "Tom"
		targetSecretKey2 := "first_friend"
		targetSecretValue2 := "Roger"
		secretValue := fmt.Sprintf(
			`{
				"name": {"first": "%s", "last": "Anderson"},
				"friends":
				[
					{"first": "Dale", "last": "Murphy"},
					{"first": "%s", "last": "Craig"},
					{"first": "Jane", "last": "Murphy"}
				]
			}`, targetSecretValue1, targetSecretValue2)
		err := CreateAWSSecretsManagerSecret(
			localstackURL,
			secretKey1, secretValue)
		Expect(err).ToNot(HaveOccurred())
		err = f.CRClient.Create(context.Background(), &esv1alpha1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "datafrom-sync",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.ExternalSecretSpec{
				SecretStoreRef: esv1alpha1.SecretStoreRef{
					Name: f.Namespace.Name,
				},
				Target: esv1alpha1.ExternalSecretTarget{
					Name: targetSecret,
				},
				Data: []esv1alpha1.ExternalSecretData{
					{
						SecretKey: targetSecretKey1,
						RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
							Key:      secretKey1,
							Property: "name.first",
						},
					},
					{
						SecretKey: targetSecretKey2,
						RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
							Key:      secretKey1,
							Property: "friends.1.first",
						},
					},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = f.WaitForSecretValue(f.Namespace.Name, targetSecret, map[string][]byte{
			targetSecretKey1: []byte(targetSecretValue1),
			targetSecretKey2: []byte(targetSecretValue2),
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should sync secrets with cluster secret store", func() {
		By("creating a AWS SM Secret")
		clusterStoreName := fmt.Sprintf("cluster-%s", f.Namespace.Name)
		targetSecretKey := "FOOB"
		secretKey := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretValue := "MYVAL"
		err := CreateAWSSecretsManagerSecret(
			localstackURL,
			secretKey, secretValue)
		Expect(err).ToNot(HaveOccurred())

		css := &esv1alpha1.ClusterSecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterStoreName,
			},
			Spec: esv1alpha1.SecretStoreSpec{
				Provider: &esv1alpha1.SecretStoreProvider{
					AWS: &esv1alpha1.AWSProvider{
						Service: esv1alpha1.AWSServiceSecretsManager,
						Region:  "us-east-1",
						Auth: esv1alpha1.AWSAuth{
							SecretRef: &esv1alpha1.AWSAuthSecretRef{
								AccessKeyID: esmeta.SecretKeySelector{
									Name:      f.Namespace.Name,
									Namespace: &f.Namespace.Name,
									Key:       "kid",
								},
								SecretAccessKey: esmeta.SecretKeySelector{
									Name:      f.Namespace.Name,
									Namespace: &f.Namespace.Name,
									Key:       "sak",
								},
							},
						},
					},
				},
			},
		}
		err = f.CRClient.Create(context.Background(), css)
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			err = f.CRClient.Delete(context.Background(), css)
			Expect(err).ToNot(HaveOccurred())
		}()

		err = f.CRClient.Create(context.Background(), &esv1alpha1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "datafrom-sync",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.ExternalSecretSpec{
				SecretStoreRef: esv1alpha1.SecretStoreRef{
					Name: clusterStoreName,
					Kind: esv1alpha1.ClusterSecretStoreKind,
				},
				Target: esv1alpha1.ExternalSecretTarget{
					Name: targetSecret,
				},
				Data: []esv1alpha1.ExternalSecretData{
					{
						SecretKey: targetSecretKey,
						RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
							Key: secretKey,
						},
					},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = f.WaitForSecretValue(f.Namespace.Name, targetSecret, map[string][]byte{
			targetSecretKey: []byte(secretValue),
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should use jwt auth tokens", func() {
		By("creating a AWS SM Secret")
		clusterStoreName := fmt.Sprintf("cluster-jwt-%s", f.Namespace.Name)
		targetSecretKey := "FOOB"
		saName := "my-sa"
		secretKey := fmt.Sprintf("%s-%s", f.Namespace.Name, "jwt-token")
		secretValue := "MYVAL"
		err := CreateAWSSecretsManagerSecret(
			localstackURL,
			secretKey, secretValue)
		Expect(err).ToNot(HaveOccurred())

		err = f.CRClient.Create(context.Background(), &v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      saName,
				Namespace: f.Namespace.Name,
				Annotations: map[string]string{
					"eks.amazonaws.com/role-arn": "arn:aws:iam::account:role/my-example-role",
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		err = f.CRClient.Create(context.Background(), &esv1alpha1.ClusterSecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterStoreName,
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
		})
		Expect(err).ToNot(HaveOccurred())

		err = f.CRClient.Create(context.Background(), &esv1alpha1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jwt-sync",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.ExternalSecretSpec{
				SecretStoreRef: esv1alpha1.SecretStoreRef{
					Name: clusterStoreName,
					Kind: esv1alpha1.ClusterSecretStoreKind,
				},
				Target: esv1alpha1.ExternalSecretTarget{
					Name: targetSecret,
				},
				Data: []esv1alpha1.ExternalSecretData{
					{
						SecretKey: targetSecretKey,
						RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
							Key: secretKey,
						},
					},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = f.WaitForSecretValue(f.Namespace.Name, targetSecret, map[string][]byte{
			targetSecretKey: []byte(secretValue),
		})
		Expect(err).ToNot(HaveOccurred())
	})
})
