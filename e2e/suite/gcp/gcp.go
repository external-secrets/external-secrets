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
package gcp

import (
	"context"
	"fmt"
	"os"

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

var _ = Describe("[gcp] ", func() {
	f := framework.New("eso-gcp")
	var secretStore *esv1alpha1.SecretStore
	projectID := "external-secrets-operator"
	credentials := os.Getenv("GCP_SM_SA_JSON")

	BeforeEach(func() {
		By("creating a secret in GCP SM")
		gcpCred := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Namespace.Name,
				Namespace: f.Namespace.Name,
			},
			StringData: map[string]string{
				"secret-access-credentials": credentials,
			},
		}
		err := f.CRClient.Create(context.Background(), gcpCred)
		Expect(err).ToNot(HaveOccurred())
		secretStore = &esv1alpha1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Namespace.Name,
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.SecretStoreSpec{
				Provider: &esv1alpha1.SecretStoreProvider{
					GCPSM: &esv1alpha1.GCPSMProvider{
						ProjectID: projectID,
						Auth: esv1alpha1.GCPSMAuth{
							SecretRef: esv1alpha1.GCPSMAuthSecretRef{
								SecretAccessKey: esmeta.SecretKeySelector{
									Name: f.Namespace.Name,
									Key:  "secret-access-credentials",
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

	It("should sync secrets", func() {
		By("creating a GCP SM Secret")
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretValue := "great-value-test"
		secret, err := createGCPSecretsManagerSecret(
			projectID,
			secretKey1, secretValue, []byte(credentials))
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
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = f.WaitForSecretValue(f.Namespace.Name, targetSecret, map[string][]byte{
			secretKey1: []byte(secretValue),
		})
		Expect(err).ToNot(HaveOccurred())

		err = deleteGCPSecretsManagerSecret(secret.Name, []byte(credentials))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should sync secrets with dataFrom", func() {
		By("creating a GCP SM Secret with JSON string")
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		targetSecretKey1 := "name"
		targetSecretValue1 := "great-name"
		targetSecretKey2 := "surname"
		targetSecretValue2 := "great-surname"
		secretValue := fmt.Sprintf("{ \"%s\": \"%s\", \"%s\": \"%s\" }", targetSecretKey1, targetSecretValue1, targetSecretKey2, targetSecretValue2)
		secret, err := createGCPSecretsManagerSecret(
			projectID,
			secretKey1, secretValue, []byte(credentials))
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

		err = deleteGCPSecretsManagerSecret(secret.Name, []byte(credentials))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should sync secrets and get inner keys", func() {
		By("creating a GCP SM Secret")
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
		secret, err := createGCPSecretsManagerSecret(
			projectID,
			secretKey1, secretValue, []byte(credentials))
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

		err = deleteGCPSecretsManagerSecret(secret.Name, []byte(credentials))
		Expect(err).ToNot(HaveOccurred())
	})

})
