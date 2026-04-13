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

package npws

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/npws/fake"
	"github.com/external-secrets/external-secrets/providers/v1/npws/npwssdk"
)

// --- helpers ----------------------------------------------------------------

func ptrStr(s string) *string { return &s }

func newValidStore() *esv1.SecretStore {
	return &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				NPWS: &esv1.NPWSProvider{
					Host: "https://npws.example.com",
					Auth: esv1.NPWSAuth{
						SecretRef: &esv1.NPWSAuthSecretRef{
							APIKey: esmeta.SecretKeySelector{
								Name: "npws-credentials",
								Key:  "api-key",
							},
						},
					},
				},
			},
		},
	}
}

// --- Capabilities -----------------------------------------------------------

func TestCapabilities(t *testing.T) {
	p := &Provider{}
	if got := p.Capabilities(); got != esv1.SecretStoreReadWrite {
		t.Errorf("expected ReadWrite, got %v", got)
	}
}

// --- Validate ---------------------------------------------------------------

func TestValidate_Ready(t *testing.T) {
	p := &Provider{api: &psrAPIAdapter{inner: npwssdk.NewPsrAPI("https://example.com")}}
	result, err := p.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != esv1.ValidationResultReady {
		t.Errorf("expected ValidationResultReady, got %v", result)
	}
}

func TestValidate_NotConnected(t *testing.T) {
	p := &Provider{api: nil}
	result, err := p.Validate()
	if err == nil {
		t.Error("expected error for nil api, got nil")
	}
	if result != esv1.ValidationResultError {
		t.Errorf("expected ValidationResultError, got %v", result)
	}
}

// --- Close ------------------------------------------------------------------

func TestClose_NilApi(t *testing.T) {
	p := &Provider{api: nil}
	if err := p.Close(context.Background()); err != nil {
		t.Errorf("Close() with nil api returned unexpected error: %v", err)
	}
}

// --- ValidateStore ----------------------------------------------------------

func TestValidateStore_Valid(t *testing.T) {
	p := &Provider{}
	warns, err := p.ValidateStore(newValidStore())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if warns != nil {
		t.Errorf("unexpected warnings: %v", warns)
	}
}

func TestValidateStore_NilProvider(t *testing.T) {
	p := &Provider{}
	store := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: nil,
		},
	}
	_, err := p.ValidateStore(store)
	if err == nil {
		t.Error("expected error for nil provider, got nil")
	}
}

func TestValidateStore_NilNPWS(t *testing.T) {
	p := &Provider{}
	store := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{NPWS: nil},
		},
	}
	_, err := p.ValidateStore(store)
	if err == nil {
		t.Error("expected error for nil NPWS config, got nil")
	}
}

func TestValidateStore_MissingHost(t *testing.T) {
	p := &Provider{}
	store := newValidStore()
	store.Spec.Provider.NPWS.Host = ""
	_, err := p.ValidateStore(store)
	if err == nil {
		t.Error("expected error for missing host, got nil")
	}
}

func TestValidateStore_NilSecretRef(t *testing.T) {
	p := &Provider{}
	store := newValidStore()
	store.Spec.Provider.NPWS.Auth.SecretRef = nil
	_, err := p.ValidateStore(store)
	if err == nil {
		t.Error("expected error for nil secretRef, got nil")
	}
}

func TestValidateStore_MissingAPIKeyName(t *testing.T) {
	p := &Provider{}
	store := newValidStore()
	store.Spec.Provider.NPWS.Auth.SecretRef.APIKey.Name = ""
	_, err := p.ValidateStore(store)
	if err == nil {
		t.Error("expected error for missing apiKey.name, got nil")
	}
}

// --- findItem / findItemByName / findFirstPasswordItem ----------------------

