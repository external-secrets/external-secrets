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

package utils

import (
	"bytes"
	"context"
	"crypto/sha3"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	tpl "text/template"
	"time"
	"unicode"

	"github.com/go-logr/logr"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/template/v2"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errParse   = "unable to parse transform template: %s"
	errExecute = "unable to execute transform template: %s"
)

var (
	errAddressesNotReady      = errors.New("addresses not ready")
	errEndpointSlicesNotReady = errors.New("endpointSlice objects not ready")
	errKeyNotFound            = errors.New("key not found")
	unicodeRegex              = regexp.MustCompile(`_U([0-9a-fA-F]{4,5})_`)
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

func RewriteMap(operations []esv1.ExternalSecretRewrite, in map[string][]byte) (map[string][]byte, error) {
	out := in
	var err error
	for i, op := range operations {
		out, err = handleRewriteOperation(op, out)
		if err != nil {
			return nil, fmt.Errorf("failed rewrite operation[%v]: %w", i, err)
		}
	}
	return out, nil
}

func handleRewriteOperation(op esv1.ExternalSecretRewrite, in map[string][]byte) (map[string][]byte, error) {
	switch {
	case op.Merge != nil:
		return RewriteMerge(*op.Merge, in)
	case op.Regexp != nil:
		return RewriteRegexp(*op.Regexp, in)
	case op.Transform != nil:
		return RewriteTransform(*op.Transform, in)
	default:
		return in, nil
	}
}

// RewriteMerge merges input values according to the operation's strategy and conflict policy.
func RewriteMerge(operation esv1.ExternalSecretRewriteMerge, in map[string][]byte) (map[string][]byte, error) {
	var out map[string][]byte

	mergedMap, conflicts, err := merge(operation, in)
	if err != nil {
		return nil, err
	}

	if operation.ConflictPolicy != esv1.ExternalSecretRewriteMergeConflictPolicyIgnore {
		if len(conflicts) > 0 {
			return nil, fmt.Errorf("merge failed with conflicts: %v", strings.Join(conflicts, ", "))
		}
	}

	switch operation.Strategy {
	case esv1.ExternalSecretRewriteMergeStrategyExtract, "":
		out = make(map[string][]byte)
		for k, v := range mergedMap {
			byteValue, err := GetByteValue(v)
			if err != nil {
				return nil, fmt.Errorf("merge failed with failed to convert value to []byte: %w", err)
			}
			out[k] = byteValue
		}
	case esv1.ExternalSecretRewriteMergeStrategyJSON:
		out = make(map[string][]byte)
		if operation.Into == "" {
			return nil, fmt.Errorf("merge failed with missing 'into' field")
		}
		mergedBytes, err := JSONMarshal(mergedMap)
		if err != nil {
			return nil, fmt.Errorf("merge failed with failed to marshal merged map: %w", err)
		}
		maps.Copy(out, in)
		out[operation.Into] = mergedBytes
	}

	return out, nil
}

// merge merges the input maps and returns the merged map and a list of conflicting keys.
func merge(operation esv1.ExternalSecretRewriteMerge, in map[string][]byte) (map[string]any, []string, error) {
	mergedMap := make(map[string]any)
	conflicts := make([]string, 0)

	// sort keys with priority keys at the end in their specified order
	keys := sortKeysWithPriority(operation, in)

	for _, key := range keys {
		value, exists := in[key]
		if !exists {
			if operation.PriorityPolicy == esv1.ExternalSecretRewriteMergePriorityPolicyIgnoreNotFound {
				continue
			}
			return nil, nil, fmt.Errorf("merge failed with key %q not found in input map", key)
		}
		var jsonMap map[string]any
		if err := json.Unmarshal(value, &jsonMap); err != nil {
			return nil, nil, fmt.Errorf("merge failed with failed to unmarshal JSON: %w", err)
		}

		for k, v := range jsonMap {
			if _, conflict := mergedMap[k]; conflict {
				conflicts = append(conflicts, k)
			}
			mergedMap[k] = v
		}
	}

	return mergedMap, conflicts, nil
}

// sortKeysWithPriority sorts keys with priority keys at the end in their specified order.
// Non-priority keys are sorted alphabetically and placed before priority keys.
func sortKeysWithPriority(operation esv1.ExternalSecretRewriteMerge, in map[string][]byte) []string {
	keys := make([]string, 0, len(in))
	for k := range in {
		if !slices.Contains(operation.Priority, k) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	keys = append(keys, operation.Priority...)
	return keys
}

// RewriteRegexp rewrites a single Regexp Rewrite Operation.
func RewriteRegexp(operation esv1.ExternalSecretRewriteRegexp, in map[string][]byte) (map[string][]byte, error) {
	out := make(map[string][]byte)
	re, err := regexp.Compile(operation.Source)
	if err != nil {
		return nil, fmt.Errorf("regexp failed with failed to compile: %w", err)
	}
	for key, value := range in {
		newKey := re.ReplaceAllString(key, operation.Target)
		out[newKey] = value
	}
	return out, nil
}

// RewriteTransform applies string transformation on each secret key name to rewrite.
func RewriteTransform(operation esv1.ExternalSecretRewriteTransform, in map[string][]byte) (map[string][]byte, error) {
	out := make(map[string][]byte)
	for key, value := range in {
		data := map[string][]byte{
			"value": []byte(key),
		}

		result, err := transform(operation.Template, data)
		if err != nil {
			return nil, fmt.Errorf("transform failed with failed to transform key: %w", err)
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

// DecodeMap decodes values from a secretMap.
func DecodeMap(strategy esv1.ExternalSecretDecodingStrategy, in map[string][]byte) (map[string][]byte, error) {
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

func Decode(strategy esv1.ExternalSecretDecodingStrategy, in []byte) ([]byte, error) {
	switch strategy {
	case esv1.ExternalSecretDecodeBase64:
		out, err := base64.StdEncoding.DecodeString(string(in))
		if err != nil {
			return nil, err
		}
		return out, nil
	case esv1.ExternalSecretDecodeBase64URL:
		out, err := base64.URLEncoding.DecodeString(string(in))
		if err != nil {
			return nil, err
		}
		return out, nil
	case esv1.ExternalSecretDecodeNone:
		return in, nil
	// default when stored version is v1alpha1
	case "":
		return in, nil
	case esv1.ExternalSecretDecodeAuto:
		out, err := Decode(esv1.ExternalSecretDecodeBase64, in)
		if err != nil {
			out, err := Decode(esv1.ExternalSecretDecodeBase64URL, in)
			if err != nil {
				return Decode(esv1.ExternalSecretDecodeNone, in)
			}
			return out, nil
		}
		return out, nil
	default:
		return nil, fmt.Errorf("decoding strategy %v is not supported", strategy)
	}
}

// ValidateKeys checks if the keys in the secret map are valid keys for a Kubernetes secret.
func ValidateKeys(log logr.Logger, in map[string][]byte) error {
	for key := range in {
		keyLength := len(key)
		if keyLength == 0 {
			delete(in, key)

			log.V(1).Info("key was deleted from the secret output because it did not exist upstream", "key", key)

			continue
		}
		if keyLength > 253 {
			return fmt.Errorf("key has length %d but max is 253: (following is truncated): %s", keyLength, key[:253])
		}
		for _, c := range key {
			if !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '-' && c != '.' && c != '_' {
				return fmt.Errorf("key has invalid character %c, only alphanumeric, '-', '.' and '_' are allowed: %s", c, key)
			}
		}
	}
	return nil
}

// ConvertKeys converts a secret map into a valid key.
// Replaces any non-alphanumeric characters depending on convert strategy.
func ConvertKeys(strategy esv1.ExternalSecretConversionStrategy, in map[string][]byte) (map[string][]byte, error) {
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

func convert(strategy esv1.ExternalSecretConversionStrategy, str string) string {
	rs := []rune(str)
	newName := make([]string, len(rs))
	for rk, rv := range rs {
		if !unicode.IsNumber(rv) &&
			!unicode.IsLetter(rv) &&
			rv != '-' &&
			rv != '.' &&
			rv != '_' {
			switch strategy {
			case esv1.ExternalSecretConversionDefault:
				newName[rk] = "_"
			case esv1.ExternalSecretConversionUnicode:
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

// ObjectHash calculates sha3 sum of the data contained in the secret.
func ObjectHash(object any) string {
	textualVersion := fmt.Sprintf("%+v", object)
	return fmt.Sprintf("%x", sha3.Sum224([]byte(textualVersion)))
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
	errNamespaceNotAllowed = errors.New("namespace should either be empty or match the namespace of the SecretStore for a namespaced SecretStore")
	errRequireNamespace    = errors.New("cluster scope requires namespace")
)

// ValidateSecretSelector just checks if the namespace field is present/absent
// depending on the secret store type.
// We MUST NOT check the name or key property here. It MAY be defaulted by the provider.
func ValidateSecretSelector(store esv1.GenericStore, ref esmeta.SecretKeySelector) error {
	clusterScope := store.GetObjectKind().GroupVersionKind().Kind == esv1.ClusterSecretStoreKind
	if clusterScope && ref.Namespace == nil {
		return errRequireNamespace
	}
	if !clusterScope && ref.Namespace != nil && *ref.Namespace != store.GetNamespace() {
		return errNamespaceNotAllowed
	}
	return nil
}

// ValidateReferentSecretSelector allows
// cluster scoped store without namespace
// this should replace above ValidateServiceAccountSelector once all providers
// support referent auth.
func ValidateReferentSecretSelector(store esv1.GenericStore, ref esmeta.SecretKeySelector) error {
	clusterScope := store.GetObjectKind().GroupVersionKind().Kind == esv1.ClusterSecretStoreKind
	if !clusterScope && ref.Namespace != nil && *ref.Namespace != store.GetNamespace() {
		return errNamespaceNotAllowed
	}
	return nil
}

// ValidateServiceAccountSelector just checks if the namespace field is present/absent
// depending on the secret store type.
// We MUST NOT check the name or key property here. It MAY be defaulted by the provider.
func ValidateServiceAccountSelector(store esv1.GenericStore, ref esmeta.ServiceAccountSelector) error {
	clusterScope := store.GetObjectKind().GroupVersionKind().Kind == esv1.ClusterSecretStoreKind
	if clusterScope && ref.Namespace == nil {
		return errRequireNamespace
	}
	if !clusterScope && ref.Namespace != nil && *ref.Namespace != store.GetNamespace() {
		return errNamespaceNotAllowed
	}
	return nil
}

// ValidateReferentServiceAccountSelector allows
// cluster scoped store without namespace
// this should replace above ValidateServiceAccountSelector once all providers
// support referent auth.
func ValidateReferentServiceAccountSelector(store esv1.GenericStore, ref esmeta.ServiceAccountSelector) error {
	clusterScope := store.GetObjectKind().GroupVersionKind().Kind == esv1.ClusterSecretStoreKind
	if !clusterScope && ref.Namespace != nil && *ref.Namespace != store.GetNamespace() {
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
	defer func() {
		_ = conn.Close()
	}()
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

	return bytes.Equal(valueByte, []byte(*valueString))
}

func ExtractSecretData(data esv1.PushSecretData, secret *corev1.Secret) ([]byte, error) {
	var (
		err   error
		value []byte
		ok    bool
	)
	if data.GetSecretKey() == "" {
		decodedMap := make(map[string]string)
		for k, v := range secret.Data {
			decodedMap[k] = string(v)
		}
		value, err = JSONMarshal(decodedMap)

		if err != nil {
			return nil, fmt.Errorf("failed to marshal secret data: %w", err)
		}
	} else {
		value, ok = secret.Data[data.GetSecretKey()]

		if !ok {
			return nil, fmt.Errorf("failed to find secret key in secret with key: %s", data.GetSecretKey())
		}
	}
	return value, nil
}

// CreateCertOpts contains options for a cert pool creation.
type CreateCertOpts struct {
	CABundle   []byte
	CAProvider *esv1.CAProvider
	StoreKind  string
	Namespace  string
	Client     client.Client
}

// FetchCACertFromSource creates a CertPool using either a CABundle directly, or
// a ConfigMap / Secret.
func FetchCACertFromSource(ctx context.Context, opts CreateCertOpts) ([]byte, error) {
	if len(opts.CABundle) == 0 && opts.CAProvider == nil {
		return nil, nil
	}

	if len(opts.CABundle) > 0 {
		pem, err := base64decode(opts.CABundle)
		if err != nil {
			return nil, fmt.Errorf("failed to decode ca bundle: %w", err)
		}

		return pem, nil
	}

	if opts.CAProvider != nil &&
		opts.StoreKind == esv1.ClusterSecretStoreKind &&
		opts.CAProvider.Namespace == nil {
		return nil, errors.New("missing namespace on caProvider secret")
	}

	switch opts.CAProvider.Type {
	case esv1.CAProviderTypeSecret:
		cert, err := getCertFromSecret(ctx, opts.Client, opts.CAProvider, opts.StoreKind, opts.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get cert from secret: %w", err)
		}

		return cert, nil
	case esv1.CAProviderTypeConfigMap:
		cert, err := getCertFromConfigMap(ctx, opts.Namespace, opts.Client, opts.CAProvider)
		if err != nil {
			return nil, fmt.Errorf("failed to get cert from configmap: %w", err)
		}

		return cert, nil
	}

	return nil, fmt.Errorf("unsupported CA provider type: %s", opts.CAProvider.Type)
}

// GetTargetNamespaces extracts namespaces based on selectors.
func GetTargetNamespaces(ctx context.Context, cl client.Client, namespaceList []string, lbs []*metav1.LabelSelector) ([]corev1.Namespace, error) {
	// make sure we don't alter the passed in slice.
	selectors := make([]*metav1.LabelSelector, 0, len(namespaceList)+len(lbs))
	for _, ns := range namespaceList {
		selectors = append(selectors, &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"kubernetes.io/metadata.name": ns,
			},
		})
	}
	selectors = append(selectors, lbs...)

	var namespaces []corev1.Namespace
	namespaceSet := make(map[string]struct{})
	for _, selector := range selectors {
		labelSelector, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return nil, fmt.Errorf("failed to convert label selector %s: %w", selector, err)
		}

		var nl corev1.NamespaceList
		err = cl.List(ctx, &nl, &client.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return nil, fmt.Errorf("failed to list namespaces by label selector %s: %w", selector, err)
		}

		for _, n := range nl.Items {
			if _, exist := namespaceSet[n.Name]; exist {
				continue
			}
			namespaceSet[n.Name] = struct{}{}
			namespaces = append(namespaces, n)
		}
	}

	return namespaces, nil
}

// NamespacePredicate can be used to watch for new or updated or deleted namespaces.
func NamespacePredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}
			return !reflect.DeepEqual(e.ObjectOld.GetLabels(), e.ObjectNew.GetLabels())
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return true
		},
	}
}

func base64decode(cert []byte) ([]byte, error) {
	if c, err := parseCertificateBytes(cert); err == nil {
		return c, nil
	}

	// try decoding and test for validity again...
	certificate, err := Decode(esv1.ExternalSecretDecodeAuto, cert)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	return parseCertificateBytes(certificate)
}

func parseCertificateBytes(certBytes []byte) ([]byte, error) {
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return nil, errors.New("failed to parse the new certificate, not valid pem data")
	}

	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return nil, fmt.Errorf("failed to validate certificate: %w", err)
	}

	return certBytes, nil
}

func getCertFromSecret(ctx context.Context, c client.Client, provider *esv1.CAProvider, storeKind, namespace string) ([]byte, error) {
	secretRef := esmeta.SecretKeySelector{
		Name: provider.Name,
		Key:  provider.Key,
	}

	if provider.Namespace != nil {
		secretRef.Namespace = provider.Namespace
	}

	cert, err := resolvers.SecretKeyRef(ctx, c, storeKind, namespace, &secretRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve secret key ref: %w", err)
	}

	return []byte(cert), nil
}

func getCertFromConfigMap(ctx context.Context, namespace string, c client.Client, provider *esv1.CAProvider) ([]byte, error) {
	objKey := client.ObjectKey{
		Name:      provider.Name,
		Namespace: namespace,
	}

	if provider.Namespace != nil {
		objKey.Namespace = *provider.Namespace
	}

	configMapRef := &corev1.ConfigMap{}
	err := c.Get(ctx, objKey, configMapRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get caProvider secret %s: %w", objKey.Name, err)
	}

	val, ok := configMapRef.Data[provider.Key]
	if !ok {
		return nil, fmt.Errorf("failed to get caProvider configMap %s -> %s", objKey.Name, provider.Key)
	}

	return []byte(val), nil
}

func CheckEndpointSlicesReady(ctx context.Context, c client.Client, svcName, svcNamespace string) error {
	var sliceList discoveryv1.EndpointSliceList
	err := c.List(ctx, &sliceList,
		client.InNamespace(svcNamespace),
		client.MatchingLabels{"kubernetes.io/service-name": svcName},
	)
	if err != nil {
		return err
	}
	if len(sliceList.Items) == 0 {
		return errEndpointSlicesNotReady
	}
	readyAddresses := 0
	for _, slice := range sliceList.Items {
		for _, ep := range slice.Endpoints {
			if ep.Conditions.Ready != nil && *ep.Conditions.Ready {
				readyAddresses += len(ep.Addresses)
			}
		}
	}
	if readyAddresses == 0 {
		return errAddressesNotReady
	}
	return nil
}

// ParseJWTClaims extracts claims from a JWT token string.
func ParseJWTClaims(tokenString string) (map[string]interface{}, error) {
	// Split the token into its three parts
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Decode the payload (the second part of the token)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("error decoding payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("error un-marshaling claims: %w", err)
	}
	return claims, nil
}

// ExtractJWTExpiration extracts the expiration time from a JWT token string.
func ExtractJWTExpiration(tokenString string) (string, error) {
	claims, err := ParseJWTClaims(tokenString)
	if err != nil {
		return "", fmt.Errorf("error getting claims: %w", err)
	}
	exp, ok := claims["exp"].(float64)
	if ok {
		return strconv.FormatFloat(exp, 'f', -1, 64), nil
	}

	return "", fmt.Errorf("exp claim not found or wrong type")
}

// FetchServiceAccountToken creates a service account token for the specified service account.
func FetchServiceAccountToken(ctx context.Context, saRef esmeta.ServiceAccountSelector, namespace string) (string, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return "", err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences: saRef.Audiences,
		},
	}
	tokenResponse, err := kubeClient.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, saRef.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}
	return tokenResponse.Status.Token, nil
}
