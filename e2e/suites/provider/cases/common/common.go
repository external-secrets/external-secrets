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
	"context"
	"fmt"
	"time"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

const (
	// Constants.
	dockerConfigExampleName    = "docker-config-example"
	dockerConfigJSONKey        = ".dockerconfigjson"
	mysecretToStringTemplating = "{{ .mysecret }}"
	sshPrivateKey              = "ssh-privatekey"

	secretValue1 = "{\"foo1\":\"foo1-val\",\"bar1\":\"bar1-val\"}"
	secretValue2 = "{\"foo2\":\"foo2-val\",\"bar2\":\"bar2-val\"}"
)

// This case creates one secret with json values and syncs them using a single .Spec.DataFrom block.
func SyncV1Alpha1(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync secrets from v1alpha1 spec", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		targetSecretKey1 := "alpha-name"
		targetSecretValue1 := "alpha-great-name"
		targetSecretKey2 := "alpha-surname"
		targetSecretValue2 := "alpha-great-surname"
		secretValue := fmt.Sprintf("{ %q: %q, %q: %q }", targetSecretKey1, targetSecretValue1, targetSecretKey2, targetSecretValue2)
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey1: {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				targetSecretKey1: []byte(targetSecretValue1),
				targetSecretKey2: []byte(targetSecretValue2),
			},
		}
		tc.ExternalSecretV1Alpha1 = &esv1alpha1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-es",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.ExternalSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: time.Second * 5},
				SecretStoreRef: esv1alpha1.SecretStoreRef{
					Name: f.Namespace.Name,
				},
				Target: esv1alpha1.ExternalSecretTarget{
					Name: framework.TargetSecretName,
				},
				DataFrom: []esv1alpha1.ExternalSecretDataRemoteRef{
					{
						Key: secretKey1,
					},
				},
			},
		}
	}
}

// This case creates multiple secrets with simple key/value pairs and syncs them using multiple .Spec.Data blocks.
// Not supported by: vault.
func SimpleDataSync(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync simple secrets from .Data[]", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretKey2 := fmt.Sprintf("%s-%s", f.Namespace.Name, "other")
		secretValue := "bar"
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey1: {Value: secretValue},
			secretKey2: {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey1: []byte(secretValue),
				secretKey2: []byte(secretValue),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: secretKey1,
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key: secretKey1,
				},
			},
			{
				SecretKey: secretKey2,
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key: secretKey2,
				},
			},
		}
	}
}

// This case creates a secret with empty target name to test if it defaults to external secret name.
// Not supported by: vault.
func SyncWithoutTargetName(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync with empty target name.", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretValue := "bar"
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey1: {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey1: []byte(secretValue),
			},
		}
		tc.ExternalSecret.Spec.Target.Name = ""
		tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: secretKey1,
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key: secretKey1,
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
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey1: {Value: secretValue1},
			secretKey2: {Value: secretValue2},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey1: []byte("foo1-val"),
				secretKey2: []byte("bar2-val"),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: secretKey1,
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key:      secretKey1,
					Property: "foo1",
				},
			},
			{
				SecretKey: secretKey2,
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key:      secretKey2,
					Property: "bar2",
				},
			},
		}
	}
}

// This case creates a secret with empty target name to test if it defaults to external secret name.
// The data is extracted from the JSON key using ref.Property.
func JSONDataWithoutTargetName(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync with empty target name, using json.", func(tc *framework.TestCase) {
		secretKey := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretValue := "{\"foo\":\"foo-val\",\"bar\":\"bar-val\"}"
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey: {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey: []byte("foo-val"),
			},
		}
		tc.ExternalSecret.Spec.Target.Name = ""
		tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: secretKey,
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key:      secretKey,
					Property: "foo",
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
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey1: {Value: secretValue1},
			secretKey2: {Value: secretValue2},
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
		tc.ExternalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Metadata: esv1beta1.ExternalSecretTemplateMetadata{
				Annotations: map[string]string{
					"example": "annotation",
				},
				Labels: map[string]string{
					"example": "label",
				},
			},
			Data: map[string]string{
				"my-data": "executed: {{ .one }}|{{ .two }}",
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: "one",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key:      secretKey1,
					Property: "foo1",
				},
			},
			{
				SecretKey: "two",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
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
		secretValue := fmt.Sprintf("{ %q: %q, %q: %q }", targetSecretKey1, targetSecretValue1, targetSecretKey2, targetSecretValue2)
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey1: {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				targetSecretKey1: []byte(targetSecretValue1),
				targetSecretKey2: []byte(targetSecretValue2),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esv1beta1.ExternalSecretDataFromRemoteRef{
			{
				Extract: &esv1beta1.ExternalSecretDataRemoteRef{
					Key: secretKey1,
				},
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
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey1: {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				targetSecretKey1: []byte(targetSecretValue1),
				targetSecretKey2: []byte(targetSecretValue2),
			},
		}
		tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: targetSecretKey1,
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key:      secretKey1,
					Property: "name.first",
				},
			},
			{
				SecretKey: targetSecretKey2,
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
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
		cloudSecretName := fmt.Sprintf("%s-%s", f.Namespace.Name, dockerConfigExampleName)
		dockerconfig := `{"auths":{"https://index.docker.io/v1/": {"auth": "c3R...zE2"}}}`
		cloudSecretValue := fmt.Sprintf(`{"dockerconfig": %s}`, dockerconfig)
		tc.Secrets = map[string]framework.SecretEntry{
			cloudSecretName: {Value: cloudSecretValue},
		}

		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				dockerConfigJSONKey: []byte(dockerconfig),
			},
		}

		tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: "mysecret",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key:      cloudSecretName,
					Property: "dockerconfig",
				},
			},
		}

		tc.ExternalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Data: map[string]string{
				dockerConfigJSONKey: mysecretToStringTemplating,
			},
		}
	}
}

