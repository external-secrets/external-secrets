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

func rsaDecrypt(scheme string, hash string, in string, privateKey string) (string, error) {

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
