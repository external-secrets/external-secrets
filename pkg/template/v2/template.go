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
	"github.com/youmark/pkcs8"
	"golang.org/x/crypto/pkcs12"
	corev1 "k8s.io/api/core/v1"
)

var tplFuncs = tpl.FuncMap{
	"pkcs12key":      pkcs12key,
	"pkcs12keyPass":  pkcs12keyPass,
	"pkcs12cert":     pkcs12cert,
	"pkcs12certPass": pkcs12certPass,

	"pemPrivateKey":  pemPrivateKey,
	"pemCertificate": pemCertificate,

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
	errConvertPrivKey       = "unable to convert pkcs12 private key: %s"
	errDecodeCertWithPass   = "unable to decode pkcs12 certificate with password: %s"
	errEncodePEMKey         = "unable to encode pem private key: %s"
	errEncodePEMCert        = "unable to encode pem certificate: %s"
)

func init() {
	fmt.Printf("calling init in v2 pkg")
	sprigFuncs := sprig.TxtFuncMap()
	delete(sprigFuncs, "env")
	delete(sprigFuncs, "expandenv")

	for k, v := range sprigFuncs {
		fmt.Printf("adding func %s\n", k)
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
	key, _, err := pkcs12.Decode([]byte(input), pass)
	if err != nil {
		return "", fmt.Errorf(errDecodePKCS12WithPass, err)
	}
	kb, err := pkcs8.ConvertPrivateKeyToPKCS8(key)
	if err != nil {
		return "", fmt.Errorf(errConvertPrivKey, err)
	}
	return string(kb), nil
}

func pkcs12key(input string) (string, error) {
	return pkcs12keyPass("", input)
}

func pkcs12certPass(pass, input string) (string, error) {
	_, cert, err := pkcs12.Decode([]byte(input), pass)
	if err != nil {
		return "", fmt.Errorf(errDecodeCertWithPass, err)
	}
	return string(cert.Raw), nil
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

func pemPrivateKey(key string) (string, error) {
	res, err := pemEncode(key, "PRIVATE KEY")
	if err != nil {
		return res, fmt.Errorf(errEncodePEMKey, err)
	}
	return res, nil
}

func pemCertificate(cert string) (string, error) {
	res, err := pemEncode(cert, "CERTIFICATE")
	if err != nil {
		return res, fmt.Errorf(errEncodePEMCert, err)
	}
	return res, nil
}
