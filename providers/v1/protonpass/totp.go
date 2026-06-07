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
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/external-secrets/external-secrets/runtime/otp"
)

// errNoTOTP indicates the item has no TOTP configured.
var errNoTOTP = errors.New("protonpass: item has no totp")

// generateTOTPCode parses an otpauth:// URI and returns the current code. The seed
// embedded in the URI is used transiently and never returned or stored.
func generateTOTPCode(uri string) (string, error) {
	return totpCodeAt(uri, time.Now())
}

// totpCodeAt is generateTOTPCode with an explicit clock, for deterministic tests.
func totpCodeAt(uri string, when time.Time) (string, error) {
	if uri == "" {
		return "", errNoTOTP
	}
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("protonpass: parse otpauth uri: %w", err)
	}
	if u.Scheme != "otpauth" {
		return "", fmt.Errorf("protonpass: not an otpauth uri (scheme %q)", u.Scheme)
	}
	q := u.Query()
	secret := q.Get("secret")
	if secret == "" {
		return "", errors.New("protonpass: otpauth uri missing secret")
	}
	opts := []otp.GeneratorOptionsFunc{otp.WithToken(secret), otp.WithWhen(when)}
	if d := q.Get("digits"); d != "" {
		n, err := strconv.Atoi(d)
		if err != nil {
			return "", fmt.Errorf("protonpass: invalid otpauth digits %q: %w", d, err)
		}
		opts = append(opts, otp.WithLength(n))
	}
	if p := q.Get("period"); p != "" {
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return "", fmt.Errorf("protonpass: invalid otpauth period %q: %w", p, err)
		}
		opts = append(opts, otp.WithTimePeriod(n))
	}
	if a := q.Get("algorithm"); a != "" {
		opts = append(opts, otp.WithAlgorithm(strings.ToLower(a)))
	}
	code, _, err := otp.GenerateCode(opts...)
	if err != nil {
		return "", fmt.Errorf("protonpass: generate totp code: %w", err)
	}
	return code, nil
}
