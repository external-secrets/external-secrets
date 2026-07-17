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

// Package crd implements an External Secrets provider that reads data from
// arbitrary Kubernetes Custom Resources (CRDs) using ServiceAccount token auth.
package crd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/tidwall/gjson"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

// GetSecret retrieves a single value from a CRD object.
// ref.Key is interpreted per store kind (see parseRemoteRefKey); ref.Property is an optional GJSON path expression.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	obj, err := c.fetchObject(ctx, ref)
	if err != nil {
		return nil, err
	}
	return extractValue(obj, ref.Property, nil)
}

// GetSecretMap returns a map of key/value pairs extracted from a CRD object.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	obj, err := c.fetchObject(ctx, ref)
	if err != nil {
		return nil, err
	}
	raw, err := extractValue(obj, ref.Property, nil)
	if err != nil {
		return nil, err
	}
	return jsonBytesToMap(raw)
}

// fetchObject validates ref.Key, enforces the whitelist, and retrieves the named CRD object.
func (c *Client) fetchObject(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (*unstructured.Unstructured, error) {
	return c.resolveWhitelistedObject(ctx, ref.Key, ref.Property)
}

// resolveWhitelistedObject validates the key, enforces the whitelist, and
// retrieves the named CRD object. Shared by GetSecret and SecretExists so both
// read paths apply the same whitelist gate; without it SecretExists could be
// used to probe the existence of objects the whitelist does not permit.
func (c *Client) resolveWhitelistedObject(ctx context.Context, key, property string) (*unstructured.Unstructured, error) {
	if key == "" {
		return nil, errors.New("crd: ref.key must not be empty")
	}
	objectName, keyNamespace, err := parseRemoteRefKey(c.storeKind, key)
	if err != nil {
		return nil, err
	}
	ns := ""
	if keyNamespace != nil {
		ns = *keyNamespace
	}
	var requestedKeys []string
	if property != "" {
		requestedKeys = []string{property}
	}
	if !c.matchesWhitelistRule(objectName, ns, requestedKeys) {
		return nil, fmt.Errorf("crd: request for %q denied by whitelist rules", key)
	}
	return c.getObject(ctx, objectName, keyNamespace)
}

// GetAllSecrets lists CRD objects whose logical keys match the store Name pattern
// (regex) and returns a map of logicalKey to serialized value.
// For SecretStore (namespaced kind), listing is limited to the store namespace and keys are object names.
// For ClusterSecretStore with a namespaced kind, listing spans all namespaces and keys are
// namespace/name. Cluster-scoped kinds use object names only.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	// Verify the caller actually has "list" permission. The preflight at store
	// bootstrap only checks "get" — moving "list" here means a SA that only
	// ever uses GetSecret does not need list rights, but anything that calls
	// dataFrom.find must.
	if c.listAccessCheck != nil {
		if err := c.listAccessCheck(ctx); err != nil {
			return nil, err
		}
	}

	list := &unstructured.UnstructuredList{}
	gvk := c.buildGVK()
	list.SetGroupVersionKind(gvk.GroupVersion().WithKind(gvk.Kind + "List"))

	var opts []kclient.ListOption
	if c.namespaced && c.storeKind != esv1.ClusterSecretStoreKind {
		// SecretStore over a namespaced kind lists within its own namespace.
		// Cluster-scoped kinds, and a ClusterSecretStore over a namespaced kind,
		// list across all namespaces (no namespace option).
		if c.namespace == "" {
			return nil, fmt.Errorf("crd: namespace is required for namespaced resource kind %q", c.store.Resource.Kind)
		}
		opts = append(opts, kclient.InNamespace(c.namespace))
	}
	if err := c.kube.List(ctx, list, opts...); err != nil {
		return nil, fmt.Errorf("crd: failed to list %s: %w", c.store.Resource.Kind, err)
	}

	var re *regexp.Regexp
	if ref.Name != nil && ref.Name.RegExp != "" {
		compiled, err := regexp.Compile(ref.Name.RegExp)
		if err != nil {
			return nil, fmt.Errorf("crd: invalid name pattern %q: %w", ref.Name.RegExp, err)
		}
		re = compiled
	}

	result := make(map[string][]byte, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		objName := item.GetName()
		objNS := item.GetNamespace()
		logicalKey := objName
		if c.namespaced && c.storeKind == esv1.ClusterSecretStoreKind {
			logicalKey = objNS + "/" + objName
		}
		if re != nil && !re.MatchString(logicalKey) {
			continue
		}
		if !c.matchesWhitelistRule(objName, objNS, nil) {
			continue
		}

		b, err := extractValue(item, "", nil)
		if err != nil {
			return nil, fmt.Errorf("crd: failed to extract value from %s/%s: %w", c.store.Resource.Kind, logicalKey, err)
		}
		result[logicalKey] = b
	}
	return esutils.ConvertKeys(ref.ConversionStrategy, result)
}