// This case creates a secret with a Docker json configuration value.
// The values from the nested data are extracted using gjson.
// Need to have a key holding dockerconfig to be supported by vault.
func DataPropertyDockerconfigJSON(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync docker configurated json secrets with template", func(tc *framework.TestCase) {
		cloudSecretName := fmt.Sprintf("%s-%s", f.Namespace.Name, dockerConfigExampleName)
		dockerconfigString := `"{\"auths\":{\"https://index.docker.io/v1/\": {\"auth\": \"c3R...zE2\"}}}"`
		dockerconfig := `{"auths":{"https://index.docker.io/v1/": {"auth": "c3R...zE2"}}}`
		cloudSecretValue := fmt.Sprintf(`{"dockerconfig": %s}`, dockerconfigString)
		tc.Secrets = map[string]framework.SecretEntry{
			cloudSecretName: {Value: cloudSecretValue},
		}

		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				dockerConfigJSONKey: []byte(dockerconfig),
			},
		}

		tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: "mysecret",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key:      cloudSecretName,
					Property: "dockerconfig",
				},
			},
		}

		tc.ExternalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Type: v1.SecretTypeDockerConfigJson,
			Data: map[string]string{
				dockerConfigJSONKey: mysecretToStringTemplating,
			},
		}
	}
}

// This case adds an ssh private key secret and synchronizes it.
// Not supported by: vault. Json parsing error.
func SSHKeySync(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync ssh key secret", func(tc *framework.TestCase) {
		sshSecretName := fmt.Sprintf("%s-%s", f.Namespace.Name, "ssh-priv-key-example")
		sshSecretValue := `-----BEGIN OPENSSH PRIVATE KEY-----
		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
		-----END OPENSSH PRIVATE KEY-----`

		tc.Secrets = map[string]framework.SecretEntry{
			sshSecretName: {Value: sshSecretValue},
		}

		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeSSHAuth,
			Data: map[string][]byte{
				sshPrivateKey: []byte(sshSecretValue),
			},
		}

		tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: "mysecret",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key: sshSecretName,
				},
			},
		}

		tc.ExternalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Type: v1.SecretTypeSSHAuth,
			Data: map[string]string{
				sshPrivateKey: mysecretToStringTemplating,
			},
		}
	}
}

// This case adds an ssh private key secret and syncs it.
func SSHKeySyncDataProperty(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should sync ssh key with provider.", func(tc *framework.TestCase) {
		cloudSecretName := fmt.Sprintf("%s-%s", f.Namespace.Name, dockerConfigExampleName)
		SSHKey := `-----BEGIN OPENSSH PRIVATE KEY-----
		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
		-----END OPENSSH PRIVATE KEY-----`
		cloudSecretValue := fmt.Sprintf(`{"ssh-auth": %q}`, SSHKey)
		tc.Secrets = map[string]framework.SecretEntry{
			cloudSecretName: {Value: cloudSecretValue},
		}

		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeSSHAuth,
			Data: map[string][]byte{
				sshPrivateKey: []byte(SSHKey),
			},
		}

		tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: "mysecret",
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key:      cloudSecretName,
					Property: "ssh-auth",
				},
			},
		}

		tc.ExternalSecret.Spec.Target.Template = &esv1beta1.ExternalSecretTemplate{
			Type: v1.SecretTypeSSHAuth,
			Data: map[string]string{
				sshPrivateKey: mysecretToStringTemplating,
			},
		}
	}
}

func DeletionPolicyDelete(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should delete secret when provider secret was deleted using .data[]", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one")
		secretKey2 := fmt.Sprintf("%s-%s", f.Namespace.Name, "other")
		secretValue := "bazz"
		tc.Secrets = map[string]framework.SecretEntry{
			secretKey1: {Value: secretValue},
			secretKey2: {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey1: []byte(secretValue),
				secretKey2: []byte(secretValue),
			},
		}

		tc.ExternalSecret.Spec.Target.DeletionPolicy = esv1beta1.DeletionPolicyDelete
		tc.ExternalSecret.Spec.Data = []esv1beta1.ExternalSecretData{
			{
				SecretKey: secretKey1,
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key: secretKey1,
				},
			},
			{
				SecretKey: secretKey2,
				RemoteRef: esv1beta1.ExternalSecretDataRemoteRef{
					Key: secretKey2,
				},
			},
		}
		tc.AfterSync = func(prov framework.SecretStoreProvider, secret *v1.Secret) {
			prov.DeleteSecret(secretKey1)
			prov.DeleteSecret(secretKey2)

			gomega.Eventually(func() bool {
				_, err := f.KubeClientSet.CoreV1().Secrets(f.Namespace.Name).Get(context.Background(), secret.Name, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, time.Minute, time.Second*5).Should(gomega.BeTrue())
		}
	}
}
