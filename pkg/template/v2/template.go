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
	"fmt"
	tpl "text/template"

	"github.com/Masterminds/sprig/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var tplFuncs = tpl.FuncMap{
	"pkcs12key":      pkcs12key,
	"pkcs12keyPass":  pkcs12keyPass,
	"pkcs12cert":     pkcs12cert,
	"pkcs12certPass": pkcs12certPass,

	"pemToPkcs12":         pemToPkcs12,
	"pemToPkcs12Pass":     pemToPkcs12Pass,
	"fullPemToPkcs12":     fullPemToPkcs12,
	"fullPemToPkcs12Pass": fullPemToPkcs12Pass,

	"filterPEM": filterPEM,

	"jwkPublicKeyPem":  jwkPublicKeyPem,
	"jwkPrivateKeyPem": jwkPrivateKeyPem,

	"toYaml":   toYAML,
	"fromYaml": fromYAML,
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
	pemTypeKey         = "PRIVATE KEY"
)

func init() {
	sprigFuncs := sprig.TxtFuncMap()
	delete(sprigFuncs, "env")
	delete(sprigFuncs, "expandenv")

	for k, v := range sprigFuncs {
		tplFuncs[k] = v
	}
}

func applyToTarget(k, val string, target esapi.TemplateTarget, secret *corev1.Secret) {
	switch target {
	case esapi.TemplateTargetAnnotations:
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string)
		}
		secret.Annotations[k] = val
	case esapi.TemplateTargetLabels:
		if secret.Labels == nil {
			secret.Labels = make(map[string]string)
		}
		secret.Labels[k] = val
	case esapi.TemplateTargetData:
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[k] = []byte(val)
	default:
	}
}

func valueScopeApply(tplMap, data map[string][]byte, target esapi.TemplateTarget, secret *corev1.Secret) error {
	for k, v := range tplMap {
		val, err := execute(k, string(v), data)
		if err != nil {
			return fmt.Errorf(errExecute, k, err)
		}
		applyToTarget(k, string(val), target, secret)
	}
	return nil
}

func mapScopeApply(tpl string, data map[string][]byte, target esapi.TemplateTarget, secret *corev1.Secret) error {
	val, err := execute(tpl, tpl, data)
	if err != nil {
		return fmt.Errorf(errExecute, tpl, err)
	}
	src := make(map[string]string)
	err = yaml.Unmarshal(val, &src)
	if err != nil {
		return fmt.Errorf("could not unmarshal template to 'map[string][]byte': %w", err)
	}
	for k, val := range src {
		applyToTarget(k, val, target, secret)
	}
	return nil
}

// Execute renders the secret data as template. If an error occurs processing is stopped immediately.
func Execute(tpl, data map[string][]byte, scope esapi.TemplateScope, target esapi.TemplateTarget, secret *corev1.Secret) error {
	if tpl == nil {
		return nil
	}
	switch scope {
	case esapi.TemplateScopeKeysAndValues:
		for _, v := range tpl {
			err := mapScopeApply(string(v), data, target, secret)
			if err != nil {
				return err
			}
		}
	case esapi.TemplateScopeValues:
		err := valueScopeApply(tpl, data, target, secret)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown scope '%v': expected 'Values' or 'KeysAndValues'", scope)
	}
	return nil
}

func execute(k, val string, data map[string][]byte) ([]byte, error) {
	strValData := make(map[string]string, len(data))
	for k := range data {
		strValData[k] = string(data[k])
	}

	t, err := tpl.New(k).
		Option("missingkey=error").
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
