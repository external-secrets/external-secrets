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

package keepersecurity

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	ksm "github.com/keeper-security/secrets-manager-go/core"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/keepersecurity/fake"
)

func enhancementsRecords(n int) []*ksm.Record {
	recs := make([]*ksm.Record, n)
	for i := 0; i < n; i++ {
		recs[i] = &ksm.Record{Uid: fmt.Sprintf("uid-%d", i)}
	}
	return recs
}

// countingMock counts GetSecrets calls and optionally throttles the first K.
func countingMock(counter *int64, recs []*ksm.Record, throttleFirst int) *fake.MockKeeperClient {
	return &fake.MockKeeperClient{
		GetSecretsFn: func(filter []string) ([]*ksm.Record, error) {
			n := atomic.AddInt64(counter, 1)
			if int(n) <= throttleFirst {
				return nil, fmt.Errorf("unable to find secret. Error: POST Error: Error: throttled, message=throttled")
			}
			if len(filter) == 0 {
				return recs, nil
			}
			var out []*ksm.Record
			for _, r := range recs {
				for _, f := range filter {
					if r.Uid == f {
						out = append(out, r)
					}
				}
			}
			return out, nil
		},
	}
}

// The shared record cache collapses a reconcile wave of N lookups to one backend call.
func TestRecordCacheCollapsesCalls(t *testing.T) {
	const n = 50

	t.Run("uncached default = one call per lookup", func(t *testing.T) {
		resetRecordCache()
		var calls int64
		c := &Client{ksmClient: countingMock(&calls, enhancementsRecords(100), 0), folderID: "fA"}
		for i := 0; i < n; i++ {
			if _, err := c.findSecretByID(context.Background(), fmt.Sprintf("uid-%d", i)); err != nil {
				t.Fatal(err)
			}
		}
		if got := atomic.LoadInt64(&calls); got != n {
			t.Fatalf("uncached: got %d backend calls, want %d", got, n)
		}
	})

	t.Run("cached = single call for the whole wave", func(t *testing.T) {
		resetRecordCache()
		t.Setenv("KEEPER_RECORD_CACHE_TTL_MS", "60000")
		var calls int64
		c := &Client{ksmClient: countingMock(&calls, enhancementsRecords(100), 0), folderID: "fB"}
		for i := 0; i < n; i++ {
			if _, err := c.findSecretByID(context.Background(), fmt.Sprintf("uid-%d", i)); err != nil {
				t.Fatal(err)
			}
		}
		if got := atomic.LoadInt64(&calls); got != 1 {
			t.Fatalf("cached: got %d backend calls, want 1", got)
		}
	})
}

// A transient throttle/429 is retried rather than surfaced as a failure.
func TestThrottleRetryRecovers(t *testing.T) {
	resetRecordCache()
	orig := waitFn
	waitFn = func(context.Context, time.Duration) error { return nil }
	defer func() { waitFn = orig }()

	var calls int64
	c := &Client{ksmClient: countingMock(&calls, enhancementsRecords(10), 2), folderID: "fC"}
	rec, err := c.findSecretByID(context.Background(), "uid-1")
	if err != nil {
		t.Fatalf("expected recovery after throttles, got error: %v", err)
	}
	if rec == nil || rec.Uid != "uid-1" {
		t.Fatalf("expected uid-1 after retry, got %v", rec)
	}
	if got := atomic.LoadInt64(&calls); got != 3 {
		t.Fatalf("expected 3 calls (2 throttled + 1 success), got %d", got)
	}
}

// A misconfigured KEEPER_THROTTLE_RETRY_ATTEMPTS must not turn the retry loop
// into a zero-attempt no-op (which would make reads return empty success).
func TestRetryAttemptsClampedToOne(t *testing.T) {
	t.Setenv("KEEPER_THROTTLE_RETRY_ATTEMPTS", "0")
	resetRecordCache()
	var calls int64
	c := &Client{ksmClient: countingMock(&calls, enhancementsRecords(2), 0), folderID: "fClamp"}
	rec, err := c.findSecretByID(context.Background(), "uid-0")
	if err != nil {
		t.Fatal(err)
	}
	if rec == nil {
		t.Fatal("expected a record, got nil (loop short-circuited to zero attempts)")
	}
	if got := atomic.LoadInt64(&calls); got != 1 {
		t.Fatalf("attempts=0 env should clamp to 1 backend call, got %d", got)
	}
}

