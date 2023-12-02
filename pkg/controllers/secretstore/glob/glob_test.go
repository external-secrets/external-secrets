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

package glob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func Test_Match(t *testing.T) {
	log := ctrl.Log.WithName("controllers").WithName("ClusterSecretStore")
	tests := []struct {
		name    string
		input   string
		pattern string
		result  bool
	}{
		{"Exact match", "hello", "hello", true},
		{"Non-match exact", "hello", "hell", false},
		{"Long glob match", "hello", "hell*", true},
		{"Short glob match", "hello", "h*", true},
		{"Glob non-match", "hello", "e*", false},
		{"Invalid pattern", "e[[a*", "e[[a*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Match(log, tt.pattern, tt.input)
			assert.Equal(t, tt.result, res)
		})
	}
}

func Test_MatchList(t *testing.T) {
	log := ctrl.Log.WithName("controllers").WithName("ClusterSecretStore")
	tests := []struct {
		name   string
		input  string
		list   []string
		result bool
	}{
		{"Glob name in list with simple wildcard", "test", []string{"*"}, true},
		{"Glob name in list with simple wildcard 2", "test", []string{"hello", "*"}, true},
		{"Glob name with prefix", "test", []string{"tes*"}, true},
		{"Glob name in list with with prefix", "test", []string{"he*", "tes*"}, true},
		{"No match", "test", []string{"he*", "ho*"}, false},
		{"No match", "test", []string{"hello", "bonjour"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := MatchStringInList(log, tt.list, tt.input)
			assert.Equal(t, tt.result, res)
		})
	}
}
