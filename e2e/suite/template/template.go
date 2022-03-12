/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/
package template

import (

	// nolint
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

var _ = Describe("[template]", Label("template"), func() {
	f := framework.New("eso-template")
	prov := newProvider(f)

	DescribeTable("sync secrets", framework.TableFunc(f, prov),
		framework.Compose("template v1", f, genericTemplate, useTemplateV1),
		framework.Compose("template v2", f, genericTemplate, useTemplateV2),
	)
})

// useTemplateV1 specifies a test case which uses the template engine v1.
func useTemplateV1(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
		EngineVersion: esv1beta1.TemplateEngineV1,
		Data: map[string]string{
			"tplv1": "executed: {{ .singlefoo | toString }}|{{ .singlebaz | toString }}",
			"other": `{{ .foo | toString }}|{{ .bar | toString }}`,
		},
	}
	tc.ExpectedSecret.Data = map[string][]byte{
		"tplv1": []byte(`executed: bar|bang`),
		"other": []byte(`barmap|bangmap`),
	}
}

// useTemplateV2 specifies a test case which uses the template engine v2.
func useTemplateV2(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
		EngineVersion: esv1beta1.TemplateEngineV2,
		Data: map[string]string{
			"tplv2":     "executed: {{ .singlefoo }}|{{ .singlebaz }}",
			"other":     `{{ .foo }}|{{ .bar }}`,
			"sprig-str": `{{ .foo | upper }}`,
			"json-ex":   `{{ $var := .singlejson | fromJson }}{{ $var.foo | toJson }}`,
		},
	}
	tc.ExpectedSecret.Data = map[string][]byte{
		"tplv2":     []byte(`executed: bar|bang`),
		"other":     []byte(`barmap|bangmap`),
		"sprig-str": []byte(`BARMAP`),
		"json-ex":   []byte(`{"bar":"baz"}`),
	}
}

// This case uses template engine v1.
func genericTemplate(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[template] should execute template v1", func(tc *framework.TestCase) {
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
		}
		tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: "singlefoo",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key: "foo",
				},
			},
			{
				SecretKey: "singlebaz",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key: "baz",
				},
			},
			{
				SecretKey: "singlejson",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key: "json",
				},
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: "map",
				},
			},
		}
	}
}
