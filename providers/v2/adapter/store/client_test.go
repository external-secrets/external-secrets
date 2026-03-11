// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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

package store

import (
	"context"
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

type fakeV2Provider struct {
	getSecretMapResponse map[string][]byte
	getSecretMapErr      error
}

func (f *fakeV2Provider) GetSecret(context.Context, esv1.ExternalSecretDataRemoteRef, *pb.ProviderReference, string) ([]byte, error) {
	return nil, nil
}

func (f *fakeV2Provider) GetSecretMap(context.Context, esv1.ExternalSecretDataRemoteRef, *pb.ProviderReference, string) (map[string][]byte, error) {
	return f.getSecretMapResponse, f.getSecretMapErr
}

func (f *fakeV2Provider) GetAllSecrets(context.Context, esv1.ExternalSecretFind, *pb.ProviderReference, string) (map[string][]byte, error) {
	return nil, nil
}

func (f *fakeV2Provider) PushSecret(context.Context, map[string][]byte, *pb.PushSecretData, *pb.ProviderReference, string) error {
	return nil
}

func (f *fakeV2Provider) DeleteSecret(context.Context, *pb.PushSecretRemoteRef, *pb.ProviderReference, string) error {
	return nil
}

func (f *fakeV2Provider) SecretExists(context.Context, *pb.PushSecretRemoteRef, *pb.ProviderReference, string) (bool, error) {
	return false, nil
}

func (f *fakeV2Provider) Validate(context.Context, *pb.ProviderReference, string) error {
	return nil
}

func (f *fakeV2Provider) Capabilities(context.Context, *pb.ProviderReference, string) (pb.SecretStoreCapabilities, error) {
	return pb.SecretStoreCapabilities_READ_WRITE, nil
}

func (f *fakeV2Provider) Close(context.Context) error {
	return nil
}

func TestGetSecretMap(t *testing.T) {
	t.Run("delegates to provider GetSecretMap", func(t *testing.T) {
		expected := map[string][]byte{
			"foo": []byte("bar"),
			"baz": []byte("qux"),
		}
		provider := &fakeV2Provider{
			getSecretMapResponse: expected,
		}
		client := NewClient(provider, &pb.ProviderReference{Name: "provider"}, "default")

		secretMap, err := client.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "sample"})
		if err != nil {
			t.Fatalf("GetSecretMap() error = %v", err)
		}
		if len(secretMap) != len(expected) {
			t.Fatalf("expected %d keys, got %d", len(expected), len(secretMap))
		}
		if string(secretMap["foo"]) != "bar" || string(secretMap["baz"]) != "qux" {
			t.Fatalf("unexpected secret map: %#v", secretMap)
		}
	})
}
