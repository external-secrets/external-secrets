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

package utils

import (
	"bytes"
	"crypto/md5" //nolint:gosec
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	tpl "text/template"
	"time"
	"unicode"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/template/v2"
)

const (
	errParse   = "unable to parse transform template: %s"
	errExecute = "unable to execute transform template: %s"
)

var (
	errKeyNotFound = errors.New("key not found")
	unicodeRegex   = regexp.MustCompile(`_U([0-9a-fA-F]{4,5})_`)
)

// JSONMarshal takes an interface and returns a new escaped and encoded byte slice.
func JSONMarshal(t any) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return bytes.TrimRight(buffer.Bytes(), "\n"), err
}

// MergeByteMap merges map of byte slices.
func MergeByteMap(dst, src map[string][]byte) map[string][]byte {
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func RewriteMap(operations []esv1beta1.ExternalSecretRewrite, in map[string][]byte) (map[string][]byte, error) {
	out := in
	var err error
	for i, op := range operations {
		if op.Regexp != nil {
			out, err = RewriteRegexp(*op.Regexp, out)
			if err != nil {
				return nil, fmt.Errorf("failed rewriting regexp operation[%v]: %w", i, err)
			}
		}
		if op.Transform != nil {
			out, err = RewriteTransform(*op.Transform, out)
			if err != nil {
				return nil, fmt.Errorf("failed rewriting transform operation[%v]: %w", i, err)
			}
		}
	}
	return out, nil
}

// RewriteRegexp rewrites a single Regexp Rewrite Operation.
func RewriteRegexp(operation esv1beta1.ExternalSecretRewriteRegexp, in map[string][]byte) (map[string][]byte, error) {
	out := make(map[string][]byte)
	re, err := regexp.Compile(operation.Source)
	if err != nil {
		return nil, err
	}
	for key, value := range in {
		newKey := re.ReplaceAllString(key, operation.Target)
		out[newKey] = value
	}
	return out, nil
}

// RewriteTransform applies string transformation on each secret key name to rewrite.
func RewriteTransform(operation esv1beta1.ExternalSecretRewriteTransform, in map[string][]byte) (map[string][]byte, error) {
	out := make(map[string][]byte)
	for key, value := range in {
		data := map[string][]byte{
			"value": []byte(key),
		}

		result, err := transform(operation.Template, data)
		if err != nil {
			return nil, err
		}

		newKey := string(result)
		out[newKey] = value
	}
	return out, nil
}

func transform(val string, data map[string][]byte) ([]byte, error) {
	strValData := make(map[string]string, len(data))
	for k := range data {
		strValData[k] = string(data[k])
	}

	t, err := tpl.New("transform").
		Funcs(template.FuncMap()).
		Parse(val)
	if err != nil {
		return nil, fmt.Errorf(errParse, err)
	}
	buf := bytes.NewBuffer(nil)
	err = t.Execute(buf, strValData)
	if err != nil {
		return nil, fmt.Errorf(errExecute, err)
	}
	return buf.Bytes(), nil
}

// DecodeValues decodes values from a secretMap.
func DecodeMap(strategy esv1beta1.ExternalSecretDecodingStrategy, in map[string][]byte) (map[string][]byte, error) {
	out := make(map[string][]byte, len(in))
	for k, v := range in {
		val, err := Decode(strategy, v)
		if err != nil {
			return nil, fmt.Errorf("failure decoding key %v: %w", k, err)
		}
		out[k] = val
	}
	return out, nil
}

func Decode(strategy esv1beta1.ExternalSecretDecodingStrategy, in []byte) ([]byte, error) {
	switch strategy {
	case esv1beta1.ExternalSecretDecodeBase64:
		out, err := base64.StdEncoding.DecodeString(string(in))
		if err != nil {
			return nil, err
		}
		return out, nil
	case esv1beta1.ExternalSecretDecodeBase64URL:
		out, err := base64.URLEncoding.DecodeString(string(in))
		if err != nil {
			return nil, err
		}
		return out, nil
	case esv1beta1.ExternalSecretDecodeNone:
		return in, nil
	// default when stored version is v1alpha1
	case "":
		return in, nil
	case esv1beta1.ExternalSecretDecodeAuto:
		out, err := Decode(esv1beta1.ExternalSecretDecodeBase64, in)
		if err != nil {
			out, err := Decode(esv1beta1.ExternalSecretDecodeBase64URL, in)
			if err != nil {
				return Decode(esv1beta1.ExternalSecretDecodeNone, in)
			}
			return out, nil
		}
		return out, nil
	default:
		return nil, fmt.Errorf("decoding strategy %v is not supported", strategy)
	}
}

func ValidateKeys(in map[string][]byte) bool {
	for key := range in {
		for _, v := range key {
			if !unicode.IsNumber(v) &&
				!unicode.IsLetter(v) &&
				v != '-' &&
				v != '.' &&
				v != '_' {
				return false
			}
		}
	}
	return true
}

// ConvertKeys converts a secret map into a valid key.
// Replaces any non-alphanumeric characters depending on convert strategy.
func ConvertKeys(strategy esv1beta1.ExternalSecretConversionStrategy, in map[string][]byte) (map[string][]byte, error) {
	out := make(map[string][]byte, len(in))
	for k, v := range in {
		key := convert(strategy, k)
		if _, exists := out[key]; exists {
			return nil, fmt.Errorf("secret name collision during conversion: %s", key)
		}
		out[key] = v
	}
	return out, nil
}

func convert(strategy esv1beta1.ExternalSecretConversionStrategy, str string) string {
	rs := []rune(str)
	newName := make([]string, len(rs))
	for rk, rv := range rs {
		if !unicode.IsNumber(rv) &&
			!unicode.IsLetter(rv) &&
			rv != '-' &&
			rv != '.' &&
			rv != '_' {
			switch strategy {
			case esv1beta1.ExternalSecretConversionDefault:
				newName[rk] = "_"
			case esv1beta1.ExternalSecretConversionUnicode:
				newName[rk] = fmt.Sprintf("_U%04x_", rv)
			default:
				newName[rk] = string(rv)
			}
		} else {
			newName[rk] = string(rv)
		}
	}
	return strings.Join(newName, "")
}

// ReverseKeys reverses a secret map into a valid key map as expected by push secrets.
// Replaces the unicode encoded representation characters back to the actual unicode character depending on convert strategy.
func ReverseKeys(strategy esv1alpha1.PushSecretConversionStrategy, in map[string][]byte) (map[string][]byte, error) {
	out := make(map[string][]byte, len(in))
	for k, v := range in {
		key := reverse(strategy, k)
		if _, exists := out[key]; exists {
			return nil, fmt.Errorf("secret name collision during conversion: %s", key)
		}
		out[key] = v
	}
	return out, nil
}

func reverse(strategy esv1alpha1.PushSecretConversionStrategy, str string) string {
	switch strategy {
	case esv1alpha1.PushSecretConversionReverseUnicode:
		matches := unicodeRegex.FindAllStringSubmatchIndex(str, -1)

		for i := len(matches) - 1; i >= 0; i-- {
			match := matches[i]
			start := match[0]
			end := match[1]
			unicodeHex := str[match[2]:match[3]]

			unicodeInt, err := strconv.ParseInt(unicodeHex, 16, 32)
			if err != nil {
				continue // Skip invalid unicode representations
			}

			unicodeChar := fmt.Sprintf("%c", unicodeInt)
			str = str[:start] + unicodeChar + str[end:]
		}

		return str
	case esv1alpha1.PushSecretConversionNone:
		return str
	default:
		return str
	}
}

// MergeStringMap performs a deep clone from src to dest.
func MergeStringMap(dest, src map[string]string) {
	for k, v := range src {
		dest[k] = v
	}
}

var (
	ErrUnexpectedKey = errors.New("unexpected key in data")
	ErrSecretType    = errors.New("can not handle secret value with type")
)

func GetByteValueFromMap(data map[string]any, key string) ([]byte, error) {
	v, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnexpectedKey, key)
	}
	return GetByteValue(v)
}
func GetByteValue(v any) ([]byte, error) {
	switch t := v.(type) {
	case string:
		return []byte(t), nil
	case map[string]any:
		return json.Marshal(t)
	case []string:
		return []byte(strings.Join(t, "\n")), nil
	case json.RawMessage:
		return t, nil
	case []byte:
		return t, nil
	// also covers int and float32 due to json.Marshal
	case float64:
		return []byte(strconv.FormatFloat(t, 'f', -1, 64)), nil
	case json.Number:
		return []byte(t.String()), nil
	case []any:
		return json.Marshal(t)
	case bool:
		return []byte(strconv.FormatBool(t)), nil
	case nil:
		return []byte(nil), nil
	default:
		return nil, fmt.Errorf("%w: %T", ErrSecretType, t)
	}
}

