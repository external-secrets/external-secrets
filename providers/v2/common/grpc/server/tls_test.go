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

package server

import (
	"path/filepath"
	"testing"
)

func TestResolveCertPath(t *testing.T) {
	t.Parallel()

	certDir := filepath.Join(string(filepath.Separator), "etc", "provider", "certs")

	tests := []struct {
		name     string
		fileName string
		want     string
		wantErr  bool
	}{
		{
			name:     "accepts simple filename",
			fileName: "ca.crt",
			want:     filepath.Join(certDir, "ca.crt"),
		},
		{
			name:     "rejects nested path",
			fileName: filepath.Join("nested", "ca.crt"),
			wantErr:  true,
		},
		{
			name:     "rejects traversal",
			fileName: filepath.Join("..", "ca.crt"),
			wantErr:  true,
		},
		{
			name:     "rejects absolute path",
			fileName: filepath.Join(string(filepath.Separator), "tmp", "ca.crt"),
			wantErr:  true,
		},
		{
			name:     "rejects windows style nested path",
			fileName: `nested\ca.crt`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveCertPath(certDir, tt.fileName)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveCertPath(%q, %q) error = nil, want error", certDir, tt.fileName)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveCertPath(%q, %q) error = %v", certDir, tt.fileName, err)
			}
			if got != tt.want {
				t.Fatalf("resolveCertPath(%q, %q) = %q, want %q", certDir, tt.fileName, got, tt.want)
			}
		})
	}
}
