/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package anywhere

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// Load the private key referenced by `privateKeyId`.
func readPrivateKeyData(privateKeyID string) (crypto.PrivateKey, error) {
	if key, err := readPKCS8PrivateKey(privateKeyID); err == nil {
		return key, nil
	}

	if key, err := readECPrivateKey(privateKeyID); err == nil {
		return key, nil
	}

	if key, err := readRSAPrivateKey(privateKeyID); err == nil {
		return key, nil
	}

	return nil, errors.New("unable to parse private key")
}

func readECPrivateKey(privateKeyId string) (ecdsa.PrivateKey, error) {
	block, err := parseDERFromPEM(privateKeyId, "EC PRIVATE KEY")
	if err != nil {
		return ecdsa.PrivateKey{}, errors.New("could not parse PEM data")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return ecdsa.PrivateKey{}, errors.New("could not parse private key")
	}

	return *privateKey, nil
}

func readRSAPrivateKey(privateKeyId string) (rsa.PrivateKey, error) {
	block, err := parseDERFromPEM(privateKeyId, "RSA PRIVATE KEY")
	if err != nil {
		return rsa.PrivateKey{}, errors.New("could not parse PEM data")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return rsa.PrivateKey{}, errors.New("could not parse private key")
	}

	return *privateKey, nil
}

func readPKCS8PrivateKey(privateKeyId string) (crypto.PrivateKey, error) {
	block, err := parseDERFromPEM(privateKeyId, "PRIVATE KEY")
	if err != nil {
		return nil, errors.New("could not parse PEM data")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("could not parse private key")
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if ok {
		return *rsaPrivateKey, nil
	}

	ecPrivateKey, ok := privateKey.(*ecdsa.PrivateKey)
	if ok {
		return *ecPrivateKey, nil
	}

	return nil, errors.New("could not parse PKCS8 private key")
}

func parseDERFromPEM(pemDataId string, blockType string) (*pem.Block, error) {
	bytes := []byte(pemDataId)

	var block *pem.Block
	for len(bytes) > 0 {
		block, bytes = pem.Decode(bytes)
		if block == nil {
			return nil, errors.New("unable to parse PEM data")
		}
		if block.Type == blockType {
			return block, nil
		}
	}
	return nil, errors.New("requested block type could not be found")
}
