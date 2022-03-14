/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package framework

import (
	"context"

	//nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/e2e/framework/log"
)

var TargetSecretName = "target-secret"

// TestCase contains the test infra to run a table driven test.
type TestCase struct {
	Framework      *Framework
	ExternalSecret *esv1beta1.ExternalSecret
	Secrets        map[string]string
	ExpectedSecret *v1.Secret
}

// SecretStoreProvider is a interface that must be implemented
// by a provider that runs the e2e test.
type SecretStoreProvider interface {
	CreateSecret(key string, val string)
	DeleteSecret(key string)
}

// TableFunc returns the main func that runs a TestCase in a table driven test.
func TableFunc(f *Framework, prov SecretStoreProvider) func(...func(*TestCase)) {
	return func(tweaks ...func(*TestCase)) {
		var err error

		// make default test case
		// and apply customization to it
		tc := makeDefaultTestCase(f)
		for _, tweak := range tweaks {
			tweak(tc)
		}

		// create secrets & defer delete
		for k, v := range tc.Secrets {
			key := k
			prov.CreateSecret(key, v)
			defer func() {
				prov.DeleteSecret(key)
			}()
		}

		// create external secret
		err = tc.Framework.CRClient.Create(context.Background(), tc.ExternalSecret)
		Expect(err).ToNot(HaveOccurred())

		// in case target name is empty
		if tc.ExternalSecret.Spec.Target.Name == "" {
			TargetSecretName = tc.ExternalSecret.ObjectMeta.Name
		}

		// wait for Kind=Secret to have the expected data
		secret, err := tc.Framework.WaitForSecretValue(tc.Framework.Namespace.Name, TargetSecretName, tc.ExpectedSecret)
		if err != nil {
			log.Logf("Did not match. Expected: %+v, Got: %+v", tc.ExpectedSecret, secret)
		}

		Expect(err).ToNot(HaveOccurred())
	}
}

func makeDefaultTestCase(f *Framework) *TestCase {
	return &TestCase{
		Framework: f,
		ExternalSecret: &esv1beta1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-es",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1beta1.ExternalSecretSpec{
				SecretStoreRef: esv1beta1.SecretStoreRef{
					Name: f.Namespace.Name,
				},
				Target: esv1beta1.ExternalSecretTarget{
					Name: TargetSecretName,
				},
			},
		},
	}
}
