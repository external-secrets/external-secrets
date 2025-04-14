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

	"github.com/external-secrets/external-secrets-e2e/framework"
	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// This case creates one secret secrets with multiple versions
// the value contains the version number
func VersionedParameter(prov *Provider) func(f *framework.Framework) (string, func(*framework.TestCase)) {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "[common] should read versioned secrets", func(tc *framework.TestCase) {
			const namePrefix = "/e2e/versioned/%s/%s"
			secretKeyOne := fmt.Sprintf(namePrefix, f.Namespace.Name, "one")
			versions := []int{1, 2, 3, 4, 5}
			valueStr := "value%d"

			tc.ExpectedSecret = &v1.Secret{
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{}, // filled below
			}

			tc.ExternalSecret.Spec.Data = make([]esapi.ExternalSecretData, len(versions))

			// create many versions
			i := 0
			for _, v := range versions {
				secretKey := fmt.Sprintf("v%d", v)
				val := fmt.Sprintf(valueStr, v)
				prov.CreateSecret(secretKeyOne, framework.SecretEntry{
					Value: val,
				})
				tc.ExpectedSecret.Data[secretKey] = []byte(val)
				tc.ExternalSecret.Spec.Data[i] = esapi.ExternalSecretData{
					SecretKey: secretKey,
					RemoteRef: esapi.ExternalSecretDataRemoteRef{
						Key:     secretKeyOne,
						Version: fmt.Sprintf("%d", v),
					},
				}
				i++
			}
		}
	}
}