// IsNil checks if an Interface is nil.
func IsNil(i any) bool {
	if i == nil {
		return true
	}
	value := reflect.ValueOf(i)
	if value.Type().Kind() == reflect.Ptr {
		return value.IsNil()
	}
	return false
}

// ObjectHash calculates md5 sum of the data contained in the secret.
//
//nolint:gosec
func ObjectHash(object any) string {
	textualVersion := fmt.Sprintf("%+v", object)
	return fmt.Sprintf("%x", md5.Sum([]byte(textualVersion)))
}

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

var (
	errNamespaceNotAllowed = errors.New("namespace not allowed with namespaced SecretStore")
	errRequireNamespace    = errors.New("cluster scope requires namespace")
)

// ValidateSecretSelector just checks if the namespace field is present/absent
// depending on the secret store type.
// We MUST NOT check the name or key property here. It MAY be defaulted by the provider.
func ValidateSecretSelector(store esv1beta1.GenericStore, ref esmeta.SecretKeySelector) error {
	clusterScope := store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind
	if clusterScope && ref.Namespace == nil {
		return errRequireNamespace
	}
	if !clusterScope && ref.Namespace != nil {
		return errNamespaceNotAllowed
	}
	return nil
}

// ValidateReferentSecretSelector allows
// cluster scoped store without namespace
// this should replace above ValidateServiceAccountSelector once all providers
// support referent auth.
func ValidateReferentSecretSelector(store esv1beta1.GenericStore, ref esmeta.SecretKeySelector) error {
	clusterScope := store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind
	if !clusterScope && ref.Namespace != nil {
		return errNamespaceNotAllowed
	}
	return nil
}

