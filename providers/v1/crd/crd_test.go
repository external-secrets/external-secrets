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
)

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

func makeWhitelistRule(name string, properties ...string) esv1.CRDProviderWhitelistRule {
	return esv1.CRDProviderWhitelistRule{Name: name, Properties: properties}
}

var testResource = esv1.CRDProviderResource{
	Group:   "example.io",
	Version: "v1alpha1",
	Kind:    "Widget",
}

func makeCRDTestStore(rules ...esv1.CRDProviderWhitelistRule) *esv1.CRDProvider {
	store := &esv1.CRDProvider{
		ServiceAccountName: "reader",
		Resource:           testResource,
	}
	if len(rules) > 0 {
		store.Whitelist = &esv1.CRDProviderWhitelist{Rules: rules}
	}
	return store
}

func makeWidgetObject(name, namespace string, spec map[string]any) *unstructured.Unstructured {
	meta := map[string]any{"name": name}
	if namespace != "" {
		meta["namespace"] = namespace
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "example.io/v1alpha1",
		"kind":       "Widget",
		"metadata":   meta,
		"spec":       spec,
	}}
}

func makeCRDClient(store *esv1.CRDProvider, namespace string, objs ...runtime.Object) *Client {
	return &Client{
		store:      store,
		namespace:  namespace,
		plural:     "widgets",
		namespaced: true,
		dynClient:  dynfake.NewSimpleDynamicClient(runtime.NewScheme(), objs...),
	}
}

func TestClientBuildGVR(t *testing.T) {
	c := &Client{store: makeCRDTestStore(), plural: "widgets", namespaced: true}
	gvr := c.buildGVR()
	if gvr.Group != testResource.Group || gvr.Version != testResource.Version || gvr.Resource != "widgets" {
		t.Fatalf("unexpected GVR: %+v", gvr)
	}
}

func TestClientGetSecretClusterScoped(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "test.external-secrets.io/v1alpha1",
		"kind":       "ClusterDBSpec",
		"metadata": map[string]any{
			"name": "clusterdbspec-sample",
		},
		"spec": map[string]any{
			"password": "cluster-secret",
		},
	}}
	store := &esv1.CRDProvider{
		ServiceAccountName: "reader",
		Resource: esv1.CRDProviderResource{
			Group:   "test.external-secrets.io",
			Version: "v1alpha1",
			Kind:    "ClusterDBSpec",
		},
	}
	// ExternalSecret lives in default; cluster-scoped Get must not use that namespace.
	c := &Client{
		store:      store,
		namespace:  "default",
		plural:     "clusterdbspecs",
		namespaced: false,
		dynClient:  dynfake.NewSimpleDynamicClient(runtime.NewScheme(), obj),
	}
	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{
		Key:      "clusterdbspec-sample",
		Property: "spec.password",
	})
	if err != nil {
		t.Fatalf("GetSecret() unexpected error: %v", err)
	}
	if string(got) != "cluster-secret" {
		t.Fatalf("GetSecret() = %q, want %q", string(got), "cluster-secret")
	}
}

func TestExtractValue(t *testing.T) {
	obj := makeWidgetObject("sample", "default", map[string]any{
		"password": "s3cr3t",
		"meta": map[string]any{
			"a": "b",
		},
		"targets": []any{
			map[string]any{"name": "app", "value": "v1"},
			map[string]any{"name": "db", "value": "v2"},
		},
	})

	t.Run("extract by property", func(t *testing.T) {
		got, err := extractValue(obj, "spec.password", nil)
		if err != nil {
			t.Fatalf("extractValue() unexpected error: %v", err)
		}
		if string(got) != "s3cr3t" {
			t.Fatalf("extractValue() = %q, want %q", string(got), "s3cr3t")
		}
	})

	t.Run("missing property", func(t *testing.T) {
		_, err := extractValue(obj, "spec.missing", nil)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("extractValue() error = %v, want not found", err)
		}
	})

	t.Run("extract selected fields", func(t *testing.T) {
		got, err := extractValue(obj, "", []string{"spec.password", "spec.meta.a"})
		if err != nil {
			t.Fatalf("extractValue() unexpected error: %v", err)
		}

		var parsed map[string]any
		if err := json.Unmarshal(got, &parsed); err != nil {
			t.Fatalf("extractValue() unmarshal error: %v", err)
		}
		if parsed["spec.password"] != "s3cr3t" {
			t.Fatalf("subset field mismatch, got %#v", parsed["spec.password"])
		}
		if parsed["spec.meta.a"] != "b" {
			t.Fatalf("subset field mismatch, got %#v", parsed["spec.meta.a"])
		}
	})

	t.Run("extract with JMESPath array expression", func(t *testing.T) {
		got, err := extractValue(obj, "spec.targets[?name=='db'].value | [0]", nil)
		if err != nil {
			t.Fatalf("extractValue() unexpected error: %v", err)
		}
		if string(got) != "v2" {
			t.Fatalf("extractValue() = %q, want %q", string(got), "v2")
		}
	})

	t.Run("invalid JMESPath expression", func(t *testing.T) {
		_, err := extractValue(obj, "spec.targets[?name=='db'", nil)
		if err == nil || !strings.Contains(err.Error(), "invalid property expression") {
			t.Fatalf("extractValue() error = %v, want invalid property expression", err)
		}
	})
}

