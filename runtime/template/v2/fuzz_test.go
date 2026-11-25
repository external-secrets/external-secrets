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

package template

import (
	"os"
	"testing"
)

func FuzzPKCS12KeyPass(f *testing.F) {
	nopass, err := os.ReadFile("_testdata/foo-nopass.pfx")
	if err != nil {
		f.Fatalf("no test data: %v", err)
	}

	withpass, err := os.ReadFile("_testdata/foo-withpass-1234.pfx")
	if err != nil {
		f.Fatalf("no test data: %v", err)
	}

	f.Add("", string(nopass))
	f.Add("1234", string(withpass))
	f.Add("wrong-password", string(nopass))

	f.Fuzz(func(t *testing.T, pass string, input string) {
		out, err := pkcs12keyPass(pass, input)
		if err != nil {
			return
		}

		if out == "" {
			t.Fatalf("empty result without mistake: pass=%q, len(input)=%d", pass, len(input))
		}

		out, err = pkcs12key(input)
		if err != nil {
			return
		}

		if out == "" {
			t.Fatalf("empty result without mistake: len(input)=%d", len(input))
		}

		out, err = pkcs12certPass(pass, input)

		if err != nil {
			return
		}

		if out == "" {
			t.Fatalf("empty result without mistake: pass=%q, len(input)=%d", pass, len(input))
		}

		out, err = pkcs12cert(input)
		if err != nil {
			return
		}

		if out == "" {
			t.Fatalf("empty result without mistake: len(input)=%d", len(input))
		}

		out, err = pemToPkcs12(pass, input)

		if err != nil {
			return
		}

		if out == "" {
			t.Fatalf("empty result without mistake: pass=%q, len(input)=%d", pass, len(input))
		}
	})
}
