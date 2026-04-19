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

// EncryptionCollectionV1 implements RSA-based encryption (Version 1).
type EncryptionCollectionV1 struct{}

// Version returns the encryption version for V1.
func (v1 *EncryptionCollectionV1) Version() EncryptionVersion {
	return EncryptionV1
}

// AreDataKeysAsymmetric returns true because V1 uses asymmetric data keys.
func (v1 *EncryptionCollectionV1) AreDataKeysAsymmetric() bool {
	return true
}

// GenerateKeyPair generates an RSA key pair for V1.
func (v1 *EncryptionCollectionV1) GenerateKeyPair() (publicKey, privateKey []byte, err error) {
	priv, err := GenerateRSAKeyPair()
	if err != nil {
		return nil, nil, err
	}
	pub := ExportRSAPublicKeyToXML(&priv.PublicKey)
	privXML := ExportRSAPrivateKeyToXML(priv)
	return []byte(pub), []byte(privXML), nil
}

// GenerateSymmetricKey generates a 32-byte AES key.
func (v1 *EncryptionCollectionV1) GenerateSymmetricKey() ([]byte, error) {
	return generateRandomBytes(32)
}

// GenerateDataKey generates asymmetric data keys (RSA) for V1.
func (v1 *EncryptionCollectionV1) GenerateDataKey() (encryptionKey, decryptionKey []byte, err error) {
	// V1 uses asymmetric data keys (RSA)
	return v1.GenerateKeyPair()
}

// EncryptWithPublicKey encrypts data with an RSA or ECC public key.
func (v1 *EncryptionCollectionV1) EncryptWithPublicKey(publicKey, plaintext []byte) ([]byte, error) {
	if IsECCKey(publicKey) {
		return EncryptWithChain(ChainEcdhHkdfSha256AesGcm, publicKey, plaintext)
	}
	return EncryptWithChain(ChainRsaPbkdf2Sha1AesCbc, publicKey, plaintext)
}

// EncryptWithPassword encrypts data with a password using PBKDF2.
func (v1 *EncryptionCollectionV1) EncryptWithPassword(password, plaintext []byte) ([]byte, error) {
	return EncryptWithChain(ChainPbkdf2Sha256_610005_AesGcmPadded, password, plaintext)
}

// EncryptWithKey encrypts data with a symmetric key using AES-GCM.
func (v1 *EncryptionCollectionV1) EncryptWithKey(key, plaintext []byte) ([]byte, error) {
	return EncryptWithChain(ChainAesGcmPadded, key, plaintext)
}

// Decrypt decrypts data by auto-detecting the encryption chain.
func (v1 *EncryptionCollectionV1) Decrypt(key, ciphertext []byte) ([]byte, error) {
	return DecryptWithChain(key, ciphertext)
}

// SignData signs data with a private key (RSA or ECDSA).
func (v1 *EncryptionCollectionV1) SignData(privateKey, data []byte) ([]byte, error) {
	kt := DetectKeyType(privateKey)
	switch kt { //nolint:exhaustive // default handles RSA and all other key types
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
		// RSA (XML format, possibly base64-encoded)
		priv, err := ParseRSAPrivateKeyFromXML(string(privateKey))
		if err != nil {
			return nil, err
		}
		return RSASign(priv, data)
	}
}

// VerifyData verifies a signature with an RSA public key.
func (v1 *EncryptionCollectionV1) VerifyData(publicKey, data, signature []byte) error {
	pub, err := ParseRSAPublicKeyFromXML(string(publicKey))
	if err != nil {
		return err
	}
	return RSAVerify(pub, data, signature)
}
