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

package safeguard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAccountSystemFilter(t *testing.T) {
	filter := buildAccountSystemFilter("dbadmin", "database-server")
	assert.Equal(t, "AccountName ieq 'dbadmin' and SystemName ieq 'database-server'", filter)
}

func TestBuildAccountSystemFilterEscapesQuotes(t *testing.T) {
	filter := buildAccountSystemFilter("o'reilly", "prod'db")
	assert.Equal(t, "AccountName ieq 'o''reilly' and SystemName ieq 'prod''db'", filter)
}

func TestParseLookupKeyFilter(t *testing.T) {
	opts, isDirect, _, err := parseLookupKey("filter:AccountName ieq 'dbadmin' and SystemName ieq 'database-server'")
	require.NoError(t, err)
	assert.False(t, isDirect)
	assert.Equal(t, "AccountName ieq 'dbadmin' and SystemName ieq 'database-server'", opts.filter)
}

func TestParseLookupKeyAccountLookup(t *testing.T) {
	opts, isDirect, _, err := parseLookupKey("dbadmin/database-server")
	require.NoError(t, err)
	assert.False(t, isDirect)
	assert.Equal(t, buildAccountSystemFilter("dbadmin", "database-server"), opts.filter)
}

func TestParseLookupKeyDirectAPIKey(t *testing.T) {
	_, isDirect, directKey, err := parseLookupKey("my-api-key-value")
	require.NoError(t, err)
	assert.True(t, isDirect)
	assert.Equal(t, "my-api-key-value", directKey)
}
