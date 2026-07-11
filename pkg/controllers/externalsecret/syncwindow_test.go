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

package externalsecret

import (
	"testing"
	"time"

	robfigcron "github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func mustParseCron(expr string) robfigcron.Schedule {
	sched, err := cronParser.Parse(expr)
	if err != nil {
		panic(expr + ": " + err.Error())
	}
	return sched
}

// TestIsWithinSyncWindow exercises the half-open / closed window boundaries
// using a daily schedule that fires at 22:00 UTC with a 2-hour duration
// (window open 22:00-00:00 UTC).
func TestIsWithinSyncWindow(t *testing.T) {
	sched := mustParseCron("0 22 * * *")
	dur := 2 * time.Hour

	// Reference firing: 2026-06-01 22:00 UTC (Monday).
	open := time.Date(2026, 6, 1, 22, 0, 0, 0, time.UTC)
	closeTime := open.Add(dur) // 2026-06-02 00:00 UTC

	tests := []struct {
		name string
		at   time.Time
		want bool
	}{
		{
			name: "at window open (inclusive)",
			at:   open,
			want: true,
		},
		{
			name: "inside window",
			at:   open.Add(30 * time.Minute),
			want: true,
		},
		{
			name: "at window close (inclusive)",
			at:   closeTime,
			want: true,
		},
		{
			name: "one second past window close",
			at:   closeTime.Add(time.Second),
			want: false,
		},
		{
			name: "one hour before window opens",
			at:   open.Add(-1 * time.Hour),
			want: false,
		},
		{
			name: "between two consecutive occurrences",
			// 03:00 UTC next day -- well past the 22:00+2h window.
			at:   open.Add(5 * time.Hour),
			want: false,
		},
		{
			name: "inside the second occurrence (next day)",
			at:   open.Add(24*time.Hour + 30*time.Minute),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWithinSyncWindow(sched, dur, tt.at)
			if got != tt.want {
				t.Errorf("at=%v: got %v, want %v", tt.at.Format(time.RFC3339), got, tt.want)
			}
		})
	}
}

// TestIsPeriodicRefreshAllowedByWindows covers the nil / empty / allow / deny /
// invalid-schedule paths and the unknown-kind default.
func TestIsPeriodicRefreshAllowedByWindows(t *testing.T) {
	// inWindow: 2026-06-01 23:00 UTC -- inside the 22:00+2h window.
	inWindow := time.Date(2026, 6, 1, 23, 0, 0, 0, time.UTC)
	// outWindow: 2026-06-01 20:00 UTC -- outside any window.
	outWindow := time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC)

	// Shared window definitions.
	validEntry := esv1.ExternalSecretSyncWindowEntry{
		Schedule: "0 22 * * *",
		Duration: metav1.Duration{Duration: 2 * time.Hour},
	}
	invalidEntry := esv1.ExternalSecretSyncWindowEntry{
		Schedule: "not-a-cron",
		Duration: metav1.Duration{Duration: 2 * time.Hour},
	}

	tests := []struct {
		name string
		sw   *esv1.ExternalSecretSyncWindows
		at   time.Time
		want bool
	}{
		{
			name: "nil SyncWindows always permits",
			sw:   nil,
			at:   inWindow,
			want: true,
		},
		{
			name: "empty Windows list always permits",
			sw:   &esv1.ExternalSecretSyncWindows{Kind: esv1.SyncWindowAllow},
			at:   inWindow,
			want: true,
		},
		// allow kind
		{
			name: "allow: at inside window -- permit",
			sw:   &esv1.ExternalSecretSyncWindows{Kind: esv1.SyncWindowAllow, Windows: []esv1.ExternalSecretSyncWindowEntry{validEntry}},
			at:   inWindow,
			want: true,
		},
		{
			name: "allow: at outside window -- block",
			sw:   &esv1.ExternalSecretSyncWindows{Kind: esv1.SyncWindowAllow, Windows: []esv1.ExternalSecretSyncWindowEntry{validEntry}},
			at:   outWindow,
			want: false,
		},
		// deny kind
		{
			name: "deny: at inside window -- block",
			sw:   &esv1.ExternalSecretSyncWindows{Kind: esv1.SyncWindowDeny, Windows: []esv1.ExternalSecretSyncWindowEntry{validEntry}},
			at:   inWindow,
			want: false,
		},
		{
			name: "deny: at outside window -- permit",
			sw:   &esv1.ExternalSecretSyncWindows{Kind: esv1.SyncWindowDeny, Windows: []esv1.ExternalSecretSyncWindowEntry{validEntry}},
			at:   outWindow,
			want: true,
		},
		// invalid schedule handling
		{
			name: "allow: only invalid schedule -- block (no window is active)",
			sw:   &esv1.ExternalSecretSyncWindows{Kind: esv1.SyncWindowAllow, Windows: []esv1.ExternalSecretSyncWindowEntry{invalidEntry}},
			at:   inWindow,
			want: false,
		},
		{
			name: "deny: only invalid schedule -- permit (no window is active)",
			sw:   &esv1.ExternalSecretSyncWindows{Kind: esv1.SyncWindowDeny, Windows: []esv1.ExternalSecretSyncWindowEntry{invalidEntry}},
			at:   inWindow,
			want: true,
		},
		{
			name: "allow: one invalid + one valid active window -- permit",
			sw: &esv1.ExternalSecretSyncWindows{
				Kind:    esv1.SyncWindowAllow,
				Windows: []esv1.ExternalSecretSyncWindowEntry{invalidEntry, validEntry},
			},
			at:   inWindow,
			want: true,
		},
		// unknown kind falls back to always-permit
		{
			name: "unknown kind -- always permit",
			sw: &esv1.ExternalSecretSyncWindows{
				Kind:    "unknown",
				Windows: []esv1.ExternalSecretSyncWindowEntry{validEntry},
			},
			at:   inWindow,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{SyncWindows: tt.sw},
			}
			got := isPeriodicRefreshAllowedByWindows(es, tt.at)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
