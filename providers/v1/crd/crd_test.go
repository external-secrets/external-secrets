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

package crd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// ── test doubles ─────────────────────────────────────────────────────────────

type testPushSecretData struct{}

func (testPushSecretData) GetMetadata() *apiextensionsv1.JSON { return nil }
func (testPushSecretData) GetSecretKey() string               { return "" }
func (testPushSecretData) GetRemoteKey() string               { return "" }
func (testPushSecretData) GetProperty() string                { return "" }

type testPushSecretRemoteRef struct {
	remoteKey string
	property  string
}

func (r testPushSecretRemoteRef) GetRemoteKey() string { return r.remoteKey }
func (r testPushSecretRemoteRef) GetProperty() string  { return r.property }

// ── helpers ──────────────────────────────────────────────────────────────────

const testStringHello = "hello"

var testResource = esv1.CRDProviderResource{
	Group: "example.io", Version: "v1alpha1", Kind: "Widget",
}

func makeStore(rules ...esv1.CRDProviderWhitelistRule) *esv1.CRDProvider {
	s := &esv1.CRDProvider{
		ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
		Resource:          testResource,
	}
	if len(rules) > 0 {
		s.Whitelist = &esv1.CRDProviderWhitelist{Rules: rules}
	}
	return s
}

func wlRule(name string, props ...string) esv1.CRDProviderWhitelistRule {
	return esv1.CRDProviderWhitelistRule{Name: name, Properties: props}
}

func wlRuleNS(ns, name string, props ...string) esv1.CRDProviderWhitelistRule {
	return esv1.CRDProviderWhitelistRule{Namespace: ns, Name: name, Properties: props}
}

func widget(name, namespace string, spec map[string]any) *unstructured.Unstructured {
	meta := map[string]any{"name": name}
	if namespace != "" {
		meta["namespace"] = namespace
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "example.io/v1alpha1", "kind": "Widget",
		"metadata": meta, "spec": spec,
	}}
}

// newTestClient builds a Client for use in unit tests.
// storeKind must be esv1.SecretStoreKind or esv1.ClusterSecretStoreKind.
func newTestClient(store *esv1.CRDProvider, storeKind, namespace string, namespaced bool, objs ...runtime.Object) *Client {
	return &Client{
		store:      store,
		namespace:  namespace,
		plural:     "widgets",
		namespaced: namespaced,
		storeKind:  storeKind,
		dynClient:  dynfake.NewSimpleDynamicClient(runtime.NewScheme(), objs...),
	}
}

// Shorthands for the two most common configurations.
func ssClient(store *esv1.CRDProvider, ns string, objs ...runtime.Object) *Client {
	return newTestClient(store, esv1.SecretStoreKind, ns, true, objs...)
}

func cssClient(store *esv1.CRDProvider, objs ...runtime.Object) *Client {
	return newTestClient(store, esv1.ClusterSecretStoreKind, "", true, objs...)
}

func ref(key, prop string) esv1.ExternalSecretDataRemoteRef {
	return esv1.ExternalSecretDataRemoteRef{Key: key, Property: prop}
}

// assertJSON unmarshals b and calls check on the result.
func assertJSON[T any](t *testing.T, b []byte, check func(*testing.T, T)) {
	t.Helper()
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, b)
	}
	check(t, v)
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestClientBuildGVR(t *testing.T) {
	c := newTestClient(makeStore(), esv1.SecretStoreKind, "", true)
	gvr := c.buildGVR()
	if gvr.Group != "example.io" || gvr.Version != "v1alpha1" || gvr.Resource != "widgets" {
		t.Fatalf("unexpected GVR: %+v", gvr)
	}
}

