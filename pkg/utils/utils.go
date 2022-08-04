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

	// nolint:gosec
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

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
				return nil, fmt.Errorf("failed rewriting operation[%v]: %w", i, err)
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

// MergeStringMap performs a deep clone from src to dest.
func MergeStringMap(dest, src map[string]string) {
	for k, v := range src {
		dest[k] = v
	}
}

// IsNil checks if an Interface is nil.
func IsNil(i interface{}) bool {
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
// nolint:gosec
func ObjectHash(object interface{}) string {
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
