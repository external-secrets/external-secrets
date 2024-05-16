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
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/pkcs12"
	gopkcs12 "software.sslmate.com/src/go-pkcs12"
)

func pkcs12keyPass(pass, input string) (string, error) {
	blocks, err := pkcs12.ToPEM([]byte(input), pass)
	if err != nil {
		return "", fmt.Errorf(errDecodePKCS12WithPass, err)
	}

	var pemData []byte
	for _, block := range blocks {
		// remove bag attributes like localKeyID, friendlyName
		block.Headers = nil
		if block.Type == pemTypeCertificate {
			continue
		}
		key, err := parsePrivateKey(block.Bytes)
		if err != nil {
			return "", err
		}
		// we use pkcs8 because it supports more key types (ecdsa, ed25519), not just RSA
		block.Bytes, err = x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return "", err
		}
		// report error if encode fails
		var buf bytes.Buffer
		if err := pem.Encode(&buf, block); err != nil {
			return "", err
		}
		pemData = append(pemData, buf.Bytes()...)
	}

	return string(pemData), nil
}

func parsePrivateKey(block []byte) (any, error) {
	if k, err := x509.ParsePKCS1PrivateKey(block); err == nil {
		return k, nil
	}
	if k, err := x509.ParsePKCS8PrivateKey(block); err == nil {
		return k, nil
	}
	if k, err := x509.ParseECPrivateKey(block); err == nil {
		return k, nil
	}
	return nil, fmt.Errorf(errParsePrivKey)
}

func pkcs12key(input string) (string, error) {
	return pkcs12keyPass("", input)
}

func pkcs12certPass(pass, input string) (string, error) {
	blocks, err := pkcs12.ToPEM([]byte(input), pass)
	if err != nil {
		return "", fmt.Errorf(errDecodeCertWithPass, err)
	}

	var pemData []byte
	for _, block := range blocks {
		if block.Type != pemTypeCertificate {
			continue
		}
		// remove bag attributes like localKeyID, friendlyName
		block.Headers = nil
		// report error if encode fails
		var buf bytes.Buffer
		if err := pem.Encode(&buf, block); err != nil {
			return "", err
		}
		pemData = append(pemData, buf.Bytes()...)
	}

	// try to order certificate chain. If it fails we return
	// the unordered raw pem data.
	// This fails if multiple leaf or disjunct certs are provided.
	ordered, err := fetchCertChains(pemData)
	if err != nil {
		return string(pemData), nil
	}

	return string(ordered), nil
}

func pkcs12cert(input string) (string, error) {
	return pkcs12certPass("", input)
}

func pemToPkcs12(cert, key string) (string, error) {
	return pemToPkcs12Pass(cert, key, "")
}

func pemToPkcs12Pass(cert, key, pass string) (string, error) {
	certPem, _ := pem.Decode([]byte(cert))

	parsedCert, err := x509.ParseCertificate(certPem.Bytes)
	if err != nil {
		return "", err
	}

	return certsToPkcs12(parsedCert, key, nil, pass)
}

func fullPemToPkcs12(cert, key string) (string, error) {
	return fullPemToPkcs12Pass(cert, key, "")
}

func fullPemToPkcs12Pass(cert, key, pass string) (string, error) {
	certPem, rest := pem.Decode([]byte(cert))

	parsedCert, err := x509.ParseCertificate(certPem.Bytes)
	if err != nil {
		return "", err
	}

	caCerts := make([]*x509.Certificate, 0)
	for len(rest) > 0 {
		caPem, restBytes := pem.Decode(rest)
		rest = restBytes

		caCert, err := x509.ParseCertificate(caPem.Bytes)
		if err != nil {
			return "", err
		}

		caCerts = append(caCerts, caCert)
	}

	return certsToPkcs12(parsedCert, key, caCerts, pass)
}

func certsToPkcs12(cert *x509.Certificate, key string, caCerts []*x509.Certificate, password string) (string, error) {
	keyPem, _ := pem.Decode([]byte(key))
	parsedKey, err := parsePrivateKey(keyPem.Bytes)
	if err != nil {
		return "", err
	}

	pfx, err := gopkcs12.Modern.Encode(parsedKey, cert, caCerts, password)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(pfx), nil
}
