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
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protowire"
)

// Proton Pass item content is a protobuf message (pass-contents item_v1, content
// format version 7). Rather than vendor a generated package and a buf toolchain —
// neither of which this repo uses — we decode and encode the handful of fields we
// need directly with the low-level protowire codec. Field numbers below mirror the
// published item_v1.proto / vault_v1.proto schemas; they are the single home for
// schema knowledge in this provider.
const (
	// Item.
	fItemMetadata    = 1
	fItemContent     = 2
	fItemExtraFields = 4
	// Metadata.
	fMetaName = 1
	fMetaNote = 2
	fMetaUUID = 3
	// Content oneof.
	fContentLogin      = 3
	fContentCreditCard = 5
	fContentIdentity   = 6
	fContentSSHKey     = 7
	fContentWifi       = 8
	fContentCustom     = 9
	// ItemLogin.
	fLoginEmail    = 1
	fLoginPassword = 2
	fLoginURLs     = 3
	fLoginTOTPURI  = 4
	fLoginUsername = 6
	// ItemCreditCard.
	fCardholderName   = 1
	fCardNumber       = 3
	fCardVerification = 4
	fCardExpiration   = 5
	fCardPIN          = 6
	// ItemIdentity (subset projected).
	fIdentityFullName     = 1
	fIdentityEmail        = 2
	fIdentityPhone        = 3
	fIdentityFirstName    = 4
	fIdentityLastName     = 6
	fIdentityOrganization = 10
	fIdentityWebsite      = 22
	fIdentityCompany      = 31
	fIdentityJobTitle     = 32
	// ItemWifi.
	fWifiSSID     = 1
	fWifiPassword = 2
	// ItemSSHKey.
	fSSHPrivateKey = 1
	fSSHPublicKey  = 2
	// ItemCustom / CustomSection.
	fCustomSections = 1
	fSectionFields  = 2
	// ExtraField and its content oneof.
	fExtraName      = 1
	fExtraTOTP      = 2
	fExtraText      = 3
	fExtraHidden    = 4
	fExtraTimestamp = 5
	// Nested single-string messages (ExtraTextField/ExtraHiddenField.content, ExtraTotp.totp_uri).
	fNestedContent = 1
	// Timestamp.
	fTimestampSeconds = 1
	// Vault.
	fVaultName = 1
)

// fieldKeyTOTP is the reserved property for a login's one-time-password code: it
// is produced on demand (a generated code, never the seed) and is deliberately
// excluded from projectItem so the seed never lands in a bulk GetSecretMap.
const fieldKeyTOTP = "totp"

// writeField is a single field to persist on a custom (extra-field) item.
type writeField struct {
	Label  string
	Value  string
	Hidden bool // hidden => secret field; otherwise a plain text field
}

// decodedItem is a decrypted item: its title (for matching) plus the verbatim
// decrypted Item message. The raw bytes are kept so updates can preserve fields
// we do not model (see setExtraField).
type decodedItem struct {
	name      string
	plaintext []byte
}

// --- low-level protobuf helpers ---

// bytesFields parses a message into field number -> all length-delimited (string
// or sub-message) values. Non-length-delimited fields are skipped. Use it when a
// caller needs several fields of one message; for a single field on a hot path,
// prefer the map-free lastBytesField.
func bytesFields(b []byte) (map[int][][]byte, error) {
	out := map[int][][]byte{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, fmt.Errorf("protonpass: parse protobuf: %w", protowire.ParseError(n))
		}
		b = b[n:]
		if typ == protowire.BytesType {
			v, m := protowire.ConsumeBytes(b)
			if m < 0 {
				return nil, fmt.Errorf("protonpass: parse protobuf bytes: %w", protowire.ParseError(m))
			}
			out[int(num)] = append(out[int(num)], v)
			b = b[m:]
			continue
		}
		m := protowire.ConsumeFieldValue(num, typ, b)
		if m < 0 {
			return nil, fmt.Errorf("protonpass: skip protobuf field: %w", protowire.ParseError(m))
		}
		b = b[m:]
	}
	return out, nil
}

