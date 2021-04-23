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
	"strings"
	tpl "text/template"

	"github.com/youmark/pkcs8"
	"golang.org/x/crypto/pkcs12"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
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

var log = ctrl.Log.WithName("template")

// Execute uses an best-effort approach to render the secret data as template.
func Execute(secret *corev1.Secret, data map[string][]byte) error {
	for k, v := range secret.Data {
		t, err := tpl.New(k).
			Funcs(tplFuncs).
			Parse(string(v))
		if err != nil {
			log.Error(err, "unable to parse template at key", "key", k)
			continue
		}
		buf := bytes.NewBuffer(nil)
		err = t.Execute(buf, data)
		if err != nil {
			log.Error(err, "unable to execute template at key", "key", k)
			continue
		}
		secret.Data[k] = buf.Bytes()
	}
	return nil
}

func pkcs12keyPass(pass string, input []byte) []byte {
	key, _, err := pkcs12.Decode(input, pass)
	if err != nil {
		log.Error(err, "unable to decode pkcs12 with password")
		return nil
	}
	kb, err := pkcs8.ConvertPrivateKeyToPKCS8(key)
	if err != nil {
		log.Error(err, "unable to convert pkcs12 private key")
		return nil
	}
	return kb
}

func pkcs12key(input []byte) []byte {
	return pkcs12keyPass("", input)
}

func pkcs12certPass(pass string, input []byte) []byte {
	_, cert, err := pkcs12.Decode(input, pass)
	if err != nil {
		log.Error(err, "unable to decode pkcs12 certificate with password")
		return nil
	}
	return cert.Raw
}

func pkcs12cert(input []byte) []byte {
	return pkcs12certPass("", input)
}

func pemPrivateKey(key []byte) string {
	buf := bytes.NewBuffer(nil)
	err := pem.Encode(buf, &pem.Block{Type: "PRIVATE KEY", Bytes: key})
	if err != nil {
		log.Error(err, "unable to encode pem private key")
		return ""
	}
	return buf.String()
}

func pemCertificate(cert []byte) string {
	buf := bytes.NewBuffer(nil)
	err := pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: cert})
	if err != nil {
		log.Error(err, "unable to encode pem certificate")
		return ""
	}
	return buf.String()
}

func base64decode(in []byte) []byte {
	out := make([]byte, len(in))
	l, err := base64.StdEncoding.Decode(out, in)
	if err != nil {
		log.Error(err, "unable to encode base64")
		return []byte("")
	}
	return out[:l]
}

func base64encode(in []byte) []byte {
	out := make([]byte, base64.StdEncoding.EncodedLen(len(in)))
	base64.StdEncoding.Encode(out, in)
	return out
}

func fromJSON(in []byte) interface{} {
	var out interface{}
	err := json.Unmarshal(in, &out)
	if err != nil {
		log.Error(err, "unable to unmarshal json")
	}
	return out
}

func toJSON(in interface{}) string {
	output, err := json.Marshal(in)
	if err != nil {
		log.Error(err, "unable to marshal json")
	}
	return string(output)
}

func toString(in []byte) string {
	return string(in)
}

func toBytes(in string) []byte {
	return []byte(in)
}
