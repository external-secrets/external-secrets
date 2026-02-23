/*
Copyright © 2026 ESO Maintainer Team

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
	"github.com/external-secrets/external-secrets/runtime/otp"
)

// generateTOTP generates a TOTP code from a base32-encoded secret.
// It delegates to the shared runtime/otp package using RFC 6238 defaults
// (SHA1, 6 digits, 30-second time step).
func generateTOTP(secret string) (string, error) {
	code, _, err := otp.GenerateCode(otp.WithToken(secret))
	return code, err
}
