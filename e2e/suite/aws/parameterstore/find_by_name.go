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
package aws

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

// This case creates multiple secrets with simple key/value pairs and syncs them using multiple .Spec.Data blocks.
// this is special because parameter store requires a leading "/" in the name.
func FindByName(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[common] should find secrets by name using .DataFrom[]", func(tc *framework.TestCase) {
		const namePrefix = "/e2e/find/name/%s/%s"
		secretKeyOne := fmt.Sprintf(namePrefix, f.Namespace.Name, "one")
		secretKeyTwo := fmt.Sprintf(namePrefix, f.Namespace.Name, "two")
		secretKeyThree := fmt.Sprintf(namePrefix, f.Namespace.Name, "three")
		secretValue := "{\"foo1\":\"foo1-val\"}"
		tc.Secrets = map[string]framework.SecretEntry{
			secretKeyOne:   {Value: secretValue},
			secretKeyTwo:   {Value: secretValue},
			secretKeyThree: {Value: secretValue},
		}
		const secNamePrefix = "_e2e_find_name_%s_%s"
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				fmt.Sprintf(secNamePrefix, f.Namespace.Name, "one"):   []byte(secretValue),
				fmt.Sprintf(secNamePrefix, f.Namespace.Name, "two"):   []byte(secretValue),
				fmt.Sprintf(secNamePrefix, f.Namespace.Name, "three"): []byte(secretValue),
			},
		}
		tc.ExternalSecret.Spec.DataFrom = []esapi.ExternalSecretDataFromRemoteRef{
			{
				Find: &esapi.ExternalSecretFind{
					Name: &esapi.FindName{
						RegExp: fmt.Sprintf("/e2e/find/name/%s.+", f.Namespace.Name),
					},
				},
			},
		}
	}
}
