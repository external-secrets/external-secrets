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

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

// This case creates multiple secrets with simple key/value pairs and syncs them using multiple .Spec.Data blocks.
func FindByTag(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should find secrets by tags using .DataFrom[]", func(tc *framework.TestCase) {
		const namePrefix = "e2e-find-name-%s-%s"
		secretKeyOne := fmt.Sprintf(namePrefix, f.Namespace.Name, "one")
		secretKeyTwo := fmt.Sprintf(namePrefix, f.Namespace.Name, "two")
		secretKeyThree := fmt.Sprintf(namePrefix, f.Namespace.Name, "three")
		tc.Secrets = map[string]framework.SecretEntry{
			secretKeyOne: {
				Value: secretValue1,
				Tags: map[string]string{
					"test": f.Namespace.Name,
				}},
			secretKeyTwo: {
				Value: secretValue1,
				Tags: map[string]string{
					"test": f.Namespace.Name,
				}},
			secretKeyThree: {
				Value: secretValue1,
				Tags: map[string]string{
					"test": f.Namespace.Name,
				}},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				fmt.Sprintf("e2e-find-name-%s-one", f.Namespace.Name):   []byte(secretValue1),
				fmt.Sprintf("e2e-find-name-%s-two", f.Namespace.Name):   []byte(secretValue1),
				fmt.Sprintf("e2e-find-name-%s-three", f.Namespace.Name): []byte(secretValue1),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esapi.ExternalSecretDataFromRemoteRef{
			{
				Find: &esapi.ExternalSecretFind{
					Tags: map[string]string{
						"test": f.Namespace.Name,
					},
				},
			},
		}
	}
}

// This case creates multiple secrets with simple key/value pairs
// this is special because parameter store requires a leading "/" in the name.
func FindByTagWithPath(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should find secrets by tags with path prefix", func(tc *framework.TestCase) {
		pathPrefix := "foobar"
		secretKeyOne := fmt.Sprintf("e2e-find-name-%s-%s", f.Namespace.Name, "one")
		secretKeyTwo := fmt.Sprintf("%s-%s-%s", pathPrefix, f.Namespace.Name, "two")
		secretKeyThree := fmt.Sprintf("%s-%s-%s", pathPrefix, f.Namespace.Name, "three")
		secretValue := "{\"foo1\":\"foo1-val\",\"bar1\":\"bar1-val\"}"
		tc.Secrets = map[string]framework.SecretEntry{
			secretKeyOne: {
				Value: secretValue,
				Tags: map[string]string{
					"test": f.Namespace.Name,
				}},
			secretKeyTwo: {
				Value: secretValue,
				Tags: map[string]string{
					"test": f.Namespace.Name,
				}},
			secretKeyThree: {
				Value: secretValue,
				Tags: map[string]string{
					"test": f.Namespace.Name,
				}},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				fmt.Sprintf("foobar-%s-two", f.Namespace.Name):   []byte(secretValue),
				fmt.Sprintf("foobar-%s-three", f.Namespace.Name): []byte(secretValue),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esapi.ExternalSecretDataFromRemoteRef{
			{
				Find: &esapi.ExternalSecretFind{
					Path: &pathPrefix,
					Tags: map[string]string{
						"test": f.Namespace.Name,
					},
				},
			},
		}
	}
}