func TestFindItem_ByProperty(t *testing.T) {
	container := &npwssdk.PsrContainer{
		Items: []npwssdk.PsrContainerItem{
			{Name: "username", ContainerItemType: npwssdk.ContainerItemText, Value: "admin"},
			{Name: "password", ContainerItemType: npwssdk.ContainerItemPassword},
			{Name: "url", ContainerItemType: npwssdk.ContainerItemURL, Value: "https://example.com"},
		},
	}

	item := findItem(container, "url")
	if item == nil || item.Name != "url" {
		t.Error("expected to find 'url' item")
	}
}

func TestFindItem_DefaultToPassword(t *testing.T) {
	container := &npwssdk.PsrContainer{
		Items: []npwssdk.PsrContainerItem{
			{Name: "username", ContainerItemType: npwssdk.ContainerItemText, Value: "admin"},
			{Name: "password", ContainerItemType: npwssdk.ContainerItemPassword},
		},
	}

	item := findItem(container, "")
	if item == nil || item.Name != "password" {
		t.Error("expected to find first password item when property is empty")
	}
}

func TestFindItem_NotFound(t *testing.T) {
	container := &npwssdk.PsrContainer{
		Items: []npwssdk.PsrContainerItem{
			{Name: "username", ContainerItemType: npwssdk.ContainerItemText},
		},
	}

	if item := findItem(container, "nonexistent"); item != nil {
		t.Error("expected nil for nonexistent item")
	}
}

// --- GetValue / SetValue on PsrContainerItem --------------------------------

func TestGetValue_TextTypes(t *testing.T) {
	textTypes := []npwssdk.ContainerItemType{
		npwssdk.ContainerItemText,
		npwssdk.ContainerItemURL,
		npwssdk.ContainerItemEmail,
		npwssdk.ContainerItemPhone,
		npwssdk.ContainerItemUserName,
		npwssdk.ContainerItemIP,
		npwssdk.ContainerItemHostName,
	}
	for _, ct := range textTypes {
		item := &npwssdk.PsrContainerItem{ContainerItemType: ct, Value: "test-value"}
		if got := item.GetValue(); got != "test-value" {
			t.Errorf("type %d: GetValue() = %q, want %q", ct, got, "test-value")
		}
	}
}

func TestGetValue_EncryptedTypes(t *testing.T) {
	for _, ct := range []npwssdk.ContainerItemType{
		npwssdk.ContainerItemPassword,
		npwssdk.ContainerItemPasswordMemo,
		npwssdk.ContainerItemOtp,
	} {
		item := &npwssdk.PsrContainerItem{
			ContainerItemType: ct,
			Value:             "encrypted-base64",    // encrypted value (not readable)
			PlainTextValue:    "decrypted-plaintext", // set after decryption
		}
		if got := item.GetValue(); got != "decrypted-plaintext" {
			t.Errorf("type %d: GetValue() = %q, want %q", ct, got, "decrypted-plaintext")
		}
	}
}

func TestGetValue_Memo(t *testing.T) {
	item := &npwssdk.PsrContainerItem{ContainerItemType: npwssdk.ContainerItemMemo, ValueMemo: ptrStr("long text")}
	if got := item.GetValue(); got != "long text" {
		t.Errorf("Memo: got %q", got)
	}
}

func TestGetSetValue_Bool(t *testing.T) {
	item := &npwssdk.PsrContainerItem{ContainerItemType: npwssdk.ContainerItemCheck}
	item.SetValue("true")
	if got := item.GetValue(); got != "true" {
		t.Errorf("Check: got %q", got)
	}
}

func TestGetSetValue_Int(t *testing.T) {
	item := &npwssdk.PsrContainerItem{ContainerItemType: npwssdk.ContainerItemInt}
	item.SetValue("42")
	if got := item.GetValue(); got != "42" {
		t.Errorf("Int: got %q", got)
	}
}

func TestGetSetValue_Decimal(t *testing.T) {
	item := &npwssdk.PsrContainerItem{ContainerItemType: npwssdk.ContainerItemDecimal}
	item.SetValue("3.14")
	if got := item.GetValue(); got != "3.14" {
		t.Errorf("Decimal: got %q", got)
	}
}

