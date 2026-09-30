package sprig

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

//TODO(evrardjp): This is fragile and needs to be removed when classification of information
//is implemented: https://github.com/external-secrets/external-secrets/pull/7026
func TestFuncMapsExcludeUnsafeFunctions(t *testing.T) {
	for _, funcs := range []map[string]interface{}{GenericFuncMap(), TxtFuncMap()} {
		for _, name := range []string{"env", "expandenv", "getHostByName", "fail"} {
			if _, ok := funcs[name]; ok {
				t.Errorf("%s should not be exposed", name)
			}
		}
	}
}

//TODO(evrardjp): This is fragile and needs to be removed when classification of information
//is implemented: https://github.com/external-secrets/external-secrets/pull/7026
func TestFunctionsDoNotLeakInputInErrors(t *testing.T) {
	const sentinel = "s3cr3t"
	tests := []struct {
		name string
		call func() error
	}{
		{"mustDateModify", func() error { _, err := mustDateModify(sentinel, time.Time{}); return err }},
		{"mustToDate", func() error { _, err := mustToDate(sentinel, "date"); return err }},
		{"mustFromJson", func() error { _, err := mustFromJson(sentinel); return err }},
		{"mustRegexFind", func() error { _, err := mustRegexFind("["+sentinel, "value"); return err }},
		{"mustRegexFindAll", func() error { _, err := mustRegexFindAll("["+sentinel, "value", -1); return err }},
		{"mustRegexMatch", func() error { _, err := mustRegexMatch("["+sentinel, "value"); return err }},
		{"mustRegexReplaceAll", func() error { _, err := mustRegexReplaceAll("["+sentinel, "value", "replacement"); return err }},
		{"mustRegexReplaceAllLiteral", func() error { _, err := mustRegexReplaceAllLiteral("["+sentinel, "value", "replacement"); return err }},
		{"mustRegexSplit", func() error { _, err := mustRegexSplit("["+sentinel, "value", -1); return err }},
		{"semver", func() error { _, err := semver(sentinel); return err }},
		{"semverCompare", func() error { _, err := semverCompare(sentinel, "1.0.0"); return err }},
		{"urlParse", func() (err error) {
			defer func() { if recovered := recover(); recovered != nil { err = fmt.Errorf("%v", recovered) } }()
			urlParse("%" + sentinel)
			return fmt.Errorf("urlParse accepted invalid input")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil { t.Fatal("expected an error") }
			if strings.Contains(err.Error(), sentinel) { t.Fatalf("error contains input sentinel: %q", err) }
		})
	}
}
