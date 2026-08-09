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

// Package decoding provides helpers for decoding ExternalSecret values.
package decoding

import (
	"encoding/base64"
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// DecodeMap decodes values from a secretMap.
func DecodeMap(strategy esv1.ExternalSecretDecodingStrategy, in map[string][]byte) (map[string][]byte, error) {
	out := make(map[string][]byte, len(in))
	for k, v := range in {
		val, err := Decode(strategy, v)
		if err != nil {
			return nil, fmt.Errorf("failure decoding key %v: %w", k, err)
		}
		out[k] = val
	}
	return out, nil
}

// Decode decodes the input byte slice according to the provided decoding strategy.
func Decode(strategy esv1.ExternalSecretDecodingStrategy, in []byte) ([]byte, error) {
	switch strategy {
	case esv1.ExternalSecretDecodeBase64:
		out, err := base64.StdEncoding.DecodeString(string(in))
		if err != nil {
			return nil, err
		}
		return out, nil
	case esv1.ExternalSecretDecodeBase64URL:
		out, err := base64.URLEncoding.DecodeString(string(in))
		if err != nil {
			return nil, err
		}
		return out, nil
	case esv1.ExternalSecretDecodeNone:
		return in, nil
	// default when stored version is v1alpha1
	case "":
		return in, nil
	case esv1.ExternalSecretDecodeAuto:
		out, err := Decode(esv1.ExternalSecretDecodeBase64, in)
		if err != nil {
			out, err := Decode(esv1.ExternalSecretDecodeBase64URL, in)
			if err != nil {
				return Decode(esv1.ExternalSecretDecodeNone, in)
			}
			return out, nil
		}
		return out, nil
	default:
		return nil, fmt.Errorf("decoding strategy %v is not supported", strategy)
	}
}
