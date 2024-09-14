//Copyright External Secrets Inc. All Rights Reserved

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