func TestGetSetValue_Date(t *testing.T) {
	item := &npwssdk.PsrContainerItem{ContainerItemType: npwssdk.ContainerItemDate}
	item.SetValue("2024-01-15T10:30:00Z")
	if got := item.GetValue(); got != "2024-01-15T10:30:00Z" {
		t.Errorf("Date: got %q", got)
	}
}

func TestGetSetValue_List(t *testing.T) {
	item := &npwssdk.PsrContainerItem{ContainerItemType: npwssdk.ContainerItemList}
	item.SetValue(`["a","b","c"]`)
	if got := item.GetValue(); got != `["a","b","c"]` {
		t.Errorf("List: got %q", got)
	}
}

func TestGetValue_Header(t *testing.T) {
	item := &npwssdk.PsrContainerItem{ContainerItemType: npwssdk.ContainerItemHeader}
	if got := item.GetValue(); got != "" {
		t.Errorf("Header: expected empty, got %q", got)
	}
}

func TestSetValue_InvalidInt(t *testing.T) {
	item := &npwssdk.PsrContainerItem{ContainerItemType: npwssdk.ContainerItemInt}
	if err := item.SetValue("not-a-number"); err == nil {
		t.Error("expected error for invalid int")
	}
}

func TestSetValue_InvalidDate(t *testing.T) {
	item := &npwssdk.PsrContainerItem{ContainerItemType: npwssdk.ContainerItemDate}
	if err := item.SetValue("not-a-date"); err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestSetValue_EncryptedSetsPlainTextValue(t *testing.T) {
	item := &npwssdk.PsrContainerItem{ContainerItemType: npwssdk.ContainerItemPassword}
	item.SetValue("new-secret")
	if item.PlainTextValue != "new-secret" {
		t.Errorf("expected PlainTextValue = %q, got %q", "new-secret", item.PlainTextValue)
	}
}

// --- test helper types for PushSecretData / PushSecretRemoteRef -------------

type testPushSecretData struct {
	remoteKey string
	property  string
	secretKey string
}

func (d *testPushSecretData) GetRemoteKey() string { return d.remoteKey }
func (d *testPushSecretData) GetProperty() string  { return d.property }
func (d *testPushSecretData) GetSecretKey() string { return d.secretKey }
func (d *testPushSecretData) GetMetadata() *apiextensionsv1.JSON {
	return nil
}

type testPushSecretRemoteRef struct {
	remoteKey string
	property  string
}

func (r *testPushSecretRemoteRef) GetRemoteKey() string { return r.remoteKey }
func (r *testPushSecretRemoteRef) GetProperty() string  { return r.property }

// --- container fixture helpers ----------------------------------------------

const testGUID = "11111111-1111-1111-1111-111111111111"

func newPasswordContainer() *npwssdk.PsrContainer {
	return &npwssdk.PsrContainer{
		ID:            testGUID,
		ContainerType: npwssdk.ContainerTypePassword,
		Info:          &npwssdk.PsrContainerInfo{ContainerName: "MyEntry"},
		Items: []npwssdk.PsrContainerItem{
			{Name: "Name", ContainerItemType: npwssdk.ContainerItemText, Value: "MyEntry"},
			{Name: "Password", ContainerItemType: npwssdk.ContainerItemPassword, Value: "ZW5jcnlwdGVk"},
			{Name: "UserName", ContainerItemType: npwssdk.ContainerItemText, Value: "admin"},
		},
	}
}

// noopUpdate is used when UpdateContainer is expected or needs to be stubbed.
func noopUpdate(_ context.Context, c *npwssdk.PsrContainer) (*npwssdk.PsrContainer, error) {
	return c, nil
}

// --- TestGetSecret ----------------------------------------------------------

func TestGetSecret(t *testing.T) {
	tests := []struct {
		name      string
		fc        *fake.Client
		ref       esv1.ExternalSecretDataRemoteRef
		want      string
		wantErr   string
		wantNoSec bool
	}{
		{
			name: "by_guid_and_property",
			fc: &fake.Client{
				GetContainerFn: func(_ context.Context, id string) (*npwssdk.PsrContainer, error) {
					if id == testGUID {
						return newPasswordContainer(), nil
					}
					return nil, nil
				},
			},
			ref:  esv1.ExternalSecretDataRemoteRef{Key: testGUID, Property: "UserName"},
			want: "admin",
		},
		{
			name: "by_name",
			fc: &fake.Client{
				GetContainerByNameFn: func(_ context.Context, name string) (*npwssdk.PsrContainer, error) {
					if name == "MyEntry" {
						return newPasswordContainer(), nil
					}
					return nil, nil
				},
			},
			ref:  esv1.ExternalSecretDataRemoteRef{Key: "MyEntry", Property: "UserName"},
			want: "admin",
		},
		{
			name: "no_property_defaults_to_password",
			fc: &fake.Client{
				GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
					return newPasswordContainer(), nil
				},
				DecryptContainerItemFn: func(_ context.Context, _ *npwssdk.PsrContainerItem, _ string) (string, error) {
					return "decrypted-pw", nil
				},
			},
			ref:  esv1.ExternalSecretDataRemoteRef{Key: testGUID},
			want: "decrypted-pw",
		},
		{
			name: "container_not_found",
			fc: &fake.Client{
				GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
					return nil, nil
				},
			},
			ref:     esv1.ExternalSecretDataRemoteRef{Key: testGUID},
			wantErr: "not found",
		},
		{
			name: "item_not_found",
			fc: &fake.Client{
				GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
					return newPasswordContainer(), nil
				},
			},
			ref:       esv1.ExternalSecretDataRemoteRef{Key: testGUID, Property: "nonexistent"},
			wantNoSec: true,
		},
		{
			name: "encrypted_item_decrypted",
			fc: &fake.Client{
				GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
					return newPasswordContainer(), nil
				},
				DecryptContainerItemFn: func(_ context.Context, _ *npwssdk.PsrContainerItem, _ string) (string, error) {
					return "s3cret", nil
				},
			},
			ref:  esv1.ExternalSecretDataRemoteRef{Key: testGUID, Property: "Password"},
			want: "s3cret",
		},
		{
			name: "api_error",
			fc: &fake.Client{
				GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
					return nil, fmt.Errorf("connection refused")
				},
			},
			ref:     esv1.ExternalSecretDataRemoteRef{Key: testGUID},
			wantErr: "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{api: tt.fc}
			got, err := p.GetSecret(context.Background(), tt.ref)

			if tt.wantNoSec {
				var nse esv1.NoSecretError
				if !errors.As(err, &nse) {
					t.Errorf("expected NoSecretError, got %v", err)
				}
				return
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", string(got), tt.want)
			}
		})
	}
}

