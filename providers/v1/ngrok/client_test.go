/*
Copyright © 2025 ESO Maintainer Team

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

package ngrok

import (
	"encoding/json"
	"errors"

	"github.com/ngrok/ngrok-api-go/v7"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/providers/v1/ngrok/fake"
)

type pushSecretRemoteRef struct {
	remoteKey string
	property  string
}

func (p pushSecretRemoteRef) GetRemoteKey() string {
	return p.remoteKey
}
func (p pushSecretRemoteRef) GetProperty() string {
	return p.property
}

type testClientOpts struct {
	vaults         []*ngrok.Vault
	secrets        []*ngrok.Secret
	secretsListErr error
	vaultName      string
}

type testClientOpt func(opts *testClientOpts)

func WithVaults(vaults ...*ngrok.Vault) testClientOpt {
	return func(opts *testClientOpts) {
		opts.vaults = vaults
	}
}

func WithSecrets(secrets ...*ngrok.Secret) testClientOpt {
	return func(opts *testClientOpts) {
		opts.secrets = secrets
	}
}

func WithSecretsListError(err error) testClientOpt {
	return func(opts *testClientOpts) {
		opts.secretsListErr = err
	}
}

func WithVaultName(vaultName string) testClientOpt {
	return func(opts *testClientOpts) {
		opts.vaultName = vaultName
	}
}

var _ = Describe("client", func() {
	var (
		s         *fake.Store
		c         *client
		vaultName string

		listVaultsErr  error
		listSecretsErr error
	)

	BeforeEach(func() {
		vaultName = "test-vault"
		listSecretsErr = nil
		listVaultsErr = nil
		s = fake.NewStore()
	})

	JustBeforeEach(func() {
		c = &client{
			vaultClient:   s.VaultClient().WithListError(listVaultsErr),
			secretsClient: s.SecretsClient().WithListError(listSecretsErr),
			vaultName:     vaultName,
		}
	})

	Describe("PushSecret", func() {
		var (
			k8Secret        *corev1.Secret
			pushData        v1alpha1.PushSecretData
			vault           *ngrok.Vault
			secret          *ngrok.Secret
			ngrokSecretName string

			pushErr error
		)

		BeforeEach(func() {
			ngrokSecretName = "secret-" + fake.GenerateRandomString(10)
			k8Secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-secret",
				},
				Data: map[string][]byte{
					"key": []byte("new value"),
					"foo": []byte("bar"),
				},
			}
			pushData = v1alpha1.PushSecretData{
				Match: v1alpha1.PushSecretMatch{
					SecretKey: "key",
					RemoteRef: v1alpha1.PushSecretRemoteRef{
						RemoteKey: ngrokSecretName,
					},
				},
			}
			vault = nil
		})

		JustBeforeEach(func(ctx SpecContext) {
			if vault != nil {
				// Set the client's vault ID. This is normally initialized by the provider's NewClient method.
				c.setVaultID(vault.ID)
			}
			pushErr = c.PushSecret(ctx, k8Secret, pushData)
		})

		When("the vault exists", func() {
			var (
				getSecretErr error
			)

			BeforeEach(func(ctx SpecContext) {
				var vaultCreateErr error
				vault, vaultCreateErr = s.VaultClient().Create(ctx, &ngrok.VaultCreate{
					Name: vaultName,
				})
				Expect(vaultCreateErr).ToNot(HaveOccurred())
			})

			// Re-fetch the secret after the push to verify it was updated
			JustBeforeEach(func(ctx SpecContext) {
				secret = nil
				iter := s.SecretsClient().List(nil)
				for iter.Next(ctx) {
					if iter.Item().Name == ngrokSecretName && iter.Item().Vault.ID == vault.ID {
						secret = iter.Item()
						break
					}
				}
				getSecretErr = iter.Err()
			})

			When("the secret does not exist", func() {
				It("should not return an error", func(ctx SpecContext) {
					Expect(pushErr).ToNot(HaveOccurred())
				})

				It("should create the ngrok secret", func(ctx SpecContext) {
					Expect(getSecretErr).ToNot(HaveOccurred())
					Expect(secret).ToNot(BeNil())
					Expect(secret.Name).To(Equal(ngrokSecretName))
					Expect(secret.ID).ToNot(BeEmpty())
					Expect(secret.Description).To(Equal(defaultDescription))
				})
			})

			When("the secret exists", func() {
				BeforeEach(func(ctx SpecContext) {
					var createErr error
					secret, createErr = s.SecretsClient().Create(ctx, &ngrok.SecretCreate{
						VaultID: vault.ID,
						Name:    ngrokSecretName,
						Value:   "old-value",
					})
					Expect(createErr).ToNot(HaveOccurred())
				})

				It("should not return an error", func(ctx SpecContext) {
					Expect(pushErr).ToNot(HaveOccurred())
				})

				It("should update the ngrok secret description", func(ctx SpecContext) {
					Expect(secret.Description).To(Equal(defaultDescription))
				})

				It("should update the ngrok secret metadata", func(ctx SpecContext) {
					// The metadata should include the sha256 of the new value.
					// sha256sum "new value" = 9c51d0b0f64dfb3662ed85ce945dd1e8f6130665c289754e4e9257a58013e61d
					Expect(secret.Metadata).To(Equal(`{"_sha256":"9c51d0b0f64dfb3662ed85ce945dd1e8f6130665c289754e4e9257a58013e61d"}`))
				})

				When("The secret key is not specified on the push data", func() {
					BeforeEach(func() {
						pushData.Match.SecretKey = ""
					})

					It("should marshal the entire secret data as JSON", func(ctx SpecContext) {
						data := map[string]string{}
						err := json.Unmarshal([]byte(secret.Metadata), &data)

						Expect(err).ToNot(HaveOccurred())
						Expect(data).To(HaveKeyWithValue("_sha256", "146ed8bb7a977ee78ee11cf262924e3ae93423c413ab6d612a8d159a0ae4e1ad"))
					})
				})

				When("the secret key does not exist in the k8s secret", func() {
					BeforeEach(func() {
						pushData.Match.SecretKey = "nonexistent-key"
					})

					It("should return an error", func(ctx SpecContext) {
						Expect(pushErr).To(HaveOccurred())
						Expect(pushErr.Error()).To(ContainSubstring("key nonexistent-key not found in secret"))
					})
				})

				When("push metadata is provided", func() {
					When("the metadata is valid", func() {
						BeforeEach(func() {
							pushData.Metadata = &apiextensionsv1.JSON{
								Raw: []byte(`
apiVersion: kubernetes.external-secrets.io/v1alpha1
kind: PushSecretMetadata
spec:
  metadata:
    environment: production
    team: frontend
  description: "my custom description"`),
							}
						})

						It("should update the ngrok secret description", func(ctx SpecContext) {
							Expect(secret.Description).To(Equal("my custom description"))
						})

						It("should update the ngrok secret metadata", func(ctx SpecContext) {
							data := map[string]string{}
							err := json.Unmarshal([]byte(secret.Metadata), &data)
							Expect(err).ToNot(HaveOccurred())
							Expect(data).To(HaveKeyWithValue("environment", "production"))
							Expect(data).To(HaveKeyWithValue("team", "frontend"))
							Expect(data).To(HaveKeyWithValue("_sha256", "9c51d0b0f64dfb3662ed85ce945dd1e8f6130665c289754e4e9257a58013e61d"))
						})
					})

					When("the metadata is invalid", func() {
						BeforeEach(func() {
							pushData.Metadata = &apiextensionsv1.JSON{
								Raw: []byte(`{ this is not valid json`),
							}
						})

						It("should return an error", func(ctx SpecContext) {
							Expect(pushErr).To(HaveOccurred())
							Expect(pushErr.Error()).To(ContainSubstring("failed to parse push secret metadata"))
						})
					})
				})
			})
		})
	})

	Describe("SecretExists", func() {
		var (
			secretName string

			exists bool
			err    error
		)

		BeforeEach(func() {
			secretName = "my-secret"
		})

		JustBeforeEach(func(ctx SpecContext) {
			exists, err = c.SecretExists(ctx, pushSecretRemoteRef{
				remoteKey: secretName,
			})
		})

		When("the vault does not exist", func() {
			It("should return exists as false without an error", func(ctx SpecContext) {
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())
			})
		})

		When("the vault exists", func() {
			var (
				vault *ngrok.Vault
			)
			BeforeEach(func(ctx SpecContext) {
				vault, err = s.VaultClient().Create(ctx, &ngrok.VaultCreate{
					Name: c.vaultName,
				})
				Expect(err).ToNot(HaveOccurred())
			})

			When("the secret does not exist", func() {
				It("should return exists as false without an error", func(ctx SpecContext) {
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())
				})
			})

			When("the secret exists", func() {
				BeforeEach(func(ctx SpecContext) {
					_, err = s.SecretsClient().Create(ctx, &ngrok.SecretCreate{
						VaultID: vault.ID,
						Name:    secretName,
						Value:   "supersecret",
					})
					Expect(err).ToNot(HaveOccurred())
				})

				It("should return exists as true without an error", func(ctx SpecContext) {
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())
				})
			})
		})

		When("an error occurs listing vaults", func() {
			BeforeEach(func() {
				listVaultsErr = errors.New("failed to list vaults")
			})

			It("should return exists as false", func() {
				Expect(exists).To(BeFalse())
			})

			It("should return the listing error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to list vaults"))
			})
		})
	})

	Describe("DeleteSecret", func() {
		var (
			secretName string

			err error
		)

		BeforeEach(func() {
			secretName = "my-secret"
		})

		JustBeforeEach(func(ctx SpecContext) {
			err = c.DeleteSecret(ctx, pushSecretRemoteRef{
				remoteKey: secretName,
			})
		})

		When("the vault does not exist", func() {
			It("should not return an error", func(ctx SpecContext) {
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("the vault exists but the secret does not", func() {
			BeforeEach(func(ctx SpecContext) {
				_, err := c.vaultClient.Create(ctx, &ngrok.VaultCreate{
					Name: c.vaultName,
				})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not return an error", func(ctx SpecContext) {
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("the vault and secret both exist", func() {
			BeforeEach(func(ctx SpecContext) {
				vault, err := s.VaultClient().Create(ctx, &ngrok.VaultCreate{
					Name: c.vaultName,
				})
				Expect(err).ToNot(HaveOccurred())
				_, err = s.SecretsClient().Create(ctx, &ngrok.SecretCreate{
					VaultID: vault.ID,
					Name:    secretName,
					Value:   "supersecret",
				})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not return an error", func(ctx SpecContext) {
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("an error occurs listing vaults", func() {
			BeforeEach(func() {
				listVaultsErr = errors.New("failed to list vaults")
			})

			It("should return the listing error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to list vaults"))
			})
		})
	})

	Describe("Validate", func() {
		var (
			result esv1.ValidationResult
			err    error
		)

		JustBeforeEach(func(ctx SpecContext) {
			result, err = c.Validate()
		})

		When("the client can list secrets", func() {
			When("there are no secrets", func() {
				It("should return ValidationResultReady without an error", func() {
					Expect(err).To(BeNil())
					Expect(result).To(Equal(esv1.ValidationResultReady))
				})
			})

			When("there are some secrets", func() {
				BeforeEach(func(ctx SpecContext) {
					vault, err := s.VaultClient().Create(ctx, &ngrok.VaultCreate{
						Name: c.vaultName,
					})
					Expect(err).ToNot(HaveOccurred())
					_, err = s.SecretsClient().Create(ctx, &ngrok.SecretCreate{
						VaultID: vault.ID,
						Name:    "my-secret",
						Value:   "supersecret",
					})
					Expect(err).ToNot(HaveOccurred())
				})

				It("should return ValidationResultReady without an error", func() {
					Expect(err).To(BeNil())
					Expect(result).To(Equal(esv1.ValidationResultReady))
				})
			})
		})

		When("the client cannot list secrets", func() {
			BeforeEach(func() {
				listSecretsErr = errors.New("failed to list secrets")
			})

			It("should return ValidationResultError with the listing error", func() {
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("failed to list secrets"))
				Expect(result).To(Equal(esv1.ValidationResultError))
			})
		})
	})

	Describe("GetSecret", func() {
		var (
			ref esv1.ExternalSecretDataRemoteRef

			secret []byte
			err    error
		)

		JustBeforeEach(func(ctx SpecContext) {
			secret, err = c.GetSecret(ctx, ref)
		})

		It("should always return an error indicating write-only operations", func(ctx SpecContext) {
			Expect(secret).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errWriteOnlyOperations))
		})
	})

	Describe("GetSecretMap", func() {
		var (
			ref esv1.ExternalSecretDataRemoteRef

			secretMap map[string][]byte
			err       error
		)

		JustBeforeEach(func(ctx SpecContext) {
			secretMap, err = c.GetSecretMap(ctx, ref)
		})

		It("should always return an error indicating write-only operations", func(ctx SpecContext) {
			Expect(secretMap).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errWriteOnlyOperations))
		})
	})

	Describe("GetAllSecrets", func() {
		var (
			find esv1.ExternalSecretFind

			secrets map[string][]byte
			err     error
		)

		JustBeforeEach(func(ctx SpecContext) {
			secrets, err = c.GetAllSecrets(ctx, find)
		})

		It("should always return an error indicating write-only operations", func(ctx SpecContext) {
			Expect(secrets).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(errWriteOnlyOperations))
		})
	})

	Describe("Close", func() {
		It("should not return an error", func(ctx SpecContext) {
			Expect(c.Close(ctx)).To(BeNil())
		})
	})
})
