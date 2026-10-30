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

// Package template provides utilities for working with JWK keys and PEM encoding.
package template

import (
	"crypto/x509"
	"errors"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

func jwkPublicKeyPem(jwkjson string) (string, error) {
	k, err := jwk.ParseKey([]byte(jwkjson))
	if err != nil {
		return "", errors.New("invalid JWK")
	}
	var rawkey any
	err = k.Raw(&rawkey)
	if err != nil {
		return "", errors.New("invalid JWK key")
	}
	mpk, err := x509.MarshalPKIXPublicKey(rawkey)
	if err != nil {
		return "", errors.New("invalid public key")
	}
	return pemEncode(mpk, "PUBLIC KEY")
}

func jwkPrivateKeyPem(jwkjson string) (string, error) {
	k, err := jwk.ParseKey([]byte(jwkjson))
	if err != nil {
		return "", errors.New("invalid JWK")
	}
	var mpk []byte
	var pk any
	err = k.Raw(&pk)
	if err != nil {
		return "", errors.New("invalid JWK key")
	}
	mpk, err = x509.MarshalPKCS8PrivateKey(pk)
	if err != nil {
		return "", errors.New("invalid private key")
	}
	return pemEncode(mpk, "PRIVATE KEY")
}
