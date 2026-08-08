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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCurvePreferences(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    []tls.CurveID
		wantErr bool
	}{
		{
			name:  "well-known curve names",
			input: []string{"X25519", "CurveP256"},
			want:  []tls.CurveID{tls.X25519, tls.CurveP256},
		},
		{
			name:  "curve name aliases",
			input: []string{"P-256", "P384"},
			want:  []tls.CurveID{tls.CurveP256, tls.CurveP384},
		},
		{
			name:  "whitespace is trimmed",
			input: []string{" X25519 ", "CurveP256"},
			want:  []tls.CurveID{tls.X25519, tls.CurveP256},
		},
		{
			name:  "nil input uses Go defaults",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty input uses Go defaults",
			input: []string{},
			want:  nil,
		},
		{
			name:    "unknown curve name",
			input:   []string{"not-a-curve"},
			wantErr: true,
		},
		{
			name:    "empty curve entry",
			input:   []string{"X25519", ""},
			wantErr: true,
		},
		{
			name:  "decimal curve ID",
			input: []string{"29"},
			want:  []tls.CurveID{tls.X25519},
		},
		{
			name:  "all four standard curves",
			input: []string{"X25519", "CurveP256", "CurveP384", "CurveP521"},
			want:  []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384, tls.CurveP521},
		},
		{
			name:  "P-521 alias",
			input: []string{"P-521"},
			want:  []tls.CurveID{tls.CurveP521},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCurvePreferences(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
