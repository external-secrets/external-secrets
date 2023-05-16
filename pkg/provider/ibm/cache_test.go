/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package ibm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const maxEntries = 10

func dataCache(t *testing.T, ttl time.Duration, maxEntries int) cacheIntf {
	t.Helper()
	return NewCache(maxEntries, ttl)
}

func TestGetData(t *testing.T) {
	tests := []struct {
		name      string
		ttl       time.Duration
		key       string
		value     []byte
		wantValue []byte
		wantFound bool
	}{
		{
			name:      "object exists in cache and has not expired",
			ttl:       30 * time.Second,
			key:       "testObject",
			value:     []byte("testValue"),
			wantValue: []byte("testValue"),
			wantFound: true,
		},
		{
			name:      "object exists in cache and will never expire",
			ttl:       0 * time.Second,
			key:       "testObject",
			value:     []byte("testValue"),
			wantValue: []byte("testValue"),
			wantFound: true,
		},
		{
			name:      "object exists in cache but has expired",
			ttl:       1 * time.Nanosecond,
			key:       "testObject",
			value:     []byte("testValue"),
			wantFound: false,
		},
		{
			name:      "object not in cache",
			ttl:       30 * time.Second,
			key:       "testObject",
			wantFound: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCache(10, tt.ttl)
			if tt.value != nil {
				c.PutData(tt.key, tt.value)
			}
			gotFound, gotValue := c.GetData(tt.key)
			assert.Equal(t, tt.wantFound, gotFound)
			assert.Equal(t, string(gotValue), string(tt.wantValue))
		})
	}
}

func TestPutData(t *testing.T) {
	t.Parallel()
	d := dataCache(t, time.Minute, maxEntries)
	d.PutData("test-key", []byte("test-value"))
}

func TestDeleteData(t *testing.T) {
	t.Parallel()
	d := dataCache(t, time.Minute, maxEntries)
	d.DeleteData("test-key")
}
