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

package hex

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

func TestGenerate(t *testing.T) {
	g := &Generator{}

	tests := []struct {
		name          string
		spec          *genv1alpha1.Hex
		genFunc       generateFunc
		expectedError string
		validate      func(t *testing.T, result []byte)
	}{
		{
			name: "generates hex string with default settings",
			spec: &genv1alpha1.Hex{
				Spec: genv1alpha1.HexSpec{
					Length: 16,
				},
			},
			genFunc: func(length int) ([]byte, error) {
				return []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE}, nil
			},
			validate: func(t *testing.T, result []byte) {
				assert.Equal(t, "deadbeefcafebabe", string(result))
				assert.Len(t, result, 16)
				_, err := hex.DecodeString(string(result))
				assert.NoError(t, err)
			},
		},
		{
			name: "generates uppercase hex string",
			spec: &genv1alpha1.Hex{
				Spec: genv1alpha1.HexSpec{
					Length:    8,
					Uppercase: true,
				},
			},
			genFunc: func(length int) ([]byte, error) {
				return []byte{0xDE, 0xAD, 0xBE, 0xEF}, nil
			},
			validate: func(t *testing.T, result []byte) {
				assert.Equal(t, "DEADBEEF", string(result))
				assert.Len(t, result, 8)
				_, err := hex.DecodeString(string(result))
				assert.NoError(t, err)
			},
		},
		{
			name: "adds prefix",
			spec: &genv1alpha1.Hex{
				Spec: genv1alpha1.HexSpec{
					Length: 8,
					Prefix: "0x",
				},
			},
			genFunc: func(length int) ([]byte, error) {
				return []byte{0xDE, 0xAD, 0xBE, 0xEF}, nil
			},
			validate: func(t *testing.T, result []byte) {
				assert.Equal(t, "0xdeadbeef", string(result))
				assert.Len(t, result, 10)
				_, err := hex.DecodeString(string(result)[2:]) // Skip prefix
				assert.NoError(t, err)
			},
		},
		{
			name: "handles odd length",
			spec: &genv1alpha1.Hex{
				Spec: genv1alpha1.HexSpec{
					Length: 7,
				},
			},
			genFunc: func(length int) ([]byte, error) {
				return []byte{0xDE, 0xAD, 0xBE, 0xEF}, nil
			},
			validate: func(t *testing.T, result []byte) {
				assert.Equal(t, "deadbee", string(result))
				assert.Len(t, result, 7)
			},
		},
		{
			name:          "fails with nil spec",
			spec:          nil,
			expectedError: errNoSpec,
		},
		{
			name: "fails with invalid length",
			spec: &genv1alpha1.Hex{
				Spec: genv1alpha1.HexSpec{
					Length: 0,
				},
			},
			expectedError: "length must be greater than 0",
		},
		{
			name: "handles generator error",
			spec: &genv1alpha1.Hex{
				Spec: genv1alpha1.HexSpec{
					Length: 8,
				},
			},
			genFunc: func(length int) ([]byte, error) {
				return nil, fmt.Errorf("random generation failed")
			},
			expectedError: "unable to generate hex string: random generation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jsonSpec *apiextensions.JSON
			if tt.spec != nil {
				data, err := yaml.Marshal(tt.spec)
				assert.NoError(t, err)
				jsonSpec = &apiextensions.JSON{Raw: data}
			}

			genFunc := tt.genFunc
			if genFunc == nil {
				genFunc = generateSecureBytes
			}

			result, state, err := g.generate(jsonSpec, genFunc)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
				assert.Nil(t, result)
				assert.Nil(t, state)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result, "hex")
			assert.Nil(t, state)

			if tt.validate != nil {
				tt.validate(t, result["hex"])
			}
		})
	}
}
