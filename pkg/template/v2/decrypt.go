package template

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"hash"
)

const (
	errParsePK = "could not apply parse of private key type to %v in %v"
)

func rsaDecrypt(scheme string, hash string, privateKeyFormat string, in string, privateKey string) (string, error) {
	switch scheme {
	case "None":
		return in, nil
	case "RSA-OAEP":

		rsaPrivateKey, err := parseRsaPrivateKeyDecrypt(privateKeyFormat, privateKey)
		if err != nil {
			return "", err
		}

		out, err := rsa.DecryptOAEP(getHash(hash), nil, rsaPrivateKey, []byte(in), nil)
		if err != nil {
			return "", err
		}
		return string(out), nil
	default:
		return "", fmt.Errorf("decrypting strategy %v is not supported", scheme)
	}
}

func parseRsaPrivateKeyDecrypt(privateKeyFormat string, privateKey string) (*rsa.PrivateKey, error) {
	pemBlock, _ := pem.Decode([]byte(privateKey))
	if pemBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	switch privateKeyFormat {
	case "None":
		return &rsa.PrivateKey{}, nil
	case "PKCS8":
		parsedPrivateKey, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
		if err != nil {
			return nil, err
		}
		rsaPrivateKey, isValid := parsedPrivateKey.(*rsa.PrivateKey)
		if !isValid {
			return nil, fmt.Errorf(errParsePK, "PKCS8", "RSA")
		}
		return rsaPrivateKey, nil
	case "PKCS1":
		rsaPrivateKey, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
		if err != nil {
			return nil, err
		}
		return rsaPrivateKey, nil
	default:
		return nil, fmt.Errorf("parse private key %v is not supported", privateKeyFormat)
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
