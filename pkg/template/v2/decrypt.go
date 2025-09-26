/*
Copyright © 2025 ESO Maintainer Team

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

package template

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/pem"
	"errors"
	"fmt"
	"hash"
)

var (
	errParsePK    = errors.New("could not parse private key")
	errRSADecrypt = errors.New("error decrypting data with RSA")
)

const (
	errSchemeNotSupported = "decryption scheme %v is not supported"
	errParseRSAPK         = "could not parse RSA private key"
	errDecodePEM          = "failed to decode PEM block"
	errWrap               = "%w: %v"
)

func rsaDecrypt(scheme, hash, in, privateKey string) (string, error) {
	switch scheme {
	case "None":
		return in, nil
	case "RSA-OAEP":

		pemBlock, _ := pem.Decode([]byte(privateKey))
		if pemBlock == nil {
			return "", fmt.Errorf(errDecodePEM)
		}

		parsedPrivateKey, err := parsePrivateKey(pemBlock.Bytes)
		if err != nil {
			return "", fmt.Errorf(errWrap, errParsePK, err)
		}

		rsaPrivateKey, isValid := parsedPrivateKey.(*rsa.PrivateKey)
		if !isValid {
			return "", fmt.Errorf(errParseRSAPK)
		}

		out, err := rsa.DecryptOAEP(getHash(hash), nil, rsaPrivateKey, []byte(in), nil)
		if err != nil {
			return "", fmt.Errorf(errWrap, errRSADecrypt, err)
		}
		return string(out), nil
	default:
		return "", fmt.Errorf(errSchemeNotSupported, scheme)
	}
}

func getHash(hash string) hash.Hash {
	switch hash {
	case "None":
		return sha256.New()
	case "SHA1":
		return crypto.SHA1.New()
	case "SHA256":
		return sha256.New()
	case "SHA512":
		return sha512.New()
	default:
		return sha256.New()
	}
}
