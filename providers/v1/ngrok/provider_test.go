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
	"fmt"
	"testing"

	"github.com/ngrok/ngrok-api-go/v7"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/ngrok/fake"
)

func newTestClusterSecretStore(provider *esv1.SecretStoreProvider) esv1.GenericStore {
	return &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterSecretStore",
		},
		Spec: esv1.SecretStoreSpec{
			Provider: provider,
		},
	}
}

func newTestNgrokClusterSecretStore(ngrokProv *esv1.NgrokProvider) esv1.GenericStore {
	return newTestClusterSecretStore(&esv1.SecretStoreProvider{
		Ngrok: ngrokProv,
	})
}

func newTestSecretStore(provider *esv1.SecretStoreProvider) esv1.GenericStore {
	return &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: "SecretStore",
		},
		Spec: esv1.SecretStoreSpec{
			Provider: provider,
		},
	}
}

func newTestNgrokSecretStore(ngrokProv *esv1.NgrokProvider) esv1.GenericStore {
	return newTestSecretStore(&esv1.SecretStoreProvider{
		Ngrok: ngrokProv,
	})
}

func newNgrokAPICredentials(name, namespace, apiKey string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"API_KEY": []byte(apiKey),
		},
	}
}

var _ = Describe("Provider", func() {
	var (
		provider *Provider
	)

	BeforeEach(func() {
		provider = &Provider{}
	})

	Describe("Capabilities", func() {
		It("should return write-only capability", func() {
			cap := provider.Capabilities()
			Expect(cap).To(Equal(esv1.SecretStoreWriteOnly))
		})
	})

	Describe("NewClient", func() {
		var (
			store            esv1.GenericStore
			ngrokStore       *fake.Store
			namespace        string
			kubeClient       kubeClient.Client
			ngrokCredentials *corev1.Secret
			vaultName        string

			// Injected errors
			vaultListErr error

			// Outputs
			err    error
			client esv1.SecretsClient
		)

		BeforeEach(func() {
			namespace = "default"
			vaultName = "vault-" + fake.GenerateRandomString(5)
			ngrokCredentials = newNgrokAPICredentials("ngrok-credentials", namespace, "secret-api-key")
			kubeClient = clientfake.NewClientBuilder().WithObjects(ngrokCredentials).Build()
			ngrokStore = fake.NewStore()
			vaultListErr = nil
		})

		JustBeforeEach(func() {
			getVaultsClient = func(_ *ngrok.ClientConfig) VaultClient {
				return ngrokStore.VaultClient().WithListError(vaultListErr)
			}
			getSecretsClient = func(_ *ngrok.ClientConfig) SecretsClient {
				return ngrokStore.SecretsClient()
			}
			client, err = provider.NewClient(GinkgoT().Context(), store, kubeClient, namespace)
		})

		Context("SecretStore", func() {
			When("the secret does not exist", func() {
				BeforeEach(func() {
					store = newTestNgrokSecretStore(&esv1.NgrokProvider{
						Vault: esv1.NgrokVault{
							Name: vaultName,
						},
						Auth: esv1.NgrokAuth{
							APIKey: &esv1.NgrokProviderSecretRef{
								SecretRef: &v1.SecretKeySelector{
									Key:  "API_KEY",
									Name: "non-existent-secret",
								},
							},
						},
					})
				})

				It("should return an error that the secret does not exist", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("secrets \"non-existent-secret\" not found"))
					Expect(client).To(BeNil())
				})
			})

			When("the store is valid", func() {
				BeforeEach(func() {
					store = newTestNgrokSecretStore(&esv1.NgrokProvider{
						Vault: esv1.NgrokVault{
							Name: vaultName,
						},
						Auth: esv1.NgrokAuth{
							APIKey: &esv1.NgrokProviderSecretRef{
								SecretRef: &v1.SecretKeySelector{
									Key:  "API_KEY",
									Name: ngrokCredentials.Name,
								},
							},
						},
					})
				})

				When("the vault does not exist", func() {
					It("Should return an error that the vault is not found", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("vault %q not found", vaultName)))
						Expect(client).To(BeNil())
					})
				})

				When("the vault exists", func() {
					BeforeEach(func() {
						_, createErr := ngrokStore.CreateVault(&ngrok.VaultCreate{
							Name: vaultName,
						})
						Expect(createErr).To(BeNil())
					})

					It("should not return an error", func() {
						Expect(err).To(BeNil())
					})

					It("should return a non-nil client", func() {
						Expect(client).NotTo(BeNil())
					})
				})

				When("there is an error listing vaults", func() {
					BeforeEach(func() {
						vaultListErr = fmt.Errorf("some error listing vaults")
					})

					It("should return the list error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("some error listing vaults"))
						Expect(client).To(BeNil())
					})
				})
			})
		})

		Context("ClusterSecretStore", func() {
			When("the store does not specify a namespace", func() {
				BeforeEach(func() {
					store = newTestNgrokClusterSecretStore(&esv1.NgrokProvider{
						Vault: esv1.NgrokVault{
							Name: vaultName,
						},
						Auth: esv1.NgrokAuth{
							APIKey: &esv1.NgrokProviderSecretRef{
								SecretRef: &v1.SecretKeySelector{
									Key:  "API_KEY",
									Name: ngrokCredentials.Name,
								},
							},
						},
					})
				})

				It("should return an error that the cluster store requires a namespace", func() {
					Expect(err).To(MatchError(errClusterStoreRequiresNamespace))
					Expect(client).To(BeNil())
				})
			})

			When("the secret does not exist", func() {
				BeforeEach(func() {
					store = newTestNgrokClusterSecretStore(&esv1.NgrokProvider{
						Vault: esv1.NgrokVault{
							Name: vaultName,
						},
						Auth: esv1.NgrokAuth{
							APIKey: &esv1.NgrokProviderSecretRef{
								SecretRef: &v1.SecretKeySelector{
									Key:       "API_KEY",
									Name:      "non-existent-secret",
									Namespace: ptr.To("some-other-namespace"),
								},
							},
						},
					})
				})

				It("should return an error that the secret does not exist", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("secrets \"non-existent-secret\" not found"))
					Expect(client).To(BeNil())
				})
			})

			When("the store is valid", func() {
				BeforeEach(func() {
					store = newTestNgrokClusterSecretStore(&esv1.NgrokProvider{
						Vault: esv1.NgrokVault{
							Name: vaultName,
						},
						Auth: esv1.NgrokAuth{
							APIKey: &esv1.NgrokProviderSecretRef{
								SecretRef: &v1.SecretKeySelector{
									Key:       "API_KEY",
									Name:      ngrokCredentials.Name,
									Namespace: ptr.To(namespace),
								},
							},
						},
					})
				})

				When("the vault does not exist", func() {
					It("Should return an error that the vault is not found", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("vault %q not found", vaultName)))
						Expect(client).To(BeNil())
					})
				})

				When("the vault exists", func() {
					BeforeEach(func() {
						_, createErr := ngrokStore.CreateVault(&ngrok.VaultCreate{
							Name: vaultName,
						})
						Expect(createErr).To(BeNil())
					})

					It("should not return an error", func() {
						Expect(err).To(BeNil())
					})

					It("should return a non-nil client", func() {
						Expect(client).NotTo(BeNil())
					})
				})
			})
		})
	})

	Describe("ValidateStore", func() {
		var (
			store esv1.GenericStore

			err      error
			warnings admission.Warnings
		)

		JustBeforeEach(func() {
			warnings, err = provider.ValidateStore(store)
		})

		When("the store is nil", func() {
			BeforeEach(func() { store = nil })

			It("Should return an invalid store error", func() {
				Expect(err).To(MatchError(errInvalidStore))
				Expect(warnings).To(BeNil())
			})
		})

		When("the provider is nil", func() {
			BeforeEach(func() { store = newTestSecretStore(nil) })

			It("Should return an invalid ngrok provider error", func() {
				Expect(err).To(MatchError(errInvalidStoreProv))
				Expect(warnings).To(BeNil())
			})
		})

		When("the ngrok provider is nil", func() {
			BeforeEach(func() { store = newTestNgrokSecretStore(nil) })

			It("Should return an invalid ngrok provider error", func() {
				Expect(err).To(MatchError(errInvalidNgrokProv))
				Expect(warnings).To(BeNil())
			})
		})

		When("the API URL is invalid", func() {
			BeforeEach(func() {
				store = newTestNgrokSecretStore(&esv1.NgrokProvider{
					APIURL: "http://example.com/path\n",
				})
			})

			It("Should return an invalid API URL error", func() {
				Expect(err).To(MatchError(errInvalidAPIURL))
				Expect(warnings).To(BeNil())
			})
		})

		When("the auth APIKey is missing", func() {
			BeforeEach(func() {
				store = newTestNgrokSecretStore(&esv1.NgrokProvider{
					Vault: esv1.NgrokVault{
						Name: "test-vault",
					},
					Auth: esv1.NgrokAuth{
						APIKey: nil,
					},
				})
			})

			It("Should return an invalid auth APIKey required error", func() {
				Expect(err).To(MatchError(errInvalidAuthAPIKeyRequired))
				Expect(warnings).To(BeNil())
			})
		})

		When("the vault name is missing", func() {
			BeforeEach(func() {
				store = newTestNgrokSecretStore(&esv1.NgrokProvider{
					Auth: esv1.NgrokAuth{
						APIKey: &esv1.NgrokProviderSecretRef{
							SecretRef: &v1.SecretKeySelector{
								Key:  "apiKey",
								Name: "ngrok-credentials",
							},
						},
					},
					Vault: esv1.NgrokVault{},
				})
			})

			It("Should return a missing vault name error", func() {
				Expect(err).To(MatchError(errMissingVaultName))
				Expect(warnings).To(BeNil())
			})
		})

		When("the store is valid", func() {
			BeforeEach(func() {
				store = newTestNgrokSecretStore(&esv1.NgrokProvider{
					Vault: esv1.NgrokVault{
						Name: "test-vault",
					},
					Auth: esv1.NgrokAuth{
						APIKey: &esv1.NgrokProviderSecretRef{
							SecretRef: &v1.SecretKeySelector{
								Key:  "apiKey",
								Name: "ngrok-credentials",
							},
						},
					},
				})
			})

			It("Should not return an error", func() {
				Expect(err).To(BeNil())
				Expect(warnings).To(BeNil())
			})
		})
	})

	// Add more Ginkgo tests here for ValidateStore, NewClient, etc.
})

func TestNgrokProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ngrok Provider Suite")
}
