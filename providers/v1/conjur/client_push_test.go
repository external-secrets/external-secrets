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

package conjur

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPolicy(t *testing.T) {
	policy := conjurPolicy("secret1", []string{"foo", "bar", "baz"})
	expected := `
- !policy
  id: secret1
  body:
  - !group
    id: delegation/consumers
    annotations:
      managed-by: "external-secrets"
      editable: "true"
  - !variable
    id: foo
    annotations:
      managed-by: "external-secrets"
  - !variable
    id: bar
    annotations:
      managed-by: "external-secrets"
  - !variable
    id: baz
    annotations:
      managed-by: "external-secrets"
  - !permit
    resource: !variable foo
    role: !group delegation/consumers
    privileges: [ read, execute ]
  - !permit
    resource: !variable bar
    role: !group delegation/consumers
    privileges: [ read, execute ]
  - !permit
    resource: !variable baz
    role: !group delegation/consumers
    privileges: [ read, execute ]`

	assert.Equal(t, expected, policy)
}
