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
	"strings"
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

type fakeV2Provider struct {
	getSecretResponse []byte
	getSecretErr      error
}

func (f *fakeV2Provider) GetSecret(context.Context, esv1.ExternalSecretDataRemoteRef, *pb.ProviderReference, string) ([]byte, error) {
	return f.getSecretResponse, f.getSecretErr
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
	t.Run("converts JSON object to byte map", func(t *testing.T) {
		provider := &fakeV2Provider{
			getSecretResponse: []byte(`{"foo":"bar","num":42,"obj":{"nested":"value"}}`),
		}
		client := NewClient(provider, &pb.ProviderReference{Name: "provider"}, "default")

		secretMap, err := client.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "sample"})
		if err != nil {
			t.Fatalf("GetSecretMap() error = %v", err)
		}

		if string(secretMap["foo"]) != "bar" {
			t.Fatalf("expected foo=bar, got %q", string(secretMap["foo"]))
		}
		if string(secretMap["num"]) != "42" {
			t.Fatalf("expected num=42, got %q", string(secretMap["num"]))
		}
		if string(secretMap["obj"]) != `{"nested":"value"}` {
			t.Fatalf("expected obj JSON value, got %q", string(secretMap["obj"]))
		}
	})

	t.Run("returns error for non JSON object payload", func(t *testing.T) {
		provider := &fakeV2Provider{
			getSecretResponse: []byte(`"plain-string"`),
		}
		client := NewClient(provider, &pb.ProviderReference{Name: "provider"}, "default")

		_, err := client.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "sample"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to decode secret as JSON object for extract") {
			t.Fatalf("expected JSON decode error, got %v", err)
		}
	})
}