// --- TestGetSecretMap -------------------------------------------------------

func TestGetSecretMap(t *testing.T) {
	tests := []struct {
		name    string
		fc      *fake.Client
		ref     esv1.ExternalSecretDataRemoteRef
		want    map[string]string
		wantErr string
	}{
		{
			name: "all_items",
			fc: &fake.Client{
				GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
					return &npwssdk.PsrContainer{
						ID: testGUID,
						Items: []npwssdk.PsrContainerItem{
							{Name: "user", ContainerItemType: npwssdk.ContainerItemText, Value: "admin"},
							{Name: "url", ContainerItemType: npwssdk.ContainerItemURL, Value: "https://x"},
						},
					}, nil
				},
			},
			ref:  esv1.ExternalSecretDataRemoteRef{Key: testGUID},
			want: map[string]string{"user": "admin", "url": "https://x"},
		},
		{
			name: "encrypted_decrypted",
			fc: &fake.Client{
				GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
					return &npwssdk.PsrContainer{
						ID: testGUID,
						Items: []npwssdk.PsrContainerItem{
							{Name: "user", ContainerItemType: npwssdk.ContainerItemText, Value: "admin"},
							{Name: "pass", ContainerItemType: npwssdk.ContainerItemPassword, Value: "enc"},
						},
					}, nil
				},
				DecryptContainerItemFn: func(_ context.Context, _ *npwssdk.PsrContainerItem, _ string) (string, error) {
					return "pw", nil
				},
			},
			ref:  esv1.ExternalSecretDataRemoteRef{Key: testGUID},
			want: map[string]string{"user": "admin", "pass": "pw"},
		},
		{
			name: "header_skipped",
			fc: &fake.Client{
				GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
					return &npwssdk.PsrContainer{
						ID: testGUID,
						Items: []npwssdk.PsrContainerItem{
							{Name: "Header1", ContainerItemType: npwssdk.ContainerItemHeader},
							{Name: "user", ContainerItemType: npwssdk.ContainerItemText, Value: "admin"},
						},
					}, nil
				},
			},
			ref:  esv1.ExternalSecretDataRemoteRef{Key: testGUID},
			want: map[string]string{"user": "admin"},
		},
		{
			name: "not_found",
			fc: &fake.Client{
				GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
					return nil, nil
				},
			},
			ref:     esv1.ExternalSecretDataRemoteRef{Key: testGUID},
			wantErr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{api: tt.fc}
			got, err := p.GetSecretMap(context.Background(), tt.ref)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("got %d items, want %d", len(got), len(tt.want))
				return
			}
			for k, wv := range tt.want {
				if string(got[k]) != wv {
					t.Errorf("key %q: got %q, want %q", k, string(got[k]), wv)
				}
			}
		})
	}
}

