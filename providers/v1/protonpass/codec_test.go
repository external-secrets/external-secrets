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

package protonpass

import (
	"strings"
	"testing"
)

func TestLoginItemMarshalProjectRoundTrip(t *testing.T) {
	blob := marshalLoginItem("demo login", "a note", "alice", "s3cret", []string{"https://example.com/x", "https://example.com/y"})
	if name := itemName(blob); name != "demo login" {
		t.Errorf("name = %q", name)
	}
	m, err := projectItem(blob)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{"username": "alice", "password": "s3cret", "url": "https://example.com/x", "note": "a note"} {
		if string(m[k]) != v {
			t.Errorf("field %q = %q, want %q", k, m[k], v)
		}
	}
	if _, ok := m["email"]; ok {
		t.Errorf("email should be absent when only username set, got %q", m["email"])
	}
}

func TestCustomItemMarshalProjectRoundTrip(t *testing.T) {
	blob := marshalCustomItem("demo database", "", []writeField{
		{Label: "Host", Value: "db.internal", Hidden: false},
		{Label: "Password", Value: "p@ss", Hidden: true},
		{Label: "Database Type", Value: "postgres", Hidden: false},
	})
	m, err := projectItem(blob)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{"Host": "db.internal", "Password": "p@ss", "Database Type": "postgres"} {
		if string(m[k]) != v {
			t.Errorf("field %q = %q, want %q", k, m[k], v)
		}
	}
}

func TestProjectExcludesTOTPSeed(t *testing.T) {
	// A login carrying a totp_uri (the seed) plus a totp-type extra field: neither
	// may appear in the projected bulk map.
	login := putString(nil, fLoginPassword, "pw")
	login = putString(login, fLoginTOTPURI, "otpauth://totp/x?secret=SEED")
	content := putMessage(nil, fContentLogin, login)
	totp := putString(nil, fNestedContent, "otpauth://totp/y?secret=SEED2")
	ef := putString(nil, fExtraName, "2fa")
	ef = putMessage(ef, fExtraTOTP, totp)
	blob := assembleItem("x", "", content, [][]byte{ef})

	m, err := projectItem(blob)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range m {
		if strings.Contains(string(v), "SEED") || strings.Contains(string(v), "otpau") {
			t.Fatalf("projection leaked a seed under %q: %q", k, v)
		}
	}
	if _, ok := m[fieldKeyTOTP]; ok {
		t.Fatal("totp must not be in the bulk projection")
	}
	if string(m["password"]) != "pw" {
		t.Errorf("password = %q", m["password"])
	}
	// The seed is reachable only via the explicit helper, for code generation.
	if loginTOTPURI(blob) != "otpauth://totp/x?secret=SEED" {
		t.Errorf("loginTOTPURI = %q", loginTOTPURI(blob))
	}
}

// TestUpdatePreservesOtherFields verifies the field-preserving update surgery:
// editing one extra field must not strip an item's typed content or its other
// fields (the round-trip-safety property generated code would give for free).
func TestUpdatePreservesOtherFields(t *testing.T) {
	blob := marshalLoginItem("acct", "n", "alice", "pw", []string{"https://x"})

	updated, err := setExtraField(blob, "token", "v1")
	if err != nil {
		t.Fatal(err)
	}
	m, err := projectItem(updated)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{"password": "pw", "username": "alice", "url": "https://x", "note": "n", "token": "v1"} {
		if string(m[k]) != v {
			t.Errorf("after add: %q = %q, want %q", k, m[k], v)
		}
	}

	updated2, err := setExtraField(updated, "token", "v2")
	if err != nil {
		t.Fatal(err)
	}
	m2, err := projectItem(updated2)
	if err != nil {
		t.Fatal(err)
	}
	if string(m2["token"]) != "v2" || string(m2["password"]) != "pw" {
		t.Errorf("after replace: token=%q password=%q", m2["token"], m2["password"])
	}

	removedBlob, removed, err := removeExtraField(updated2, "token")
	if err != nil || !removed {
		t.Fatalf("remove: removed=%v err=%v", removed, err)
	}
	m3, err := projectItem(removedBlob)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m3["token"]; ok {
		t.Error("token still present after remove")
	}
	if string(m3["password"]) != "pw" {
		t.Error("password lost after remove")
	}
}

