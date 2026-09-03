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

package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

// These builders construct item_v1 bytes matching the REAL proto definition:
//   message Item {
//     Metadata metadata = 1;
//     Content  content  = 2;   // <- oneof wrapper the decoder must traverse
//     PlatformSpecific platform_specific = 3;  // skipped
//     repeated ExtraField extra_fields = 4;
//   }
//   message Content { oneof content { ItemNote note=2; ItemLogin login=3; ... } }
//   message ItemLogin { item_email=1; password=2; urls=3; totp_uri=4; passkeys=5; item_username=6; }

func buildMetadata(name, note string) []byte {
	var b []byte
	if name != "" {
		b = protowire.AppendTag(b, metaFieldName, protowire.BytesType)
		b = protowire.AppendString(b, name)
	}
	if note != "" {
		b = protowire.AppendTag(b, metaFieldNote, protowire.BytesType)
		b = protowire.AppendString(b, note)
	}
	return b
}

func buildLogin(email, username, password string, urls []string) []byte {
	var b []byte
	if email != "" {
		b = protowire.AppendTag(b, loginFieldEmail, protowire.BytesType)
		b = protowire.AppendString(b, email)
	}
	if password != "" {
		b = protowire.AppendTag(b, loginFieldPassword, protowire.BytesType)
		b = protowire.AppendString(b, password)
	}
	for _, u := range urls {
		b = protowire.AppendTag(b, loginFieldURLs, protowire.BytesType)
		b = protowire.AppendString(b, u)
	}
	// totp_uri (field 4) omitted; the seed is never projected.
	if username != "" {
		b = protowire.AppendTag(b, loginFieldUsername, protowire.BytesType)
		b = protowire.AppendString(b, username)
	}
	return b
}

// buildContentLogin wraps a login payload in the Content oneof (login = field 3).
func buildContentLogin(login []byte) []byte {
	var b []byte
	if len(login) > 0 {
		b = protowire.AppendTag(b, contentFieldLogin, protowire.BytesType)
		b = protowire.AppendBytes(b, login)
	}
	return b
}

func buildExtraText(name, content string) []byte {
	var b []byte
	b = protowire.AppendTag(b, extraFieldName, protowire.BytesType)
	b = protowire.AppendString(b, name)
	var inner []byte
	inner = protowire.AppendTag(inner, 1, protowire.BytesType)
	inner = protowire.AppendString(inner, content)
	b = protowire.AppendTag(b, extraFieldText, protowire.BytesType)
	b = protowire.AppendBytes(b, inner)
	return b
}

// buildItem assembles an Item with metadata (field 1), content (field 2),
// and extra fields (field 4).
func buildItem(metadata, content []byte, extras ...[]byte) []byte {
	var b []byte
	if len(metadata) > 0 {
		b = protowire.AppendTag(b, itemFieldMetadata, protowire.BytesType)
		b = protowire.AppendBytes(b, metadata)
	}
	if len(content) > 0 {
		b = protowire.AppendTag(b, itemFieldContent, protowire.BytesType)
		b = protowire.AppendBytes(b, content)
	}
	for _, ef := range extras {
		b = protowire.AppendTag(b, itemFieldExtraField, protowire.BytesType)
		b = protowire.AppendBytes(b, ef)
	}
	return b
}

func TestProjectLoginItem(t *testing.T) {
	content := buildItem(
		buildMetadata("My Login", "A note"),
		buildContentLogin(buildLogin("alice@example.com", "alice", "p@ss", []string{"https://example.com"})),
		buildExtraText("API Key", "secret"),
		buildExtraText("empty", ""),
	)

	got, err := Project(content)
	require.NoError(t, err)

	assert.Equal(t, []byte("My Login"), got["title"])
	assert.Equal(t, []byte("A note"), got["note"])
	assert.Equal(t, []byte("alice"), got["username"])
	assert.Equal(t, []byte("p@ss"), got["password"])
	assert.Equal(t, []byte("alice@example.com"), got["email"])
	assert.Equal(t, []byte("https://example.com"), got["url"])
	// Extra fields surface unconditionally when the name is present.
	assert.Equal(t, []byte("secret"), got["API Key"])
	assert.Equal(t, []byte(""), got["empty"])
}

func TestProjectEmptyTypedFieldsAreOmitted(t *testing.T) {
	content := buildItem(
		buildMetadata("No Password", ""),
		buildContentLogin(buildLogin("alice@example.com", "alice", "", nil)),
	)
	got, err := Project(content)
	require.NoError(t, err)

	assert.Equal(t, []byte("No Password"), got["title"])
	// Empty login password must not be present.
	_, ok := got["password"]
	assert.False(t, ok)
	// Empty note must not be present.
	_, ok = got["note"]
	assert.False(t, ok)
	// email and username are non-empty, so they remain.
	assert.Equal(t, []byte("alice@example.com"), got["email"])
	assert.Equal(t, []byte("alice"), got["username"])
}

func TestProjectSecureNote(t *testing.T) {
	// A secure-note item: Content holds an ItemNote (field 2), which is empty.
	var noteContent []byte
	noteContent = protowire.AppendTag(noteContent, contentFieldNote, protowire.BytesType)
	noteContent = protowire.AppendBytes(noteContent, nil)

	content := buildItem(buildMetadata("Only Note", "some note"), noteContent)
	got, err := Project(content)
	require.NoError(t, err)
	assert.Equal(t, []byte("Only Note"), got["title"])
	assert.Equal(t, []byte("some note"), got["note"])
	_, ok := got["password"]
	assert.False(t, ok)
}

func TestProjectSkipsPlatformSpecific(t *testing.T) {
	// Item with a platform_specific (field 3) payload must not corrupt projection.
	platform := []byte{0x0a, 0x03, 'x', 'y', 'z'}
	var item []byte
	item = protowire.AppendTag(item, itemFieldMetadata, protowire.BytesType)
	item = protowire.AppendBytes(item, buildMetadata("T", ""))
	item = protowire.AppendTag(item, 3, protowire.BytesType) // platform_specific
	item = protowire.AppendBytes(item, platform)
	got, err := Project(item)
	require.NoError(t, err)
	assert.Equal(t, []byte("T"), got["title"])
}

func TestProjectInvalidTag(t *testing.T) {
	_, err := Project([]byte{0xff})
	require.Error(t, err)
}

func TestProjectRejectsInvalidWireBytes(t *testing.T) {
	// A metadata field claiming 100 bytes that are not present is truncated.
	var b []byte
	b = protowire.AppendTag(b, itemFieldMetadata, protowire.BytesType)
	b = protowire.AppendVarint(b, 100)
	_, err := Project(b)
	require.Error(t, err)
}

func TestProjectSkipsUnknownItemFields(t *testing.T) {
	// Item with a platform_specific (field 3) payload must project successfully
	// and must NOT surface the unknown field.
	platform := []byte{0x0a, 0x03, 'x', 'y', 'z'}
	var item []byte
	item = protowire.AppendTag(item, itemFieldMetadata, protowire.BytesType)
	item = protowire.AppendBytes(item, buildMetadata("T", ""))
	item = protowire.AppendTag(item, 3, protowire.BytesType) // platform_specific
	item = protowire.AppendBytes(item, platform)
	got, err := Project(item)
	require.NoError(t, err)
	assert.Equal(t, []byte("T"), got["title"])
	_, ok := got["xyz"]
	assert.False(t, ok)
}
