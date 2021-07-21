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
package gcp

import (
	"fmt"
	"os"

	// nolint
	. "github.com/onsi/ginkgo"

	// nolint
	. "github.com/onsi/ginkgo/extensions/table"
	v1 "k8s.io/api/core/v1"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

var _ = Describe("[gcp] ", func() {
	f := framework.New("eso-gcp")
	credentials := os.Getenv("GCP_SM_SA_JSON")
	projectID := os.Getenv("GCP_PROJECT_ID")
	prov := newgcpProvider(f, credentials, projectID)

	DescribeTable("sync secrets", framework.TableFunc(f, prov)) // Entry(common.SimpleDataSync(f)),
	// Entry(common.JSONDataWithProperty(f)),
	// Entry(common.JSONDataFromSync(f)),
	// Entry(common.NestedJSONWithGJSON(f)),
	// Entry(common.JSONDataWithTemplate(f)),
	// Entry(mySimpleTest(f)),

})

func mySimpleTest(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[Joey] A simple test for testing test testability", func(tc *framework.TestCase) {
		secretKey1 := fmt.Sprintf("%s-%s", f.Namespace.Name, "one") // Maps tc.Secret on cloud with External Secret on Cluster
		secretValue := "Hello there!"

		// 1. Secret that will be pushed to cloud
		tc.Secrets = map[string]string{
			secretKey1: secretValue,
		}

		// 2. What we get back from the cloud provider
		tc.ExternalSecret.Spec.Data = []esv1alpha1.ExternalSecretData{
			{
				SecretKey: secretKey1,
				RemoteRef: esv1alpha1.ExternalSecretDataRemoteRef{
					Key: secretKey1,
				},
			},
		}

		// 3. Local representation of a secret that will be compared against 2
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKey1: []byte(secretValue),
			},
		}
	}
}
