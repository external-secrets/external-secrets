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
	"context"
	"fmt"
	"time"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	// nolint
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("[template]", Label("template"), func() {
	f := framework.New("templating")
	prov := newProvider(f)
	fakeSecretClient := fake.New()

	DescribeTable("sync secrets", framework.TableFuncWithExternalSecret(f, prov),
		framework.Compose("template v1", f, genericExternalSecretTemplate, useTemplateV1),
		framework.Compose("template v2", f, genericExternalSecretTemplate, useTemplateV2),
	)

	DescribeTable("push secret", framework.TableFuncWithPushSecret(f, prov, fakeSecretClient),
		framework.Compose("template", f, genericPushSecretTemplate, useTemplateWithPushSecret),
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
func genericExternalSecretTemplate(f *framework.Framework) (string, func(*framework.TestCase)) {
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

// This case uses template engine v1.
func genericPushSecretTemplate(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[template] should execute template v1", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		tc.PushSecretSource = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretKey1,
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"singlefoo": []byte("bar"),
			},
			Type: v1.SecretTypeOpaque,
		}
		tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
			Secret: esv1alpha1.PushSecretSecret{
				Name: secretKey1,
			},
		}
		tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{
			{
				Match: esv1alpha1.PushSecretMatch{
					SecretKey: "singlefoo",
					RemoteRef: esv1alpha1.PushSecretRemoteRef{
						RemoteKey: "key",
						Property:  "singlefoo",
					},
				},
			},
		}
		tc.VerifyPushSecretOutcome = func(sourcePs *esv1alpha1.PushSecret, pushClient esv1beta1.SecretsClient) {
			gomega.Eventually(func() bool {
				s := &esv1alpha1.PushSecret{}
				err := tc.Framework.CRClient.Get(context.Background(), types.NamespacedName{Name: tc.PushSecret.Name, Namespace: tc.PushSecret.Namespace}, s)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				for i := range s.Status.Conditions {
					c := s.Status.Conditions[i]
					if c.Type == esv1alpha1.PushSecretReady && c.Status == v1.ConditionTrue {
						return true
					}
				}

				return false
			}, time.Minute*1, time.Second*5).Should(gomega.BeTrue())

			// create an external secret that fetches the created remote secret
			// and check the value
			exampleOutput := "example-output"
			es := &esv1beta1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2e-es",
					Namespace: f.Namespace.Name,
				},
				Spec: esv1beta1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Second * 5},
					SecretStoreRef: esv1beta1.SecretStoreRef{
						Name: f.Namespace.Name,
					},
					Target: esv1beta1.ExternalSecretTarget{
						Name: exampleOutput,
					},
					Data: []esv1beta1.ExternalSecretData{
						{
							SecretKey: exampleOutput,
							RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
								Key: "key",
							},
						},
					},
				},
			}

			err := tc.Framework.CRClient.Create(context.Background(), es)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			outputSecret := &v1.Secret{}
			err = wait.PollImmediate(time.Second*5, time.Second*15, func() (bool, error) {
				err := f.CRClient.Get(context.Background(), types.NamespacedName{
					Namespace: f.Namespace.Name,
					Name:      exampleOutput,
				}, outputSecret)
				if apierrors.IsNotFound(err) {
					return false, nil
				}
				return true, nil
			})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			v, ok := outputSecret.Data[exampleOutput]
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(string(v)).To(gomega.Equal("executed: BAR"))
		}
	}
}

// useTemplateWithPushSecret specifies a test case which uses the template engine v1.
func useTemplateWithPushSecret(tc *framework.TestCase) {
	tc.PushSecret.Spec.Template = &esv1beta1.ExternalSecretTemplate{
		EngineVersion: esv1beta1.TemplateEngineV2,
		Data: map[string]string{
			"singlefoo": "executed: {{ .singlefoo | upper }}",
		},
	}
}
