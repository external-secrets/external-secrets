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
package previder

import (
	"context"
	"reflect"
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestSecretManagerCapabilities(t *testing.T) {
	previderProvider := &SecretManager{}
	if previderProvider.Capabilities() != esv1.SecretStoreReadOnly {
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
	path := "some/path"
	for _, tc := range []struct {
		name      string
		ref       esv1.ExternalSecretFind
		want      map[string][]byte
		tokenType string
		wantErr   bool
	}{
		{
			name: "no criteria returns every secret",
			ref:  esv1.ExternalSecretFind{},
			want: map[string][]byte{
				"secret1": []byte("secret1content"),
				"secret2": []byte("secret2content"),
				"other1":  []byte("other1content"),
			},
		},
		{
			name: "regexp selects a subset",
			ref:  esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "^secret"}},
			want: map[string][]byte{
				"secret1": []byte("secret1content"),
				"secret2": []byte("secret2content"),
			},
		},
		{
			name: "regexp matching nothing returns an empty map",
			ref:  esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "^nomatch"}},
			want: map[string][]byte{},
		},
		{
			name:    "invalid regexp is an error",
			ref:     esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "[unterminated"}},
			wantErr: true,
		},
		{
			name:      "a ReadOnly token cannot enumerate",
			ref:       esv1.ExternalSecretFind{},
			tokenType: "ReadOnly",
			wantErr:   true,
		},
		{
			name:      "an EnvironmentAdmin token cannot enumerate",
			ref:       esv1.ExternalSecretFind{},
			tokenType: "EnvironmentAdmin",
			wantErr:   true,
		},
		{
			name:    "tags are not supported",
			ref:     esv1.ExternalSecretFind{Tags: map[string]string{"env": "prod"}},
			wantErr: true,
		},
		{
			name:    "path is not supported",
			ref:     esv1.ExternalSecretFind{Path: &path},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tokenType := tc.tokenType
			if tokenType == "" {
				tokenType = "ReadWrite"
			}
			previderProvider := &SecretManager{VaultClient: &PreviderVaultFakeClient{}, TokenType: tokenType}
			got, err := previderProvider.GetAllSecrets(context.Background(), tc.ref)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got result %v", got)
				}
				if got != nil {
					t.Errorf("expected no result alongside the error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSecretManagerGetSecret(t *testing.T) {
	previderProvider := &SecretManager{VaultClient: &PreviderVaultFakeClient{}}
	ctx := context.Background()
	ref := esv1.ExternalSecretDataRemoteRef{Key: "secret1"}
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
	ref := esv1.ExternalSecretDataRemoteRef{Key: "secret3"}
	_, err := previderProvider.GetSecret(ctx, ref)
	if err == nil {
		t.Errorf("Secret found while non were expected")
	}
}

func TestSecretManagerGetSecretMap(t *testing.T) {
	previderProvider := &SecretManager{VaultClient: &PreviderVaultFakeClient{}}
	ctx := context.Background()
	key := "secret1"

	ref := esv1.ExternalSecretDataRemoteRef{Key: key}
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
	if err != nil || validate != esv1.ValidationResultReady {
		t.Errorf("Could not validate")
	}
}

func TestSecretManagerValidateStore(t *testing.T) {
	previderProvider := &SecretManager{}
	store := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Previder: &esv1.PreviderProvider{
					Auth: esv1.PreviderAuth{
						SecretRef: &esv1.PreviderAuthSecretRef{
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

	store = &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Previder: &esv1.PreviderProvider{
					Auth: esv1.PreviderAuth{
						SecretRef: &esv1.PreviderAuthSecretRef{
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

	store = &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Previder: &esv1.PreviderProvider{
					Auth: esv1.PreviderAuth{
						SecretRef: &esv1.PreviderAuthSecretRef{
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
