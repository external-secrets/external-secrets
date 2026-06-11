/*
Copyright © 2026 SSH Communications

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

package privx

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	jwt "github.com/golang-jwt/jwt/v5"
)

var (
	// errEd25519KeyType is returned when the Ed25519 key type does not match the expected type.
	errEd25519KeyType = errors.New("Ed25519 key type mismatch")
)

// signingMethodEd25519 is to circumvent alg:ed25519 expectation of PrivX.
type signingMethodEd25519 struct{}

func (m signingMethodEd25519) Alg() string { return "Ed25519" }

// Sign signs the JWT signing string using Ed25519.
// Note: Comments are in English by request.
func (m signingMethodEd25519) Sign(signingString string, key any) ([]byte, error) {
	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: got %T", errEd25519KeyType, key)
	}
	sig := ed25519.Sign(priv, []byte(signingString))
	return sig, nil
}

func (m signingMethodEd25519) Verify(signingString string, signature []byte, key any) error {
	pub, ok := key.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("%w: got %T", errEd25519KeyType, key)
	}
	if !ed25519.Verify(pub, []byte(signingString), signature) {
		return jwt.ErrSignatureInvalid
	}
	return nil
}

func init() {
	// Register custom method so parser can resolve alg="Ed25519".
	jwt.RegisterSigningMethod("Ed25519", func() jwt.SigningMethod {
		return signingMethodEd25519{}
	})
}

// SigningMethodEd25519 is an optional helper if you want a handle without relying on init order.
func SigningMethodEd25519() jwt.SigningMethod {
	return signingMethodEd25519{}
}
