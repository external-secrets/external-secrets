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
	"context"
	"fmt"
	"net/http"

	vault "github.com/hashicorp/vault/api"

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

var _ = Describe("[vault] ", func() {
	f := framework.New("eso-vault")
	var secretStore *esv1alpha1.SecretStore

	BeforeEach(func() {
		By("creating an secret store for vault")
		vaultCreds := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Namespace.Name,
				Namespace: f.Namespace.Name,
			},
			StringData: map[string]string{
				"token": "root", // vault dev-mode default token
			},
		}
		err := f.CRClient.Create(context.Background(), vaultCreds)
		Expect(err).ToNot(HaveOccurred())
		secretStore = &esv1alpha1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      f.Namespace.Name,
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.SecretStoreSpec{
				Provider: &esv1alpha1.SecretStoreProvider{
					Vault: &esv1alpha1.VaultProvider{
						Version: esv1alpha1.VaultKVStoreV2,
						Path:    "secret",
						Server:  "http://vault.default:8200",
						Auth: esv1alpha1.VaultAuth{
							TokenSecretRef: &esmeta.SecretKeySelector{
								Name: f.Namespace.Name,
								Key:  "token",
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
		secretKey := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretProp := "example"
		secretValue := "bar"
		targetSecret := "target-secret"

		By("creating a vault secret")
		vc, err := vault.NewClient(&vault.Config{
			Address: "http://vault.default:8200",
		})
		Expect(err).ToNot(HaveOccurred())
		vc.SetToken("root") // dev-mode default token
		req := vc.NewRequest(http.MethodPost, fmt.Sprintf("/v1/secret/data/%s", secretKey))
		err = req.SetJSONBody(map[string]interface{}{
			"data": map[string]string{
				secretProp: secretValue,
			},
		})
		Expect(err).ToNot(HaveOccurred())
		_, err = vc.RawRequestWithContext(context.Background(), req)
		Expect(err).ToNot(HaveOccurred())

		By("creating ExternalSecret")
		err = f.CRClient.Create(context.Background(), &esv1alpha1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "simple-sync",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.ExternalSecretSpec{
				SecretStoreRef: esv1alpha1.SecretStoreRef{
					Name: secretStore.Name,
				},
				Target: esv1alpha1.ExternalSecretTarget{
					Name: targetSecret,
				},
				Data: []esv1alpha1.ExternalSecretData{
					{
						SecretKey: secretKey,
						RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
							Key:      secretKey,
							Property: secretProp,
						},
					},
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())
		_, err = f.WaitForSecretValue(f.Namespace.Name, targetSecret, map[string][]byte{
			secretKey: []byte(secretValue),
		})
		Expect(err).ToNot(HaveOccurred())
	})
})
