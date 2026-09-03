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

// Package codec decodes the Proton Pass item_v1 protobuf content and projects
// it into a flat label->value map for ExternalSecret consumption.
package codec

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
)

// Item message field numbers (item_v1.proto -> message Item).
const (
	itemFieldMetadata   = 1 // Metadata sub-message
	itemFieldContent    = 2 // Content sub-message (holds the oneof)
	itemFieldExtraField = 4 // repeated ExtraField
)

// Content oneof member field numbers (message Content).
const (
	contentFieldNote  = 2 // ItemNote (secure note; empty)
	contentFieldLogin = 3 // ItemLogin
)

// Metadata sub-message field numbers (message Metadata).
const (
	metaFieldName = 1
	metaFieldNote = 2
)

// ItemLogin sub-message field numbers (message ItemLogin).
const (
	loginFieldEmail    = 1 // item_email
	loginFieldPassword = 2 // password
	loginFieldURLs     = 3 // repeated urls
	loginFieldTOTP     = 4 // totp_uri; the seed is not exposed
	loginFieldUsername = 6 // item_username
)

// ExtraField message field numbers (message ExtraField).
const (
	extraFieldName   = 1
	extraFieldText   = 3 // ExtraTextField{content=1}
	extraFieldHidden = 4 // ExtraHiddenField{content=1}
)

// Project decodes item_v1 content and returns the projected label->value map.
func Project(content []byte) (map[string][]byte, error) {
	result := make(map[string][]byte)

	var metadata, contentMsg []byte
	var extraFields [][]byte

	b := content
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, fmt.Errorf("codec: invalid tag: %w", protowire.ParseError(n))
		}
		b = b[n:]
		switch int32(num) {
		case itemFieldMetadata:
			if typ == protowire.BytesType {
				v, n := protowire.ConsumeBytes(b)
				if n < 0 {
					return nil, fmt.Errorf("codec: invalid metadata: %w", protowire.ParseError(n))
				}
				metadata = v
				b = b[n:]
				continue
			}
		case itemFieldContent:
			if typ == protowire.BytesType {
				v, n := protowire.ConsumeBytes(b)
				if n < 0 {
					return nil, fmt.Errorf("codec: invalid content: %w", protowire.ParseError(n))
				}
				contentMsg = v
				b = b[n:]
				continue
			}
		case itemFieldExtraField:
			if typ == protowire.BytesType {
				v, n := protowire.ConsumeBytes(b)
				if n < 0 {
					return nil, fmt.Errorf("codec: invalid extra field: %w", protowire.ParseError(n))
				}
				extraFields = append(extraFields, v)
				b = b[n:]
				continue
			}
		}
		// Skip unknown fields (platform_specific=3, etc.).
		n = protowire.ConsumeFieldValue(num, typ, b)
		if n < 0 {
			return nil, fmt.Errorf("codec: invalid field: %w", protowire.ParseError(n))
		}
		b = b[n:]
	}

	if len(metadata) > 0 {
		projectMetadata(metadata, result)
	}
	if len(contentMsg) > 0 {
		projectContent(contentMsg, result)
	}
	for _, ef := range extraFields {
		projectExtraField(ef, result)
	}

	return result, nil
}

// projectContent parses the Content oneof and dispatches the typed item.
func projectContent(b []byte, result map[string][]byte) {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return
		}
		b = b[n:]
		if typ != protowire.BytesType {
			n = protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return
			}
			b = b[n:]
			continue
		}
		v, n := protowire.ConsumeBytes(b)
		if n < 0 {
			return
		}
		switch int32(num) {
		case contentFieldLogin:
			projectLogin(v, result)
		case contentFieldNote:
			// ItemNote is empty; the note text lives in Metadata.note.
		}
		b = b[n:]
	}
}

func projectMetadata(b []byte, result map[string][]byte) {
	var name, note []byte
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return
		}
		b = b[n:]
		switch int32(num) {
		case metaFieldName:
			if typ == protowire.BytesType {
				if v, n := protowire.ConsumeBytes(b); n >= 0 {
					name = v
					b = b[n:]
					continue
				}
			}
		case metaFieldNote:
			if typ == protowire.BytesType {
				if v, n := protowire.ConsumeBytes(b); n >= 0 {
					note = v
					b = b[n:]
					continue
				}
			}
		}
		n = protowire.ConsumeFieldValue(num, typ, b)
		if n < 0 {
			return
		}
		b = b[n:]
	}
	if len(name) > 0 {
		result["title"] = name
	}
	if len(note) > 0 {
		result["note"] = note
	}
}

func projectLogin(b []byte, result map[string][]byte) {
	var email, username, password []byte
	var urls [][]byte
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return
		}
		b = b[n:]
		switch int32(num) {
		case loginFieldEmail:
			if typ == protowire.BytesType {
				if v, n := protowire.ConsumeBytes(b); n >= 0 {
					email = v
					b = b[n:]
					continue
				}
			}
		case loginFieldPassword:
			if typ == protowire.BytesType {
				if v, n := protowire.ConsumeBytes(b); n >= 0 {
					password = v
					b = b[n:]
					continue
				}
			}
		case loginFieldURLs:
			if typ == protowire.BytesType {
				if v, n := protowire.ConsumeBytes(b); n >= 0 {
					urls = append(urls, v)
					b = b[n:]
					continue
				}
			}
		case loginFieldUsername:
			if typ == protowire.BytesType {
				if v, n := protowire.ConsumeBytes(b); n >= 0 {
					username = v
					b = b[n:]
					continue
				}
			}
		case loginFieldTOTP:
			// totp_uri holds the long-lived seed; never surface it.
		}
		n = protowire.ConsumeFieldValue(num, typ, b)
		if n < 0 {
			return
		}
		b = b[n:]
	}
	// Typed/built-in fields are listed only when non-empty.
	if len(username) > 0 {
		result["username"] = username
	}
	if len(password) > 0 {
		result["password"] = password
	}
	if len(email) > 0 {
		result["email"] = email
	}
	if len(urls) > 0 {
		result["url"] = joinStrings(urls)
	}
}

func projectExtraField(b []byte, result map[string][]byte) {
	var name []byte
	var value []byte
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return
		}
		b = b[n:]
		switch int32(num) {
		case extraFieldName:
			if typ == protowire.BytesType {
				if v, n := protowire.ConsumeBytes(b); n >= 0 {
					name = v
					b = b[n:]
					continue
				}
			}
		case extraFieldText, extraFieldHidden:
			if typ == protowire.BytesType {
				if sub, n := protowire.ConsumeBytes(b); n >= 0 {
					value = extraContent(sub)
					b = b[n:]
					continue
				}
			}
		}
		n = protowire.ConsumeFieldValue(num, typ, b)
		if n < 0 {
			return
		}
		b = b[n:]
	}
	// Extra fields are surfaced unconditionally when the name is present.
	if len(name) > 0 {
		result[string(name)] = value
	}
}

// extraContent reads field 1 (content) of a text/hidden extra field.
func extraContent(b []byte) []byte {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil
		}
		b = b[n:]
		if num == 1 && typ == protowire.BytesType {
			if v, n := protowire.ConsumeBytes(b); n >= 0 {
				return v
			}
		}
		n = protowire.ConsumeFieldValue(num, typ, b)
		if n < 0 {
			return nil
		}
		b = b[n:]
	}
	return nil
}

func joinStrings(values [][]byte) []byte {
	var parts []string
	for _, v := range values {
		s := string(v)
		if s != "" {
			parts = append(parts, s)
		}
	}
	return []byte(strings.Join(parts, ","))
}