// ValidateServiceAccountSelector just checks if the namespace field is present/absent
// depending on the secret store type.
// We MUST NOT check the name or key property here. It MAY be defaulted by the provider.
func ValidateServiceAccountSelector(store esv1beta1.GenericStore, ref esmeta.ServiceAccountSelector) error {
	clusterScope := store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind
	if clusterScope && ref.Namespace == nil {
		return errRequireNamespace
	}
	if !clusterScope && ref.Namespace != nil {
		return errNamespaceNotAllowed
	}
	return nil
}

// ValidateReferentServiceAccountSelector allows
// cluster scoped store without namespace
// this should replace above ValidateServiceAccountSelector once all providers
// support referent auth.
func ValidateReferentServiceAccountSelector(store esv1beta1.GenericStore, ref esmeta.ServiceAccountSelector) error {
	clusterScope := store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind
	if !clusterScope && ref.Namespace != nil {
		return errNamespaceNotAllowed
	}
	return nil
}

func NetworkValidate(endpoint string, timeout time.Duration) error {
	hostname, err := url.Parse(endpoint)

	if err != nil {
		return fmt.Errorf("could not parse url: %w", err)
	}

	host := hostname.Hostname()
	port := hostname.Port()

	if port == "" {
		port = "443"
	}

	url := fmt.Sprintf("%v:%v", host, port)
	conn, err := net.DialTimeout("tcp", url, timeout)
	if err != nil {
		return fmt.Errorf("error accessing external store: %w", err)
	}
	defer conn.Close()
	return nil
}

func Deref[V any](v *V) V {
	if v == nil {
		// Create zero value
		var res V
		return res
	}
	return *v
}

func Ptr[T any](i T) *T {
	return &i
}

func ConvertToType[T any](obj any) (T, error) {
	var v T

	data, err := json.Marshal(obj)
	if err != nil {
		return v, fmt.Errorf("failed to marshal object: %w", err)
	}

	if err = json.Unmarshal(data, &v); err != nil {
		return v, fmt.Errorf("failed to unmarshal object: %w", err)
	}

	return v, nil
}

// FetchValueFromMetadata fetches a key from a metadata if it exists. It will recursively look in
// embedded values as well. Must be a unique key, otherwise it will just return the first
// occurrence.
func FetchValueFromMetadata[T any](key string, data *apiextensionsv1.JSON, def T) (t T, _ error) {
	if data == nil {
		return def, nil
	}

	m := map[string]any{}
	if err := json.Unmarshal(data.Raw, &m); err != nil {
		return t, fmt.Errorf("failed to parse JSON raw data: %w", err)
	}

	v, err := dig[T](key, m)
	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return def, nil
		}
	}

	return v, nil
}

func dig[T any](key string, data map[string]any) (t T, _ error) {
	if v, ok := data[key]; ok {
		c, k := v.(T)
		if !k {
			return t, fmt.Errorf("failed to convert value to the desired type; was: %T", v)
		}

		return c, nil
	}

	for _, v := range data {
		if ty, ok := v.(map[string]any); ok {
			return dig[T](key, ty)
		}
	}

	return t, errKeyNotFound
}

func CompareStringAndByteSlices(valueString *string, valueByte []byte) bool {
	if valueString == nil {
		return false
	}
	stringToByteSlice := []byte(*valueString)
	if len(stringToByteSlice) != len(valueByte) {
		return false
	}

	for sb := range valueByte {
		if stringToByteSlice[sb] != valueByte[sb] {
			return false
		}
	}

	return true
}