// --- TestGetAllSecrets ------------------------------------------------------

func TestGetAllSecrets(t *testing.T) {
	t.Run("returns_all", func(t *testing.T) {
		fc := &fake.Client{
			GetContainerListFn: func(_ context.Context, _ npwssdk.ContainerType, _ *npwssdk.PsrContainerListFilter) ([]npwssdk.PsrContainer, error) {
				return []npwssdk.PsrContainer{
					{
						Info:  &npwssdk.PsrContainerInfo{ContainerName: "Entry1"},
						Items: []npwssdk.PsrContainerItem{{Name: "Password", ContainerItemType: npwssdk.ContainerItemPassword, Value: "enc1"}},
					},
					{
						Info:  &npwssdk.PsrContainerInfo{ContainerName: "Entry2"},
						Items: []npwssdk.PsrContainerItem{{Name: "Password", ContainerItemType: npwssdk.ContainerItemPassword, Value: "enc2"}},
					},
				}, nil
			},
			DecryptContainerItemFn: func(_ context.Context, item *npwssdk.PsrContainerItem, _ string) (string, error) {
				if item.Value == "enc1" {
					return "pw1", nil
				}
				return "pw2", nil
			},
		}
		p := &Provider{api: fc}
		got, err := p.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if len(got) != 2 {
			t.Errorf("expected 2 entries, got %d", len(got))
			return
		}
		if string(got["Entry1"]) != "pw1" {
			t.Errorf("Entry1: got %q, want %q", string(got["Entry1"]), "pw1")
		}
		if string(got["Entry2"]) != "pw2" {
			t.Errorf("Entry2: got %q, want %q", string(got["Entry2"]), "pw2")
		}
	})

	t.Run("regex_filter", func(t *testing.T) {
		fc := &fake.Client{
			GetContainerListFn: func(_ context.Context, _ npwssdk.ContainerType, _ *npwssdk.PsrContainerListFilter) ([]npwssdk.PsrContainer, error) {
				return []npwssdk.PsrContainer{
					{
						Info:  &npwssdk.PsrContainerInfo{ContainerName: "ProdDB"},
						Items: []npwssdk.PsrContainerItem{{Name: "Password", ContainerItemType: npwssdk.ContainerItemPassword, Value: "enc"}},
					},
					{
						Info:  &npwssdk.PsrContainerInfo{ContainerName: "DevDB"},
						Items: []npwssdk.PsrContainerItem{{Name: "Password", ContainerItemType: npwssdk.ContainerItemPassword, Value: "enc"}},
					},
					{
						Info:  &npwssdk.PsrContainerInfo{ContainerName: "StagingDB"},
						Items: []npwssdk.PsrContainerItem{{Name: "Password", ContainerItemType: npwssdk.ContainerItemPassword, Value: "enc"}},
					},
				}, nil
			},
			DecryptContainerItemFn: func(_ context.Context, _ *npwssdk.PsrContainerItem, _ string) (string, error) {
				return "secret", nil
			},
		}
		p := &Provider{api: fc}
		got, err := p.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
			Name: &esv1.FindName{RegExp: "^Prod"},
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if len(got) != 1 {
			t.Errorf("expected 1 entry, got %d", len(got))
			return
		}
		if _, ok := got["ProdDB"]; !ok {
			t.Errorf("expected ProdDB in result")
		}
	})

	t.Run("decrypt_fail_skipped", func(t *testing.T) {
		fc := &fake.Client{
			GetContainerListFn: func(_ context.Context, _ npwssdk.ContainerType, _ *npwssdk.PsrContainerListFilter) ([]npwssdk.PsrContainer, error) {
				return []npwssdk.PsrContainer{
					{
						Info:  &npwssdk.PsrContainerInfo{ContainerName: "First"},
						Items: []npwssdk.PsrContainerItem{{Name: "Password", ContainerItemType: npwssdk.ContainerItemPassword, Value: "bad"}},
					},
					{
						Info:  &npwssdk.PsrContainerInfo{ContainerName: "Second"},
						Items: []npwssdk.PsrContainerItem{{Name: "Password", ContainerItemType: npwssdk.ContainerItemPassword, Value: "good"}},
					},
				}, nil
			},
			DecryptContainerItemFn: func(_ context.Context, item *npwssdk.PsrContainerItem, _ string) (string, error) {
				if item.Value == "bad" {
					return "", fmt.Errorf("decrypt failed")
				}
				return "ok", nil
			},
		}
		p := &Provider{api: fc}
		got, err := p.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if len(got) != 1 {
			t.Errorf("expected 1 entry, got %d", len(got))
			return
		}
		if string(got["Second"]) != "ok" {
			t.Errorf("Second: got %q, want %q", string(got["Second"]), "ok")
		}
	})
}

