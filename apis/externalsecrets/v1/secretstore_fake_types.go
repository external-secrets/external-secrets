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

package v1

// FakeProvider configures a fake provider that returns static values.
type FakeProvider struct {
	Data             []FakeProviderData `json:"data"`
	ValidationResult *ValidationResult  `json:"validationResult,omitempty"`
}

// FakeProviderData defines a key-value pair with optional version for the fake provider.
type FakeProviderData struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version string `json:"version,omitempty"`
}
