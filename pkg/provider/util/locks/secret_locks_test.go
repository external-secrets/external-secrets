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
package locks

import "testing"

func TestSecretLocks_TryLock(t *testing.T) {
	t.Parallel()

	secretName := "test-secret"

	tests := []struct {
		desc       string
		preprocess func(locks *SecretLocks) chan bool
		expected   bool
	}{
		{
			desc: "No conflict occurs and hold lock successfully",
			preprocess: func(locks *SecretLocks) chan bool {
				ch := make(chan bool)
				go func() {
					ch <- true
				}()
				return ch
			},
			expected: true,
		},
		{
			desc: "Conflict occurs and cannot hold lock",
			preprocess: func(locks *SecretLocks) chan bool {
				ch := make(chan bool)
				go func() {
					_, ok := locks.TryLock(secretName)
					ch <- ok
				}()
				return ch
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			locks := SecretLocks{}
			ch := tc.preprocess(&locks)

			success := <-ch
			if !success {
				t.Fatal("preprocessing failed")
			}

			if _, got := locks.TryLock(secretName); got != tc.expected {
				t.Fatalf("received an unepceted result: got: %v, expected: %v", got, tc.expected)
			}
		})
	}
}
