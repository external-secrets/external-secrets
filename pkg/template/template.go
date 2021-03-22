package template

import (
	"bytes"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
	tpl "text/template"

	"github.com/youmark/pkcs8"

	"golang.org/x/crypto/pkcs12"
	corev1 "k8s.io/api/core/v1"
)

var tplFuncs = tpl.FuncMap{
	"pkcs12key":      PKCS12Key,
	"pkcs12keyPass":  PKCS12KeyPass,
	"pkcs12cert":     PKCS12Cert,
	"pkcs12certPass": PKCS12CertPass,

	// TODO: fromJSON, toJSON, base64encode
	"pemPrivateKey":  PemPrivateKey,
	"pemCertificate": PemCertificate,
	"base64decode":   Base64Decode,
	"toString":       toString,
	"upper":          strings.ToUpper,
	"lower":          strings.ToLower,
}

// Execute uses an best-effort approach to render the template
func Execute(secret *corev1.Secret, data map[string][]byte) error {
	for k, v := range secret.Data {
		t, err := tpl.New(k).
			Funcs(tplFuncs).
			Parse(string(v))
		if err != nil {
			return fmt.Errorf("unable to parse template at key %s: %w", k, err)
		}
		buf := bytes.NewBuffer(nil)
		err = t.Execute(buf, data)
		if err != nil {
			return fmt.Errorf("unable to execute template at key %s: %w", k, err)
		}
		secret.Data[k] = buf.Bytes()
	}
	return nil
}

func PKCS12KeyPass(pass string, input []byte) []byte {
	key, _, err := pkcs12.Decode(input, pass)
	if err != nil {
		return nil
	}
	kb, err := pkcs8.ConvertPrivateKeyToPKCS8(key)
	if err != nil {
		return nil
	}
	return kb
}

func PKCS12Key(input []byte) []byte {
	return PKCS12KeyPass("", input)
}

func PKCS12CertPass(pass string, input []byte) []byte {
	_, cert, err := pkcs12.Decode(input, pass)
	if err != nil {
		return nil
	}
	return cert.Raw
}

func PKCS12Cert(input []byte) []byte {
	return PKCS12CertPass("", input)
}

func PemPrivateKey(key []byte) string {
	buf := bytes.NewBuffer(nil)
	err := pem.Encode(buf, &pem.Block{Type: "PRIVATE KEY", Bytes: key})
	if err != nil {
		return ""
	}
	return buf.String()
}

func PemCertificate(cert []byte) string {
	buf := bytes.NewBuffer(nil)
	err := pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: cert})
	if err != nil {
		return ""
	}
	return buf.String()
}

func Base64Decode(in []byte) []byte {
	out := make([]byte, len(in))
	l, err := base64.StdEncoding.Decode(out, in)
	if err != nil {
		return []byte(err.Error())
	}
	return out[:l]
}

func toString(in []byte) string {
	return string(in)
}
