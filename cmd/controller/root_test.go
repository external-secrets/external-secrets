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

package controller

import "testing"

func TestStoreRequeueIntervalDefault(t *testing.T) {
	flag := rootCmd.Flags().Lookup("store-requeue-interval")
	if flag == nil {
		t.Fatal("store-requeue-interval flag not found")
	}

	if flag.DefValue != "30s" {
		t.Fatalf("expected store-requeue-interval default 30s, got %q", flag.DefValue)
	}
}
