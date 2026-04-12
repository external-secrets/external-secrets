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

// Package crd implements an External Secrets provider that reads data from
// arbitrary Kubernetes Custom Resources (CRDs) using ServiceAccount token auth.
package crd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/jmespath/go-jmespath"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

// GetSecret retrieves a single value from a CRD object.
// ref.Key is interpreted per store kind (see parseRemoteRefKey); ref.Property is an optional JMESPath expression.
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
	if ref.Key == "" {
		return nil, errors.New("crd: ref.key must not be empty")
	}
	objectName, keyNamespace, err := parseRemoteRefKey(c.storeKind, ref.Key)
	if err != nil {
		return nil, err
	}
	ns := ""
	if keyNamespace != nil {
		ns = *keyNamespace
	}
	var requestedKeys []string
	if ref.Property != "" {
		requestedKeys = []string{ref.Property}
	}
	allowed, err := c.matchesWhitelistRule(objectName, ns, requestedKeys)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("crd: request for %q denied by whitelist rules", ref.Key)
	}
	return c.getObject(ctx, ref.Key)
}

// GetAllSecrets lists CRD objects whose logical keys match the store Name pattern
// (regex) and returns a map of logicalKey to serialised value.
// For SecretStore (namespaced kind), listing is limited to the store namespace and keys are object names.
// For ClusterSecretStore with a namespaced kind, listing is all namespaces unless remoteNamespace is set;
// keys are namespace/name. Cluster-scoped kinds use object names only.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	gvr := c.buildGVR()

	var (
		list *unstructured.UnstructuredList
		err  error
	)
	resourceInterface := c.dynClient.Resource(gvr)
	if !c.namespaced || (c.storeKind == esv1.ClusterSecretStoreKind && c.store.RemoteNamespace == "") {
		list, err = resourceInterface.List(ctx, metav1.ListOptions{})
	} else {
		ns := c.namespace
		if c.storeKind == esv1.ClusterSecretStoreKind {
			ns = c.store.RemoteNamespace
		}
		if ns == "" {
			return nil, fmt.Errorf("crd: namespace is required for namespaced resource kind %q", c.store.Resource.Kind)
		}
		list, err = resourceInterface.Namespace(ns).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("crd: failed to list %s: %w", c.store.Resource.Kind, err)
	}

	var re *regexp.Regexp
	if ref.Name != nil && ref.Name.RegExp != "" {
		re, err = regexp.Compile(ref.Name.RegExp)
		if err != nil {
			return nil, fmt.Errorf("crd: invalid name pattern %q: %w", ref.Name.RegExp, err)
		}
	}

	result := make(map[string][]byte, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		objName := item.GetName()
		objNS := item.GetNamespace()
		logicalKey := objName
		if c.namespaced && c.storeKind == esv1.ClusterSecretStoreKind && c.store.RemoteNamespace == "" {
			logicalKey = objNS + "/" + objName
		}
		if re != nil && !re.MatchString(logicalKey) {
			continue
		}
		allowed, err := c.matchesWhitelistRule(objName, objNS, nil)
		if err != nil {
			return nil, err
		}
		if !allowed {
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

// SecretExists returns true when the named CRD object exists.
func (c *Client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	_, err := c.getObject(ctx, ref.GetRemoteKey())
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
	return esv1.ValidationResultReady, nil
}

// Close is a no-op for the CRD provider.
func (c *Client) Close(_ context.Context) error {
	return nil
}

// buildGVR returns the GroupVersionResource using the plural name resolved
// at client construction time via server-side discovery.
func (c *Client) buildGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    c.store.Resource.Group,
		Version:  c.store.Resource.Version,
		Resource: c.plural,
	}
}