func TestClientGetSecret(t *testing.T) {
	richSpec := map[string]any{
		"password": "pw1",
		"foo":      map[string]any{"bar": int64(42), "baz": testStringHello},
		"nested":   []any{map[string]any{"key": "ep", "val": "db:5432"}, map[string]any{"key": "fqdn", "val": "u:p@db"}},
	}

	tests := []struct {
		name       string
		client     func() *Client
		ref        esv1.ExternalSecretDataRemoteRef
		wantStr    string
		wantErrIs  error
		wantErrMsg string
		checkFn    func(*testing.T, []byte)
	}{
		// ── SecretStore ──
		{
			name: "empty key", client: func() *Client { return ssClient(makeStore(), "ns1", widget("x", "ns1", richSpec)) },
			ref: ref("", ""), wantErrMsg: "must not be empty",
		},
		{
			name: "slash rejected", client: func() *Client { return ssClient(makeStore(), "ns1", widget("x", "ns1", richSpec)) },
			ref: ref("a/b", ""), wantErrMsg: "must not contain '/'",
		},
		{
			name: "missing object", client: func() *Client { return ssClient(makeStore(), "ns1", widget("x", "ns1", richSpec)) },
			ref: ref("does-not-exist", ""), wantErrIs: esv1.NoSecretError{},
		},
		{
			name: "scalar property", client: func() *Client { return ssClient(makeStore(), "ns1", widget("item-a", "ns1", richSpec)) },
			ref: ref("item-a", "spec.password"), wantStr: "pw1",
		},
		{
			name: "nested scalar via dot path", client: func() *Client { return ssClient(makeStore(), "ns1", widget("item-a", "ns1", richSpec)) },
			ref: ref("item-a", "spec.foo.bar"), wantStr: "42",
		},
		{
			name: "JMESPath on array", client: func() *Client { return ssClient(makeStore(), "ns1", widget("item-a", "ns1", richSpec)) },
			ref: ref("item-a", "spec.nested[?key=='fqdn'].val | [0]"), wantStr: "u:p@db",
		},
		{
			name:   "nested object returns JSON",
			client: func() *Client { return ssClient(makeStore(), "ns1", widget("item-a", "ns1", richSpec)) },
			ref:    ref("item-a", "spec.foo"),
			checkFn: func(t *testing.T, b []byte) {
				assertJSON(t, b, func(t *testing.T, m map[string]any) {
					if m["bar"] != float64(42) || m["baz"] != testStringHello {
						t.Fatalf("spec.foo = %v", m)
					}
				})
			},
		},
		{
			name:   "array property returns JSON array",
			client: func() *Client { return ssClient(makeStore(), "ns1", widget("item-a", "ns1", richSpec)) },
			ref:    ref("item-a", "spec.nested"),
			checkFn: func(t *testing.T, b []byte) {
				assertJSON(t, b, func(t *testing.T, arr []map[string]any) {
					if len(arr) != 2 || arr[0]["key"] != "ep" {
						t.Fatalf("spec.nested = %v", arr)
					}
				})
			},
		},

		// ── ClusterSecretStore: namespaced kind ──
		{
			name: "CSS: namespace/name resolves", client: func() *Client { return cssClient(makeStore(), widget("item-a", "ns1", richSpec)) },
			ref: ref("ns1/item-a", "spec.password"), wantStr: "pw1",
		},
		{
			name: "CSS: bare name rejected for namespaced kind", client: func() *Client { return cssClient(makeStore(), widget("item-a", "ns1", richSpec)) },
			ref: ref("item-a", ""), wantErrMsg: "namespace/objectName",
		},

		// ── ClusterSecretStore: cluster-scoped kind ──
		{
			name: "CSS cluster-scoped: bare name resolves",
			client: func() *Client {
				return newTestClient(makeStore(), esv1.ClusterSecretStoreKind, "default", false,
					widget("global", "", map[string]any{"password": "x"}))
			},
			ref: ref("global", "spec.password"), wantStr: "x",
		},
		{
			name: "CSS cluster-scoped: slash rejected",
			client: func() *Client {
				return newTestClient(makeStore(), esv1.ClusterSecretStoreKind, "default", false,
					widget("global", "", map[string]any{"password": "x"}))
			},
			ref: ref("ns/global", "spec.password"), wantErrMsg: "does not allow '/'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.client().GetSecret(context.Background(), tt.ref)
			switch {
			case tt.wantErrMsg != "":
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Fatalf("error = %v, want %q", err, tt.wantErrMsg)
				}
			case tt.wantErrIs != nil:
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("error = %v, want %T", err, tt.wantErrIs)
				}
			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.wantStr != "" && string(got) != tt.wantStr {
					t.Fatalf("= %q, want %q", string(got), tt.wantStr)
				}
				if tt.checkFn != nil {
					tt.checkFn(t, got)
				}
			}
		})
	}
}

