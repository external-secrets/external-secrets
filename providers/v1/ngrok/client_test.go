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

package ngrok

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ngrok/ngrok-api-go/v9"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/providers/v1/ngrok/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("client", func() {
	var (
		c          *client
		vaultsAPI  *fake.VaultClient
		secretsAPI *fake.SecretsClient
		vaultName  string
	)

	BeforeEach(func() {
		vaultName = "test-vault"
		vaultsAPI = &fake.VaultClient{}
		secretsAPI = &fake.SecretsClient{}
	})

	JustBeforeEach(func() {
		c = &client{
			vaultClient:   vaultsAPI,
			secretsClient: secretsAPI,
			vaultName:     vaultName,
		}
	})

	Describe("PushSecret", func() {
		var (
			k8Secret        *corev1.Secret
			pushData        v1alpha1.PushSecretData
			vault           *ngrok.Vault
			ngrokSecretName string

			pushErr error

			createCalledWith *ngrok.SecretCreate
			updateCalledWith *ngrok.SecretUpdate
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
			createCalledWith = nil
			updateCalledWith = nil
		})

		JustBeforeEach(func(ctx SpecContext) {
			if vault != nil {
				c.setVaultID(vault.ID)
			}
			pushErr = c.PushSecret(ctx, k8Secret, pushData)
		})

		When("the vault does not exist", func() {
			BeforeEach(func() {
				vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
					return fake.NewIter([]*ngrok.Vault{}, nil)
				}
			})
			It("should return an error", func(ctx SpecContext) {
				Expect(pushErr).To(HaveOccurred())
				Expect(pushErr.Error()).To(ContainSubstring("failed to verify vault name still matches ID: failed to refresh vault ID: vault does not exist"))
			})
		})

		When("an error occurs listing vaults", func() {
			BeforeEach(func() {
				vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
					return fake.NewIter([]*ngrok.Vault{}, errors.New("failed to list vaults"))
				}
			})
			It("should return an error", func(ctx SpecContext) {
				Expect(pushErr).To(HaveOccurred())
				Expect(pushErr.Error()).To(ContainSubstring("failed to list vaults"))
			})
		})

		When("the vault exists", func() {
			BeforeEach(func(ctx SpecContext) {
				vault = &ngrok.Vault{ID: "vault-123", Name: vaultName}
				vaultsAPI.GetFn = func(ctx context.Context, id string) (*ngrok.Vault, error) {
					if id == vault.ID {
						return vault, nil
					}
					return nil, errors.New("not found")
				}
				vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
					return fake.NewIter([]*ngrok.Vault{vault}, nil)
				}
			})

			When("an error occurs listing secrets", func() {
				BeforeEach(func() {
					secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
						return fake.NewIter([]*ngrok.Secret{}, errors.New("failed to list secrets"))
					}
				})
				It("should return an error", func(ctx SpecContext) {
					Expect(pushErr).To(HaveOccurred())
					Expect(pushErr.Error()).To(ContainSubstring("failed to get secret: failed to list secrets"))
				})
			})

			When("the secret does not exist", func() {
				BeforeEach(func() {
					secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
						return fake.NewIter([]*ngrok.Secret{}, nil)
					}
					secretsAPI.CreateFn = func(ctx context.Context, req *ngrok.SecretCreate) (*ngrok.Secret, error) {
						createCalledWith = req
						return &ngrok.Secret{ID: "sec-123", Name: req.Name, Description: req.Description}, nil
					}
				})

				It("should not return an error", func(ctx SpecContext) {
					Expect(pushErr).ToNot(HaveOccurred())
				})

				It("should create the ngrok secret", func(ctx SpecContext) {
					Expect(createCalledWith).ToNot(BeNil())
					Expect(createCalledWith.Name).To(Equal(ngrokSecretName))
					Expect(createCalledWith.Description).To(Equal(defaultDescription))
					Expect(createCalledWith.VaultID).To(Equal(vault.ID))
					Expect(createCalledWith.Value).To(Equal("new value"))
				})

				When("secret creation fails", func() {
					BeforeEach(func() {
						secretsAPI.CreateFn = func(ctx context.Context, req *ngrok.SecretCreate) (*ngrok.Secret, error) {
							return nil, errors.New("failed to create secret")
						}
					})
					It("should return an error", func(ctx SpecContext) {
						Expect(pushErr).To(HaveOccurred())
						Expect(pushErr.Error()).To(ContainSubstring("failed to create secret"))
					})
				})
			})

			When("the secret exists", func() {
				var existingSecret *ngrok.Secret
				BeforeEach(func(ctx SpecContext) {
					existingSecret = &ngrok.Secret{
						ID:   "sec-123",
						Name: ngrokSecretName,
						Vault: ngrok.Ref{
							ID: vault.ID,
						},
					}
					secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
						return fake.NewIter([]*ngrok.Secret{existingSecret}, nil)
					}
					secretsAPI.UpdateFn = func(ctx context.Context, req *ngrok.SecretUpdate) (*ngrok.Secret, error) {
						updateCalledWith = req
						return &ngrok.Secret{ID: req.ID}, nil
					}
				})

				It("should not return an error", func(ctx SpecContext) {
					Expect(pushErr).ToNot(HaveOccurred())
				})

				It("should update the ngrok secret description", func(ctx SpecContext) {
					Expect(updateCalledWith).ToNot(BeNil())
					Expect(updateCalledWith.Description).ToNot(BeNil())
					Expect(*updateCalledWith.Description).To(Equal(defaultDescription))
				})

				It("should update the ngrok secret metadata", func(ctx SpecContext) {
					Expect(updateCalledWith).ToNot(BeNil())
					Expect(updateCalledWith.Metadata).ToNot(BeNil())
					// sha256sum "new value" = 9c51d0b0f64dfb3662ed85ce945dd1e8f6130665c289754e4e9257a58013e61d
					Expect(*updateCalledWith.Metadata).To(Equal(`{"_sha256":"9c51d0b0f64dfb3662ed85ce945dd1e8f6130665c289754e4e9257a58013e61d"}`))
				})

				When("secret update fails", func() {
					BeforeEach(func() {
						secretsAPI.UpdateFn = func(ctx context.Context, req *ngrok.SecretUpdate) (*ngrok.Secret, error) {
							return nil, errors.New("failed to update secret")
						}
					})
					It("should return an error", func(ctx SpecContext) {
						Expect(pushErr).To(HaveOccurred())
						Expect(pushErr.Error()).To(ContainSubstring("failed to update secret"))
					})
				})

				When("The secret key is not specified on the push data", func() {
					BeforeEach(func() {
						pushData.Match.SecretKey = ""
					})

					It("should marshal the entire secret data as JSON", func(ctx SpecContext) {
						Expect(updateCalledWith).ToNot(BeNil())
						Expect(updateCalledWith.Metadata).ToNot(BeNil())

						data := map[string]string{}
						err := json.Unmarshal([]byte(*updateCalledWith.Metadata), &data)

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
							Expect(updateCalledWith).ToNot(BeNil())
							Expect(updateCalledWith.Description).ToNot(BeNil())
							Expect(*updateCalledWith.Description).To(Equal("my custom description"))
						})

						It("should update the ngrok secret metadata", func(ctx SpecContext) {
							Expect(updateCalledWith).ToNot(BeNil())
							Expect(updateCalledWith.Metadata).ToNot(BeNil())
							data := map[string]string{}
							err := json.Unmarshal([]byte(*updateCalledWith.Metadata), &data)
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

		When("the cached vault ID no longer matches the vault name", func() {
			BeforeEach(func() {
				// Seed a stale cached vault ID. The vault it points to has since
				// been renamed, so the client must refresh the ID before pushing.
				vault = &ngrok.Vault{ID: "stale-vault-id", Name: vaultName}
				vaultsAPI.GetFn = func(ctx context.Context, id string) (*ngrok.Vault, error) {
					return &ngrok.Vault{ID: id, Name: "renamed-out-from-under-us"}, nil
				}
				vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
					return fake.NewIter([]*ngrok.Vault{{ID: "fresh-vault-id", Name: vaultName}}, nil)
				}
				secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
					return fake.NewIter([]*ngrok.Secret{}, nil)
				}
				secretsAPI.CreateFn = func(ctx context.Context, req *ngrok.SecretCreate) (*ngrok.Secret, error) {
					createCalledWith = req
					return &ngrok.Secret{ID: "sec-123"}, nil
				}
			})

			It("should refresh the vault ID and push using the refreshed ID", func(ctx SpecContext) {
				Expect(pushErr).ToNot(HaveOccurred())
				Expect(createCalledWith).ToNot(BeNil())
				Expect(createCalledWith.VaultID).To(Equal("fresh-vault-id"))
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
			BeforeEach(func() {
				vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
					return fake.NewIter([]*ngrok.Vault{}, nil)
				}
			})
			It("should return exists as false without an error", func(ctx SpecContext) {
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())
			})
		})

		When("the vault exists", func() {
			var vault *ngrok.Vault
			BeforeEach(func(ctx SpecContext) {
				vault = &ngrok.Vault{ID: "vault-123", Name: vaultName}
				vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
					return fake.NewIter([]*ngrok.Vault{vault}, nil)
				}
			})

			When("the secret does not exist", func() {
				BeforeEach(func() {
					secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
						return fake.NewIter([]*ngrok.Secret{}, nil)
					}
				})
				It("should return exists as false without an error", func(ctx SpecContext) {
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())
				})
			})

			When("the secret exists", func() {
				BeforeEach(func(ctx SpecContext) {
					secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
						return fake.NewIter([]*ngrok.Secret{
							{ID: "sec-123", Name: secretName, Vault: ngrok.Ref{ID: vault.ID}},
						}, nil)
					}
				})

				It("should return exists as true without an error", func(ctx SpecContext) {
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())
				})
			})

			When("an error occurs listing secrets", func() {
				BeforeEach(func(ctx SpecContext) {
					secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
						return fake.NewIter([]*ngrok.Secret{}, errors.New("failed to list secrets"))
					}
				})

				It("should return an error", func(ctx SpecContext) {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("error fetching secret: failed to list secrets"))
				})
			})
		})

		When("an error occurs listing vaults", func() {
			BeforeEach(func() {
				vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
					return fake.NewIter([]*ngrok.Vault{}, errors.New("failed to list vaults"))
				}
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

	Describe("getVaultByName", func() {
		var (
			vault      *ngrok.Vault
			fetched    *ngrok.Vault
			err        error
			listPaging *ngrok.FilteredPaging
		)

		BeforeEach(func(ctx SpecContext) {
			vault = &ngrok.Vault{ID: "vault-123", Name: vaultName}
			vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
				listPaging = paging
				return fake.NewIter([]*ngrok.Vault{vault}, nil)
			}
		})

		JustBeforeEach(func(ctx SpecContext) {
			fetched, err = c.getVaultByName(ctx, vaultName)
		})

		It("should return the matching vault", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(fetched).ToNot(BeNil())
			Expect(fetched.ID).To(Equal(vault.ID))
		})

		It("should filter vaults by name", func() {
			Expect(listPaging).ToNot(BeNil())
			Expect(listPaging.Filter).ToNot(BeNil())
			Expect(*listPaging.Filter).To(Equal(`obj.name == "test-vault"`))
		})
	})

	Describe("getSecretByVaultIDAndName", func() {
		var (
			targetVaultID string
			found         *ngrok.Secret
			err           error
			secretName    string
			listPaging    *ngrok.FilteredPaging
		)

		BeforeEach(func(ctx SpecContext) {
			secretName = "shared-name"
			targetVaultID = "vault-target"
		})

		JustBeforeEach(func(ctx SpecContext) {
			found, err = c.getSecretByVaultIDAndName(ctx, targetVaultID, secretName)
		})

		When("a secret with the same name exists in multiple vaults", func() {
			BeforeEach(func(ctx SpecContext) {
				secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
					listPaging = paging
					return fake.NewIter([]*ngrok.Secret{
						{ID: "sec-1", Name: secretName, Vault: ngrok.Ref{ID: "vault-other"}},
						{ID: "sec-2", Name: secretName, Vault: ngrok.Ref{ID: targetVaultID}},
					}, nil)
				}
			})

			It("should return the secret from the target vault", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(found).ToNot(BeNil())
				Expect(found.ID).To(Equal("sec-2"))
				Expect(found.Vault.ID).To(Equal(targetVaultID))
			})

			It("should filter secrets by name before checking the vault", func() {
				Expect(listPaging).ToNot(BeNil())
				Expect(listPaging.Filter).ToNot(BeNil())
				Expect(*listPaging.Filter).To(Equal(`obj.name == "shared-name"`))
			})
		})

		When("only another vault has a secret with that name", func() {
			BeforeEach(func(ctx SpecContext) {
				secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
					return fake.NewIter([]*ngrok.Secret{
						{ID: "sec-1", Name: secretName, Vault: ngrok.Ref{ID: "vault-other"}},
					}, nil)
				}
			})

			It("should report the secret as missing from the target vault", func() {
				Expect(found).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("secret 'shared-name' does not exist")))
				Expect(err).To(MatchError(ContainSubstring(errVaultSecretDoesNotExist.Error())))
			})
		})
	})

	Describe("DeleteSecret", func() {
		var (
			secretName string
			err        error
			deletedID  string
		)

		BeforeEach(func() {
			secretName = "my-secret"
			deletedID = ""
		})

		JustBeforeEach(func(ctx SpecContext) {
			err = c.DeleteSecret(ctx, pushSecretRemoteRef{
				remoteKey: secretName,
			})
		})

		When("the vault does not exist", func() {
			BeforeEach(func() {
				vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
					return fake.NewIter([]*ngrok.Vault{}, nil)
				}
			})
			It("should not return an error", func(ctx SpecContext) {
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("the vault exists but the secret does not", func() {
			BeforeEach(func(ctx SpecContext) {
				vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
					return fake.NewIter([]*ngrok.Vault{{ID: "vault-123", Name: vaultName}}, nil)
				}
				secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
					return fake.NewIter([]*ngrok.Secret{}, nil)
				}
			})

			It("should not return an error", func(ctx SpecContext) {
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("the vault and secret both exist", func() {
			BeforeEach(func(ctx SpecContext) {
				vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
					return fake.NewIter([]*ngrok.Vault{{ID: "vault-123", Name: vaultName}}, nil)
				}
				secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
					return fake.NewIter([]*ngrok.Secret{
						{ID: "sec-123", Name: secretName, Vault: ngrok.Ref{ID: "vault-123"}},
					}, nil)
				}
				secretsAPI.DeleteFn = func(ctx context.Context, id string) error {
					deletedID = id
					return nil
				}
			})

			It("should not return an error and delete is called", func(ctx SpecContext) {
				Expect(err).ToNot(HaveOccurred())
				Expect(deletedID).To(Equal("sec-123"))
			})

			When("secret deletion fails", func() {
				BeforeEach(func(ctx SpecContext) {
					secretsAPI.DeleteFn = func(ctx context.Context, id string) error {
						return errors.New("failed to delete secret")
					}
				})

				It("should return an error", func(ctx SpecContext) {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to delete secret"))
				})
			})
		})

		When("an error occurs listing secrets", func() {
			BeforeEach(func(ctx SpecContext) {
				vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
					return fake.NewIter([]*ngrok.Vault{{ID: "vault-123", Name: vaultName}}, nil)
				}
				secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
					return fake.NewIter([]*ngrok.Secret{}, errors.New("failed to list secrets"))
				}
			})

			It("should return an error", func(ctx SpecContext) {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to list secrets"))
			})
		})

		When("an error occurs listing vaults", func() {
			BeforeEach(func() {
				vaultsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Vault] {
					return fake.NewIter([]*ngrok.Vault{}, errors.New("failed to list vaults"))
				}
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
				BeforeEach(func() {
					secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
						return fake.NewIter([]*ngrok.Secret{}, nil)
					}
				})
				It("should return ValidationResultReady without an error", func() {
					Expect(err).To(BeNil())
					Expect(result).To(Equal(esv1.ValidationResultReady))
				})
			})

			When("there are some secrets", func() {
				BeforeEach(func(ctx SpecContext) {
					secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
						return fake.NewIter([]*ngrok.Secret{{ID: "sec-1"}}, nil)
					}
				})

				It("should return ValidationResultReady without an error", func() {
					Expect(err).To(BeNil())
					Expect(result).To(Equal(esv1.ValidationResultReady))
				})
			})
		})

		When("the client cannot list secrets", func() {
			BeforeEach(func() {
				secretsAPI.ListFn = func(paging *ngrok.FilteredPaging) ngrok.Iter[*ngrok.Secret] {
					return fake.NewIter([]*ngrok.Secret{}, errors.New("failed to list secrets"))
				}
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
