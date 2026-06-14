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
package passworddepot

import (
	"context"
	"encoding/json"
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type staticPasswordDepotClient struct {
	secret SecretEntry
}

func (s staticPasswordDepotClient) GetSecret(_, _ string) (SecretEntry, error) {
	return s.secret, nil
}

func TestPasswordDepotGetSecretWithoutProperty(t *testing.T) {
	provider := &PasswordDepot{
		client: staticPasswordDepotClient{
			secret: SecretEntry{
				Name:  mySecret,
				Login: "user",
				Pass:  "secret",
			},
		},
		database: someDB,
	}

	got, err := provider.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: mySecret})
	if err != nil {
		t.Fatalf("expected full secret JSON, got error: %v", err)
	}

	var decoded map[string]string
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("expected JSON encoded secret, got %q: %v", string(got), err)
	}
	if decoded["name"] != mySecret {
		t.Fatalf("unexpected name: %q", decoded["name"])
	}
	if decoded["login"] != "user" {
		t.Fatalf("unexpected login: %q", decoded["login"])
	}
	if decoded["pass"] != "secret" {
		t.Fatalf("unexpected pass: %q", decoded["pass"])
	}
}

func TestPasswordDepotGetSecretMapUninitialized(t *testing.T) {
	provider := &PasswordDepot{}

	_, err := provider.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: mySecret})
	if err == nil {
		t.Fatal("expected error for uninitialized provider")
	}
	if err.Error() != errUninitalizedPasswordDepotProvider {
		t.Fatalf("unexpected error: %v", err)
	}
}