func TestExtractValue(t *testing.T) {
	obj := widget("sample", "default", map[string]any{
		"password": "s3cr3t",
		"meta":     map[string]any{"a": "b"},
		"targets":  []any{map[string]any{"name": "app", "value": "v1"}, map[string]any{"name": "db", "value": "v2"}},
	})

	tests := []struct {
		name       string
		property   string
		fields     []string
		wantStr    string
		wantErrMsg string
		checkFn    func(*testing.T, []byte)
	}{
		{name: "by property", property: "spec.password", wantStr: "s3cr3t"},
		{name: "missing property", property: "spec.missing", wantErrMsg: "not found"},
		{name: "invalid JMESPath", property: "spec.targets[?name=='db'", wantErrMsg: "invalid property expression"},
		{name: "JMESPath array", property: "spec.targets[?name=='db'].value | [0]", wantStr: "v2"},
		{
			name: "selected fields", fields: []string{"spec.password", "spec.meta.a"},
			checkFn: func(t *testing.T, b []byte) {
				assertJSON(t, b, func(t *testing.T, m map[string]any) {
					if m["spec.password"] != "s3cr3t" || m["spec.meta.a"] != "b" {
						t.Fatalf("subset = %v", m)
					}
				})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractValue(obj, tt.property, tt.fields)
			if tt.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Fatalf("error = %v, want %q", err, tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantStr != "" && string(got) != tt.wantStr {
				t.Fatalf("= %q, want %q", string(got), tt.wantStr)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, got)
			}
		})
	}
}

func TestJSONBytesToMap(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		checkFn func(*testing.T, map[string][]byte)
	}{
		{
			name: "mixed value types",
			raw:  `{"a":"x","b":1}`,
			checkFn: func(t *testing.T, got map[string][]byte) {
				if string(got["a"]) != "x" || string(got["b"]) != "1" {
					t.Fatalf("got %v", got)
				}
			},
		},
		{
			name: "non-object falls back to value key",
			raw:  `"hello"`,
			checkFn: func(t *testing.T, got map[string][]byte) {
				if string(got["value"]) != `"hello"` {
					t.Fatalf(`["value"] = %q`, string(got["value"]))
				}
			},
		},
		{
			name: "nested object preserved as JSON",
			raw:  `{"user":"admin","foo":{"bar":42,"baz":"hello"}}`,
			checkFn: func(t *testing.T, got map[string][]byte) {
				if string(got["user"]) != "admin" {
					t.Fatalf(`["user"] = %q`, string(got["user"]))
				}
				assertJSON(t, got["foo"], func(t *testing.T, m map[string]any) {
					if m["bar"] != float64(42) || m["baz"] != testStringHello {
						t.Fatalf("foo = %v", m)
					}
				})
			},
		},
		{
			name: "array preserved as JSON",
			raw:  `{"items":[{"key":"a","val":"1"},{"key":"b","val":"2"}]}`,
			checkFn: func(t *testing.T, got map[string][]byte) {
				assertJSON(t, got["items"], func(t *testing.T, items []map[string]any) {
					if len(items) != 2 || items[0]["key"] != "a" {
						t.Fatalf("items = %v", items)
					}
				})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonBytesToMap([]byte(tt.raw))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.checkFn(t, got)
		})
	}
}

func TestClientGetSecretMap(t *testing.T) {
	obj := widget("item-a", "ns1", map[string]any{
		"map":    map[string]any{"a": "x", "b": int64(1)},
		"foo":    map[string]any{"bar": int64(42), "baz": testStringHello},
		"nested": []any{map[string]any{"key": "ep", "val": "db:5432"}, map[string]any{"key": "fqdn", "val": "u:p@db"}},
	})
	c := ssClient(makeStore(), "ns1", obj)

	tests := []struct {
		name    string
		ref     esv1.ExternalSecretDataRemoteRef
		checkFn func(*testing.T, map[string][]byte)
	}{
		{
			name: "flat sub-object",
			ref:  ref("item-a", "spec.map"),
			checkFn: func(t *testing.T, got map[string][]byte) {
				if string(got["a"]) != "x" || string(got["b"]) != "1" {
					t.Fatalf("got %v", got)
				}
			},
		},
		{
			name: "spec returns nested objects as JSON",
			ref:  ref("item-a", "spec"),
			checkFn: func(t *testing.T, got map[string][]byte) {
				assertJSON(t, got["foo"], func(t *testing.T, m map[string]any) {
					if m["bar"] != float64(42) || m["baz"] != testStringHello {
						t.Fatalf("foo = %v", m)
					}
				})
				assertJSON(t, got["nested"], func(t *testing.T, arr []map[string]any) {
					if len(arr) != 2 || arr[0]["key"] != "ep" {
						t.Fatalf("nested = %v", arr)
					}
				})
			},
		},
		{
			name: "spec.foo returns flat map",
			ref:  ref("item-a", "spec.foo"),
			checkFn: func(t *testing.T, got map[string][]byte) {
				if string(got["bar"]) != "42" || string(got["baz"]) != testStringHello {
					t.Fatalf("got %v", got)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.GetSecretMap(context.Background(), tt.ref)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.checkFn(t, got)
		})
	}
}

func TestClientGetAllSecrets(t *testing.T) {
	objA := widget("app-a", "ns1", map[string]any{"password": "a"})
	objB := widget("sys-b", "ns1", map[string]any{"password": "b"})

	tests := []struct {
		name       string
		client     func() *Client
		find       esv1.ExternalSecretFind
		wantKeys   []string
		wantErrMsg string
	}{
		{
			name: "no filter returns all", wantKeys: []string{"app-a", "sys-b"},
			client: func() *Client { return ssClient(makeStore(), "ns1", objA, objB) },
		},
		{
			name: "regexp filters list", wantKeys: []string{"sys-b"},
			client: func() *Client { return ssClient(makeStore(), "ns1", objA, objB) },
			find:   esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "^sys-.*$"}},
		},
		{
			name: "invalid regex", wantErrMsg: "invalid name pattern",
			client: func() *Client { return ssClient(makeStore(), "ns1", objA) },
			find:   esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "("}},
		},
		{
			name: "whitelist name rule", wantKeys: []string{"app-a"},
			client: func() *Client { return ssClient(makeStore(wlRule("^app-.*$")), "ns1", objA, objB) },
		},
		{
			name: "CSS namespaced kind uses namespace/name keys", wantKeys: []string{"ns1/app-a", "ns2/sys-b"},
			client: func() *Client {
				return cssClient(makeStore(),
					widget("app-a", "ns1", map[string]any{"password": "a"}),
					widget("sys-b", "ns2", map[string]any{"password": "b"}))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.client().GetAllSecrets(context.Background(), tt.find)
			if tt.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Fatalf("error = %v, want %q", err, tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.wantKeys) {
				t.Fatalf("len = %d, want %d; keys: %v", len(got), len(tt.wantKeys), got)
			}
			for _, k := range tt.wantKeys {
				if _, ok := got[k]; !ok {
					t.Fatalf("missing key %q", k)
				}
			}
		})
	}
}

