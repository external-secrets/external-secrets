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

package secretmanager

import (
	"context"
	"errors"
	"fmt"
	"testing"

	smsV2 "github.com/cloudru-tech/secret-manager-sdk/api/v2"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/cloudru/secretmanager/fake"
)

const (
	keyID        = "50000000-4000-3000-2000-100000000001"
	anotherKeyID = "50000000-4000-3000-2000-100000000002"
)

var (
	errInvalidSecretID = errors.New("secret id is invalid")
	errInternal        = errors.New("internal server error")
)

func TestClientGetSecret(t *testing.T) {
	tests := []struct {
		name        string
		ref         esv1beta1.ExternalSecretDataRemoteRef
		setup       func(mock *fake.MockSecretProvider)
		wantPayload []byte
		wantErr     error
	}{
		{
			name: "success",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     uuid.NewString(),
				Version: "1",
			},
			setup: func(mock *fake.MockSecretProvider) {
				mock.MockAccessSecretVersion([]byte("secret"), nil)
			},
			wantPayload: []byte("secret"),
			wantErr:     nil,
		},
		{
			name: "success_named_secret",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "very_secret",
				Version: "1",
			},
			setup: func(mock *fake.MockSecretProvider) {
				// before it should find the secret by the name.
				mock.MockListSecrets([]*smsV2.Secret{
					{
						Id:   keyID,
						Name: "very_secret",
					},
				}, nil)
				mock.MockAccessSecretVersion([]byte("secret"), nil)
			},
			wantPayload: []byte("secret"),
			wantErr:     nil,
		},
		{
			name: "success_multikv",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      uuid.NewString(),
				Version:  "1",
				Property: "another.secret",
			},
			setup: func(mock *fake.MockSecretProvider) {
				mock.MockAccessSecretVersion([]byte(`{"some": "value", "another": {"secret": "another_value"}}`), nil)
			},
			wantPayload: []byte("another_value"),
			wantErr:     nil,
		},
		{
			name: "error_access_secret",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     uuid.NewString(),
				Version: "1",
			},
			setup: func(mock *fake.MockSecretProvider) {
				mock.MockAccessSecretVersion(nil, errInvalidSecretID)
			},
			wantPayload: nil,
			wantErr:     errInvalidSecretID,
		},
		{
			name: "error_access_named_secret",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "very_secret",
				Version: "1",
			},
			setup: func(mock *fake.MockSecretProvider) {
				mock.MockAccessSecretVersionPath(nil, errInternal)
			},
			wantPayload: nil,
			wantErr:     errInternal,
		},
		{
			name: "error_access_named_secret:invalid_version",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     "very_secret",
				Version: "hello",
			},
			setup: func(mock *fake.MockSecretProvider) {
			},
			wantPayload: nil,
			wantErr:     ErrInvalidSecretVersion,
		},
		{
			name: "error_multikv:invalid_json",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      keyID,
				Version:  "1",
				Property: "some",
			},
			setup: func(mock *fake.MockSecretProvider) {
				mock.MockAccessSecretVersion([]byte(`"some": "value"`), nil)
			},
			wantPayload: nil,
			wantErr:     fmt.Errorf(`expecting the secret %q in JSON format, could not access property "some"`, keyID),
		},
		{
			name: "error_multikv:not_found",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      keyID,
				Version:  "1",
				Property: "unexpected",
			},
			setup: func(mock *fake.MockSecretProvider) {
				mock.MockAccessSecretVersion([]byte(`{"some": "value"}`), nil)
			},
			wantPayload: nil,
			wantErr:     fmt.Errorf(`the requested property "unexpected" does not exist in secret %q`, keyID),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &fake.MockSecretProvider{}
			tt.setup(mock)
			c := &Client{
				apiClient: mock,
				projectID: "123",
			}

			got, gotErr := c.GetSecret(context.Background(), tt.ref)

			tassert.Equal(t, tt.wantPayload, got)
			tassert.Equal(t, tt.wantErr, gotErr)
		})
	}
}

