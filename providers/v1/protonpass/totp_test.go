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
	"encoding/base32"
	"errors"
	"testing"
	"time"
)

// rfc6238Secret is the 20-byte seed from RFC 6238 Appendix B (SHA-1), base32-encoded.
func rfc6238Secret() string {
	return base32.StdEncoding.EncodeToString([]byte("12345678912345678912"))
}

func TestTOTPCodeAtKnownVectors(t *testing.T) {
	secret := rfc6238Secret()
	when := time.Date(1970, 1, 1, 0, 0, 59, 0, time.UTC)

	// digits=6 (default) and digits=8, against the same vectors runtime/otp pins.
	got6, err := totpCodeAt("otpauth://totp/Example?secret="+secret+"&algorithm=SHA1&digits=6&period=30", when)
	if err != nil {
		t.Fatalf("digits=6: %v", err)
	}
	if got6 != "016480" {
		t.Errorf("digits=6 code = %q, want %q", got6, "016480")
	}

	got8, err := totpCodeAt("otpauth://totp/Example?secret="+secret+"&digits=8&period=30", when)
	if err != nil {
		t.Fatalf("digits=8: %v", err)
	}
	if got8 != "71016480" {
		t.Errorf("digits=8 code = %q, want %q", got8, "71016480")
	}
}

func TestTOTPCodeErrors(t *testing.T) {
	now := time.Now()
	if _, err := totpCodeAt("", now); !errors.Is(err, errNoTOTP) {
		t.Errorf("empty uri: want errNoTOTP, got %v", err)
	}
	if _, err := totpCodeAt("https://example.com", now); err == nil {
		t.Error("non-otpauth scheme should error")
	}
	if _, err := totpCodeAt("otpauth://totp/x?digits=6", now); err == nil {
		t.Error("missing secret should error")
	}
}
