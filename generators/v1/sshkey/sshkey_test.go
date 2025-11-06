/*
Copyright © 2025 ESO Maintainer Team

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

package sshkey

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

func TestGenerate(t *testing.T) {
	g := &Generator{}

	tests := []struct {
		name        string
		jsonSpec    *apiextensions.JSON
		wantErr     bool
		expectedErr string
		validate    func(t *testing.T, result map[string][]byte)
	}{
		{
			name:        "nil spec should return error",
			jsonSpec:    nil,
			wantErr:     true,
			expectedErr: errNoSpec,
		},
		{
			name:     "empty spec should use defaults",
			jsonSpec: &apiextensions.JSON{Raw: []byte(`{"spec":{}}`)},
			wantErr:  false,
			validate: func(t *testing.T, result map[string][]byte) {
				assert.Contains(t, result, "privateKey")
				assert.Contains(t, result, "publicKey")
				assert.True(t, len(result["privateKey"]) > 0)
				assert.True(t, len(result["publicKey"]) > 0)
				// Should contain RSA private key header
				assert.Contains(t, string(result["privateKey"]), "BEGIN OPENSSH PRIVATE KEY")
				// Should contain ssh-rsa public key
				assert.True(t, strings.HasPrefix(string(result["publicKey"]), "ssh-rsa "))
			},
		},
		{
			name:     "rsa key with custom size",
			jsonSpec: &apiextensions.JSON{Raw: []byte(`{"spec":{"keyType":"rsa","keySize":4096}}`)},
			wantErr:  false,
			validate: func(t *testing.T, result map[string][]byte) {
				assert.Contains(t, result, "privateKey")
				assert.Contains(t, result, "publicKey")
				assert.True(t, len(result["privateKey"]) > 0)
				assert.True(t, len(result["publicKey"]) > 0)
			},
		},
		{
			name:     "ed25519 key",
			jsonSpec: &apiextensions.JSON{Raw: []byte(`{"spec":{"keyType":"ed25519"}}`)},
			wantErr:  false,
			validate: func(t *testing.T, result map[string][]byte) {
				assert.Contains(t, result, "privateKey")
				assert.Contains(t, result, "publicKey")
				assert.True(t, len(result["privateKey"]) > 0)
				assert.True(t, len(result["publicKey"]) > 0)
				// Should contain ed25519 public key
				assert.True(t, strings.HasPrefix(string(result["publicKey"]), "ssh-ed25519 "))
			},
		},
		{
			name:     "ed25519 key with explicit keySize (should be ignored)",
			jsonSpec: &apiextensions.JSON{Raw: []byte(`{"spec":{"keyType":"ed25519","keySize":4096}}`)},
			wantErr:  false,
			validate: func(t *testing.T, result map[string][]byte) {
				assert.Contains(t, result, "privateKey")
				assert.Contains(t, result, "publicKey")
				assert.True(t, len(result["privateKey"]) > 0)
				assert.True(t, len(result["publicKey"]) > 0)
				// Should contain ed25519 public key (keySize should be ignored)
				assert.True(t, strings.HasPrefix(string(result["publicKey"]), "ssh-ed25519 "))
			},
		},
		{
			name:     "key with comment",
			jsonSpec: &apiextensions.JSON{Raw: []byte(`{"spec":{"keyType":"rsa","comment":"test@example.com"}}`)},
			wantErr:  false,
			validate: func(t *testing.T, result map[string][]byte) {
				assert.Contains(t, result, "privateKey")
				assert.Contains(t, result, "publicKey")
				// Should contain the comment in public key
				assert.Contains(t, string(result["publicKey"]), "test@example.com")
			},
		},
		{
			name:        "unsupported key type",
			jsonSpec:    &apiextensions.JSON{Raw: []byte(`{"spec":{"keyType":"unsupported"}}`)},
			wantErr:     true,
			expectedErr: "unsupported key type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := g.Generate(context.Background(), tt.jsonSpec, nil, "")

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
				return
			}

			assert.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestCleanup(t *testing.T) {
	g := &Generator{}
	err := g.Cleanup(context.Background(), nil, nil, nil, "")
	assert.NoError(t, err)
}

func TestParseSpec(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected *genv1alpha1.SSHKey
		wantErr  bool
	}{
		{
			name: "valid spec",
			data: []byte(`{"spec":{"keyType":"rsa","keySize":2048,"comment":"test"}}`),
			expected: &genv1alpha1.SSHKey{
				Spec: genv1alpha1.SSHKeySpec{
					KeyType: "rsa",
					KeySize: func() *int { i := 2048; return &i }(),
					Comment: "test",
				},
			},
			wantErr: false,
		},
		{
			name:    "empty spec",
			data:    []byte(`{"spec":{}}`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSpec(tt.data)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.expected != nil {
				assert.Equal(t, tt.expected.Spec.KeyType, result.Spec.KeyType)
				assert.Equal(t, tt.expected.Spec.KeySize, result.Spec.KeySize)
				assert.Equal(t, tt.expected.Spec.Comment, result.Spec.Comment)
			}
		})
	}
}
