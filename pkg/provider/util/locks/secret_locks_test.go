//Copyright External Secrets Inc. All Rights Reserved

package locks

import (
	"strings"
	"testing"
)

func TestTryLock(t *testing.T) {
	t.Parallel()

	providerName := "test-provider"
	secretName := "test-secret"

	tests := []struct {
		desc       string
		preprocess func() chan error
		expected   string
	}{
		{
			desc: "No conflict occurs and hold lock successfully",
			preprocess: func() chan error {
				ch := make(chan error)
				go func() {
					ch <- nil
				}()
				return ch
			},
			expected: "",
		},
		{
			desc: "Conflict occurs and cannot hold lock",
			preprocess: func() chan error {
				ch := make(chan error)
				go func() {
					_, err := TryLock(providerName, secretName)
					ch <- err
				}()
				return ch
			},
			expected: "failed to acquire lock: provider: test-provider, secret: test-secret: unable to access secret since it is locked",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			// Evacuate the sharedLocks temporarily
			tmp := sharedLocks
			sharedLocks = &secretLocks{}
			defer func() {
				sharedLocks = tmp
			}()

			ch := tc.preprocess()

			err := <-ch
			if err != nil {
				t.Fatalf("preprocessing failed: %v", err)
			}

			_, got := TryLock(providerName, secretName)
			if got != nil {
				if tc.expected == "" {
					t.Fatalf("received an unexpected error: %v", got)
				}

				if !strings.Contains(got.Error(), tc.expected) {
					t.Fatalf("error %q is supposed to contain %q", got, tc.expected)
				}
				return
			}

			if tc.expected != "" {
				t.Fatal("expected to receive an error but got nil")
			}
		})
	}
}
