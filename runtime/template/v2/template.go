/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

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
	"strings"
	tpl "text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/feature"
)

var tplFuncs = tpl.FuncMap{
	"pkcs12key":      pkcs12key,
	"pkcs12keyPass":  pkcs12keyPass,
	"pkcs12cert":     pkcs12cert,
	"pkcs12certPass": pkcs12certPass,

	"pemToPkcs12":               pemToPkcs12,
	"pemToPkcs12Pass":           pemToPkcs12Pass,
	"fullPemToPkcs12":           fullPemToPkcs12,
	"fullPemToPkcs12Pass":       fullPemToPkcs12Pass,
	"pemTruststoreToPKCS12":     pemTruststoreToPKCS12,
	"pemTruststoreToPKCS12Pass": pemTruststoreToPKCS12Pass,

	"filterPEM":       filterPEM,
	"filterCertChain": filterCertChain,

	"jwkPublicKeyPem":  jwkPublicKeyPem,
	"jwkPrivateKeyPem": jwkPrivateKeyPem,

	"toYaml":   toYAML,
	"fromYaml": fromYAML,

	"getSecretKey": getSecretKey,
	"rsaDecrypt":   rsaDecrypt,
}

var leftDelim, rightDelim string

// FuncMap returns the template function map so other templating calls can use the same extra functions.
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
	fs := pflag.NewFlagSet("template", pflag.ExitOnError)
	fs.StringVar(&leftDelim, "template-left-delimiter", "{{", "templating left delimiter")
	fs.StringVar(&rightDelim, "template-right-delimiter", "}}", "templating right delimiter")
	feature.Register(feature.Feature{
		Flags: fs,
	})
}

func applyToTarget(k string, val []byte, target string, obj client.Object) error {
	target = strings.ToLower(target)
	switch target {
	case "annotations":
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[k] = string(val)
		obj.SetAnnotations(annotations)
	case "labels":
		labels := obj.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[k] = string(val)
		obj.SetLabels(labels)
	case "data":
		if err := setField(obj, "data", k, val); err != nil {
			return fmt.Errorf("failed to set data field on object: %w", err)
		}
	case "spec":
		if err := setField(obj, "spec", k, val); err != nil {
			return fmt.Errorf("failed to set data field on object: %w", err)
		}
	default:
		parts := strings.Split(target, ".")
		if len(parts) == 0 {
			return fmt.Errorf("invalid path: %s", target)
		}

		unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return fmt.Errorf("failed to convert object to unstructured: %w", err)
		}

		// Navigate to the parent of the target field
		current := unstructured
		for i := range len(parts) - 1 {
			part := parts[i]
			if current[part] == nil {
				current[part] = make(map[string]any)
			}
			next, ok := current[part].(map[string]any)
			if !ok {
				return fmt.Errorf("path %s is not a map at segment %s", target, part)
			}
			current = next
		}

		// Set the value at the final key
		lastPart := parts[len(parts)-1]
		current[lastPart] = tryParseYAML(val)

		// Convert back to the original object type
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, obj); err != nil {
			return fmt.Errorf("failed to convert unstructured to object: %w", err)
		}
	}

	// all fields have been nilled out if they weren't set.
	if obj.GetLabels() == nil {
		obj.SetLabels(make(map[string]string))
	}
	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(make(map[string]string))
	}

	return nil
}

func valueScopeApply(tplMap, data map[string][]byte, target string, secret client.Object) error {
	for k, v := range tplMap {
		val, err := execute(k, string(v), data)
		if err != nil {
			return fmt.Errorf(errExecute, k, err)
		}
		if err := applyToTarget(k, val, target, secret); err != nil {
			return fmt.Errorf("failed to apply to target: %w", err)
		}
	}
	return nil
}

func mapScopeApply(tpl string, data map[string][]byte, target string, secret client.Object) error {
	val, err := execute(tpl, tpl, data)
	if err != nil {
		return fmt.Errorf(errExecute, tpl, err)
	}
	// TODO: need to figure out larger values?
	fmt.Println("motherfucking value:", string(val))
	src := make(map[string]string)
	err = yaml.Unmarshal(val, &src)
	if err != nil {
		return fmt.Errorf("could not unmarshal template to 'map[string][]byte': %w", err)
	}
	for k, val := range src {
		if err := applyToTarget(k, []byte(val), target, secret); err != nil {
			return fmt.Errorf("failed to apply to target: %w", err)
		}
	}
	return nil
}

// Execute renders the secret data as template. If an error occurs processing is stopped immediately.
func Execute(tpl, data map[string][]byte, scope esapi.TemplateScope, target string, secret client.Object) error {
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
		Delims(leftDelim, rightDelim).
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

// setData sets the data field of the object.
func setField(obj client.Object, field, k string, val []byte) error {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return fmt.Errorf("failed to convert object to unstructured: %w", err)
	}
	_, ok := m[field]
	if !ok {
		m[field] = map[string]any{}
	}
	specMap, ok := m[field].(map[string]any)
	if !ok {
		return fmt.Errorf("failed to convert data to map[string][]byte")
	}

	specMap[k] = tryParseYAML(val)
	m[field] = specMap

	// Convert back to the original object type
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(m, obj); err != nil {
		return fmt.Errorf("failed to convert unstructured to object: %w", err)
	}
	return nil
}

// tryParseYAML attempts to parse a string value as YAML, returns original value if parsing fails.
func tryParseYAML(value any) any {
	str, ok := value.(string)
	if !ok {
		return value
	}

	var parsed any
	if err := yaml.Unmarshal([]byte(str), &parsed); err == nil {
		return parsed
	}

	return value
}
