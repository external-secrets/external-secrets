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

package controller

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
)

var (
	cipherIDs  = map[string]uint16{}
	versionIDs = map[string]uint16{}
	curveNames = map[string]tls.CurveID{}
)

func init() {
	// See https://pkg.go.dev/crypto/tls#CipherSuites for cipher names.
	for _, cs := range append(tls.CipherSuites(), tls.InsecureCipherSuites()...) {
		cipherIDs[cs.Name] = cs.ID
	}
	// See https://pkg.go.dev/crypto/tls#VersionName for version names.
	for _, v := range []uint16{tls.VersionTLS10, tls.VersionTLS11, tls.VersionTLS12, tls.VersionTLS13} {
		versionIDs[strings.TrimPrefix(tls.VersionName(v), "TLS ")] = v
	}
	// See https://pkg.go.dev/crypto/tls#CurveID for curve names.
	for _, c := range []tls.CurveID{
		tls.CurveP256, tls.CurveP384, tls.CurveP521, tls.X25519,
		tls.X25519MLKEM768, tls.SecP256r1MLKEM768, tls.SecP384r1MLKEM1024,
	} {
		curveNames[c.String()] = c
		curveNames[strconv.FormatUint(uint64(c), 10)] = c
	}
	// Legacy aliases predating crypto/tls's own CurveID.String() naming.
	curveNames["P-256"] = tls.CurveP256
	curveNames["P256"] = tls.CurveP256
	curveNames["P-384"] = tls.CurveP384
	curveNames["P384"] = tls.CurveP384
	curveNames["P-521"] = tls.CurveP521
	curveNames["P521"] = tls.CurveP521
}

// parseCipherSuites converts a comma separated list of TLS cipher suite names
// to their IDs. Returns (nil, nil) for an empty string so tls.Config uses Go
// defaults.
func parseCipherSuites(cipherListString string) ([]uint16, error) {
	if cipherListString == "" {
		return nil, nil
	}
	cipherList := strings.Split(cipherListString, ",")
	ret := make([]uint16, 0, len(cipherList))
	for _, c := range cipherList {
		id, ok := cipherIDs[c]
		if !ok {
			return ret, fmt.Errorf("cipher %s was not found", c)
		}
		ret = append(ret, id)
	}
	return ret, nil
}

// parseTLSVersion converts a human-readable TLS version (for example "1.1")
// to the value accepted by tls.Config (for example 0x0302).
// Returns an error for unrecognized version strings.
func parseTLSVersion(version string) (uint16, error) {
	if v, ok := versionIDs[version]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("unsupported TLS minimum version %q; valid values are 1.0, 1.1, 1.2, 1.3", version)
}

// parseCurvePreferences converts curve names to tls.CurveID values, preserving order.
// An empty slice returns (nil, nil) so tls.Config uses Go defaults.
//
// Each entry may be a well-known name matching crypto/tls constants (for example
// X25519, CurveP256, CurveP384, CurveP521, or the hybrid post-quantum groups) or
// a decimal tls.CurveID as supported by the Go toolchain.
func parseCurvePreferences(names []string) ([]tls.CurveID, error) {
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
	if id, ok := curveNames[name]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("unknown tls curve preference %q (use a name like X25519, CurveP256, CurveP384, CurveP521, or a decimal CurveID)", name)
}

// disableHTTP2 is a TLS configuration function that disables HTTP/2.
func disableHTTP2(cfg *tls.Config) {
	cfg.NextProtos = []string{"http/1.1"}
}

// buildTLSConfigFuncs assembles a slice of tls.Config mutators from the current
// flag values. It is shared across all subcommands (controller, webhook, certcontroller).
func buildTLSConfigFuncs(ciphers, minVer string, curves []string, http2 bool) ([]func(*tls.Config), error) {
	var opts []func(*tls.Config)

	if !http2 {
		opts = append(opts, disableHTTP2)
	}

	var cipherIDs []uint16
	if ciphers != "" {
		ids, err := parseCipherSuites(ciphers)
		if err != nil {
			return nil, fmt.Errorf("unable to parse tls ciphers: %w", err)
		}
		if len(ids) > 0 {
			cipherIDs = ids
			opts = append(opts, func(cfg *tls.Config) {
				cfg.CipherSuites = ids
			})
		}
	}

	var minVersion uint16
	if minVer != "" {
		ver, err := parseTLSVersion(minVer)
		if err != nil {
			return nil, fmt.Errorf("unable to parse tls min version: %w", err)
		}
		minVersion = ver
		opts = append(opts, func(cfg *tls.Config) {
			cfg.MinVersion = ver
		})
	}

	if http2 {
		if err := checkHTTP2Ciphers(cipherIDs, minVersion); err != nil {
			return nil, err
		}
	}

	if len(curves) > 0 {
		filtered := make([]string, 0, len(curves))
		for _, c := range curves {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			filtered = append(filtered, c)
		}
		if len(filtered) > 0 {
			curveIDs, err := parseCurvePreferences(filtered)
			if err != nil {
				return nil, fmt.Errorf("unable to parse tls curve preferences: %w", err)
			}
			if len(curveIDs) > 0 {
				opts = append(opts, func(cfg *tls.Config) {
					cfg.CurvePreferences = curveIDs
				})
			}
		}
	}

	return opts, nil
}

// checkHTTP2Ciphers mirrors net/http's internal HTTP/2 server configuration check:
// an explicit TLS 1.0-1.2 CipherSuites list must include at least one HTTP/2
// mandatory-to-implement (MTI) cipher, or negotiation will fail. TLS 1.3 ignores
// CipherSuites entirely, so an explicit MinVersion of 1.3 skips the check.
func checkHTTP2Ciphers(cipherIDs []uint16, minVersion uint16) error {
	if len(cipherIDs) == 0 || minVersion >= tls.VersionTLS13 {
		return nil
	}
	for _, id := range cipherIDs {
		if id == tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 || id == tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 {
			return nil
		}
	}
	return fmt.Errorf("http2: --tls-ciphers is missing an HTTP/2-required AES_128_GCM_SHA256 cipher " +
		"(need at least one of TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 or TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, " +
		"or set --tls-min-version=1.3)")
}
