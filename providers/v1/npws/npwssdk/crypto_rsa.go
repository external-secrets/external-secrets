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
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"math/big"
)

// rsaKeyValue represents the .NET XML RSA key format.
type rsaKeyValue struct {
	XMLName  xml.Name `xml:"RSAKeyValue"`
	Modulus  string   `xml:"Modulus"`
	Exponent string   `xml:"Exponent"`
	P        string   `xml:"P,omitempty"`
	Q        string   `xml:"Q,omitempty"`
	DP       string   `xml:"DP,omitempty"`
	DQ       string   `xml:"DQ,omitempty"`
	InverseQ string   `xml:"InverseQ,omitempty"`
	D        string   `xml:"D,omitempty"`
}

// ParseRSAPublicKeyFromXML parses a Base64-encoded .NET XML RSA public key.
func ParseRSAPublicKeyFromXML(base64XML string) (*rsa.PublicKey, error) {
	xmlBytes, err := base64.StdEncoding.DecodeString(base64XML)
	if err != nil {
		return nil, fmt.Errorf("RSA: decoding base64: %w", err)
	}

	var kv rsaKeyValue
	if err := xml.Unmarshal(xmlBytes, &kv); err != nil {
		return nil, fmt.Errorf("RSA: parsing XML: %w", err)
	}

	n, err := base64ToBigInt(kv.Modulus)
	if err != nil {
		return nil, fmt.Errorf("RSA: parsing Modulus: %w", err)
	}
	e, err := base64ToBigInt(kv.Exponent)
	if err != nil {
		return nil, fmt.Errorf("RSA: parsing Exponent: %w", err)
	}

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// ParseRSAPrivateKeyFromXML parses a Base64-encoded .NET XML RSA private key.
func ParseRSAPrivateKeyFromXML(base64XML string) (*rsa.PrivateKey, error) {
	xmlBytes, err := base64.StdEncoding.DecodeString(base64XML)
	if err != nil {
		return nil, fmt.Errorf("RSA: decoding base64: %w", err)
	}

	var kv rsaKeyValue
	if err := xml.Unmarshal(xmlBytes, &kv); err != nil {
		return nil, fmt.Errorf("RSA: parsing XML: %w", err)
	}

	n, err := base64ToBigInt(kv.Modulus)
	if err != nil {
		return nil, fmt.Errorf("RSA: parsing Modulus: %w", err)
	}
	e, err := base64ToBigInt(kv.Exponent)
	if err != nil {
		return nil, fmt.Errorf("RSA: parsing Exponent: %w", err)
	}
	d, err := base64ToBigInt(kv.D)
	if err != nil {
		return nil, fmt.Errorf("RSA: parsing D: %w", err)
	}
	p, err := base64ToBigInt(kv.P)
	if err != nil {
		return nil, fmt.Errorf("RSA: parsing P: %w", err)
	}
	q, err := base64ToBigInt(kv.Q)
	if err != nil {
		return nil, fmt.Errorf("RSA: parsing Q: %w", err)
	}
	dp, err := base64ToBigInt(kv.DP)
	if err != nil {
		return nil, fmt.Errorf("RSA: parsing DP: %w", err)
	}
	dq, err := base64ToBigInt(kv.DQ)
	if err != nil {
		return nil, fmt.Errorf("RSA: parsing DQ: %w", err)
	}
	qi, err := base64ToBigInt(kv.InverseQ)
	if err != nil {
		return nil, fmt.Errorf("RSA: parsing InverseQ: %w", err)
	}

	key := &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: n,
			E: int(e.Int64()),
		},
		D:      d,
		Primes: []*big.Int{p, q},
		Precomputed: rsa.PrecomputedValues{
			Dp:   dp,
			Dq:   dq,
			Qinv: qi,
		},
	}

	if err := key.Validate(); err != nil {
		return nil, fmt.Errorf("RSA: invalid key: %w", err)
	}

	return key, nil
}

// ExportRSAPublicKeyToXML exports an RSA public key in .NET XML format, Base64-encoded.
func ExportRSAPublicKeyToXML(pub *rsa.PublicKey) string {
	kv := rsaKeyValue{
		Modulus:  base64.StdEncoding.EncodeToString(pub.N.Bytes()),
		Exponent: base64.StdEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
	xmlBytes, _ := xml.Marshal(kv)
	return base64.StdEncoding.EncodeToString(xmlBytes)
}

// ExportRSAPrivateKeyToXML exports an RSA private key in .NET XML format, Base64-encoded.
func ExportRSAPrivateKeyToXML(priv *rsa.PrivateKey) string {
	priv.Precompute()
	kv := rsaKeyValue{
		Modulus:  base64.StdEncoding.EncodeToString(priv.N.Bytes()),
		Exponent: base64.StdEncoding.EncodeToString(big.NewInt(int64(priv.E)).Bytes()),
		D:        base64.StdEncoding.EncodeToString(priv.D.Bytes()),
		P:        base64.StdEncoding.EncodeToString(priv.Primes[0].Bytes()),
		Q:        base64.StdEncoding.EncodeToString(priv.Primes[1].Bytes()),
		DP:       base64.StdEncoding.EncodeToString(priv.Precomputed.Dp.Bytes()),
		DQ:       base64.StdEncoding.EncodeToString(priv.Precomputed.Dq.Bytes()),
		InverseQ: base64.StdEncoding.EncodeToString(priv.Precomputed.Qinv.Bytes()),
	}
	xmlBytes, _ := xml.Marshal(kv)
	return base64.StdEncoding.EncodeToString(xmlBytes)
}

// GenerateRSAKeyPair generates a new RSA key pair.
func GenerateRSAKeyPair() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

// RSAEncrypt encrypts data with an RSA public key using PKCS#1 v1.5 padding.
func RSAEncrypt(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, pub, plaintext)
}

// RSADecrypt decrypts data with an RSA private key using PKCS#1 v1.5 padding.
func RSADecrypt(priv *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
}

// RSASign signs data with SHA-512 and RSA PKCS#1 v1.5.
func RSASign(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	hash := sha512.Sum512(data)
	return rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA512, hash[:])
}

// RSAVerify verifies an RSA SHA-512 PKCS#1 v1.5 signature.
func RSAVerify(pub *rsa.PublicKey, data, signature []byte) error {
	hash := sha512.Sum512(data)
	return rsa.VerifyPKCS1v15(pub, crypto.SHA512, hash[:], signature)
}

// base64ToBigInt decodes a Base64 string to a big.Int (big-endian unsigned).
func base64ToBigInt(s string) (*big.Int, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(b), nil
}
