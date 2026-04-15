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

package delinea

import (
	"reflect"
	"testing"
)

func TestMissingRequiredEnvFromConfig(t *testing.T) {
	t.Parallel()

	cfg := config{}
	want := []string{
		"DELINEA_TENANT",
		"DELINEA_CLIENT_ID",
		"DELINEA_CLIENT_SECRET",
	}

	if got := missingRequiredEnvFromConfig(cfg); !reflect.DeepEqual(got, want) {
		t.Fatalf("missingRequiredEnvFromConfig() = %v, want %v", got, want)
	}
}

func TestMissingRequiredEnvFromConfigComplete(t *testing.T) {
	t.Parallel()

	cfg := config{
		tenant:       "tenant",
		clientID:     "client",
		clientSecret: "secret",
	}

	if got := missingRequiredEnvFromConfig(cfg); len(got) != 0 {
		t.Fatalf("missingRequiredEnvFromConfig() = %v, want none", got)
	}
}
