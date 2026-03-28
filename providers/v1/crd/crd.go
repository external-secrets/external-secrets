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
// ref.Key is the object name; ref.Property is an optional JMESPath expression.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if ref.Key == "" {
		return nil, errors.New("crd: ref.key (object name) must not be empty")
	}
	requestedKeys := requestedPropertyKeys(ref.Property)
	allowed, err := c.matchesWhitelistRule(ref.Key, requestedKeys)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("crd: request for %q denied by whitelist rules", ref.Key)
	}
	obj, err := c.getObject(ctx, ref.Key)
	if err != nil {
		return nil, err
	}
	return extractValue(obj, ref.Property, nil)
}

// GetSecretMap returns a map of key/value pairs extracted from a CRD object.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if ref.Key == "" {
		return nil, errors.New("crd: ref.key (object name) must not be empty")
	}
	requestedKeys := requestedPropertyKeys(ref.Property)
	allowed, err := c.matchesWhitelistRule(ref.Key, requestedKeys)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("crd: request for %q denied by whitelist rules", ref.Key)
	}
	obj, err := c.getObject(ctx, ref.Key)
	if err != nil {
		return nil, err
	}
	raw, err := extractValue(obj, ref.Property, nil)
	if err != nil {
		return nil, err
	}
	return jsonBytesToMap(raw)
}

// GetAllSecrets lists CRD objects whose names match the store Name pattern
// (regex) and returns a map of objectName to serialised value.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	gvr := c.buildGVR()

	var (
		list *unstructured.UnstructuredList
		err  error
	)
	ri := c.dynClient.Resource(gvr)
	if c.namespaced {
		if c.namespace == "" {
			return nil, fmt.Errorf("crd: namespace is required for namespaced resource kind %q", c.store.Resource.Kind)
		}
		list, err = ri.Namespace(c.namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = ri.List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("crd: failed to list %s: %w", c.store.Resource.Kind, err)
	}

	namePattern := ""
	if ref.Name != nil {
		namePattern = ref.Name.RegExp
	}
	var re *regexp.Regexp
	if namePattern != "" {
		re, err = regexp.Compile(namePattern)
		if err != nil {
			return nil, fmt.Errorf("crd: invalid name pattern %q: %w", namePattern, err)
		}
	}

	result := make(map[string][]byte, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		name := item.GetName()
		if re != nil && !re.MatchString(name) {
			continue
		}
		allowed, err := c.matchesWhitelistRule(name, nil)
		if err != nil {
			return nil, err
		}
		if !allowed {
			continue
		}

		b, err := extractValue(item, "", nil)
		if err != nil {
			return nil, fmt.Errorf("crd: failed to extract value from %s/%s: %w", c.store.Resource.Kind, name, err)
		}
		result[name] = b
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

// getObject fetches a single named CRD object from the cluster.
func (c *Client) getObject(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	gvr := c.buildGVR()
	var (
		obj *unstructured.Unstructured
		err error
	)
	ri := c.dynClient.Resource(gvr)
	if c.namespaced {
		if c.namespace == "" {
			return nil, fmt.Errorf("crd: namespace is required for namespaced resource kind %q", c.store.Resource.Kind)
		}
		obj, err = ri.Namespace(c.namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		obj, err = ri.Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, esv1.NoSecretError{}
		}
		return nil, fmt.Errorf("crd: failed to get %s/%s: %w", c.store.Resource.Kind, name, err)
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

func requestedPropertyKeys(property string) []string {
	if property == "" {
		return nil
	}
	return []string{property}
}

func (c *Client) matchesWhitelistRule(name string, requestedKeys []string) (bool, error) {
	if c.store.Whitelist == nil || len(c.store.Whitelist.Rules) == 0 {
		return true, nil
	}

	for _, rule := range c.store.Whitelist.Rules {
		nameMatches := true
		if rule.Name != "" {
			re, err := regexp.Compile(rule.Name)
			if err != nil {
				return false, fmt.Errorf("crd: invalid whitelist name regex %q: %w", rule.Name, err)
			}
			nameMatches = re.MatchString(name)
		}
		if !nameMatches {
			continue
		}

		if len(rule.Properties) == 0 {
			return true, nil
		}

		if len(requestedKeys) == 0 {
			continue
		}

		allKeysMatch := true
		for _, key := range requestedKeys {
			keyMatched := false
			for _, propPattern := range rule.Properties {
				re, err := regexp.Compile(propPattern)
				if err != nil {
					return false, fmt.Errorf("crd: invalid whitelist property regex %q: %w", propPattern, err)
				}
				if re.MatchString(key) {
					keyMatched = true
					break
				}
			}
			if !keyMatched {
				allKeysMatch = false
				break
			}
		}

		if allKeysMatch {
			return true, nil
		}
	}

	return false, nil
}
