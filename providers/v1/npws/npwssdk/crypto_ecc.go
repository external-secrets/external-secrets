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
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

const (
	// P-521 curve: each coordinate is ceil(521/8) = 66 bytes.
	eccP521ComponentSize = 66
	// PLAIN ECDSA signature: r || s, each 66 bytes for P-521.
	eccP521SignatureSize = eccP521ComponentSize * 2
)

// GenerateECDHKeyPair generates a new ECDH P-521 key pair.
func GenerateECDHKeyPair() (*ecdh.PrivateKey, error) {
	return ecdh.P521().GenerateKey(rand.Reader)
}

// ECDHSharedSecret computes the ECDH shared secret between a private key and a peer's public key.
// Leading zero bytes are stripped to match BouncyCastle behavior.
func ECDHSharedSecret(priv *ecdh.PrivateKey, pub *ecdh.PublicKey) ([]byte, error) {
	secret, err := priv.ECDH(pub)
	if err != nil {
		return nil, fmt.Errorf("ECDH: %w", err)
	}
	// Strip leading zero bytes (BouncyCastle compatibility)
	for len(secret) > 0 && secret[0] == 0 {
		secret = secret[1:]
	}
	return secret, nil
}

// ECDSASign signs data with ECDSA P-521 SHA-256 in PLAIN format (raw r||s concatenation).
// Each component is zero-padded to 66 bytes.
func ECDSASign(priv *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, priv, hash[:])
	if err != nil {
		return nil, fmt.Errorf("ECDSA sign: %w", err)
	}

	sig := make([]byte, eccP521SignatureSize)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	// Right-align (zero-pad on the left)
	copy(sig[eccP521ComponentSize-len(rBytes):eccP521ComponentSize], rBytes)
	copy(sig[eccP521SignatureSize-len(sBytes):eccP521SignatureSize], sBytes)
	return sig, nil
}

// ECDSAVerify verifies a PLAIN-format ECDSA P-521 SHA-256 signature.
func ECDSAVerify(pub *ecdsa.PublicKey, data, signature []byte) error {
	if len(signature) != eccP521SignatureSize {
		return fmt.Errorf("ECDSA verify: signature must be %d bytes, got %d", eccP521SignatureSize, len(signature))
	}

	r := new(big.Int).SetBytes(signature[:eccP521ComponentSize])
	s := new(big.Int).SetBytes(signature[eccP521ComponentSize:])

	hash := sha256.Sum256(data)
	if !ecdsa.Verify(pub, hash[:], r, s) {
		return fmt.Errorf("ECDSA verify: invalid signature")
	}
	return nil
}

// ECDHPrivateToECDSA converts an ECDH P-521 private key to an ECDSA private key.
func ECDHPrivateToECDSA(priv *ecdh.PrivateKey) (*ecdsa.PrivateKey, error) {
	// The raw ECDH private key bytes are the scalar D
	rawBytes := priv.Bytes()
	d := new(big.Int).SetBytes(rawBytes)

	curve := elliptic.P521()
	x, y := curve.ScalarBaseMult(d.Bytes())

	return &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: d,
	}, nil
}

// ECDHPublicToECDSA converts an ECDH P-521 public key to an ECDSA public key.
func ECDHPublicToECDSA(pub *ecdh.PublicKey) (*ecdsa.PublicKey, error) {
	// ECDH public key raw bytes are the uncompressed point (0x04 || X || Y)
	rawBytes := pub.Bytes()
	x, y := elliptic.Unmarshal(elliptic.P521(), rawBytes)
	if x == nil {
		return nil, fmt.Errorf("ECDSA: failed to unmarshal public key")
	}
	return &ecdsa.PublicKey{
		Curve: elliptic.P521(),
		X:     x,
		Y:     y,
	}, nil
}
