// /*
// Copyright © The ESO Authors
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

package npwssdk

import (
	"crypto/sha1" //nolint:gosec // SHA1 required for C# PBKDF2 compatibility
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
)

// Pbkdf2SHA1 derives a key using PBKDF2 with SHA-1.
func Pbkdf2SHA1(password, salt []byte, iterations, keyLen int) []byte {
	return pbkdf2.Key(password, salt, iterations, keyLen, sha1.New)
}

// Pbkdf2SHA256 derives a key using PBKDF2 with SHA-256.
func Pbkdf2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	return pbkdf2.Key(password, salt, iterations, keyLen, sha256.New)
}

// HkdfSHA256 derives a key using HKDF with SHA-256.
// The info parameter is empty (matching the C# implementation).
func HkdfSHA256(secret, salt []byte, keyLen int) ([]byte, error) {
	reader := hkdf.New(sha256.New, secret, salt, nil)
	key := make([]byte, keyLen)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, fmt.Errorf("HKDF-SHA256: %w", err)
	}
	return key, nil
}