// --- TestPushSecret ---------------------------------------------------------

func TestPushSecret(t *testing.T) {
	t.Run("update_existing", func(t *testing.T) {
		updateCalled := false
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return newPasswordContainer(), nil
			},
			UpdateContainerFn: func(_ context.Context, c *npwssdk.PsrContainer) (*npwssdk.PsrContainer, error) {
				updateCalled = true
				return c, nil
			},
		}
		p := &Provider{api: fc}
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("new-value")}}
		data := &testPushSecretData{remoteKey: testGUID, property: "UserName", secretKey: "key"}
		err := p.PushSecret(context.Background(), secret, data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !updateCalled {
			t.Errorf("expected UpdateContainerFn to be called")
		}
	})

	t.Run("value_unchanged", func(t *testing.T) {
		updateCalled := false
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return newPasswordContainer(), nil
			},
			UpdateContainerFn: func(_ context.Context, c *npwssdk.PsrContainer) (*npwssdk.PsrContainer, error) {
				updateCalled = true
				return c, nil
			},
		}
		p := &Provider{api: fc}
		// "admin" is the current value for UserName
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("admin")}}
		data := &testPushSecretData{remoteKey: testGUID, property: "UserName", secretKey: "key"}
		err := p.PushSecret(context.Background(), secret, data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if updateCalled {
			t.Errorf("expected UpdateContainerFn NOT to be called when value is unchanged")
		}
	})

	t.Run("new_item", func(t *testing.T) {
		updateCalled := false
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return newPasswordContainer(), nil
			},
			UpdateContainerFn: func(_ context.Context, c *npwssdk.PsrContainer) (*npwssdk.PsrContainer, error) {
				updateCalled = true
				return c, nil
			},
		}
		p := &Provider{api: fc}
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("new-field-value")}}
		data := &testPushSecretData{remoteKey: testGUID, property: "NewField", secretKey: "key"}
		err := p.PushSecret(context.Background(), secret, data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !updateCalled {
			t.Errorf("expected UpdateContainerFn to be called for new item")
		}
	})

	t.Run("create_container", func(t *testing.T) {
		addCalled := false
		userIDCalled := false
		fc := &fake.Client{
			GetContainerByNameFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return nil, nil
			},
			GetCurrentUserIDFn: func() string {
				userIDCalled = true
				return "user-org-unit-id"
			},
			AddContainerFn: func(_ context.Context, c *npwssdk.PsrContainer, parentOrgUnitID string) (*npwssdk.PsrContainer, error) {
				addCalled = true
				return c, nil
			},
		}
		p := &Provider{api: fc}
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("pw-value")}}
		data := &testPushSecretData{remoteKey: "NewEntry", property: "", secretKey: "key"}
		err := p.PushSecret(context.Background(), secret, data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !addCalled {
			t.Errorf("expected AddContainerFn to be called")
		}
		if !userIDCalled {
			t.Errorf("expected GetCurrentUserIDFn to be called")
		}
	})

	t.Run("guid_not_found", func(t *testing.T) {
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return nil, nil
			},
		}
		p := &Provider{api: fc}
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("val")}}
		data := &testPushSecretData{remoteKey: testGUID, property: "Password", secretKey: "key"}
		err := p.PushSecret(context.Background(), secret, data)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected error containing 'not found', got %v", err)
		}
	})

	t.Run("guid_property_on_create", func(t *testing.T) {
		fc := &fake.Client{
			GetContainerByNameFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return nil, nil
			},
		}
		p := &Provider{api: fc}
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("val")}}
		data := &testPushSecretData{remoteKey: "NewEntry", property: testGUID, secretKey: "key"}
		err := p.PushSecret(context.Background(), secret, data)
		if err == nil || !strings.Contains(err.Error(), "cannot create") {
			t.Errorf("expected error containing 'cannot create', got %v", err)
		}
	})

	t.Run("dataname_change", func(t *testing.T) {
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return newPasswordContainer(), nil
			},
			UpdateContainerFn: noopUpdate,
		}
		p := &Provider{api: fc}
		// Pushing a different value to the "Name" field would change the DataName
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("DifferentName")}}
		data := &testPushSecretData{remoteKey: testGUID, property: "Name", secretKey: "key"}
		err := p.PushSecret(context.Background(), secret, data)
		if err == nil || !strings.Contains(err.Error(), "would change") {
			t.Errorf("expected error containing 'would change', got %v", err)
		}
	})
}

