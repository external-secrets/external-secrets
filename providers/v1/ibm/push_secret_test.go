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

package ibm

import (
	"errors"
	"reflect"
	"testing"

	sm "github.com/IBM/secrets-manager-go-sdk/v2/secretsmanagerv2"
	corev1 "k8s.io/api/core/v1"

	fakesm "github.com/external-secrets/external-secrets/providers/v1/ibm/fake"
	testingfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
	"github.com/external-secrets/external-secrets/runtime/util/locks"
)

const pushGroupID = "0b3a3d8e-4c0e-4f53-9a8a-1d4a3f2c6e10"

func newPushTestProvider() (*providerIBM, *fakesm.InMemoryClient) {
	backend := fakesm.NewInMemoryClient(map[string]string{"team": pushGroupID})
	return &providerIBM{IBMClient: backend}, backend
}

func newSourceSecret(data map[string]string) *corev1.Secret {
	secret := &corev1.Secret{Data: map[string][]byte{}}
	for k, v := range data {
		secret.Data[k] = []byte(v)
	}
	return secret
}

func newArbitrarySecret(id, name, payload string, managed bool) *sm.ArbitrarySecret {
	secret := &sm.ArbitrarySecret{
		ID: &id, Name: &name, Payload: &payload, VersionsTotal: new(int64(1)),
		SecretType: new(sm.Secret_SecretType_Arbitrary), SecretGroupID: new("default"),
	}
	if managed {
		secret.CustomMetadata = map[string]any{managedByKey: managedByValue}
	}
	return secret
}

func newKVSecret(id, name string, data map[string]any, managed bool) *sm.KVSecret {
	secret := &sm.KVSecret{
		ID: &id, Name: &name, Data: data, VersionsTotal: new(int64(1)),
		SecretType: new(sm.Secret_SecretType_Kv), SecretGroupID: new(pushGroupID),
	}
	if managed {
		secret.CustomMetadata = map[string]any{managedByKey: managedByValue}
	}
	return secret
}

