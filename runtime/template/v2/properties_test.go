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

package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromProperties(t *testing.T) {
	input := `# database settings
username = admin
connection: jdbc:postgresql://db.example/app?ssl=true
escaped\ key = escaped\ value
message = hello \
    world
placeholder = ${PASSWORD}
symbol = \u263A
`

	actual, err := fromProperties(input)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"connection":  "jdbc:postgresql://db.example/app?ssl=true",
		"escaped key": "escaped value",
		"message":     "hello world",
		"placeholder": "${PASSWORD}",
		"symbol":      "☺",
		"username":    "admin",
	}, actual)
}

func TestFromPropertiesReturnsParseErrors(t *testing.T) {
	_, err := fromProperties(`invalid = \u12`)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unable to parse properties")
	assert.ErrorContains(t, err, "invalid unicode literal")
}
