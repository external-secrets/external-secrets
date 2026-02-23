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

// Package protonpass implements a provider for Proton Pass using the pass-cli binary.
package protonpass

import "encoding/json"

// itemListResponse represents the response from pass-cli item list --output json.
type itemListResponse struct {
	Items []item `json:"items"`
}

// item represents a Proton Pass item from the CLI output.
type item struct {
	ID         string      `json:"id"`
	ShareID    string      `json:"share_id"`
	VaultID    string      `json:"vault_id"`
	Content    itemContent `json:"content"`
	State      string      `json:"state"`
	Flags      []string    `json:"flags"`
	CreateTime string      `json:"create_time"`
}

// itemContent represents the content wrapper of an item.
type itemContent struct {
	Title       string          `json:"title"`
	Note        string          `json:"note"`
	ItemUUID    string          `json:"item_uuid"`
	Content     itemTypeContent `json:"content"`
	ExtraFields []extraField    `json:"extra_fields"`
}

// itemTypeContent represents the type-specific content (Login, Note, etc.).
type itemTypeContent struct {
	Login *loginContent `json:"Login,omitempty"`
	Note  *noteContent  `json:"Note,omitempty"`
}

// loginContent represents login-specific item data.
type loginContent struct {
	Email    string          `json:"email"`
	Username string          `json:"username"`
	Password string          `json:"password"`
	URLs     []string        `json:"urls"`
	TOTPUri  string          `json:"totp_uri"`
	Passkeys json.RawMessage `json:"passkeys,omitempty"`
}

// noteContent represents note-specific item data.
type noteContent struct{}

// extraField represents an extra field in a Proton Pass item.
type extraField struct {
	FieldName string `json:"field_name"`
	Type      string `json:"type"`
	Value     string `json:"value"`
}

// Fields returns all fields of the item as a flat key-value map.
// This centralizes field extraction logic used by both GetSecret and GetSecretMap.
func (item *item) Fields() map[string]string {
	fields := make(map[string]string)

	if item.Content.Note != "" {
		fields["note"] = item.Content.Note
	}

	if login := item.Content.Content.Login; login != nil {
		if login.Username != "" {
			fields["username"] = login.Username
		}
		if login.Password != "" {
			fields["password"] = login.Password
		}
		if login.Email != "" {
			fields["email"] = login.Email
		}
		if login.TOTPUri != "" {
			fields["totpSecret"] = login.TOTPUri
		}
	}

	for _, field := range item.Content.ExtraFields {
		fields[field.FieldName] = field.Value
	}

	return fields
}

// vault represents a Proton Pass vault from the CLI output.
type vault struct {
	VaultID string `json:"vault_id"`
	Name    string `json:"name"`
}
