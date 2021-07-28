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
	return "[common] should sync docker configurated json secrets with template simple", func(tc *framework.TestCase) {
		cloudSecretName := fmt.Sprintf("%s-%s", f.Namespace.Name, "docker-config-example")
		dockerconfig := `{"auths":{"https://index.docker.io/v1/": {"auth": "c3R...zE2"}}}`
		cloudSecretValue := fmt.Sprintf(`{"dockerconfig": %s}`, dockerconfig)
		tc.Secrets = map[string]string{
			cloudSecretName: cloudSecretValue,
		}

		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				".dockerconfigjson": []byte(dockerconfig),
			},
		}

		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: "mysecret",
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      cloudSecretName,
					Property: "dockerconfig",
				},
			},
		}

		tc.ExternalSecret.Spec.Target.Template = &esv1alpha1.ExternalSecretTemplate{
			Data: map[string]string{
				".dockerconfigjson": "{{ .mysecret | toString }}",
			},
		}
	}
}

// This case creates a secret with a Docker json configuration value.
// The values from the nested data are extracted using gjson.
// Need to have a key holding dockerconfig to be supported by vault.
func DataPropertyDockerconfigJSON(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync docker configurated json secrets with template", func(tc *framework.TestCase) {
		cloudSecretName := fmt.Sprintf("%s-%s", f.Namespace.Name, "docker-config-example")
		dockerconfigString := `"{\"auths\":{\"https://index.docker.io/v1/\": {\"auth\": \"c3R...zE2\"}}}"`
		dockerconfig := `{"auths":{"https://index.docker.io/v1/": {"auth": "c3R...zE2"}}}`
		cloudSecretValue := fmt.Sprintf(`{"dockerconfig": %s}`, dockerconfigString)
		tc.Secrets = map[string]string{
			cloudSecretName: cloudSecretValue,
		}

		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				".dockerconfigjson": []byte(dockerconfig),
			},
		}

		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: "mysecret",
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      cloudSecretName,
					Property: "dockerconfig",
				},
			},
		}

		tc.ExternalSecret.Spec.Target.Template = &esv1alpha1.ExternalSecretTemplate{
			Type: v1.SecretTypeDockerConfigJson,
			Data: map[string]string{
				".dockerconfigjson": "{{ .mysecret | toString }}",
			},
		}
	}
}

// This case adds an ssh private key secret and sybcs it.
// Not supported by: vault. Json parsing error.
func SSHKeySync(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync ssh key secret", func(tc *framework.TestCase) {
		sshSecretName := fmt.Sprintf("%s-%s", f.Namespace.Name, "ssh-priv-key-example")
		sshSecretValue := `-----BEGIN OPENSSH PRIVATE KEY-----
		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
		-----END OPENSSH PRIVATE KEY-----`

		tc.Secrets = map[string]string{
			sshSecretName: sshSecretValue,
		}

		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				"ssh-privatekey": []byte(sshSecretValue),
			},
		}

		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: "mysecret",
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: sshSecretName,
				},
			},
		}

		tc.ExternalSecret.Spec.Target.Template = &esv1alpha1.ExternalSecretTemplate{
			Data: map[string]string{
				"ssh-privatekey": "{{ .mysecret | toString }}",
			},
		}
	}
}

// This case adds an ssh private key secret and syncs it.
// Supported by vault. But does not work with any form of line breaks as standard ssh key.
func SSHKeySyncDataProperty(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync ssh key with provider.", func(tc *framework.TestCase) {
		cloudSecretName := fmt.Sprintf("%s-%s", f.Namespace.Name, "docker-config-example")
		//SSHKey := "EY2NNWddRADTFdNvEojrCwo+DUxy6va2JltQAbxmhyvSZsL1eYsutunsKEwonGSru0Zd+m z5DHJOOQdHEsH3AAAACmFub3RoZXJvbmU= -----END OPENSSH PRIVATE KEY-----"
		SSHKey := `-----BEGIN OPENSSH PRIVATE KEY-----
		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
		-----END OPENSSH PRIVATE KEY-----`
		cloudSecretValue := fmt.Sprintf(`{"ssh-auth": "%s"}`, SSHKey)
		tc.Secrets = map[string]string{
			cloudSecretName: cloudSecretValue,
		}

		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeSSHAuth,
			Data: map[string][]byte{
				"ssh-privatekey": []byte(SSHKey),
			},
		}

		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: "mysecret",
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key:      cloudSecretName,
					Property: "ssh-auth",
				},
			},
		}

		tc.ExternalSecret.Spec.Target.Template = &esv1alpha1.ExternalSecretTemplate{
			Type: v1.SecretTypeSSHAuth,
			Data: map[string]string{
				"ssh-privatekey": "{{ .mysecret | toString }}",
			},
		}
	}
}
