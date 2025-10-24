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

package acr

import (
	"testing"
)

func TestValidateACRRegistry(t *testing.T) {
	tests := []struct {
		name      string
		registry  string
		wantError bool
	}{
		{
			name:      "valid registry - .io",
			registry:  "myregistry.azurecr.io",
			wantError: false,
		},
		{
			name:      "valid registry - .cn",
			registry:  "myregistry.azurecr.cn",
			wantError: false,
		},
		{
			name:      "valid registry - .de",
			registry:  "myregistry.azurecr.de",
			wantError: false,
		},
		{
			name:      "valid registry - .us",
			registry:  "myregistry.azurecr.us",
			wantError: false,
		},
		{
			name:      "valid registry - with dashes",
			registry:  "my-registry-name.azurecr.io",
			wantError: false,
		},
		{
			name:      "invalid registry - empty",
			registry:  "",
			wantError: true,
		},
		{
			name:      "invalid registry - SSRF attempt localhost",
			registry:  "localhost",
			wantError: true,
		},
		{
			name:      "invalid registry - SSRF attempt IP",
			registry:  "192.168.1.1",
			wantError: true,
		},
		{
			name:      "invalid registry - wrong domain",
			registry:  "myregistry.example.com",
			wantError: true,
		},
		{
			name:      "invalid registry - wrong TLD",
			registry:  "myregistry.azurecr.com",
			wantError: true,
		},
		{
			name:      "invalid registry - missing domain",
			registry:  "azurecr.io",
			wantError: true,
		},
		{
			name:      "invalid registry - uppercase",
			registry:  "MyRegistry.azurecr.io",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateACRRegistry(tt.registry)
			if tt.wantError && err == nil {
				t.Errorf("validateACRRegistry() expected error but got none for registry: %s", tt.registry)
			}
			if !tt.wantError && err != nil {
				t.Errorf("validateACRRegistry() unexpected error: %v for registry: %s", err, tt.registry)
			}
		})
	}
}