// dataFrom.find.path folder-path resolution and prefix matching.
func TestFolderTreePathMatching(t *testing.T) {
	tree := buildFolderTree([]*ksm.KeeperFolder{
		{FolderUid: "a", Name: "Production"},
		{FolderUid: "b", ParentUid: "a", Name: "Databases"},
		{FolderUid: "c", ParentUid: "b", Name: "MySQL"},
	})
	for uid, want := range map[string]string{
		"a": "Production", "b": "Production/Databases", "c": "Production/Databases/MySQL", "missing": "",
	} {
		if got := tree.pathOf(uid); got != want {
			t.Errorf("pathOf(%q)=%q want %q", uid, got, want)
		}
	}
	cases := []struct {
		path, want string
		match      bool
	}{
		{"Production/Databases/MySQL", "Production/Databases", true},
		{"Production/Web", "Production/Databases", false},
		{"Production/Databases", "Production/Databases", true},
		{"anything", "", true},
	}
	for _, tc := range cases {
		if got := pathMatchesPrefix(tc.path, tc.want); got != tc.match {
			t.Errorf("pathMatchesPrefix(%q,%q)=%v want %v", tc.path, tc.want, got, tc.match)
		}
	}
}

// find is best-effort: a record that can't be represented is skipped, not fatal.
func TestFindSkipsUnparseableRecords(t *testing.T) {
	resetRecordCache()
	c := &Client{ksmClient: &fake.MockKeeperClient{
		GetSecretsFn: func([]string) ([]*ksm.Record, error) {
			return []*ksm.Record{{Uid: "bad-no-rawjson"}}, nil // empty RawJson -> getValidKeeperSecret fails
		},
	}, folderID: "fBE"}
	got, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: ".*"}})
	if err != nil {
		t.Fatalf("find must skip unparseable records, not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 records (bad record skipped), got %d", len(got))
	}
}

// Keeper Notation is detected and resolved against the record set.
func TestKeeperNotation(t *testing.T) {
	for _, k := range []string{"keeper://abc/field/password", "keeper://abc/custom_field/token", "keeper://abc/file/cert.pem"} {
		if !isKeeperNotation(k) {
			t.Errorf("expected %q to be detected as notation", k)
		}
	}
	// Without the keeper:// prefix, keys are treated as plain UID/title (no notation).
	for _, k := range []string{"abc/field/login", "P6RKlo32gu4IA5ZZrPvLXg", "my-record-title"} {
		if isKeeperNotation(k) {
			t.Errorf("did not expect %q to be detected as notation", k)
		}
	}

	resetRecordCache()
	c := &Client{ksmClient: &fake.MockKeeperClient{
		GetSecretsFn:   func([]string) ([]*ksm.Record, error) { return enhancementsRecords(3), nil },
		FindNotationFn: func(_ []*ksm.Record, _ string) ([]interface{}, error) { return []interface{}{"s3cr3t"}, nil },
	}, folderID: "fN"}
	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "keeper://abc/field/password"})
	if err != nil {
		t.Fatalf("GetSecret(notation): %v", err)
	}
	if string(got) != "s3cr3t" {
		t.Fatalf("notation value = %q, want %q", string(got), "s3cr3t")
	}
}

// Validate performs a real read: Ready on success, Error on auth/config failure,
// Unknown on a transient throttle (so the store is not hard-failed).
func TestValidate(t *testing.T) {
	resetRecordCache()
	orig := waitFn
	waitFn = func(context.Context, time.Duration) error { return nil }
	defer func() { waitFn = orig }()

	tests := []struct {
		name    string
		mock    *fake.MockKeeperClient
		want    esv1.ValidationResult
		wantErr bool
	}{
		{
			name:    "ready",
			mock:    &fake.MockKeeperClient{GetSecretsFn: func([]string) ([]*ksm.Record, error) { return enhancementsRecords(1), nil }},
			want:    esv1.ValidationResultReady,
			wantErr: false,
		},
		{
			name:    "config error",
			mock:    &fake.MockKeeperClient{GetSecretsFn: func([]string) ([]*ksm.Record, error) { return nil, fmt.Errorf("invalid configuration") }},
			want:    esv1.ValidationResultError,
			wantErr: true,
		},
		{
			name: "throttle -> unknown",
			mock: &fake.MockKeeperClient{GetSecretsFn: func([]string) ([]*ksm.Record, error) {
				return nil, fmt.Errorf("POST Error: Error: throttled, message=throttled")
			}},
			want:    esv1.ValidationResultUnknown,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRecordCache()
			c := &Client{ksmClient: tt.mock, folderID: "fV-" + tt.name}
			got, err := c.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err=%v wantErr=%v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
