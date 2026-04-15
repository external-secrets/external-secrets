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