func TestEmptyExtraFieldRoundTrips(t *testing.T) {
	// An explicitly-present custom field with an empty value must survive
	// projection — Proton's own field resolver lists extra fields unconditionally,
	// so a pushed empty value round-trips rather than vanishing. (A typed field
	// that is merely unset stays suppressed; see the email assertion above.)
	blob := marshalCustomItem("svc", "", []writeField{{Label: "token", Value: "", Hidden: true}})
	m, err := projectItem(blob)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := m["token"]
	if !ok {
		t.Fatal("empty extra field must be present in projection")
	}
	if len(v) != 0 {
		t.Errorf("token = %q, want empty", v)
	}
}

func TestUpdateDeleteHandleDuplicateLabels(t *testing.T) {
	// An item that already carries two top-level fields with the same label:
	// update must not leave a stale duplicate that wins on read, and delete must
	// remove every occurrence.
	login := putMessage(nil, fContentLogin, putString(nil, fLoginPassword, "pw"))
	dup := func(v string) []byte { return buildExtraField("dup", v, true) }
	blob := assembleItem("dupitem", "", login, [][]byte{dup("old1"), dup("old2")})

	updated, err := setExtraField(blob, "dup", "new")
	if err != nil {
		t.Fatal(err)
	}
	m, err := projectItem(updated)
	if err != nil {
		t.Fatal(err)
	}
	if string(m["dup"]) != "new" {
		t.Errorf("update with duplicate labels read back %q, want %q (stale duplicate won)", m["dup"], "new")
	}

	removed, ok, err := removeExtraField(blob, "dup")
	if err != nil || !ok {
		t.Fatalf("remove: ok=%v err=%v", ok, err)
	}
	m2, err := projectItem(removed)
	if err != nil {
		t.Fatal(err)
	}
	if _, present := m2["dup"]; present {
		t.Error("duplicate label survived delete")
	}
}

func TestMarshalProducesNonEmptyWire(t *testing.T) {
	if len(marshalLoginItem("n", "", "u", "p", nil)) == 0 {
		t.Fatal("empty marshal")
	}
}

// customItemWithSectionField builds a custom item carrying one field inside a
// CustomSection (the shape the Proton Pass app produces) so the write path can be
// exercised against a section-resident label.
func customItemWithSectionField(label, value string) []byte {
	field := buildExtraField(label, value, true)
	section := putMessage(nil, fSectionFields, field)
	custom := putMessage(nil, fCustomSections, section)
	content := putMessage(nil, fContentCustom, custom)
	return assembleItem("sectioned", "", content, nil)
}

func TestWriteRefusesSectionField(t *testing.T) {
	blob := customItemWithSectionField("apiKey", "v1")

	// The field is readable (projectItem flattens section fields)...
	m, err := projectItem(blob)
	if err != nil {
		t.Fatal(err)
	}
	if string(m["apiKey"]) != "v1" {
		t.Fatalf("section field not projected: %q", m["apiKey"])
	}

	// ...but editing it must be refused rather than duplicated at the top level.
	if _, err := setExtraField(blob, "apiKey", "v2"); err == nil {
		t.Error("setExtraField must refuse a section-resident label")
	}
	if _, _, err := removeExtraField(blob, "apiKey"); err == nil {
		t.Error("removeExtraField must refuse a section-resident label")
	}

	// A genuinely new label is still added as a top-level field.
	updated, err := setExtraField(blob, "newField", "x")
	if err != nil {
		t.Fatalf("setExtraField(new) = %v", err)
	}
	m2, err := projectItem(updated)
	if err != nil {
		t.Fatal(err)
	}
	if string(m2["newField"]) != "x" || string(m2["apiKey"]) != "v1" {
		t.Errorf("new=%q apiKey=%q", m2["newField"], m2["apiKey"])
	}
}