func TestJSONBytesToMap(t *testing.T) {
	t.Run("object with mixed value types", func(t *testing.T) {
		raw := []byte(`{"a":"x","b":1}`)
		got, err := jsonBytesToMap(raw)
		if err != nil {
			t.Fatalf("jsonBytesToMap() unexpected error: %v", err)
		}
		if string(got["a"]) != "x" {
			t.Fatalf("jsonBytesToMap()[a] = %q, want %q", string(got["a"]), "x")
		}
		if string(got["b"]) != "1" {
			t.Fatalf("jsonBytesToMap()[b] = %q, want %q", string(got["b"]), "1")
		}
	})

	t.Run("non-object payload falls back to value key", func(t *testing.T) {
		raw := []byte(`"hello"`)
		got, err := jsonBytesToMap(raw)
		if err != nil {
			t.Fatalf("jsonBytesToMap() unexpected error: %v", err)
		}
		if string(got["value"]) != `"hello"` {
			t.Fatalf("jsonBytesToMap()[value] = %q, want %q", string(got["value"]), `"hello"`)
		}
	})
}

func TestClientGetSecret(t *testing.T) {
	obj := makeWidgetObject("item-a", "ns1", map[string]any{"password": "pw1"})
	c := makeCRDClient(makeCRDTestStore(), "ns1", obj)

	t.Run("empty key", func(t *testing.T) {
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{})
		if err == nil || !strings.Contains(err.Error(), "must not be empty") {
			t.Fatalf("GetSecret() error = %v, want key validation error", err)
		}
	})

	t.Run("returns property value", func(t *testing.T) {
		got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "item-a", Property: "spec.password"})
		if err != nil {
			t.Fatalf("GetSecret() unexpected error: %v", err)
		}
		if string(got) != "pw1" {
			t.Fatalf("GetSecret() = %q, want %q", string(got), "pw1")
		}
	})

	t.Run("missing object maps to NoSecretError", func(t *testing.T) {
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "does-not-exist"})
		if !errors.Is(err, esv1.NoSecretError{}) {
			t.Fatalf("GetSecret() error = %v, want NoSecretError", err)
		}
	})
}

func TestClientGetSecretMap(t *testing.T) {
	obj := makeWidgetObject("item-a", "ns1", map[string]any{
		"map": map[string]any{"a": "x", "b": int64(1)},
	})
	c := makeCRDClient(makeCRDTestStore(), "ns1", obj)

	got, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "item-a", Property: "spec.map"})
	if err != nil {
		t.Fatalf("GetSecretMap() unexpected error: %v", err)
	}
	if string(got["a"]) != "x" {
		t.Fatalf("GetSecretMap()[a] = %q, want %q", string(got["a"]), "x")
	}
	if string(got["b"]) != "1" {
		t.Fatalf("GetSecretMap()[b] = %q, want %q", string(got["b"]), "1")
	}
}

func TestClientGetAllSecrets(t *testing.T) {
	objA := makeWidgetObject("app-a", "ns1", map[string]any{"password": "a"})
	objB := makeWidgetObject("sys-b", "ns1", map[string]any{"password": "b"})

	t.Run("find.name regex filtering", func(t *testing.T) {
		c := makeCRDClient(makeCRDTestStore(), "ns1", objA, objB)
		got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
		if err != nil {
			t.Fatalf("GetAllSecrets() unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("GetAllSecrets() len = %d, want 2", len(got))
		}
		if _, ok := got["app-a"]; !ok {
			t.Fatalf("GetAllSecrets() missing expected key app-a")
		}
		if _, ok := got["sys-b"]; !ok {
			t.Fatalf("GetAllSecrets() missing expected key sys-b")
		}
	})

	t.Run("find.name regexp filters list", func(t *testing.T) {
		c := makeCRDClient(makeCRDTestStore(), "ns1", objA, objB)
		got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
			Name: &esv1.FindName{RegExp: "^sys-.*$"},
		})
		if err != nil {
			t.Fatalf("GetAllSecrets() unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("GetAllSecrets() len = %d, want 1", len(got))
		}
		if _, ok := got["sys-b"]; !ok {
			t.Fatalf("GetAllSecrets() missing expected key sys-b")
		}
	})

	t.Run("invalid regex returns error", func(t *testing.T) {
		c := makeCRDClient(makeCRDTestStore(), "ns1", objA)
		_, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "("}})
		if err == nil || !strings.Contains(err.Error(), "invalid name pattern") {
			t.Fatalf("GetAllSecrets() error = %v, want invalid pattern error", err)
		}
	})

	t.Run("whitelist name rule filters list", func(t *testing.T) {
		c := makeCRDClient(makeCRDTestStore(makeWhitelistRule("^app-.*$")), "ns1", objA, objB)
		got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
		if err != nil {
			t.Fatalf("GetAllSecrets() unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("GetAllSecrets() len = %d, want 1", len(got))
		}
		if _, ok := got["app-a"]; !ok {
			t.Fatalf("GetAllSecrets() missing expected key app-a")
		}
	})
}

