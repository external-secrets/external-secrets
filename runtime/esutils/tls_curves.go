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

package esutils

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ParseCurvePreferences converts curve names to tls.CurveID values, preserving order.
// An empty slice returns (nil, nil) so tls.Config uses Go defaults.
//
// Each entry may be a well-known name matching crypto/tls constants (for example
// X25519, CurveP256, CurveP384, CurveP521) or a decimal tls.CurveID as supported
// by the Go toolchain (for example hybrid post-quantum groups when available).
func ParseCurvePreferences(names []string) ([]tls.CurveID, error) {
	if len(names) == 0 {
		return nil, nil
	}
	out := make([]tls.CurveID, 0, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			return nil, fmt.Errorf("empty curve preference entry")
		}
		id, err := parseCurveID(name)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func parseCurveID(name string) (tls.CurveID, error) {
	if isAllDecimal(name) {
		u, err := strconv.ParseUint(name, 10, 16)
		if err != nil {
			return 0, fmt.Errorf("invalid tls curve id %q: %w", name, err)
		}
		return tls.CurveID(u), nil
	}

	switch name {
	case "X25519":
		return tls.X25519, nil
	case "CurveP256", "P-256", "P256":
		return tls.CurveP256, nil
	case "CurveP384", "P-384", "P384":
		return tls.CurveP384, nil
	case "CurveP521", "P-521", "P521":
		return tls.CurveP521, nil
	default:
		return 0, fmt.Errorf("unknown tls curve preference %q (use a name like X25519 or a decimal CurveID)", name)
	}
}

func isAllDecimal(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