func TestClientMiscMethods(t *testing.T) {
	c := ssClient(makeStore(), "ns1")
	if err := c.PushSecret(context.Background(), nil, testPushSecretData{}); err == nil {
		t.Fatal("PushSecret() expected error")
	}
	if err := c.DeleteSecret(context.Background(), testPushSecretRemoteRef{}); err == nil {
		t.Fatal("DeleteSecret() expected error")
	}
	if got, err := c.Validate(); err != nil || got != esv1.ValidationResultReady {
		t.Fatalf("Validate() = (%v, %v), want (%v, nil)", got, err, esv1.ValidationResultReady)
	}
	if err := c.Close(context.Background()); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}
}

func TestClientSecretExists(t *testing.T) {
	obj := widget("item-a", "ns1", map[string]any{"password": "pw1"})
	c := ssClient(makeStore(), "ns1", obj)

	if exists, err := c.SecretExists(context.Background(), testPushSecretRemoteRef{remoteKey: "item-a"}); err != nil || !exists {
		t.Fatalf("SecretExists(item-a) = (%v, %v), want (true, nil)", exists, err)
	}
	if exists, err := c.SecretExists(context.Background(), testPushSecretRemoteRef{remoteKey: "missing"}); err != nil || exists {
		t.Fatalf("SecretExists(missing) = (%v, %v), want (false, nil)", exists, err)
	}
}

