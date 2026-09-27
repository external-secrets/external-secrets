package clusterexternalsecret

import (
	"strings"
	"testing"
)

func TestBuildCESFinalizer(t *testing.T) {
	r := &Reconciler{}

	tests := []struct {
		name    string
		cesName string
	}{
		{name: "short name, well under limit", cesName: "my-secret"},
		{name: "exactly 59 chars (boundary, still fits)", cesName: strings.Repeat("a", 59)},
		{name: "60 chars (one over, must truncate)", cesName: strings.Repeat("b", 60)},
		{name: "61 chars, from the actual issue repro", cesName: "registry-pull-credentials-for-shared-artifact-proxy-cache-tok"},
		{name: "max allowed k8s name, 253 chars", cesName: strings.Repeat("c", 253)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finalizer := r.buildCESFinalizer(tt.cesName)

			parts := strings.SplitN(finalizer, "/", 2)
			if len(parts) != 2 {
				t.Fatalf("finalizer %q missing '/' separator", finalizer)
			}
			domain, namePart := parts[0], parts[1]

			if domain != cesFinalizerDomain {
				t.Errorf("domain = %q, want %q", domain, cesFinalizerDomain)
			}
			if len(namePart) > 63 {
				t.Errorf("name part %q is %d bytes, want <= 63", namePart, len(namePart))
			}

			// Determinism: calling it again must give the exact same string.
			again := r.buildCESFinalizer(tt.cesName)
			if again != finalizer {
				t.Errorf("buildCESFinalizer not deterministic: %q vs %q", finalizer, again)
			}
		})
	}
}

// TestBuildCESFinalizer_BackwardCompatible locks in that short names produce
// the exact same finalizer string the old, pre-fix implementation did.
// If this ever fails, existing clusters would get new finalizer names on
// upgrade and orphan the old ones.
func TestBuildCESFinalizer_BackwardCompatible(t *testing.T) {
	r := &Reconciler{}
	cesName := "my-short-ces-name"

	got := r.buildCESFinalizer(cesName)
	want := "externalsecrets.external-secrets.io/ces-" + cesName

	if got != want {
		t.Errorf("buildCESFinalizer(%q) = %q, want %q (pre-fix output)", cesName, got, want)
	}
}

// TestBuildCESFinalizer_NoCollision guards against two different long names
// that share the same 50-char truncated prefix producing the same finalizer.
func TestBuildCESFinalizer_NoCollision(t *testing.T) {
	r := &Reconciler{}

	base := strings.Repeat("x", 50) // shared 50-char prefix
	nameA := base + "-suffix-one-that-is-long-enough"
	nameB := base + "-suffix-two-that-is-long-enough"

	finalizerA := r.buildCESFinalizer(nameA)
	finalizerB := r.buildCESFinalizer(nameB)

	if finalizerA == finalizerB {
		t.Errorf("two different CES names produced the same finalizer: %q", finalizerA)
	}
}
