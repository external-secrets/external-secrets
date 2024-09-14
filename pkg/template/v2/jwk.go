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

package template

import (
	"crypto/x509"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

func jwkPublicKeyPem(jwkjson string) (string, error) {
	k, err := jwk.ParseKey([]byte(jwkjson))
	if err != nil {
		return "", err
	}
	var rawkey any
	err = k.Raw(&rawkey)
	if err != nil {
		return "", err
	}
	mpk, err := x509.MarshalPKIXPublicKey(rawkey)
	if err != nil {
		return "", err
	}
	return pemEncode(string(mpk), "PUBLIC KEY")
}

func jwkPrivateKeyPem(jwkjson string) (string, error) {
	k, err := jwk.ParseKey([]byte(jwkjson))
	if err != nil {
		return "", err
	}
	var mpk []byte
	var pk any
	err = k.Raw(&pk)
	if err != nil {
		return "", err
	}
	mpk, err = x509.MarshalPKCS8PrivateKey(pk)
	if err != nil {
		return "", err
	}
	return pemEncode(string(mpk), "PRIVATE KEY")
}
