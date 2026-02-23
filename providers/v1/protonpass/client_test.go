/*
Copyright © 2026 ESO Maintainer Team

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

package protonpass

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestParseKey(t *testing.T) {
	tests := []struct {
		key       string
		wantItem  string
		wantField string
	}{
		{"my-item", "my-item", ""},
		{"my-item/password", "my-item", "password"},
		{"my-item/nested/path", "my-item", "nested/path"},
		{"", "", ""},
		{"/", "", ""},
	}

	for _, tc := range tests {
		item, field := parseKey(tc.key)
		assert.Equal(t, tc.wantItem, item, "key=%q item", tc.key)
		assert.Equal(t, tc.wantField, field, "key=%q field", tc.key)
	}
}

func TestMatchesFind(t *testing.T) {
	makeItem := func(title string) item {
		return item{Content: itemContent{Title: title}}
	}

	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name    string
		item    item
		ref     esv1.ExternalSecretFind
		match   bool
		wantErr string
	}{
		{
			name:  "no filter matches all",
			item:  makeItem("anything"),
			ref:   esv1.ExternalSecretFind{},
			match: true,
		},
		{
			name: "regexp substring match",
			item: makeItem("my-secret-item"),
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{RegExp: "secret"},
			},
			match: true,
		},
		{
			name: "regexp no match",
			item: makeItem("my-item"),
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{RegExp: "secret"},
			},
			match: false,
		},
		{
			name: "regexp anchored pattern matches",
			item: makeItem("prod-database"),
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{RegExp: "^prod-.*"},
			},
			match: true,
		},
		{
			name: "regexp anchored pattern rejects",
			item: makeItem("staging-database"),
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{RegExp: "^prod-.*"},
			},
			match: false,
		},
		{
			name: "regexp character class",
			item: makeItem("secret-42"),
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{RegExp: `secret-\d+`},
			},
			match: true,
		},
		{
			name: "invalid regexp returns error",
			item: makeItem("anything"),
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{RegExp: "[invalid"},
			},
			wantErr: "invalid regexp",
		},
		{
			name: "path prefix match",
			item: makeItem("prod/database"),
			ref: esv1.ExternalSecretFind{
				Path: strPtr("prod/"),
			},
			match: true,
		},
		{
			name: "path prefix no match",
			item: makeItem("staging/database"),
			ref: esv1.ExternalSecretFind{
				Path: strPtr("prod/"),
			},
			match: false,
		},
		{
			name: "name matches but path doesn't",
			item: makeItem("staging/secret-item"),
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{RegExp: "secret"},
				Path: strPtr("prod/"),
			},
			match: false,
		},
		{
			name: "nil name with non-nil path",
			item: makeItem("prod/item"),
			ref: esv1.ExternalSecretFind{
				Path: strPtr("prod/"),
			},
			match: true,
		},
		{
			name: "non-nil name with nil path",
			item: makeItem("my-secret"),
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{RegExp: "secret"},
			},
			match: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := matchesFind(tc.item, tc.ref)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.match, got)
		})
	}
}

func TestGetSecret(t *testing.T) {
	ctx := context.Background()

	loginItem := &item{
		ID: "item-1",
		Content: itemContent{
			Title: "my-login",
			Content: itemTypeContent{
				Login: &loginContent{
					Username: "user",
					Password: "secret123",
					Email:    "user@example.com",
				},
			},
			ExtraFields: []extraField{
				{FieldName: "apiKey", Value: "key-abc"},
			},
		},
	}

	tests := []struct {
		name    string
		cli     *fakeCLI
		ref     esv1.ExternalSecretDataRemoteRef
		want    string
		wantErr string
	}{
		{
			name: "simple key defaults to password field",
			cli: &fakeCLI{
				resolveItemIDResults: map[string]string{"my-login": "item-1"},
				getItemResults:       map[string]*item{"item-1": loginItem},
			},
			ref:  esv1.ExternalSecretDataRemoteRef{Key: "my-login"},
			want: "secret123",
		},
		{
			name: "key with item/field format",
			cli: &fakeCLI{
				resolveItemIDResults: map[string]string{"my-login": "item-1"},
				getItemResults:       map[string]*item{"item-1": loginItem},
			},
			ref:  esv1.ExternalSecretDataRemoteRef{Key: "my-login/username"},
			want: "user",
		},
		{
			name: "property overrides field from key",
			cli: &fakeCLI{
				resolveItemIDResults: map[string]string{"my-login": "item-1"},
				getItemResults:       map[string]*item{"item-1": loginItem},
			},
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "my-login/password",
				Property: "email",
			},
			want: "user@example.com",
		},
		{
			name: "login failure propagates error",
			cli: &fakeCLI{
				loginErr: errors.New("auth failed"),
			},
			ref:     esv1.ExternalSecretDataRemoteRef{Key: "my-login"},
			wantErr: "failed to ensure login",
		},
		{
			name: "item not found propagates error",
			cli: &fakeCLI{
				resolveItemIDErr: errItemNotFound,
			},
			ref:     esv1.ExternalSecretDataRemoteRef{Key: "nonexistent"},
			wantErr: "not found",
		},
		{
			name: "extracts extra fields",
			cli: &fakeCLI{
				resolveItemIDResults: map[string]string{"my-login": "item-1"},
				getItemResults:       map[string]*item{"item-1": loginItem},
			},
			ref:  esv1.ExternalSecretDataRemoteRef{Key: "my-login/apiKey"},
			want: "key-abc",
		},
		{
			name: "field not found returns error",
			cli: &fakeCLI{
				resolveItemIDResults: map[string]string{"my-login": "item-1"},
				getItemResults:       map[string]*item{"item-1": loginItem},
			},
			ref:     esv1.ExternalSecretDataRemoteRef{Key: "my-login/nonexistent"},
			wantErr: "field not found",
		},
		{
			name: "extracts note field",
			cli: &fakeCLI{
				resolveItemIDResults: map[string]string{"my-note": "item-3"},
				getItemResults: map[string]*item{
					"item-3": {
						ID: "item-3",
						Content: itemContent{
							Note: "my secret note",
						},
					},
				},
			},
			ref:  esv1.ExternalSecretDataRemoteRef{Key: "my-note/note"},
			want: "my secret note",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &provider{cli: tc.cli}
			got, err := p.GetSecret(ctx, tc.ref)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(got))
		})
	}
}

func TestGetSecretMap(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		cli     *fakeCLI
		ref     esv1.ExternalSecretDataRemoteRef
		want    map[string]string
		wantErr string
	}{
		{
			name: "login item",
			cli: &fakeCLI{
				resolveItemIDResults: map[string]string{"my-login": "item-1"},
				getItemResults: map[string]*item{
					"item-1": {
						ID: "item-1",
						Content: itemContent{
							Note: "a note",
							Content: itemTypeContent{
								Login: &loginContent{
									Username: "user",
									Password: "pass",
									Email:    "user@example.com",
									TOTPUri:  "otpauth://totp/test",
								},
							},
						},
					},
				},
			},
			ref: esv1.ExternalSecretDataRemoteRef{Key: "my-login"},
			want: map[string]string{
				"username":   "user",
				"password":   "pass",
				"email":      "user@example.com",
				"totpSecret": "otpauth://totp/test",
				"note":       "a note",
			},
		},
		{
			name: "item with extra fields",
			cli: &fakeCLI{
				resolveItemIDResults: map[string]string{"my-item": "item-3"},
				getItemResults: map[string]*item{
					"item-3": {
						ID: "item-3",
						Content: itemContent{
							ExtraFields: []extraField{
								{FieldName: "apiKey", Value: "key-123"},
								{FieldName: "region", Value: "us-east-1"},
							},
						},
					},
				},
			},
			ref: esv1.ExternalSecretDataRemoteRef{Key: "my-item"},
			want: map[string]string{
				"apiKey": "key-123",
				"region": "us-east-1",
			},
		},
		{
			name: "empty item",
			cli: &fakeCLI{
				resolveItemIDResults: map[string]string{"empty": "item-4"},
				getItemResults: map[string]*item{
					"item-4": {
						ID:      "item-4",
						Content: itemContent{},
					},
				},
			},
			ref:  esv1.ExternalSecretDataRemoteRef{Key: "empty"},
			want: map[string]string{},
		},
		{
			name: "login failure propagates error",
			cli: &fakeCLI{
				loginErr: errors.New("auth failed"),
			},
			ref:     esv1.ExternalSecretDataRemoteRef{Key: "my-login"},
			wantErr: "failed to ensure login",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &provider{cli: tc.cli}
			got, err := p.GetSecretMap(ctx, tc.ref)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			result := make(map[string]string, len(got))
			for k, v := range got {
				result[k] = string(v)
			}
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestGetAllSecrets(t *testing.T) {
	ctx := context.Background()

	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name    string
		cli     *fakeCLI
		vault   string
		ref     esv1.ExternalSecretFind
		want    map[string]string
		wantErr string
	}{
		{
			name: "returns login passwords keyed by title",
			cli: &fakeCLI{
				listItemsResults: map[string][]item{
					"myvault": {
						{ID: "item-1", Content: itemContent{Title: "Login A"}},
						{ID: "item-2", Content: itemContent{Title: "Login B"}},
					},
				},
				getItemResults: map[string]*item{
					"item-1": {
						ID: "item-1",
						Content: itemContent{
							Title:   "Login A",
							Content: itemTypeContent{Login: &loginContent{Password: "pass-a"}},
						},
					},
					"item-2": {
						ID: "item-2",
						Content: itemContent{
							Title:   "Login B",
							Content: itemTypeContent{Login: &loginContent{Password: "pass-b"}},
						},
					},
				},
			},
			vault: "myvault",
			ref:   esv1.ExternalSecretFind{},
			want: map[string]string{
				"Login A": "pass-a",
				"Login B": "pass-b",
			},
		},
		{
			name: "name filter narrows results",
			cli: &fakeCLI{
				listItemsResults: map[string][]item{
					"myvault": {
						{ID: "item-1", Content: itemContent{Title: "prod-db"}},
						{ID: "item-2", Content: itemContent{Title: "staging-db"}},
					},
				},
				getItemResults: map[string]*item{
					"item-1": {
						ID: "item-1",
						Content: itemContent{
							Title:   "prod-db",
							Content: itemTypeContent{Login: &loginContent{Password: "prod-pass"}},
						},
					},
				},
			},
			vault: "myvault",
			ref: esv1.ExternalSecretFind{
				Name: &esv1.FindName{RegExp: "prod"},
			},
			want: map[string]string{
				"prod-db": "prod-pass",
			},
		},
		{
			name: "path filter narrows results",
			cli: &fakeCLI{
				listItemsResults: map[string][]item{
					"myvault": {
						{ID: "item-1", Content: itemContent{Title: "prod/db"}},
						{ID: "item-2", Content: itemContent{Title: "staging/db"}},
					},
				},
				getItemResults: map[string]*item{
					"item-1": {
						ID: "item-1",
						Content: itemContent{
							Title:   "prod/db",
							Content: itemTypeContent{Login: &loginContent{Password: "prod-pass"}},
						},
					},
				},
			},
			vault: "myvault",
			ref: esv1.ExternalSecretFind{
				Path: strPtr("prod/"),
			},
			want: map[string]string{
				"prod/db": "prod-pass",
			},
		},
		{
			name: "silently skips items where GetItem fails",
			cli: &fakeCLI{
				listItemsResults: map[string][]item{
					"myvault": {
						{ID: "item-1", Content: itemContent{Title: "ok-item"}},
						{ID: "item-bad", Content: itemContent{Title: "bad-item"}},
					},
				},
				getItemResults: map[string]*item{
					"item-1": {
						ID: "item-1",
						Content: itemContent{
							Title:   "ok-item",
							Content: itemTypeContent{Login: &loginContent{Password: "ok-pass"}},
						},
					},
					// item-bad is missing, so GetItem will return errItemNotFound
				},
			},
			vault: "myvault",
			ref:   esv1.ExternalSecretFind{},
			want: map[string]string{
				"ok-item": "ok-pass",
			},
		},
		{
			name: "non-login items excluded",
			cli: &fakeCLI{
				listItemsResults: map[string][]item{
					"myvault": {
						{ID: "item-1", Content: itemContent{Title: "A Note"}},
					},
				},
				getItemResults: map[string]*item{
					"item-1": {
						ID: "item-1",
						Content: itemContent{
							Title:   "A Note",
							Note:    "just a note",
							Content: itemTypeContent{Note: &noteContent{}},
						},
					},
				},
			},
			vault: "myvault",
			ref:   esv1.ExternalSecretFind{},
			want:  map[string]string{},
		},
		{
			name: "login failure propagates error",
			cli: &fakeCLI{
				loginErr: errors.New("auth failed"),
			},
			vault:   "myvault",
			ref:     esv1.ExternalSecretFind{},
			wantErr: "failed to ensure login",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &provider{cli: tc.cli, vault: tc.vault}
			got, err := p.GetAllSecrets(ctx, tc.ref)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			result := make(map[string]string, len(got))
			for k, v := range got {
				result[k] = string(v)
			}
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestUnsupportedOperations(t *testing.T) {
	ctx := context.Background()
	p := &provider{cli: &fakeCLI{}}

	err := p.PushSecret(ctx, nil, nil)
	assert.ErrorIs(t, err, errPushSecretNotSupported)

	err = p.DeleteSecret(ctx, nil)
	assert.ErrorIs(t, err, errDeleteSecretNotSupported)

	_, err = p.SecretExists(ctx, nil)
	assert.ErrorIs(t, err, errSecretExistsNotSupported)
}
