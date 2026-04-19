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

import "crypto/ecdh"

// EncryptionCollectionV2 implements ECC-based encryption (Version 2).
type EncryptionCollectionV2 struct{}

// Version returns the encryption version for V2.
func (v2 *EncryptionCollectionV2) Version() EncryptionVersion {
	return EncryptionV2
}

// AreDataKeysAsymmetric returns false because V2 uses symmetric data keys.
func (v2 *EncryptionCollectionV2) AreDataKeysAsymmetric() bool {
	return false
}

// GenerateKeyPair generates an ECC (P-521) key pair for V2.
func (v2 *EncryptionCollectionV2) GenerateKeyPair() (publicKey, privateKey []byte, err error) {
	priv, err := GenerateECDHKeyPair()
	if err != nil {
		return nil, nil, err
	}
	return priv.PublicKey().Bytes(), priv.Bytes(), nil
}

// GenerateSymmetricKey generates a 32-byte AES key.
func (v2 *EncryptionCollectionV2) GenerateSymmetricKey() ([]byte, error) {
	return generateRandomBytes(32)
}

// GenerateDataKey generates symmetric data keys for V2.
func (v2 *EncryptionCollectionV2) GenerateDataKey() (encryptionKey, decryptionKey []byte, err error) {
	// V2 uses symmetric data keys
	key, err := generateRandomBytes(32)
	if err != nil {
		return nil, nil, err
	}
	return key, key, nil
}

// EncryptWithPublicKey encrypts data with an ECC or RSA public key.
func (v2 *EncryptionCollectionV2) EncryptWithPublicKey(publicKey, plaintext []byte) ([]byte, error) {
	// For ECC keys, use ECDH chain. For RSA keys (legacy data), use RSA chain.
	if IsRSAKey(publicKey) {
		return EncryptWithChain(ChainRsaPbkdf2Sha1AesCbc, publicKey, plaintext)
	}
	return EncryptWithChain(ChainEcdhHkdfSha256AesGcm, publicKey, plaintext)
}

// EncryptWithPassword encrypts data with a password using PBKDF2.
func (v2 *EncryptionCollectionV2) EncryptWithPassword(password, plaintext []byte) ([]byte, error) {
	return EncryptWithChain(ChainPbkdf2Sha256_610005_AesGcmPadded, password, plaintext)
}

// EncryptWithKey encrypts data with a symmetric key using AES-GCM.
func (v2 *EncryptionCollectionV2) EncryptWithKey(key, plaintext []byte) ([]byte, error) {
	return EncryptWithChain(ChainAesGcmPadded, key, plaintext)
}

// Decrypt decrypts data by auto-detecting the encryption chain.
func (v2 *EncryptionCollectionV2) Decrypt(key, ciphertext []byte) ([]byte, error) {
	return DecryptWithChain(key, ciphertext)
}

// SignData signs data with a private key (RSA or ECDSA).
func (v2 *EncryptionCollectionV2) SignData(privateKey, data []byte) ([]byte, error) {
	kt := DetectKeyType(privateKey)
	switch kt { //nolint:exhaustive // default handles raw P-521 scalar bytes
	case KeyTypeRSA:
		priv, err := ParseRSAPrivateKeyFromXML(string(privateKey))
		if err != nil {
			return nil, err
		}
		return RSASign(priv, data)
	case KeyTypeECCPrivate:
		ecdhPriv, err := parseECCPrivateKey(privateKey)
		if err != nil {
			return nil, err
		}
		ecdsaPriv, err := ECDHPrivateToECDSA(ecdhPriv)
		if err != nil {
			return nil, err
		}
		return ECDSASign(ecdsaPriv, data)
	default:
		// Try raw 66-byte P-521 scalar
		ecdhPriv, err := ecdh.P521().NewPrivateKey(privateKey)
		if err != nil {
			return nil, err
		}
		ecdsaPriv, err := ECDHPrivateToECDSA(ecdhPriv)
		if err != nil {
			return nil, err
		}
		return ECDSASign(ecdsaPriv, data)
	}
}

// VerifyData verifies a signature with an RSA or ECC public key.
func (v2 *EncryptionCollectionV2) VerifyData(publicKey, data, signature []byte) error {
	if IsRSAKey(publicKey) {
		pub, err := ParseRSAPublicKeyFromXML(string(publicKey))
		if err != nil {
			return err
		}
		return RSAVerify(pub, data, signature)
	}
	// ECC
	ecdhPub, err := ecdh.P521().NewPublicKey(publicKey)
	if err != nil {
		return err
	}
	ecdsaPub, err := ECDHPublicToECDSA(ecdhPub)
	if err != nil {
		return err
	}
	return ECDSAVerify(ecdsaPub, data, signature)
}