func TestClientMiscMethods(t *testing.T) {
	c := makeCRDClient(makeCRDTestStore(), "ns1")

	if err := c.PushSecret(context.Background(), nil, testPushSecretData{}); err == nil {
		t.Fatalf("PushSecret() expected error")
	}
	if err := c.DeleteSecret(context.Background(), testPushSecretRemoteRef{}); err == nil {
		t.Fatalf("DeleteSecret() expected error")
	}

	if got, err := c.Validate(); err != nil || got != esv1.ValidationResultReady {
		t.Fatalf("Validate() = (%v, %v), want (%v, nil)", got, err, esv1.ValidationResultReady)
	}
	if err := c.Close(context.Background()); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}
}

func TestClientSecretExists(t *testing.T) {
	obj := makeWidgetObject("item-a", "ns1", map[string]any{"password": "pw1"})
	c := makeCRDClient(makeCRDTestStore(), "ns1", obj)

	exists, err := c.SecretExists(context.Background(), testPushSecretRemoteRef{remoteKey: "item-a"})
	if err != nil || !exists {
		t.Fatalf("SecretExists(item-a) = (%v, %v), want (true, nil)", exists, err)
	}

	exists, err = c.SecretExists(context.Background(), testPushSecretRemoteRef{remoteKey: "missing"})
	if err != nil || exists {
		t.Fatalf("SecretExists(missing) = (%v, %v), want (false, nil)", exists, err)
	}
}

func TestWhitelistMatching(t *testing.T) {
	obj := makeWidgetObject("item-a", "ns1", map[string]any{"password": "pw1"})

	tests := []struct {
		name       string
		rules      []esv1.CRDProviderWhitelistRule
		ref        esv1.ExternalSecretDataRemoteRef
		wantVal    string
		wantErrMsg string
	}{
		{
			name:       "denied when no rule matches",
			rules:      []esv1.CRDProviderWhitelistRule{makeWhitelistRule("^allowed-.*$")},
			ref:        esv1.ExternalSecretDataRemoteRef{Key: "item-a", Property: "spec.password"},
			wantErrMsg: "denied by whitelist",
		},
		{
			name:    "allowed by name rule only",
			rules:   []esv1.CRDProviderWhitelistRule{makeWhitelistRule("^item-.*$")},
			ref:     esv1.ExternalSecretDataRemoteRef{Key: "item-a", Property: "spec.password"},
			wantVal: "pw1",
		},
		{
			name:       "denied when property does not match rule",
			rules:      []esv1.CRDProviderWhitelistRule{makeWhitelistRule("^item-.*$", "^spec\\.allowed$")},
			ref:        esv1.ExternalSecretDataRemoteRef{Key: "item-a", Property: "spec.password"},
			wantErrMsg: "denied by whitelist",
		},
		{
			name:    "allowed when both name and property match",
			rules:   []esv1.CRDProviderWhitelistRule{makeWhitelistRule("^item-.*$", "^spec\\.password$")},
			ref:     esv1.ExternalSecretDataRemoteRef{Key: "item-a", Property: "spec.password"},
			wantVal: "pw1",
		},
		{
			name: "allowed when name is empty and one of two properties matches",
			rules: []esv1.CRDProviderWhitelistRule{{
				Properties: []string{"^spec\\.username$", "^spec\\.password$"},
			}},
			ref:     esv1.ExternalSecretDataRemoteRef{Key: "item-a", Property: "spec.password"},
			wantVal: "pw1",
		},
		{
			name: "denied when name is empty and none of two properties match",
			rules: []esv1.CRDProviderWhitelistRule{{
				Properties: []string{"^spec\\.username$", "^spec\\.token$"},
			}},
			ref:        esv1.ExternalSecretDataRemoteRef{Key: "item-a", Property: "spec.password"},
			wantErrMsg: "denied by whitelist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := makeCRDClient(makeCRDTestStore(tt.rules...), "ns1", obj)
			got, err := c.GetSecret(context.Background(), tt.ref)
			if tt.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Fatalf("GetSecret() error = %v, want %q", err, tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetSecret() unexpected error: %v", err)
			}
			if string(got) != tt.wantVal {
				t.Fatalf("GetSecret() = %q, want %q", string(got), tt.wantVal)
			}
		})
	}
}