func TestClientGetSecretMap(t *testing.T) {
	tests := []struct {
		name        string
		ref         esv1beta1.ExternalSecretDataRemoteRef
		setup       func(mock *fake.MockSecretProvider)
		wantPayload map[string][]byte
		wantErr     error
	}{
		{
			name: "success",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     keyID,
				Version: "1",
			},
			setup: func(mock *fake.MockSecretProvider) {
				mock.MockAccessSecretVersion([]byte(`{"some": "value", "another": "value", "foo": {"bar": "baz"}}`), nil)
			},
			wantPayload: map[string][]byte{
				"some":    []byte("value"),
				"another": []byte("value"),
				"foo":     []byte(`{"bar": "baz"}`),
			},
			wantErr: nil,
		},
		{
			name: "error_access_secret",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     keyID,
				Version: "1",
			},
			setup: func(mock *fake.MockSecretProvider) {
				mock.MockAccessSecretVersion(nil, errInvalidSecretID)
			},
			wantPayload: nil,
			wantErr:     errInvalidSecretID,
		},
		{
			name: "error_not_json",
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:     keyID,
				Version: "1",
			},
			setup: func(mock *fake.MockSecretProvider) {
				mock.MockAccessSecretVersion([]byte(`top_secret`), nil)
			},
			wantPayload: nil,
			wantErr:     fmt.Errorf(`expecting the secret %q in JSON format`, keyID),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &fake.MockSecretProvider{}
			tt.setup(mock)
			c := &Client{
				apiClient: mock,
				projectID: "123",
			}

			got, gotErr := c.GetSecretMap(context.Background(), tt.ref)

			tassert.Equal(t, tt.wantErr, gotErr)
			tassert.Equal(t, len(tt.wantPayload), len(got))
			for k, v := range tt.wantPayload {
				tassert.Equal(t, v, got[k])
			}
		})
	}
}

func TestClientGetAllSecrets(t *testing.T) {
	tests := []struct {
		name        string
		ref         esv1beta1.ExternalSecretFind
		setup       func(mock *fake.MockSecretProvider)
		wantPayload map[string][]byte
		wantErr     error
	}{
		{
			name: "success",
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{RegExp: "secret.*"},
				Tags: map[string]string{
					"env": "prod",
				},
			},
			setup: func(mock *fake.MockSecretProvider) {
				mock.MockListSecrets([]*smsV2.Secret{
					{Id: keyID, Name: "secret1", Path: "secret1"},
					{Id: anotherKeyID, Name: "secret", Path: "storage/secret"},
				}, nil)

				mock.MockAccessSecretVersion([]byte(`{"some": "value", "another": "value", "foo": {"bar": "baz"}}`), nil)
				mock.MockAccessSecretVersion([]byte(`{"second_secret": "prop_value"}`), nil)
			},
			wantPayload: map[string][]byte{
				"secret1":        []byte(`{"some": "value", "another": "value", "foo": {"bar": "baz"}}`),
				"storage/secret": []byte(`{"second_secret": "prop_value"}`),
			},
			wantErr: nil,
		},
		{
			name: "success_not_json",
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{RegExp: "secr.*"},
				Tags: map[string]string{
					"env": "prod",
				},
			},
			setup: func(mock *fake.MockSecretProvider) {
				mock.MockListSecrets([]*smsV2.Secret{
					{Id: keyID, Name: "secret", Path: "secret"},
					{Id: anotherKeyID, Name: "secret2", Path: "storage/secret"},
				}, nil)
				mock.MockListSecrets(nil, nil) // mock next call

				mock.MockAccessSecretVersion([]byte(`{"some": "value", "another": "value", "foo": {"bar": "baz"}}`), nil)
				mock.MockAccessSecretVersion([]byte(`top_secret`), nil)
			},
			wantPayload: map[string][]byte{
				"secret":         []byte(`{"some": "value", "another": "value", "foo": {"bar": "baz"}}`),
				"storage/secret": []byte(`top_secret`),
			},
			wantErr: nil,
		},
		{
			name:        "error_no_filters",
			ref:         esv1beta1.ExternalSecretFind{},
			wantPayload: nil,
			wantErr:     errors.New("at least one of the following fields must be set: tags, name"),
		},
		{
			name: "error_list_secrets",
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{RegExp: "label.*"},
				Tags: map[string]string{
					"env": "prod",
				},
			},
			setup: func(mock *fake.MockSecretProvider) {
				mock.MockListSecrets(nil, errInternal)
			},
			wantPayload: nil,
			wantErr:     fmt.Errorf("failed to list secrets: %w", errInternal),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &fake.MockSecretProvider{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			c := &Client{
				apiClient: mock,
				projectID: "123",
			}
			got, gotErr := c.GetAllSecrets(context.Background(), tt.ref)

			tassert.Equal(t, tt.wantErr, gotErr)
			tassert.Equal(t, len(tt.wantPayload), len(got))
			for k, v := range tt.wantPayload {
				tassert.Equal(t, v, got[k])
			}
		})
	}
}
