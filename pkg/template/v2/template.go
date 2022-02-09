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
	"encoding/pem"
	"fmt"
	tpl "text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/lestrrat-go/jwx/jwk"
	"golang.org/x/crypto/pkcs12"
	corev1 "k8s.io/api/core/v1"
)

var tplFuncs = tpl.FuncMap{
	"pkcs12key":      pkcs12key,
	"pkcs12keyPass":  pkcs12keyPass,
	"pkcs12cert":     pkcs12cert,
	"pkcs12certPass": pkcs12certPass,

	"jwkPublicKeyPem":  jwkPublicKeyPem,
	"jwkPrivateKeyPem": jwkPrivateKeyPem,
}

// So other templating calls can use the same extra functions.
func FuncMap() tpl.FuncMap {
	return tplFuncs
}

const (
	errParse                = "unable to parse template at key %s: %s"
	errExecute              = "unable to execute template at key %s: %s"
	errDecodePKCS12WithPass = "unable to decode pkcs12 with password: %s"
	errDecodeCertWithPass   = "unable to decode pkcs12 certificate with password: %s"
	errParsePrivKey         = "unable to parse private key type"

	pemTypeCertificate = "CERTIFICATE"
)

func init() {
	sprigFuncs := sprig.TxtFuncMap()
	delete(sprigFuncs, "env")
	delete(sprigFuncs, "expandenv")

	for k, v := range sprigFuncs {
		tplFuncs[k] = v
	}
}

// Execute renders the secret data as template. If an error occurs processing is stopped immediately.
func Execute(tpl, data map[string][]byte, secret *corev1.Secret) error {
	if tpl == nil {
		return nil
	}
	for k, v := range tpl {
		val, err := execute(k, string(v), data)
		if err != nil {
			return fmt.Errorf(errExecute, k, err)
		}
		secret.Data[k] = val
	}
	return nil
}

func execute(k, val string, data map[string][]byte) ([]byte, error) {
	strValData := make(map[string]string, len(data))
	for k := range data {
		strValData[k] = string(data[k])
	}

	t, err := tpl.New(k).
		Funcs(tplFuncs).
		Parse(val)
	if err != nil {
		return nil, fmt.Errorf(errParse, k, err)
	}
	buf := bytes.NewBuffer(nil)
	err = t.Execute(buf, strValData)
	if err != nil {
		return nil, fmt.Errorf(errExecute, k, err)
	}
	return buf.Bytes(), nil
}

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

func parsePrivateKey(block []byte) (interface{}, error) {
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

func jwkPublicKeyPem(jwkjson string) (string, error) {
	k, err := jwk.ParseKey([]byte(jwkjson))
	if err != nil {
		return "", err
	}
	var rawkey interface{}
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
	var pk interface{}
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

func pemEncode(thing, kind string) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := pem.Encode(buf, &pem.Block{Type: kind, Bytes: []byte(thing)})
	return buf.String(), err
}
