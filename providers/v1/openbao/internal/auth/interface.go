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

// Package auth provides a minimal, mockable interface to OpenBao auth methods.
//
// For provider internal use only, API might change at any point in time.
package auth

import (
	"github.com/openbao/openbao/api/v2"
)

// Factory provides a minimal, mockable interface to OpenBao auth methods
// (minimal in a sense that it exposes exactly the parameters that are needed by
// the provider). There are two implementation available:
//
// - [DefaultAuthMethodFactory]
//
// - [MockFactory] a mock for testing.
type Factory interface {
	UserPass(username, password, mount string) (api.AuthMethod, error)
	AppRole(id, secret, mount string) (api.AuthMethod, error)
}
