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
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	tpl "text/template"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/youmark/pkcs8"
	"golang.org/x/crypto/pkcs12"
	corev1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var tplFuncs = tpl.FuncMap{
	"pkcs12key":      pkcs12key,
	"pkcs12keyPass":  pkcs12keyPass,
	"pkcs12cert":     pkcs12cert,
	"pkcs12certPass": pkcs12certPass,

	"pemPrivateKey":  pemPrivateKey,
	"pemCertificate": pemCertificate,
	"base64decode":   base64decode,
	"base64encode":   base64encode,
	"fromJSON":       fromJSON,
	"toJSON":         toJSON,

	"jwkPublicKeyPem":  jwkPublicKeyPem,
	"jwkPrivateKeyPem": jwkPrivateKeyPem,

	"toString": toString,
	"toBytes":  toBytes,
	"upper":    strings.ToUpper,
	"lower":    strings.ToLower,
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
	errDecodeBase64         = "unable to decode base64: %s"
	errUnmarshalJSON        = "unable to unmarshal json: %s"
	errMarshalJSON          = "unable to marshal json: %s"
)

// Execute renders the secret data as template. If an error occurs processing is stopped immediately.
func Execute(tpl, data map[string][]byte, _ esapi.TemplateScope, _ esapi.TemplateTarget, secret *corev1.Secret) error {
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
	t, err := tpl.New(k).
		Option("missingkey=error").
		Funcs(tplFuncs).
		Parse(val)
	if err != nil {
		return nil, fmt.Errorf(errParse, k, err)
	}
	buf := bytes.NewBuffer(nil)
	err = t.Execute(buf, data)
	if err != nil {
		return nil, fmt.Errorf(errExecute, k, err)
	}
	return buf.Bytes(), nil
}

func pkcs12keyPass(pass string, input []byte) ([]byte, error) {
	key, _, err := pkcs12.Decode(input, pass)
	if err != nil {
		return nil, fmt.Errorf(errDecodePKCS12WithPass, err)
	}
	kb, err := pkcs8.ConvertPrivateKeyToPKCS8(key)
	if err != nil {
		return nil, fmt.Errorf(errConvertPrivKey, err)
	}
	return kb, nil
}

func pkcs12key(input []byte) ([]byte, error) {
	return pkcs12keyPass("", input)
}

func pkcs12certPass(pass string, input []byte) ([]byte, error) {
	_, cert, err := pkcs12.Decode(input, pass)
	if err != nil {
		return nil, fmt.Errorf(errDecodeCertWithPass, err)
	}
	return cert.Raw, nil
}

func pkcs12cert(input []byte) ([]byte, error) {
	return pkcs12certPass("", input)
}

func jwkPublicKeyPem(jwkjson []byte) (string, error) {
	k, err := jwk.ParseKey(jwkjson)
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
	return pemEncode(mpk, "PUBLIC KEY")
}

func jwkPrivateKeyPem(jwkjson []byte) (string, error) {
	k, err := jwk.ParseKey(jwkjson)
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
	return pemEncode(mpk, "PRIVATE KEY")
}

func pemEncode(thing []byte, kind string) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := pem.Encode(buf, &pem.Block{Type: kind, Bytes: thing})
	return buf.String(), err
}

func pemPrivateKey(key []byte) (string, error) {
	res, err := pemEncode(key, "PRIVATE KEY")
	if err != nil {
		return res, fmt.Errorf(errEncodePEMKey, err)
	}
	return res, nil
}

func pemCertificate(cert []byte) (string, error) {
	res, err := pemEncode(cert, "CERTIFICATE")
	if err != nil {
		return res, fmt.Errorf(errEncodePEMCert, err)
	}
	return res, nil
}

func base64decode(in []byte) ([]byte, error) {
	out := make([]byte, len(in))
	l, err := base64.StdEncoding.Decode(out, in)
	if err != nil {
		return nil, fmt.Errorf(errDecodeBase64, err)
	}
	return out[:l], nil
}

func base64encode(in []byte) []byte {
	out := make([]byte, base64.StdEncoding.EncodedLen(len(in)))
	base64.StdEncoding.Encode(out, in)
	return out
}

func fromJSON(in []byte) (any, error) {
	var out any
	err := json.Unmarshal(in, &out)
	if err != nil {
		return nil, fmt.Errorf(errUnmarshalJSON, err)
	}
	return out, nil
}

func toJSON(in any) (string, error) {
	output, err := json.Marshal(in)
	if err != nil {
		return "", fmt.Errorf(errMarshalJSON, err)
	}
	return string(output), nil
}

func toString(in []byte) string {
	return string(in)
}

func toBytes(in string) []byte {
	return []byte(in)
}
