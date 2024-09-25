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
package previder

import (
	"context"
	"testing"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestSecretManagerCapabilities(t *testing.T) {
	previderProvider := &SecretManager{}
	if previderProvider.Capabilities() != esv1beta1.SecretStoreReadOnly {
		t.Errorf("Store does not return correct value for capabilities")
	}
}

func TestSecretManagerClose(t *testing.T) {
	previderProvider := &SecretManager{}
	ctx := context.Background()
	if previderProvider.Close(ctx) != nil {
		t.Errorf("Store close acts different than expected")
	}
}

func TestSecretManagerGetAllSecrets(t *testing.T) {
	previderProvider := &SecretManager{}
	ctx := context.Background()
	ref := esv1beta1.ExternalSecretFind{}
	result, err := previderProvider.GetAllSecrets(ctx, ref)
	if result != nil || err == nil {
		t.Errorf("Store close acts different than expected")
	}
}

func TestSecretManagerGetSecret(t *testing.T) {
	previderProvider := &SecretManager{VaultClient: &PreviderVaultFakeClient{}}
	ctx := context.Background()
	ref := esv1beta1.ExternalSecretDataRemoteRef{Key: "secret1"}
	returnedSecret, err := previderProvider.GetSecret(ctx, ref)
	if err != nil {
		t.Errorf("Secret not found")
	}
	if string(returnedSecret) != "secret1content" {
		t.Errorf("Wrong secret returned")
	}
}

func TestSecretManagerGetSecretNotExisting(t *testing.T) {
	previderProvider := &SecretManager{VaultClient: &PreviderVaultFakeClient{}}
	ctx := context.Background()
	ref := esv1beta1.ExternalSecretDataRemoteRef{Key: "secret3"}
	_, err := previderProvider.GetSecret(ctx, ref)
	if err == nil {
		t.Errorf("Secret found while non were expected")
	}
}

func TestSecretManagerGetSecretMap(t *testing.T) {
	previderProvider := &SecretManager{VaultClient: &PreviderVaultFakeClient{}}
	ctx := context.Background()
	key := "secret1"

	ref := esv1beta1.ExternalSecretDataRemoteRef{Key: key}
	returnedSecret, err := previderProvider.GetSecretMap(ctx, ref)
	if err != nil {
		t.Errorf("Secret not found")
	}
	if value, ok := returnedSecret[key]; !ok || string(value) != "secret1content" {
		t.Errorf("Key not found or wrong secret returned")
	}
}

func TestSecretManagerValidate(t *testing.T) {
	previderProvider := &SecretManager{VaultClient: &PreviderVaultFakeClient{}}
	validate, err := previderProvider.Validate()
	if err != nil || validate != esv1beta1.ValidationResultReady {
		t.Errorf("Could not validate")
	}
}

func TestSecretManagerValidateStore(t *testing.T) {
	previderProvider := &SecretManager{}
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Previder: &esv1beta1.PreviderProvider{
					Auth: esv1beta1.PreviderAuth{
						SecretRef: &esv1beta1.PreviderAuthSecretRef{
							AccessToken: v1.SecretKeySelector{
								Name: "token",
								Key:  "key",
							},
						},
					},
				},
			},
		},
	}

	result, err := previderProvider.ValidateStore(store)
	if result != nil || err != nil {
		t.Errorf("Store Validation acts different than expected")
	}

	store = &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Previder: &esv1beta1.PreviderProvider{
					Auth: esv1beta1.PreviderAuth{
						SecretRef: &esv1beta1.PreviderAuthSecretRef{
							AccessToken: v1.SecretKeySelector{
								Name: "token",
							},
						},
					},
				},
			},
		},
	}

	result, err = previderProvider.ValidateStore(store)
	if result != nil || err == nil {
		t.Errorf("Store Validation key is not checked")
	}

	store = &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Previder: &esv1beta1.PreviderProvider{
					Auth: esv1beta1.PreviderAuth{
						SecretRef: &esv1beta1.PreviderAuthSecretRef{
							AccessToken: v1.SecretKeySelector{
								Key: "token",
							},
						},
					},
				},
			},
		},
	}

	result, err = previderProvider.ValidateStore(store)
	if result != nil || err == nil {
		t.Errorf("Store Validation name is not checked")
	}
}