// TestWhitelistMatching covers all whitelist filter dimensions: name, namespace,
// properties, and combinations — as a single table-driven test.
func TestWhitelistMatching(t *testing.T) {
	obj := widget("item-a", "ns1", map[string]any{"password": "pw1"})

	tests := []struct {
		name       string
		client     func() *Client
		ref        esv1.ExternalSecretDataRemoteRef
		wantVal    string
		wantErrMsg string
	}{
		// ── name-only rules (SecretStore) ──
		{
			name: "denied when no rule matches", wantErrMsg: "denied by whitelist",
			client: func() *Client { return ssClient(makeStore(wlRule("^allowed-.*$")), "ns1", obj) },
			ref:    ref("item-a", "spec.password"),
		},
		{
			name: "allowed by name rule", wantVal: "pw1",
			client: func() *Client { return ssClient(makeStore(wlRule("^item-.*$")), "ns1", obj) },
			ref:    ref("item-a", "spec.password"),
		},

		// ── name + properties ──
		{
			name: "denied when property does not match", wantErrMsg: "denied by whitelist",
			client: func() *Client { return ssClient(makeStore(wlRule("^item-.*$", `^spec\.allowed$`)), "ns1", obj) },
			ref:    ref("item-a", "spec.password"),
		},
		{
			name: "allowed when both name and property match", wantVal: "pw1",
			client: func() *Client { return ssClient(makeStore(wlRule("^item-.*$", `^spec\.password$`)), "ns1", obj) },
			ref:    ref("item-a", "spec.password"),
		},

		// ── properties-only ──
		{
			name: "allowed when one of two properties matches", wantVal: "pw1",
			client: func() *Client {
				return ssClient(makeStore(esv1.CRDProviderWhitelistRule{Properties: []string{`^spec\.username$`, `^spec\.password$`}}), "ns1", obj)
			},
			ref: ref("item-a", "spec.password"),
		},
		{
			name: "denied when no property matches", wantErrMsg: "denied by whitelist",
			client: func() *Client {
				return ssClient(makeStore(esv1.CRDProviderWhitelistRule{Properties: []string{`^spec\.username$`, `^spec\.token$`}}), "ns1", obj)
			},
			ref: ref("item-a", "spec.password"),
		},

		// ── namespace rules (ClusterSecretStore) ──
		{
			name: "CSS: namespace allows matching NS", wantVal: "pw1",
			client: func() *Client { return cssClient(makeStore(wlRuleNS("^ns1$", "")), obj) },
			ref:    ref("ns1/item-a", "spec.password"),
		},
		{
			name: "CSS: namespace denies non-matching NS", wantErrMsg: "denied by whitelist",
			client: func() *Client { return cssClient(makeStore(wlRuleNS("^prod$", "")), obj) },
			ref:    ref("ns1/item-a", "spec.password"),
		},
		{
			name: "CSS: namespace regex pattern", wantVal: "pw1",
			client: func() *Client { return cssClient(makeStore(wlRuleNS("^ns.*$", "")), obj) },
			ref:    ref("ns1/item-a", "spec.password"),
		},
		{
			name: "CSS: namespace + name both must match", wantErrMsg: "denied by whitelist",
			client: func() *Client { return cssClient(makeStore(wlRuleNS("^ns1$", "^other-.*$")), obj) },
			ref:    ref("ns1/item-a", "spec.password"),
		},
		{
			name: "SecretStore ignores namespace rule", wantVal: "pw1",
			client: func() *Client { return ssClient(makeStore(wlRuleNS("^prod$", "")), "ns1", obj) },
			ref:    ref("item-a", "spec.password"),
		},
		{
			name: "invalid namespace regex", wantErrMsg: "invalid whitelist namespace regex",
			client: func() *Client { return cssClient(makeStore(wlRuleNS("(invalid", "")), obj) },
			ref:    ref("ns1/item-a", "spec.password"),
		},
		{
			// Regression: namespace rule must not match cluster-scoped objects.
			name: "CSS: namespace rule does not match cluster-scoped object", wantErrMsg: "denied by whitelist",
			client: func() *Client {
				return newTestClient(makeStore(wlRuleNS("^prod$", "")), esv1.ClusterSecretStoreKind, "", false,
					widget("item-a", "", map[string]any{"password": "pw1"}))
			},
			ref: ref("item-a", "spec.password"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.client().GetSecret(context.Background(), tt.ref)
			if tt.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Fatalf("error = %v, want %q", err, tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.wantVal {
				t.Fatalf("= %q, want %q", string(got), tt.wantVal)
			}
		})
	}
}

// TestWhitelistGetAllSecrets verifies namespace whitelist filtering in GetAllSecrets.
func TestWhitelistGetAllSecrets(t *testing.T) {
	o1 := widget("app-a", "ns1", map[string]any{"password": "a"})
	o2 := widget("app-b", "ns2", map[string]any{"password": "b"})

	tests := []struct {
		name     string
		rules    []esv1.CRDProviderWhitelistRule
		wantKeys []string
	}{
		{name: "allow only ns1", rules: []esv1.CRDProviderWhitelistRule{wlRuleNS("^ns1$", "")}, wantKeys: []string{"ns1/app-a"}},
		{name: "ns1 + name rule", rules: []esv1.CRDProviderWhitelistRule{wlRuleNS("^ns1$", ""), wlRuleNS("", "^app-b$")}, wantKeys: []string{"ns1/app-a", "ns2/app-b"}},
		{name: "filter to ns2", rules: []esv1.CRDProviderWhitelistRule{wlRuleNS("^ns2$", "")}, wantKeys: []string{"ns2/app-b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cssClient(makeStore(tt.rules...), o1, o2)
			got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.wantKeys) {
				t.Fatalf("len = %d, want %d; keys: %v", len(got), len(tt.wantKeys), got)
			}
			for _, k := range tt.wantKeys {
				if _, ok := got[k]; !ok {
					t.Fatalf("missing key %q; got %v", k, got)
				}
			}
		})
	}
}
