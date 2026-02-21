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

// Package mysterybox provides a client interface and implementations for Nebius Mysterybox service.
package mysterybox

import "context"

// Client is an interface that contains the main methods to interact with Secret Service.
type Client interface {
	GetSecret(ctx context.Context, token, secretID, versionID string) (*Payload, error)
	GetSecretByKey(ctx context.Context, token, secretID, versionID, key string) (*PayloadEntry, error)
	Close() error
}

// Payload represents a secret version payload returned by the Nebius Mysterybox service.
// It contains the version identifier and the list of key/value entries.
type Payload struct {
	VersionID string
	Entries   []Entry
}

// PayloadEntry represents a single entry from a secret version payload identified by key.
type PayloadEntry struct {
	VersionID string
	Entry     Entry
}

// Entry is a key/value item within a secret payload.
// Only one of StringValue or BinaryValue is expected to be set depending on the secret's data type.
type Entry struct {
	Key         string
	StringValue string
	BinaryValue []byte
}
