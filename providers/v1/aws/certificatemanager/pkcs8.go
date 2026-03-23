/*
Copyright © The ESO Authors

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

package certificatemanager

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // required for PBKDF2 HMAC-SHA1 per RFC 8018
	"crypto/sha256"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"hash"
)

const passphraseLength = 32

var passphraseChars = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

var (
	oidPBES2          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 13}
	oidPBKDF2         = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 12}
	oidHMACWithSHA1   = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 7}
	oidHMACWithSHA256 = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 9}
	oidAES128CBC      = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 2}
	oidAES192CBC      = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 22}
	oidAES256CBC      = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 42}
)

type encryptedPrivateKeyInfo struct {
	EncryptionAlgorithm pkiAlgorithmIdentifier
	EncryptedData       []byte
}

type pkiAlgorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

type pbes2Params struct {
	KeyDerivationFunc pkiAlgorithmIdentifier
	EncryptionScheme  pkiAlgorithmIdentifier
}

type pbkdf2Params struct {
	Salt           []byte
	IterationCount int
	KeyLength      int                    `asn1:"optional"`
	PRF            pkiAlgorithmIdentifier `asn1:"optional"`
}

// generatePassphrase returns a random alphanumeric passphrase safe for
// the ACM ExportCertificate API (no #, $, or % characters).
var generatePassphrase = func() ([]byte, error) {
	b := make([]byte, passphraseLength)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	for i, v := range b {
		b[i] = passphraseChars[int(v)%len(passphraseChars)]
	}
	return b, nil
}

// decryptPKCS8PEM decodes an "ENCRYPTED PRIVATE KEY" PEM block and returns the
// decrypted key as a "PRIVATE KEY" PEM block. If the PEM is already unencrypted
// it is returned as-is.
func decryptPKCS8PEM(encryptedPEM, passphrase []byte) ([]byte, error) {
	block, _ := pem.Decode(encryptedPEM)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	if block.Type == "PRIVATE KEY" {
		return encryptedPEM, nil
	}
	if block.Type != "ENCRYPTED PRIVATE KEY" {
		return nil, fmt.Errorf("unexpected PEM type: %s", block.Type)
	}

	decryptedDER, err := decryptPKCS8(block.Bytes, passphrase)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: decryptedDER,
	}), nil
}

func decryptPKCS8(encryptedDER, passphrase []byte) ([]byte, error) {
	var epki encryptedPrivateKeyInfo
	if _, err := asn1.Unmarshal(encryptedDER, &epki); err != nil {
		return nil, fmt.Errorf("failed to parse EncryptedPrivateKeyInfo: %w", err)
	}
	if !epki.EncryptionAlgorithm.Algorithm.Equal(oidPBES2) {
		return nil, fmt.Errorf("unsupported encryption algorithm: %v", epki.EncryptionAlgorithm.Algorithm)
	}

	var params pbes2Params
	if _, err := asn1.Unmarshal(epki.EncryptionAlgorithm.Parameters.FullBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to parse PBES2 parameters: %w", err)
	}
	if !params.KeyDerivationFunc.Algorithm.Equal(oidPBKDF2) {
		return nil, fmt.Errorf("unsupported KDF: %v", params.KeyDerivationFunc.Algorithm)
	}

	var kdfParams pbkdf2Params
	if _, err := asn1.Unmarshal(params.KeyDerivationFunc.Parameters.FullBytes, &kdfParams); err != nil {
		return nil, fmt.Errorf("failed to parse PBKDF2 parameters: %w", err)
	}

	hashFunc := prfHash(kdfParams.PRF.Algorithm)
	if hashFunc == nil {
		return nil, fmt.Errorf("unsupported PBKDF2 PRF: %v", kdfParams.PRF.Algorithm)
	}

	keySize, err := aesCBCKeySize(params.EncryptionScheme.Algorithm)
	if err != nil {
		return nil, err
	}

	var iv []byte
	if _, err := asn1.Unmarshal(params.EncryptionScheme.Parameters.FullBytes, &iv); err != nil {
		return nil, fmt.Errorf("failed to parse IV: %w", err)
	}

	key, err := pbkdf2.Key(hashFunc, string(passphrase), kdfParams.Salt, kdfParams.IterationCount, keySize)
	if err != nil {
		return nil, fmt.Errorf("pbkdf2 key derivation failed: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	if len(epki.EncryptedData)%block.BlockSize() != 0 {
		return nil, errors.New("encrypted data is not a multiple of the block size")
	}

	plaintext := make([]byte, len(epki.EncryptedData))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, epki.EncryptedData)

	return removePKCS7Padding(plaintext, block.BlockSize())
}

func prfHash(oid asn1.ObjectIdentifier) func() hash.Hash {
	switch {
	case oid == nil, oid.Equal(oidHMACWithSHA1):
		return sha1.New
	case oid.Equal(oidHMACWithSHA256):
		return sha256.New
	default:
		return nil
	}
}

func aesCBCKeySize(oid asn1.ObjectIdentifier) (int, error) {
	switch {
	case oid.Equal(oidAES128CBC):
		return 16, nil
	case oid.Equal(oidAES192CBC):
		return 24, nil
	case oid.Equal(oidAES256CBC):
		return 32, nil
	default:
		return 0, fmt.Errorf("unsupported encryption scheme: %v", oid)
	}
}

func removePKCS7Padding(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data")
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > blockSize || padLen > len(data) {
		return nil, errors.New("invalid PKCS#7 padding")
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if data[i] != byte(padLen) { //nolint:gosec // padLen is bounded by blockSize (max 16)
			return nil, errors.New("invalid PKCS#7 padding")
		}
	}
	return data[:len(data)-padLen], nil
}
