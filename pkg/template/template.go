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
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	tpl "text/template"

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
	"base64decode":   base64decode,
	"base64encode":   base64encode,
	"fromJSON":       fromJSON,
	"toJSON":         toJSON,

	"toString": toString,
	"toBytes":  toBytes,
	"upper":    strings.ToUpper,
	"lower":    strings.ToLower,
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
func Execute(secret *corev1.Secret, data map[string][]byte) error {
	for k, v := range secret.Data {
		t, err := tpl.New(k).
			Funcs(tplFuncs).
			Parse(string(v))
		if err != nil {
			return fmt.Errorf(errParse, k, err)
		}
		buf := bytes.NewBuffer(nil)
		err = t.Execute(buf, data)
		if err != nil {
			return fmt.Errorf(errExecute, k, err)
		}
		secret.Data[k] = buf.Bytes()
	}
	return nil
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

func pemPrivateKey(key []byte) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := pem.Encode(buf, &pem.Block{Type: "PRIVATE KEY", Bytes: key})
	if err != nil {
		return "", fmt.Errorf(errEncodePEMKey, err)
	}
	return buf.String(), err
}

func pemCertificate(cert []byte) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: cert})
	if err != nil {
		return "", fmt.Errorf(errEncodePEMCert, err)
	}
	return buf.String(), nil
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

func fromJSON(in []byte) (interface{}, error) {
	var out interface{}
	err := json.Unmarshal(in, &out)
	if err != nil {
		return nil, fmt.Errorf(errUnmarshalJSON, err)
	}
	return out, nil
}

func toJSON(in interface{}) (string, error) {
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