// SecretExists returns true when the named CRD object exists and is permitted
// by the whitelist. The whitelist is enforced here (not just in GetSecret) so
// this cannot be used to probe for objects outside the allowed set.
func (c *Client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	_, err := c.resolveWhitelistedObject(ctx, ref.GetRemoteKey(), ref.GetProperty())
	if err != nil {
		if errors.Is(err, esv1.NoSecretError{}) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Validate checks that the provider is correctly configured.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	// A referent ClusterSecretStore cannot be validated at store-creation time:
	// the ServiceAccount namespace is only known once an ExternalSecret consumes
	// the store. Report "unknown" rather than a false "ready".
	if c.referent {
		return esv1.ValidationResultUnknown, nil
	}
	return esv1.ValidationResultReady, nil
}

// Close is a no-op for the CRD provider.
func (c *Client) Close(_ context.Context) error {
	return nil
}

// buildGVK returns the GroupVersionKind of the configured target resource. The
// controller-runtime client's RESTMapper resolves this to the correct resource
// and scope at request time.
func (c *Client) buildGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   c.store.Resource.Group,
		Version: c.store.Resource.Version,
		Kind:    c.store.Resource.Kind,
	}
}

// getObject fetches a CRD object from the already-parsed remoteRef.key
// components (see parseRemoteRefKey). Callers parse the key once and pass the
// object name and optional namespace in, so the key is not re-parsed here.
func (c *Client) getObject(ctx context.Context, objName string, keyNS *string) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(c.buildGVK())

	if c.namespaced {
		var requestNS string
		switch {
		case keyNS != nil:
			requestNS = *keyNS
		case c.storeKind == esv1.SecretStoreKind:
			requestNS = c.namespace
		default:
			return nil, fmt.Errorf("crd: namespaced resource kind %q requires remoteRef.key in the form namespace/objectName when using ClusterSecretStore", c.store.Resource.Kind)
		}
		if requestNS == "" {
			return nil, fmt.Errorf("crd: namespace is required for namespaced resource kind %q", c.store.Resource.Kind)
		}
		if err := c.kube.Get(ctx, kclient.ObjectKey{Namespace: requestNS, Name: objName}, obj); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, esv1.NoSecretError{}
			}
			return nil, fmt.Errorf("crd: failed to get %s %s/%s: %w", c.store.Resource.Kind, requestNS, objName, err)
		}
		return obj, nil
	}

	if keyNS != nil {
		return nil, fmt.Errorf("crd: cluster-scoped resource kind %q does not allow '/' in remoteRef.key (use object name only)", c.store.Resource.Kind)
	}
	if err := c.kube.Get(ctx, kclient.ObjectKey{Name: objName}, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, esv1.NoSecretError{}
		}
		return nil, fmt.Errorf("crd: failed to get %s/%s: %w", c.store.Resource.Kind, objName, err)
	}
	return obj, nil
}

// extractValue serializes an unstructured object (or a sub-field) to bytes.
// property is a GJSON path expression taking precedence over fields; it uses the
// same syntax as the Kubernetes provider (see
// https://github.com/tidwall/gjson/blob/master/SYNTAX.md) so the property
// dialect is consistent across ESO providers.
// fields is the store-level Properties list restricting which fields are included.
func extractValue(obj *unstructured.Unstructured, property string, fields []string) ([]byte, error) {
	raw, err := json.Marshal(obj.Object)
	if err != nil {
		return nil, fmt.Errorf("crd: failed to marshal object: %w", err)
	}

	if property != "" {
		res := gjson.GetBytes(raw, property)
		if !res.Exists() {
			return nil, fmt.Errorf("crd: property %q not found in object %q", property, obj.GetName())
		}
		// String leaves are returned unwrapped; everything else (objects,
		// arrays, numbers, booleans) is returned as its raw JSON.
		if res.Type == gjson.String {
			return []byte(res.Str), nil
		}
		return []byte(res.Raw), nil
	}

	if len(fields) > 0 {
		subset := make(map[string]any, len(fields))
		for _, f := range fields {
			res := gjson.GetBytes(raw, f)
			if res.Exists() {
				subset[f] = res.Value()
			}
		}
		return esutils.JSONMarshal(subset)
	}

	return raw, nil
}

