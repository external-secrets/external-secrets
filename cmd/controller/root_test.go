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
