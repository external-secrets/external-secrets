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
func FindByName(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should find secrets by name using .DataFrom[]", func(tc *framework.TestCase) {
		const namePrefix = "e2e_find_name_%s_%s"
		secretKeyOne := fmt.Sprintf(namePrefix, f.Namespace.Name, "one")
		secretKeyTwo := fmt.Sprintf(namePrefix, f.Namespace.Name, "two")
		secretKeyThree := fmt.Sprintf(namePrefix, f.Namespace.Name, "three")
		secretValue := "{\"foo1\":\"foo1-val\"}"
		tc.Secrets = map[string]framework.SecretEntry{
			secretKeyOne:   {Value: secretValue},
			secretKeyTwo:   {Value: secretValue},
			secretKeyThree: {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKeyOne:   []byte(secretValue),
				secretKeyTwo:   []byte(secretValue),
				secretKeyThree: []byte(secretValue),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esapi.ExternalSecretDataFromRemoteRef{
			{
				Find: &esapi.ExternalSecretFind{
					Name: &esapi.FindName{
						RegExp: fmt.Sprintf("e2e_find_name_%s.+", f.Namespace.Name),
					},
				},
			},
		}
	}
}

func FindByNameWithPath(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should find secrets by name with path", func(tc *framework.TestCase) {
		secretKeyOne := fmt.Sprintf("e2e-find-name-%s-one", f.Namespace.Name)
		secretKeyTwo := fmt.Sprintf("%s-two", f.Namespace.Name)
		secretKeythree := fmt.Sprintf("%s-three", f.Namespace.Name)
		secretValue := "{\"foo1\":\"foo1-val\"}"
		tc.Secrets = map[string]framework.SecretEntry{
			secretKeyOne:   {Value: secretValue},
			secretKeyTwo:   {Value: secretValue},
			secretKeythree: {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				fmt.Sprintf("%s-two", f.Namespace.Name):   []byte(secretValue),
				fmt.Sprintf("%s-three", f.Namespace.Name): []byte(secretValue),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esapi.ExternalSecretDataFromRemoteRef{
			{
				Find: &esapi.ExternalSecretFind{
					Path: &f.Namespace.Name,
					Name: &esapi.FindName{
						RegExp: ".+",
					},
				},
			},
		}
	}
}