// jsonBytesToMap converts a JSON byte slice to map[string][]byte.
// String values are unwrapped (JSON quotes removed); non-string values
// (objects, arrays, numbers, booleans) are kept as raw JSON bytes.
//
// When the input is valid JSON but not an object (e.g. a bare string
// `"hello"` or an array `[1,2]`), it cannot be mapped to key/value
// pairs. In that case the raw payload is returned under a single
// "value" key. This is intentional: the input always originates from
// extractValue which already validated it via json.Marshal, so a
// non-object result is expected for non-map properties.
func jsonBytesToMap(raw []byte) (map[string][]byte, error) {
	var kv map[string]json.RawMessage
	if err := json.Unmarshal(raw, &kv); err != nil {
		return map[string][]byte{"value": raw}, nil
	}
	out := make(map[string][]byte, len(kv))
	for k, v := range kv {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			out[k] = []byte(s)
		} else {
			out[k] = v
		}
	}
	return out, nil
}

// compiledWhitelistRule is a pre-validated, pre-compiled form of
// CRDProviderWhitelistRule. Patterns are compiled once at Client construction
// and reused on every read instead of recompiling on the hot path.
type compiledWhitelistRule struct {
	name       *regexp.Regexp   // nil when the rule does not constrain the object name
	namespace  *regexp.Regexp   // nil when the rule does not constrain the namespace
	properties []*regexp.Regexp // empty when the rule does not constrain properties
}

// compileWhitelistRules validates and compiles every regex in the whitelist.
// Returns nil with no error when the whitelist is unset or has no rules.
// Empty rules (no name, no namespace, no properties) are rejected because they
// would match anything and silently widen access.
func compileWhitelistRules(wl *esv1.CRDProviderWhitelist) ([]compiledWhitelistRule, error) {
	if wl == nil || len(wl.Rules) == 0 {
		return nil, nil
	}
	rules := make([]compiledWhitelistRule, 0, len(wl.Rules))
	for i, r := range wl.Rules {
		if r.Name == "" && r.Namespace == "" && len(r.Properties) == 0 {
			return nil, fmt.Errorf("crd: whitelist.rules[%d]: %w", i, errEmptyWhitelistRule)
		}
		var cr compiledWhitelistRule
		if r.Name != "" {
			re, err := regexp.Compile(r.Name)
			if err != nil {
				return nil, fmt.Errorf("crd: invalid whitelist.rules[%d].name regex %q: %w", i, r.Name, err)
			}
			cr.name = re
		}
		if r.Namespace != "" {
			re, err := regexp.Compile(r.Namespace)
			if err != nil {
				return nil, fmt.Errorf("crd: invalid whitelist.rules[%d].namespace regex %q: %w", i, r.Namespace, err)
			}
			cr.namespace = re
		}
		if len(r.Properties) > 0 {
			cr.properties = make([]*regexp.Regexp, 0, len(r.Properties))
			for j, p := range r.Properties {
				re, err := regexp.Compile(p)
				if err != nil {
					return nil, fmt.Errorf("crd: invalid whitelist.rules[%d].properties[%d] regex %q: %w", i, j, p, err)
				}
				cr.properties = append(cr.properties, re)
			}
		}
		rules = append(rules, cr)
	}
	return rules, nil
}

// matchesWhitelistRule checks whether the given object (identified by its bare
// name and namespace) is permitted by the store's whitelist rules.
// objectName is always the bare name without any namespace prefix.
// namespace is the object's namespace; it is only considered when the store is
// a ClusterSecretStore and rule.namespace is set – for SecretStore the field
// is ignored because the namespace is implicitly fixed to the store namespace.
func (c *Client) matchesWhitelistRule(objectName, namespace string, requestedKeys []string) bool {
	if len(c.whitelistRules) == 0 {
		return true
	}
	for _, rule := range c.whitelistRules {
		if rule.name != nil && !rule.name.MatchString(objectName) {
			continue
		}
		// Namespace check: only evaluated for ClusterSecretStore. Cluster-scoped
		// objects (namespace=="") never match a namespace rule — the rule
		// explicitly targets namespaced objects.
		if rule.namespace != nil && c.storeKind == esv1.ClusterSecretStoreKind {
			if namespace == "" || !rule.namespace.MatchString(namespace) {
				continue
			}
		}
		if len(rule.properties) == 0 {
			return true
		}
		if len(requestedKeys) == 0 {
			continue
		}
		allMatched := true
		for _, key := range requestedKeys {
			matched := false
			for _, re := range rule.properties {
				if re.MatchString(key) {
					matched = true
					break
				}
			}
			if !matched {
				allMatched = false
				break
			}
		}
		if allMatched {
			return true
		}
	}
	return false
}
