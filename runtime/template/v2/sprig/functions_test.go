package sprig

import "testing"

func TestFuncMapsExcludeUnsafeFunctions(t *testing.T) {
	for _, funcs := range []map[string]interface{}{GenericFuncMap(), TxtFuncMap()} {
		for _, name := range []string{"env", "expandenv", "getHostByName", "fail"} {
			if _, ok := funcs[name]; ok {
				t.Errorf("%s should not be exposed", name)
			}
		}
	}
}