// lastBytesField returns the last length-delimited value of field num (nil if
// absent) without allocating a field map — for single-field reads on hot paths
// (item title, totp URI, vault name). A malformed message yields nil; callers
// already treat an unreadable item as absent.
func lastBytesField(b []byte, num protowire.Number) []byte {
	var val []byte
	for len(b) > 0 {
		n, typ, tn := protowire.ConsumeTag(b)
		if tn < 0 {
			return nil
		}
		b = b[tn:]
		if n == num && typ == protowire.BytesType {
			v, vn := protowire.ConsumeBytes(b)
			if vn < 0 {
				return nil
			}
			val = v
			b = b[vn:]
			continue
		}
		fn := protowire.ConsumeFieldValue(n, typ, b)
		if fn < 0 {
			return nil
		}
		b = b[fn:]
	}
	return val
}

// lastVarint returns the last varint value of field num in a message.
func lastVarint(b []byte, num protowire.Number) (uint64, bool) {
	var val uint64
	var found bool
	for len(b) > 0 {
		n, typ, tn := protowire.ConsumeTag(b)
		if tn < 0 {
			return 0, false
		}
		b = b[tn:]
		if n == num && typ == protowire.VarintType {
			v, vn := protowire.ConsumeVarint(b)
			if vn < 0 {
				return 0, false
			}
			val, found = v, true
			b = b[vn:]
			continue
		}
		fn := protowire.ConsumeFieldValue(n, typ, b)
		if fn < 0 {
			return 0, false
		}
		b = b[fn:]
	}
	return val, found
}

func lastString(f map[int][][]byte, num int) string {
	v := f[num]
	if len(v) == 0 {
		return ""
	}
	return string(v[len(v)-1])
}

func repeatedString(f map[int][][]byte, num int) []string {
	raw := f[num]
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, string(v))
	}
	return out
}

func subMessage(f map[int][][]byte, num int) ([]byte, bool) {
	v := f[num]
	if len(v) == 0 {
		return nil, false
	}
	return v[len(v)-1], true
}

// optFields parses the sub-message at field num, returning (fields, true) when
// present and (nil, false) when absent. It folds the presence check and the parse
// into one step so callers need not pre-check then re-look-up.
func optFields(f map[int][][]byte, num int) (map[int][][]byte, bool, error) {
	b, ok := subMessage(f, num)
	if !ok {
		return nil, false, nil
	}
	m, err := bytesFields(b)
	return m, true, err
}

func putString(b []byte, num protowire.Number, s string) []byte {
	if s == "" {
		return b
	}
	b = protowire.AppendTag(b, num, protowire.BytesType)
	return protowire.AppendBytes(b, []byte(s))
}

func putMessage(b []byte, num protowire.Number, v []byte) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	return protowire.AppendBytes(b, v)
}

// --- decoding ---

// itemName extracts metadata.name from decrypted item content.
func itemName(plaintext []byte) string {
	return string(lastBytesField(lastBytesField(plaintext, fItemMetadata), fMetaName))
}

// vaultName extracts Vault.name from decrypted vault content.
func vaultName(plaintext []byte) string {
	return string(lastBytesField(plaintext, fVaultName))
}

// loginTOTPURI returns the raw otpauth seed URI of a login item, if any. It is
// used only to generate a code on explicit request; the value is never written
// to a Secret.
func loginTOTPURI(plaintext []byte) string {
	login := lastBytesField(lastBytesField(plaintext, fItemContent), fContentLogin)
	return string(lastBytesField(login, fLoginTOTPURI))
}

