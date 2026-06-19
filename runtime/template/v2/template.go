/*
Copyright © The ESO Authors

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
	"maps"
	"strconv"
	"strings"
	tpl "text/template"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/feature"
	"github.com/external-secrets/external-secrets/runtime/template/v2/sprig"
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
	"certSANs":        certSANs,

	"jwkPublicKeyPem":  jwkPublicKeyPem,
	"jwkPrivateKeyPem": jwkPrivateKeyPem,

	"toYaml":   toYAML,
	"fromYaml": fromYAML,

	"rsaDecrypt": rsaDecrypt,
}

var leftDelim, rightDelim string

var (
	errConvertingToUnstructured = "failed to convert object to unstructured: %w"
	errConvertingToObject       = "failed to convert unstructured to object: %w"
)

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
	maps.Copy(tplFuncs, sprig.TxtFuncMap())
	fs := pflag.NewFlagSet("template", pflag.ExitOnError)
	fs.StringVar(&leftDelim, "template-left-delimiter", "{{", "templating left delimiter")
	fs.StringVar(&rightDelim, "template-right-delimiter", "}}", "templating right delimiter")
	feature.Register(feature.Feature{
		Flags: fs,
	})
}

func applyToTarget(k string, val []byte, target string, obj client.Object) error {
	// Match the well-known top-level targets case-insensitively, but keep the
	// original case of target so nested custom-resource paths preserve
	// mixed-case segments (e.g. spec.headers.customRequestHeaders). Issue #6458.
	switch strings.ToLower(target) {
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
			return fmt.Errorf("failed to set spec field on object: %w", err)
		}
	default:
		tokens, err := parseTargetPath(target)
		if err != nil {
			return err
		}
		// Set the value at the target path, converting []byte to string to avoid
		// base64 encoding when serializing.
		leaf := func(any) any { return tryParseYAML(string(val)) }
		if err := applyAtPath(obj, target, tokens, leaf); err != nil {
			return err
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

	// See applyToTarget: switch on the lowercased target but keep the original
	// case so nested paths are not lowercased (issue #6458).
	switch strings.ToLower(target) {
	case "annotations", "labels", "data":
		// normal route
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

		// we are done
		return nil
	}

	// for more complex path, we need to navigate to the last element of the path
	// creating objects in that path if they don't exist and then apply the parsed
	// structure at that location to the entire object.
	var parsed any
	if err := yaml.Unmarshal(val, &parsed); err != nil {
		return fmt.Errorf("could not unmarshal template YAML: %w", err)
	}

	return applyParsedToPath(parsed, target, secret)
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
		return fmt.Errorf(errConvertingToUnstructured, err)
	}
	_, ok := m[field]
	if !ok {
		m[field] = map[string]any{}
	}
	specMap, ok := m[field].(map[string]any)
	if !ok {
		return fmt.Errorf("failed to convert data to map[string][]byte")
	}

	// Secrets require base64-encoded []byte values in the data field
	// Other resources (ConfigMaps, custom resources) need plain string values
	_, isSecret := obj.(*corev1.Secret)
	if isSecret {
		// For Secrets, keep as []byte (will be base64-encoded during serialization)
		specMap[k] = val
	} else {
		// For generic (ConfigMaps, custom resources), use plain strings
		specMap[k] = string(val)
	}
	m[field] = specMap

	// Convert back to the original object type
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(m, obj); err != nil {
		return fmt.Errorf(errConvertingToObject, err)
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

// applyParsedToPath applies a parsed YAML structure to a specific path in the object.
func applyParsedToPath(parsed any, target string, obj client.Object) error {
	tokens, err := parseTargetPath(target)
	if err != nil {
		return err
	}

	// navigate to the last element of the path and apply the entire struct at that location.
	// MERGE the parsed content into existing map content instead of replacing it.
	leaf := func(existing any) any {
		existingMap, existingOk := existing.(map[string]any)
		parsedMap, parsedOk := parsed.(map[string]any)
		if existingOk && parsedOk {
			maps.Copy(existingMap, parsedMap)
			return existingMap
		}
		// existing or parsed value is not a map, replace entirely.
		// this might break if people are trying to overwrite
		// fields that aren't supposed to do that. but that's
		// on the user to keep in mind. If they are trying to
		// update a number field with a complex value, that's
		// going to error on update anyway.
		return parsed
	}

	// single value, aka "spec": replace entirely to preserve historical behavior.
	if len(tokens) == 1 && tokens[0].kind == tokenKey {
		leaf = func(any) any { return parsed }
	}

	return applyAtPath(obj, target, tokens, leaf)
}

// applyAtPath applies leaf at the token path within obj's unstructured form and
// writes the result back into obj. target is used only for error messages.
func applyAtPath(obj client.Object, target string, tokens []pathToken, leaf func(existing any) any) error {
	unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return fmt.Errorf(errConvertingToUnstructured, err)
	}

	updated, err := setAtPath(unstructured, tokens, leaf)
	if err != nil {
		return fmt.Errorf("failed to set path %s: %w", target, err)
	}

	updatedMap, ok := updated.(map[string]any)
	if !ok {
		return fmt.Errorf("failed to set path %s: expected object root, got %T", target, updated)
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(updatedMap, obj); err != nil {
		return fmt.Errorf(errConvertingToObject, err)
	}

	return nil
}

// pathTokenKind distinguishes a map-key access from a slice-index access in a target path.
type pathTokenKind int

const (
	tokenKey pathTokenKind = iota
	tokenIndex
)

// pathToken is a single step in a parsed target path.
type pathToken struct {
	kind pathTokenKind
	name string
	idx  int
}

// maxArrayIndex limits the memory assignation for setAtPath.
const maxArrayIndex = 10000

// parseTargetPath is a pure function parsing and validating a dotted target path, with optional slice notation, into tokens.
// Each step is a pathToken of kind "tokenKey" or "tokenIndex".
//
// e.g.
//
//		input:  "spec.rules[0].from[0].source.notRemoteIpBlocks"
//		output: [
//	  	{key,  "spec"},
//	  	{key,  "rules"},
//	  	{index, 0},
//	  	{key,  "from"},
//	  	{index, 0},
//	  	{key,  "source"},
//	  	{key,  "notRemoteIpBlocks"},
//		]
func parseTargetPath(target string) ([]pathToken, error) {
	var tokens []pathToken
	for seg := range strings.SplitSeq(target, ".") {
		if seg == "" {
			return nil, fmt.Errorf("invalid path %q: empty segment", target)
		}

		name, rest := seg, ""
		if i := strings.IndexByte(seg, '['); i >= 0 {
			name, rest = seg[:i], seg[i:]
		}
		if name != "" {
			tokens = append(tokens, pathToken{kind: tokenKey, name: name})
		} else if rest == "" {
			return nil, fmt.Errorf("invalid path %q: empty segment", target)
		}

		for rest != "" {
			if rest[0] != '[' {
				return nil, fmt.Errorf("invalid path %q: expected '[' near %q", target, rest)
			}
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				return nil, fmt.Errorf("invalid path %q: unterminated '['", target)
			}
			idx, err := strconv.Atoi(rest[1:end])
			if err != nil {
				return nil, fmt.Errorf("invalid path %q: bad index %q", target, rest[1:end])
			}
			if idx < 0 {
				return nil, fmt.Errorf("invalid path %q: negative index %d", target, idx)
			}
			if idx > maxArrayIndex {
				return nil, fmt.Errorf("invalid path %q: index %d exceeds maximum (10000)", target, idx)
			}
			tokens = append(tokens, pathToken{kind: tokenIndex, idx: idx})
			rest = rest[end+1:]
		}
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("invalid path: %s", target)
	}
	if tokens[0].kind != tokenKey {
		return nil, fmt.Errorf("invalid path %q: path must start with a key", target)
	}
	return tokens, nil
}

// setAtPath walks node following tokens, creating intermediate maps and slices as needed,
// and applies leaf to the value at the final location. It returns the possibly-reallocated
// node, since growing a slice replaces its header.
func setAtPath(node any, tokens []pathToken, leaf func(existing any) any) (any, error) {
	tok := tokens[0]
	switch tok.kind {
	case tokenKey:
		return setMap(node, tokens, leaf, tok)
	case tokenIndex:
		return setIndex(node, tokens, leaf, tok)
	default:
		return nil, fmt.Errorf("unknown path token kind %d", tok.kind)
	}
}

// setIndex sets a value in a slice at index.
func setIndex(node any, tokens []pathToken, leaf func(existing any) any, tok pathToken) (any, error) {
	s, ok := node.([]any)
	if !ok && node != nil {
		return nil, fmt.Errorf("expected array at index %d but found %T", tok.idx, node)
	}
	if tok.idx >= len(s) {
		grown := make([]any, tok.idx+1)
		copy(grown, s)
		s = grown
	}
	if err := setLeaf(
		func() any { return s[tok.idx] },
		func(v any) { s[tok.idx] = v },
		tokens, leaf,
	); err != nil {
		return nil, err
	}
	return s, nil
}

// setMap sets a value in a map at token name.
func setMap(node any, tokens []pathToken, leaf func(existing any) any, tok pathToken) (any, error) {
	m, ok := node.(map[string]any)
	if !ok && node != nil {
		return nil, fmt.Errorf("expected map at key %q but found %T", tok.name, node)
	}
	if len(m) == 0 {
		m = make(map[string]any)
	}
	if err := setLeaf(
		func() any { return m[tok.name] },
		func(v any) { m[tok.name] = v },
		tokens, leaf,
	); err != nil {
		return nil, err
	}
	return m, nil
}

// setLeaf writes leaf into a container via get/set closures, recursing when
// the path continues. The closures capture the container (slice or map).
// set is called to set a value in a container, be that a slice or a map.
// get is called to retrieve a value from a container, either a slice or a map.
// This abstraction is necessary because there are no shared slice/map operation even
// in Typed Parameters, so these cannot be reasonably abstracted. At least, it makes
// the entire logic a bit more readable.
func setLeaf(get func() any, set func(any), tokens []pathToken, leaf func(existing any) any) error {
	if len(tokens) == 1 {
		set(leaf(get()))
		return nil
	}
	child, err := setAtPath(get(), tokens[1:], leaf)
	if err != nil {
		return err
	}
	set(child)
	return nil
}
