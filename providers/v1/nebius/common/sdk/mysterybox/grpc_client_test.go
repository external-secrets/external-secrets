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

package mysterybox

import (
	"bytes"
	"context"
	"errors"
	"testing"

	mbox "github.com/nebius/gosdk/proto/nebius/mysterybox/v1"
	tassert "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	defaultVersion     = "version1"
	alternativeVersion = "version2"
	notFoundError      = "not found"
)

type FakePayloadService struct {
	data map[string]map[string]*mbox.SecretPayload
}

func (f *FakePayloadService) Get(_ context.Context, r *mbox.GetPayloadRequest, _ ...grpc.CallOption) (*mbox.SecretPayload, error) {
	version := extractVersionFromRequest(r.VersionId)
	val, ok := f.data[r.GetSecretId()][version]
	if !ok {
		return nil, errors.New("secret not found")
	}
	return val, nil
}

func (f *FakePayloadService) GetByKey(_ context.Context, r *mbox.GetPayloadByKeyRequest, _ ...grpc.CallOption) (*mbox.SecretPayloadEntry, error) {
	version := extractVersionFromRequest(r.VersionId)
	payload, ok := f.data[r.GetSecretId()][version]
	if !ok {
		return nil, errors.New("secret not found")
	}

	for _, p := range payload.GetData() {
		if p.Key == r.GetKey() {
			return &mbox.SecretPayloadEntry{VersionId: payload.VersionId, Data: &mbox.Payload{Key: p.Key, Payload: p.Payload}}, nil
		}
	}

	return nil, errors.New(notFoundError)
}

func InitFakePayloadService() *FakePayloadService {
	mysteryboxData := map[string]map[string]*mbox.SecretPayload{}
	mysteryboxData["secret1Id"] = make(map[string]*mbox.SecretPayload)
	mysteryboxData["secret1Id"][defaultVersion] = &mbox.SecretPayload{
		VersionId: defaultVersion,
		Data: []*mbox.Payload{
			{
				Key:     "key1",
				Payload: &mbox.Payload_StringValue{StringValue: "test string secret"},
			}, {
				Key:     "key2",
				Payload: &mbox.Payload_BinaryValue{BinaryValue: []byte("test byte secret")},
			},
		},
	}
	mysteryboxData["secret2Id"] = make(map[string]*mbox.SecretPayload)
	mysteryboxData["secret2Id"][defaultVersion] = &mbox.SecretPayload{
		VersionId: defaultVersion,
		Data: []*mbox.Payload{
			{
				Key:     "key3",
				Payload: &mbox.Payload_StringValue{StringValue: "test string secret"},
			},
		},
	}
	mysteryboxData["secret2Id"][alternativeVersion] = &mbox.SecretPayload{
		VersionId: alternativeVersion,
		Data: []*mbox.Payload{
			{
				Key:     "key3",
				Payload: &mbox.Payload_StringValue{StringValue: "test string secret alternative"},
			},
		},
	}
	return &FakePayloadService{
		data: mysteryboxData,
	}
}

func TestGetSecret(t *testing.T) {
	t.Parallel()
	client := &GrpcClient{PayloadService: InitFakePayloadService()}

	tests := []struct {
		name     string
		secretID string
		expected map[string][]byte
		version  string
		wantErr  string
	}{
		{
			name:     "Get secret's payload",
			secretID: "secret1Id",
			expected: map[string][]byte{
				"key1": []byte("test string secret"),
				"key2": []byte("test byte secret"),
			}},
		{
			name:     "Get secret's payload by version",
			secretID: "secret2Id",
			expected: map[string][]byte{
				"key3": []byte("test string secret alternative"),
			},
			version: alternativeVersion,
		},
		{
			name:     "Get secret's payload by version not found",
			secretID: "secret2Id",
			wantErr:  notFoundError,
			version:  "another-version",
		},
		{
			name:     "Not found secret",
			secretID: "nope",
			wantErr:  notFoundError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload, err := client.GetSecret(context.Background(), "token", tt.secretID, tt.version)
			if tt.wantErr != "" {
				tassert.Error(t, err)
				tassert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.version == "" {
				tassert.Equal(t, defaultVersion, payload.VersionID, "Payload version must be default")
			} else {
				tassert.Equal(t, tt.version, payload.VersionID)
			}

			expected := make(map[string][]byte, len(tt.expected))
			for k, v := range tt.expected {
				expected[k] = v
			}

			tassert.Equal(t, len(payload.Entries), len(expected))
			for _, entry := range payload.Entries {
				value, _ := expected[entry.Key]
				if (entry.BinaryValue != nil && bytes.Equal(value, entry.BinaryValue)) || (entry.StringValue != "" && bytes.Equal(value, []byte(entry.StringValue))) {
					delete(expected, entry.Key)
					continue
				}
			}
			tassert.Empty(t, expected, "not all expected entries found: %+v", expected)
		})
	}
}

func TestGetSecretByKey(t *testing.T) {
	t.Parallel()

	client := &GrpcClient{PayloadService: InitFakePayloadService()}

	tests := []struct {
		name     string
		secretID string
		key      string
		wantStr  string
		wantBin  []byte
		wantErr  string
	}{
		{name: "Get secret's string payload", secretID: "secret1Id", key: "key1", wantStr: "test string secret"},
		{name: "Get secret's binary payload", secretID: "secret1Id", key: "key2", wantBin: []byte("test byte secret")},
		{name: "Not found key", secretID: "secret1Id", key: "missing", wantErr: notFoundError},
		{name: "Not found secret", secretID: "nope", key: "any", wantErr: notFoundError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			payload, err := client.GetSecretByKey(context.Background(), "token", tt.secretID, "", tt.key)
			if tt.wantErr != "" {
				tassert.Error(t, err)
				tassert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, payload)
			require.NotNil(t, payload.Entry)

			if tt.wantStr != "" {
				tassert.Nil(t, payload.Entry.BinaryValue)
				tassert.Equal(t, tt.wantStr, payload.Entry.StringValue)
			}
			if tt.wantBin != nil {
				tassert.Empty(t, payload.Entry.StringValue)
				tassert.Equal(t, tt.wantBin, payload.Entry.BinaryValue)
			}
		})
	}
}

func extractVersionFromRequest(requestVersion string) string {
	if requestVersion == "" {
		return defaultVersion
	}
	return requestVersion
}