// projectItem flattens a decrypted item to label->value pairs for GetSecretMap /
// dataFrom extract. The TOTP seed is never emitted (a login's totp_uri and any
// totp-type extra field are skipped); request the generated code via the reserved
// "totp" property. Keys are raw labels; the controller converts them to valid
// Secret keys (esutils.ConvertKeys) per the user's conversionStrategy.
func projectItem(plaintext []byte) (map[string][]byte, error) {
	top, err := bytesFields(plaintext)
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	// set is for typed/built-in fields (note, login password, card number, …). An
	// unset proto3 scalar is indistinguishable from "" on the wire, so an empty
	// value means "absent" and is suppressed — matching Proton's own field
	// resolver, which lists typed fields only when non-empty.
	set := func(k, v string) {
		if k != "" && v != "" {
			out[k] = []byte(v)
		}
	}
	// setPresent is for explicit extra/section fields, whose presence is recorded
	// distinctly from their value. Proton lists these unconditionally, so an empty
	// value is surfaced (and a pushed empty value round-trips) — only an empty
	// label is dropped, since it is unaddressable.
	setPresent := func(k, v string) {
		if k != "" {
			out[k] = []byte(v)
		}
	}

	meta, ok, err := optFields(top, fItemMetadata)
	if err != nil {
		return nil, err
	}
	if ok {
		set("note", lastString(meta, fMetaNote))
	}

	content, ok, err := optFields(top, fItemContent)
	if err != nil {
		return nil, err
	}
	if ok {
		if err := projectContent(content, set, setPresent); err != nil {
			return nil, err
		}
	}

	for _, ef := range top[fItemExtraFields] {
		if err := collectField(setPresent, ef); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// projectContent emits the named fields of a Content oneof. Exactly one arm
// applies; the per-type field-label mapping is the user-facing contract. Typed
// fields use set (empty == absent); a custom item's section fields use setPresent
// (surfaced even when empty, like top-level extra fields).
func projectContent(cf map[int][][]byte, set, setPresent func(k, v string)) error {
	if lf, ok, err := optFields(cf, fContentLogin); err != nil {
		return err
	} else if ok {
		username := lastString(lf, fLoginUsername)
		if username == "" {
			username = lastString(lf, fLoginEmail)
		}
		set("username", username)
		set("password", lastString(lf, fLoginPassword))
		set("email", lastString(lf, fLoginEmail))
		if u := repeatedString(lf, fLoginURLs); len(u) > 0 {
			set("url", u[0])
		}
		// totp_uri (seed) intentionally omitted — see fieldKeyTOTP.
		return nil
	}
	if cc, ok, err := optFields(cf, fContentCreditCard); err != nil {
		return err
	} else if ok {
		set("cardholderName", lastString(cc, fCardholderName))
		set("number", lastString(cc, fCardNumber))
		set("verificationNumber", lastString(cc, fCardVerification))
		set("expirationDate", lastString(cc, fCardExpiration))
		set("pin", lastString(cc, fCardPIN))
		return nil
	}
	if id, ok, err := optFields(cf, fContentIdentity); err != nil {
		return err
	} else if ok {
		set("fullName", lastString(id, fIdentityFullName))
		set("email", lastString(id, fIdentityEmail))
		set("phoneNumber", lastString(id, fIdentityPhone))
		set("firstName", lastString(id, fIdentityFirstName))
		set("lastName", lastString(id, fIdentityLastName))
		set("organization", lastString(id, fIdentityOrganization))
		set("website", lastString(id, fIdentityWebsite))
		set("company", lastString(id, fIdentityCompany))
		set("jobTitle", lastString(id, fIdentityJobTitle))
		return nil
	}
	if w, ok, err := optFields(cf, fContentWifi); err != nil {
		return err
	} else if ok {
		set("ssid", lastString(w, fWifiSSID))
		set("password", lastString(w, fWifiPassword))
		return nil
	}
	if s, ok, err := optFields(cf, fContentSSHKey); err != nil {
		return err
	} else if ok {
		set("privateKey", lastString(s, fSSHPrivateKey))
		set("publicKey", lastString(s, fSSHPublicKey))
		return nil
	}
	if cu, ok, err := optFields(cf, fContentCustom); err != nil {
		return err
	} else if ok {
		for _, sec := range cu[fCustomSections] {
			sf, err := bytesFields(sec)
			if err != nil {
				return err
			}
			for _, ef := range sf[fSectionFields] {
				if err := collectField(setPresent, ef); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// collectField adds a text/hidden/timestamp extra field to the projection. The
// field is surfaced whenever its content arm is present, even with an empty value
// (setPresent), matching Proton's unconditional extra-field listing. TOTP extra
// fields are skipped — their value is the otpauth seed.
func collectField(setPresent func(k, v string), efBytes []byte) error {
	f, err := bytesFields(efBytes)
	if err != nil {
		return err
	}
	name := lastString(f, fExtraName)
	if name == "" {
		return nil
	}
	if nf, ok, err := optFields(f, fExtraText); err != nil {
		return err
	} else if ok {
		setPresent(name, lastString(nf, fNestedContent))
		return nil
	}
	if nf, ok, err := optFields(f, fExtraHidden); err != nil {
		return err
	} else if ok {
		setPresent(name, lastString(nf, fNestedContent))
		return nil
	}
	if tsf, ok, err := optFields(f, fExtraTimestamp); err != nil {
		return err
	} else if ok {
		if inner, ok := subMessage(tsf, fNestedContent); ok {
			if secs, ok := lastVarint(inner, fTimestampSeconds); ok {
				//nolint:gosec // protobuf Timestamp.seconds is a real epoch value
				setPresent(name, time.Unix(int64(secs), 0).UTC().Format("2006-01-02T15:04:05Z"))
			}
		}
	}
	// A totp-type extra field carries the otpauth seed and is never projected.
	return nil
}

// --- encoding (create) ---

// marshalLoginItem builds a Proton Pass Login item.
func marshalLoginItem(name, note, username, password string, urls []string) []byte {
	var login []byte
	login = putString(login, fLoginPassword, password)
	for _, u := range urls {
		login = putString(login, fLoginURLs, u)
	}
	login = putString(login, fLoginUsername, username)
	content := putMessage(nil, fContentLogin, login)
	return assembleItem(name, note, content, nil)
}

// marshalCustomItem builds a custom item whose data lives in hidden/text extra
// fields — the representation a pushed k8s Secret maps onto.
func marshalCustomItem(name, note string, fields []writeField) []byte {
	content := putMessage(nil, fContentCustom, nil) // empty ItemCustom{}
	extra := make([][]byte, 0, len(fields))
	for _, f := range fields {
		extra = append(extra, buildExtraField(f.Label, f.Value, f.Hidden))
	}
	return assembleItem(name, note, content, extra)
}

// assembleItem encodes metadata + content + extra_fields into an Item.
func assembleItem(name, note string, content []byte, extraFields [][]byte) []byte {
	var meta []byte
	meta = putString(meta, fMetaName, name)
	meta = putString(meta, fMetaNote, note)
	meta = putString(meta, fMetaUUID, uuid.NewString())

	var item []byte
	item = putMessage(item, fItemMetadata, meta)
	item = putMessage(item, fItemContent, content)
	for _, ef := range extraFields {
		item = putMessage(item, fItemExtraFields, ef)
	}
	return item
}

// buildExtraField encodes one ExtraField with a hidden or text value.
func buildExtraField(label, value string, hidden bool) []byte {
	var ef []byte
	ef = putString(ef, fExtraName, label)
	nested := putString(nil, fNestedContent, value)
	if hidden {
		ef = putMessage(ef, fExtraHidden, nested)
	} else {
		ef = putMessage(ef, fExtraText, nested)
	}
	return ef
}

// --- encoding (update): field-preserving surgery ---

// setExtraField returns item content with label set to value as a hidden extra
// field, replacing any existing top-level extra field with the same label. Every
// other top-level field (metadata, typed content, and any field we do not model)
// is preserved byte-for-byte, so updating one field never strips the rest.
//
// Writes operate on top-level extra_fields only. A label that a read surfaced
// from inside a custom item's section (see projectContent) cannot be edited here,
// so writing it is refused rather than silently appended as a duplicate.
func setExtraField(plaintext []byte, label, value string) ([]byte, error) {
	preserved, extra, err := splitExtraFields(plaintext)
	if err != nil {
		return nil, err
	}
	idxs, err := findExtra(extra, label)
	if err != nil {
		return nil, err
	}
	if len(idxs) == 0 {
		if err := refuseSectionField(plaintext, label); err != nil {
			return nil, err
		}
		extra = append(extra, buildExtraField(label, value, true))
		return reassemble(preserved, extra), nil
	}
	// Replace the first occurrence and drop any duplicates, so a pre-existing
	// duplicate label can't leave a stale value that later wins on read.
	extra[idxs[0]] = buildExtraField(label, value, true)
	for i := len(idxs) - 1; i >= 1; i-- {
		extra = slices.Delete(extra, idxs[i], idxs[i]+1)
	}
	return reassemble(preserved, extra), nil
}

// removeExtraField returns item content with the named top-level extra field
// removed (preserving all other fields) and whether a field was removed. As with
// setExtraField, this acts on top-level extra_fields only; deleting a label that
// lives in a section is refused rather than silently reported as a no-op.
func removeExtraField(plaintext []byte, label string) ([]byte, bool, error) {
	preserved, extra, err := splitExtraFields(plaintext)
	if err != nil {
		return nil, false, err
	}
	idxs, err := findExtra(extra, label)
	if err != nil {
		return nil, false, err
	}
	if len(idxs) == 0 {
		if err := refuseSectionField(plaintext, label); err != nil {
			return nil, false, err
		}
		return reassemble(preserved, extra), false, nil
	}
	// Remove every occurrence so a duplicate label cannot survive the delete.
	for i := len(idxs) - 1; i >= 0; i-- {
		extra = slices.Delete(extra, idxs[i], idxs[i]+1)
	}
	return reassemble(preserved, extra), true, nil
}

// refuseSectionField returns an error when label names a field that lives inside
// a custom item's section. projectItem surfaces section fields on read, but the
// write surgery above can only edit top-level extra_fields, so editing a section
// field would silently corrupt the item (a duplicate top-level field on push, a
// no-op on delete). Refusing keeps the read/write namespaces honest.
func refuseSectionField(plaintext []byte, label string) error {
	inSection, err := labelInSection(plaintext, label)
	if err != nil {
		return err
	}
	if inSection {
		return fmt.Errorf("protonpass: field %q lives in a custom-item section, which this provider cannot edit; "+
			"move it out of the section in the Proton Pass app, or target a top-level field", label)
	}
	return nil
}

// labelInSection reports whether label names a field inside any CustomSection of
// a custom item (as opposed to a top-level extra field).
func labelInSection(plaintext []byte, label string) (bool, error) {
	top, err := bytesFields(plaintext)
	if err != nil {
		return false, err
	}
	content, ok, err := optFields(top, fItemContent)
	if err != nil || !ok {
		return false, err
	}
	custom, ok, err := optFields(content, fContentCustom)
	if err != nil || !ok {
		return false, err
	}
	for _, sec := range custom[fCustomSections] {
		sf, err := bytesFields(sec)
		if err != nil {
			return false, err
		}
		for _, ef := range sf[fSectionFields] {
			f, err := bytesFields(ef)
			if err != nil {
				return false, err
			}
			if lastString(f, fExtraName) == label {
				return true, nil
			}
		}
	}
	return false, nil
}

// findExtra returns the indices of every extra field named label, in order.
func findExtra(extra [][]byte, label string) ([]int, error) {
	var idxs []int
	for i, ef := range extra {
		f, err := bytesFields(ef)
		if err != nil {
			return nil, err
		}
		if lastString(f, fExtraName) == label {
			idxs = append(idxs, i)
		}
	}
	return idxs, nil
}

// splitExtraFields returns the raw bytes of all top-level fields except
// extra_fields (preserved verbatim, in order) and the value bytes of each
// extra_fields entry.
func splitExtraFields(plaintext []byte) (preserved []byte, extra [][]byte, err error) {
	b := plaintext
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, nil, fmt.Errorf("protonpass: parse item: %w", protowire.ParseError(n))
		}
		vlen := protowire.ConsumeFieldValue(num, typ, b[n:])
		if vlen < 0 {
			return nil, nil, fmt.Errorf("protonpass: parse item field: %w", protowire.ParseError(vlen))
		}
		field := b[:n+vlen]
		if int(num) == fItemExtraFields && typ == protowire.BytesType {
			v, _ := protowire.ConsumeBytes(b[n:])
			extra = append(extra, v)
		} else {
			preserved = append(preserved, field...)
		}
		b = b[n+vlen:]
	}
	return preserved, extra, nil
}

// reassemble appends the extra fields after the preserved fields. Re-ordering
// extra_fields to the end is deliberate and semantically safe (protobuf field
// order is not significant).
func reassemble(preserved []byte, extra [][]byte) []byte {
	out := make([]byte, 0, len(preserved))
	out = append(out, preserved...)
	for _, ef := range extra {
		out = putMessage(out, fItemExtraFields, ef)
	}
	return out
}