func TestPushSecret(t *testing.T) {
	source := newSourceSecret(map[string]string{"token": "s3cr3t", "user": "admin"})

	t.Run("creates an arbitrary secret in the default group", func(t *testing.T) {
		ibm, backend := newPushTestProvider()
		err := ibm.PushSecret(t.Context(), source, testingfake.PushSecretData{SecretKey: "token", RemoteKey: "default/arbitrary/app-token"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, ok := backend.Find("app-token", sm.Secret_SecretType_Arbitrary, "default").(*sm.ArbitrarySecret)
		if !ok {
			t.Fatal("secret was not created")
		}
		if *got.Payload != "s3cr3t" || !isManagedByESO(got.CustomMetadata) {
			t.Errorf("unexpected secret: payload %q, custom metadata %v", *got.Payload, got.CustomMetadata)
		}
	})

	t.Run("creates a kv secret in a custom group from one property", func(t *testing.T) {
		ibm, backend := newPushTestProvider()
		err := ibm.PushSecret(t.Context(), source, testingfake.PushSecretData{SecretKey: "token", RemoteKey: "team/kv/app", Property: "api.token"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, ok := backend.Find("app", sm.Secret_SecretType_Kv, pushGroupID).(*sm.KVSecret)
		if !ok {
			t.Fatal("secret was not created")
		}
		if want := map[string]any{"api.token": "s3cr3t"}; !reflect.DeepEqual(got.Data, want) {
			t.Errorf("unexpected data: got %v, want %v", got.Data, want)
		}
	})

	t.Run("pushes the whole Kubernetes secret to a kv secret when no secret key is set", func(t *testing.T) {
		ibm, backend := newPushTestProvider()
		err := ibm.PushSecret(t.Context(), source, testingfake.PushSecretData{RemoteKey: "team/kv/app"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := backend.Find("app", sm.Secret_SecretType_Kv, pushGroupID).(*sm.KVSecret)
		if want := map[string]any{"token": "s3cr3t", "user": "admin"}; !reflect.DeepEqual(got.Data, want) {
			t.Errorf("unexpected data: got %v, want %v", got.Data, want)
		}
	})

	t.Run("merges a property into an existing kv secret", func(t *testing.T) {
		ibm, backend := newPushTestProvider()
		backend.Put(newKVSecret(secretUUID, "app", map[string]any{"other": "kept", "token": "old"}, true))
		err := ibm.PushSecret(t.Context(), source, testingfake.PushSecretData{SecretKey: "token", RemoteKey: "team/kv/app", Property: "token"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := backend.Find("app", sm.Secret_SecretType_Kv, pushGroupID).(*sm.KVSecret)
		if want := map[string]any{"other": "kept", "token": "s3cr3t"}; !reflect.DeepEqual(got.Data, want) {
			t.Errorf("unexpected data: got %v, want %v", got.Data, want)
		}
		if *got.VersionsTotal != 2 {
			t.Errorf("expected a new version, got %d versions", *got.VersionsTotal)
		}
	})

	t.Run("leaves the secret alone when the value is unchanged", func(t *testing.T) {
		ibm, backend := newPushTestProvider()
		backend.Put(newArbitrarySecret(secretUUID, "app-token", "s3cr3t", true))
		err := ibm.PushSecret(t.Context(), source, testingfake.PushSecretData{SecretKey: "token", RemoteKey: "default/arbitrary/app-token"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := backend.Find("app-token", sm.Secret_SecretType_Arbitrary, "default").(*sm.ArbitrarySecret)
		if *got.VersionsTotal != 1 {
			t.Errorf("expected no new version, got %d versions", *got.VersionsTotal)
		}
	})

	t.Run("updates an existing secret by ID", func(t *testing.T) {
		ibm, backend := newPushTestProvider()
		backend.Put(newArbitrarySecret(secretUUID, "app-token", "old", true))
		err := ibm.PushSecret(t.Context(), source, testingfake.PushSecretData{SecretKey: "token", RemoteKey: secretUUID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := backend.Find("app-token", sm.Secret_SecretType_Arbitrary, "default").(*sm.ArbitrarySecret)
		if *got.Payload != "s3cr3t" {
			t.Errorf("unexpected payload %q", *got.Payload)
		}
	})

	t.Run("refuses to overwrite a secret not managed by external-secrets", func(t *testing.T) {
		ibm, backend := newPushTestProvider()
		backend.Put(newArbitrarySecret(secretUUID, "app-token", "old", false))
		err := ibm.PushSecret(t.Context(), source, testingfake.PushSecretData{SecretKey: "token", RemoteKey: "default/arbitrary/app-token"})
		if !errors.Is(err, errNotManagedByESO) {
			t.Fatalf("expected errNotManagedByESO, got %v", err)
		}
		got := backend.Find("app-token", sm.Secret_SecretType_Arbitrary, "default").(*sm.ArbitrarySecret)
		if *got.Payload != "old" {
			t.Errorf("secret was overwritten with %q", *got.Payload)
		}
	})

	t.Run("retries later while another push holds the secret", func(t *testing.T) {
		ibm, backend := newPushTestProvider()
		unlock, err := locks.TryLock(ProviderIBMSM, "default/arbitrary/app-token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer unlock()
		err = ibm.PushSecret(t.Context(), source, testingfake.PushSecretData{SecretKey: "token", RemoteKey: "default/arbitrary/app-token"})
		if !errors.Is(err, locks.ErrConflict) {
			t.Fatalf("expected locks.ErrConflict, got %v", err)
		}
		if backend.Find("app-token", sm.Secret_SecretType_Arbitrary, "default") != nil {
			t.Error("secret was created while locked")
		}
	})

	errorCases := []struct {
		name      string
		data      testingfake.PushSecretData
		wantError string
	}{
		{
			name:      "refuses to create a secret by ID",
			data:      testingfake.PushSecretData{SecretKey: "token", RemoteKey: secretUUID},
			wantError: "cannot be created by ID",
		},
		{
			name:      "rejects a property for arbitrary secrets",
			data:      testingfake.PushSecretData{SecretKey: "token", RemoteKey: "default/arbitrary/app", Property: "token"},
			wantError: "property is not supported for secret type arbitrary",
		},
		{
			name:      "rejects secret types that cannot be pushed",
			data:      testingfake.PushSecretData{SecretKey: "token", RemoteKey: "default/username_password/app"},
			wantError: "push is not supported for secret type username_password",
		},
		{
			name:      "rejects a kv value that is not a JSON object",
			data:      testingfake.PushSecretData{SecretKey: "token", RemoteKey: "default/kv/app"},
			wantError: "kv secrets need a JSON object",
		},
		{
			name:      "reports an unknown secret group",
			data:      testingfake.PushSecretData{SecretKey: "token", RemoteKey: "nope/arbitrary/app"},
			wantError: "secret group nope not found",
		},
	}
	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			ibm, _ := newPushTestProvider()
			err := ibm.PushSecret(t.Context(), source, tc.data)
			if !ErrorContains(err, tc.wantError) {
				t.Errorf("unexpected error: got %v, want %q", err, tc.wantError)
			}
		})
	}
}

func TestDeleteSecret(t *testing.T) {
	t.Run("deletes a managed secret", func(t *testing.T) {
		ibm, backend := newPushTestProvider()
		backend.Put(newArbitrarySecret(secretUUID, "app-token", "s3cr3t", true))
		if err := ibm.DeleteSecret(t.Context(), testingfake.PushSecretData{RemoteKey: "default/arbitrary/app-token"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if backend.Find("app-token", sm.Secret_SecretType_Arbitrary, "default") != nil {
			t.Error("secret was not deleted")
		}
	})

	t.Run("removes one property from a kv secret and keeps the rest", func(t *testing.T) {
		ibm, backend := newPushTestProvider()
		backend.Put(newKVSecret(secretUUID, "app", map[string]any{"other": "kept", "token": "s3cr3t"}, true))
		if err := ibm.DeleteSecret(t.Context(), testingfake.PushSecretData{RemoteKey: "team/kv/app", Property: "token"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, ok := backend.Find("app", sm.Secret_SecretType_Kv, pushGroupID).(*sm.KVSecret)
		if !ok {
			t.Fatal("secret was deleted")
		}
		if want := map[string]any{"other": "kept"}; !reflect.DeepEqual(got.Data, want) {
			t.Errorf("unexpected data: got %v, want %v", got.Data, want)
		}
	})

	t.Run("deletes a kv secret when its last property is removed", func(t *testing.T) {
		ibm, backend := newPushTestProvider()
		backend.Put(newKVSecret(secretUUID, "app", map[string]any{"token": "s3cr3t"}, true))
		if err := ibm.DeleteSecret(t.Context(), testingfake.PushSecretData{RemoteKey: "team/kv/app", Property: "token"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if backend.Find("app", sm.Secret_SecretType_Kv, pushGroupID) != nil {
			t.Error("secret was not deleted")
		}
	})

	t.Run("leaves secrets not managed by external-secrets alone", func(t *testing.T) {
		ibm, backend := newPushTestProvider()
		backend.Put(newArbitrarySecret(secretUUID, "app-token", "s3cr3t", false))
		if err := ibm.DeleteSecret(t.Context(), testingfake.PushSecretData{RemoteKey: "default/arbitrary/app-token"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if backend.Find("app-token", sm.Secret_SecretType_Arbitrary, "default") == nil {
			t.Error("unmanaged secret was deleted")
		}
	})

	t.Run("ignores a missing secret", func(t *testing.T) {
		ibm, _ := newPushTestProvider()
		if err := ibm.DeleteSecret(t.Context(), testingfake.PushSecretData{RemoteKey: "default/arbitrary/missing"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestSecretExists(t *testing.T) {
	ibm, backend := newPushTestProvider()
	backend.Put(newKVSecret(secretUUID, "app", map[string]any{"token": "s3cr3t"}, true))

	t.Run("finds an existing secret", func(t *testing.T) {
		exists, err := ibm.SecretExists(t.Context(), testingfake.PushSecretData{RemoteKey: "team/kv/app"})
		if err != nil || !exists {
			t.Errorf("got exists=%v, err=%v; want true, nil", exists, err)
		}
	})

	t.Run("reports a missing secret", func(t *testing.T) {
		exists, err := ibm.SecretExists(t.Context(), testingfake.PushSecretData{RemoteKey: "team/kv/missing"})
		if err != nil || exists {
			t.Errorf("got exists=%v, err=%v; want false, nil", exists, err)
		}
	})

	t.Run("checks the property of a kv secret", func(t *testing.T) {
		exists, err := ibm.SecretExists(t.Context(), testingfake.PushSecretData{RemoteKey: "team/kv/app", Property: "token"})
		if err != nil || !exists {
			t.Errorf("present property: got exists=%v, err=%v; want true, nil", exists, err)
		}
		exists, err = ibm.SecretExists(t.Context(), testingfake.PushSecretData{RemoteKey: "team/kv/app", Property: "user"})
		if err != nil || exists {
			t.Errorf("missing property: got exists=%v, err=%v; want false, nil", exists, err)
		}
	})
}