// --- TestDeleteSecret -------------------------------------------------------

func TestDeleteSecret(t *testing.T) {
	t.Run("whole_entry", func(t *testing.T) {
		deleteCalled := false
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return newPasswordContainer(), nil
			},
			DeleteContainerFn: func(_ context.Context, _ *npwssdk.PsrContainer) error {
				deleteCalled = true
				return nil
			},
		}
		p := &Provider{api: fc, deletionPolicyWholeEntry: true}
		ref := &testPushSecretRemoteRef{remoteKey: testGUID, property: "UserName"}
		err := p.DeleteSecret(context.Background(), ref)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !deleteCalled {
			t.Errorf("expected DeleteContainerFn to be called")
		}
	})

	t.Run("single_field", func(t *testing.T) {
		updateCalled := false
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return newPasswordContainer(), nil // 3 items: Name, Password, UserName
			},
			UpdateContainerFn: func(_ context.Context, c *npwssdk.PsrContainer) (*npwssdk.PsrContainer, error) {
				updateCalled = true
				if len(c.Items) != 2 {
					t.Errorf("expected 2 items after removal, got %d", len(c.Items))
				}
				return c, nil
			},
		}
		p := &Provider{api: fc, deletionPolicyWholeEntry: false}
		ref := &testPushSecretRemoteRef{remoteKey: testGUID, property: "UserName"}
		err := p.DeleteSecret(context.Background(), ref)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !updateCalled {
			t.Errorf("expected UpdateContainerFn to be called")
		}
	})

	t.Run("container_gone", func(t *testing.T) {
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return nil, nil
			},
		}
		p := &Provider{api: fc}
		ref := &testPushSecretRemoteRef{remoteKey: testGUID, property: "Password"}
		err := p.DeleteSecret(context.Background(), ref)
		if err != nil {
			t.Errorf("expected no error when container is gone, got %v", err)
		}
	})

	t.Run("item_gone", func(t *testing.T) {
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return newPasswordContainer(), nil
			},
		}
		p := &Provider{api: fc, deletionPolicyWholeEntry: false}
		ref := &testPushSecretRemoteRef{remoteKey: testGUID, property: "nonexistent"}
		err := p.DeleteSecret(context.Background(), ref)
		if err != nil {
			t.Errorf("expected no error when item is gone, got %v", err)
		}
	})

	t.Run("last_field_deletes", func(t *testing.T) {
		deleteCalled := false
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				// Container with Name + Password only
				return &npwssdk.PsrContainer{
					ID: testGUID,
					Items: []npwssdk.PsrContainerItem{
						{Name: "Name", ContainerItemType: npwssdk.ContainerItemText, Value: "MyEntry"},
						{Name: "Password", ContainerItemType: npwssdk.ContainerItemPassword, Value: "enc"},
					},
				}, nil
			},
			DeleteContainerFn: func(_ context.Context, _ *npwssdk.PsrContainer) error {
				deleteCalled = true
				return nil
			},
		}
		p := &Provider{api: fc, deletionPolicyWholeEntry: false}
		ref := &testPushSecretRemoteRef{remoteKey: testGUID, property: "Password"}
		err := p.DeleteSecret(context.Background(), ref)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !deleteCalled {
			t.Errorf("expected DeleteContainerFn to be called when only DataName candidate left")
		}
	})

	t.Run("dataname_delete", func(t *testing.T) {
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return newPasswordContainer(), nil // 3 items: Name, Password, UserName
			},
		}
		p := &Provider{api: fc, deletionPolicyWholeEntry: false}
		// Try to delete the "Name" text field which is the DataName source
		ref := &testPushSecretRemoteRef{remoteKey: testGUID, property: "Name"}
		err := p.DeleteSecret(context.Background(), ref)
		if err == nil || !strings.Contains(err.Error(), "would change") {
			t.Errorf("expected error containing 'would change', got %v", err)
		}
	})
}

// --- TestSecretExists -------------------------------------------------------

func TestSecretExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return newPasswordContainer(), nil
			},
		}
		p := &Provider{api: fc}
		ref := &testPushSecretRemoteRef{remoteKey: testGUID}
		exists, err := p.SecretExists(context.Background(), ref)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !exists {
			t.Errorf("expected exists=true")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		fc := &fake.Client{
			GetContainerFn: func(_ context.Context, _ string) (*npwssdk.PsrContainer, error) {
				return nil, fmt.Errorf("not found")
			},
		}
		p := &Provider{api: fc}
		ref := &testPushSecretRemoteRef{remoteKey: testGUID}
		exists, err := p.SecretExists(context.Background(), ref)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if exists {
			t.Errorf("expected exists=false")
		}
	})
}

// --- TestClose_Logout -------------------------------------------------------

func TestClose_Logout(t *testing.T) {
	logoutCalled := false
	fc := &fake.Client{
		LogoutFn: func(_ context.Context) error {
			logoutCalled = true
			return nil
		},
	}
	p := &Provider{api: fc}
	err := p.Close(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !logoutCalled {
		t.Errorf("expected LogoutFn to be called")
	}
}