// getObject fetches a CRD object using remoteRef.key semantics (see parseRemoteRefKey).
func (c *Client) getObject(ctx context.Context, remoteKey string) (*unstructured.Unstructured, error) {
	objName, keyNS, err := parseRemoteRefKey(c.storeKind, remoteKey)
	if err != nil {
		return nil, err
	}

	gvr := c.buildGVR()
	ri := c.dynClient.Resource(gvr)

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
		obj, err := ri.Namespace(requestNS).Get(ctx, objName, metav1.GetOptions{})
		if err != nil {
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
	obj, err := ri.Get(ctx, objName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, esv1.NoSecretError{}
		}
		return nil, fmt.Errorf("crd: failed to get %s/%s: %w", c.store.Resource.Kind, objName, err)
	}
	return obj, nil
}

// extractValue serialises an unstructured object (or a sub-field) to bytes.
// property is a JMESPath expression taking precedence over fields.
// fields is the store-level Properties list restricting which fields are included.
func extractValue(obj *unstructured.Unstructured, property string, fields []string) ([]byte, error) {
	raw, err := json.Marshal(obj.Object)
	if err != nil {
		return nil, fmt.Errorf("crd: failed to marshal object: %w", err)
	}

	if property != "" {
		val, err := jmespath.Search(property, obj.Object)
		if err != nil {
			return nil, fmt.Errorf("crd: invalid property expression %q: %w", property, err)
		}
		if val == nil {
			return nil, fmt.Errorf("crd: property %q not found in object %q", property, obj.GetName())
		}
		s, ok := val.(string)
		if ok {
			return []byte(s), nil
		}
		b, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("crd: failed to marshal property %q in object %q: %w", property, obj.GetName(), err)
		}
		return b, nil
	}

	if len(fields) > 0 {
		subset := make(map[string]any, len(fields))
		for _, f := range fields {
			val, err := jmespath.Search(f, obj.Object)
			if err != nil {
				return nil, fmt.Errorf("crd: invalid property expression %q: %w", f, err)
			}
			if val != nil {
				subset[f] = val
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

// matchesWhitelistRule checks whether the given object (identified by its bare
// name and namespace) is permitted by the store's whitelist rules.
// objectName is always the bare name without any namespace prefix.
// namespace is the object's namespace; it is only considered when the store is
// a ClusterSecretStore and rule.Namespace is set – for SecretStore the field
// is ignored because the namespace is implicitly fixed to the store namespace.
func (c *Client) matchesWhitelistRule(objectName, namespace string, requestedKeys []string) (bool, error) {
	if c.store.Whitelist == nil || len(c.store.Whitelist.Rules) == 0 {
		return true, nil
	}

	for _, rule := range c.store.Whitelist.Rules {
		// Name check
		if rule.Name != "" {
			re, err := regexp.Compile(rule.Name)
			if err != nil {
				return false, fmt.Errorf("crd: invalid whitelist name regex %q: %w", rule.Name, err)
			}
			if !re.MatchString(objectName) {
				continue
			}
		}

		// Namespace check: only evaluated for ClusterSecretStore when the rule
		// specifies a namespace pattern. Cluster-scoped objects (namespace=="")
		// never match a namespace rule — the rule explicitly targets namespaced
		// objects.
		if rule.Namespace != "" && c.storeKind == esv1.ClusterSecretStoreKind {
			if namespace == "" {
				continue
			}
			re, err := regexp.Compile(rule.Namespace)
			if err != nil {
				return false, fmt.Errorf("crd: invalid whitelist namespace regex %q: %w", rule.Namespace, err)
			}
			if !re.MatchString(namespace) {
				continue
			}
		}

		if len(rule.Properties) == 0 {
			return true, nil
		}
		if len(requestedKeys) == 0 {
			continue
		}

		allMatched := true
		for _, key := range requestedKeys {
			matched := false
			for _, pattern := range rule.Properties {
				re, err := regexp.Compile(pattern)
				if err != nil {
					return false, fmt.Errorf("crd: invalid whitelist property regex %q: %w", pattern, err)
				}
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
			return true, nil
		}
	}

	return false, nil
}
