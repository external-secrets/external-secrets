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
package common

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

// This case creates multiple secrets with simple key/value pairs and syncs them using multiple .Spec.Data blocks.
// Not supported by: vault.
func SimpleDataSync(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync simple secrets from .Data[]", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretKey2 := fmt.Sprintf("%s-%s", f.Namespace.Name, "other")
		secretValue := "bar"
		tc.Secrets = map[string]string{
			secretKey1: secretValue,
			secretKey2: secretValue,
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey1: []byte(secretValue),
				secretKey2: []byte(secretValue),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: secretKey1,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: secretKey1,
				},
			},
			{
				SecretKey: secretKey2,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: secretKey2,
				},
			},
		}
	}
}

// This case creates multiple secrets with json values and syncs them using multiple .Spec.Data blocks.
// The data is extracted from the JSON key using ref.Property.
func JSONDataWithProperty(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync multiple secrets from .Data[]", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretKey2 := fmt.Sprintf("%s-%s", f.Namespace.Name, "two")
		secretValue1 := "{\"foo1\":\"foo1-val\",\"bar1\":\"bar1-val\"}"
		secretValue2 := "{\"foo2\":\"foo2-val\",\"bar2\":\"bar2-val\"}"
		tc.Secrets = map[string]string{
			secretKey1: secretValue1,
			secretKey2: secretValue2,
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey1: []byte("foo1-val"),
				secretKey2: []byte("bar2-val"),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: secretKey1,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      secretKey1,
					Property: "foo1",
				},
			},
			{
				SecretKey: secretKey2,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      secretKey2,
					Property: "bar2",
				},
			},
		}
	}
}

// This case creates multiple secrets with json values and renders a template.
// The data is extracted from the JSON key using ref.Property.
func JSONDataWithTemplate(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync json secrets with template", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretKey2 := fmt.Sprintf("%s-%s", f.Namespace.Name, "other")
		secretValue1 := "{\"foo1\":\"foo1-val\",\"bar1\":\"bar1-val\"}"
		secretValue2 := "{\"foo2\":\"foo2-val\",\"bar2\":\"bar2-val\"}"
		tc.Secrets = map[string]string{
			secretKey1: secretValue1,
			secretKey2: secretValue2,
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"example": "annotation",
				},
				Labels: map[string]string{
					"example": "label",
				},
			},
			Data: map[string][]byte{
				"my-data": []byte(`executed: foo1-val|bar2-val`),
			},
		}
		tc.ExternalSecret.Spec.Target.Template = &esv1alpha1.ExternalSecretTemplate{
			Metadata: esv1alpha1.ExternalSecretTemplateMetadata{
				Annotations: map[string]string{
					"example": "annotation",
				},
				Labels: map[string]string{
					"example": "label",
				},
			},
			Data: map[string]string{
				"my-data": "executed: {{ .one | toString }}|{{ .two | toString }}",
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: "one",
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      secretKey1,
					Property: "foo1",
				},
			},
			{
				SecretKey: "two",
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      secretKey2,
					Property: "bar2",
				},
			},
		}
	}
}

// This case creates one secret with json values and syncs them using a single .Spec.DataFrom block.
func JSONDataFromSync(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync secrets with dataFrom", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		targetSecretKey1 := "name"
		targetSecretValue1 := "great-name"
		targetSecretKey2 := "surname"
		targetSecretValue2 := "great-surname"
		secretValue := fmt.Sprintf("{ \"%s\": \"%s\", \"%s\": \"%s\" }", targetSecretKey1, targetSecretValue1, targetSecretKey2, targetSecretValue2)
		tc.Secrets = map[string]string{
			secretKey1: secretValue,
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				targetSecretKey1: []byte(targetSecretValue1),
				targetSecretKey2: []byte(targetSecretValue2),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esv1alpha1.ExternalSecretDataRemoteRef{
			{
				Key: secretKey1,
			},
		}
	}
}

// This case creates a secret with a nested json value. It is synced into two secrets.
// The values from the nested data are extracted using gjson.
// not supported by: vault.
func NestedJSONWithGJSON(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync nested json secrets and get inner keys", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		targetSecretKey1 := "firstname"
		targetSecretValue1 := "Tom"
		targetSecretKey2 := "first_friend"
		targetSecretValue2 := "Roger"
		secretValue := fmt.Sprintf(
			`{
				"name": {"first": "%s", "last": "Anderson"},
				"friends":
				[
					{"first": "Dale", "last": "Murphy"},
					{"first": "%s", "last": "Craig"},
					{"first": "Jane", "last": "Murphy"}
				]
			}`, targetSecretValue1, targetSecretValue2)
		tc.Secrets = map[string]string{
			secretKey1: secretValue,
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				targetSecretKey1: []byte(targetSecretValue1),
				targetSecretKey2: []byte(targetSecretValue2),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: targetSecretKey1,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      secretKey1,
					Property: "name.first",
				},
			},
			{
				SecretKey: targetSecretKey2,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      secretKey1,
					Property: "friends.1.first",
				},
			},
		}
	}
}

// This case creates a secret with a Docker json configuration value.
// The values from the nested data are extracted using gjson.
// not supported by: vault.
func DockerJSONConfig(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[specific] should do something I guess but not sure what!", func(tc *framework.TestCase) {
		cloudSecretName := fmt.Sprintf("%s-%s", f.Namespace.Name, "docker-config-example")
		cloudSecretValue := `{"auths":{"https://index.docker.io/v1/": {"auth": "c3R...zE2"}}}`

		tc.Secrets = map[string]string{
			cloudSecretName: cloudSecretValue,
		}

		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				".dockerconfigjson": []byte(cloudSecretValue),
			},
		}

		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: "mysecret",
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: cloudSecretName,
				},
			},
		}

		tc.ExternalSecret.Spec.Target.Template = &esv1alpha1.ExternalSecretTemplate{
			Metadata: esv1alpha1.ExternalSecretTemplateMetadata{
				Annotations: map[string]string{
					"example": "annotation",
				},
				Labels: map[string]string{
					"example": "label",
				},
			},
			Data: map[string]string{
				".dockerconfigjson": "{{ .mysecret | toString }}",
			},
		}

	}
}
