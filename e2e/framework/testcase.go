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
	"time"

	//nolint
	"github.com/external-secrets/external-secrets-e2e/framework/log"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var TargetSecretName = "target-secret"

// TestCase contains the test infra to run a table driven test.
type TestCase struct {
	Framework               *Framework
	ExternalSecret          *esv1beta1.ExternalSecret
	ExternalSecretV1Alpha1  *esv1alpha1.ExternalSecret
	PushSecret              *esv1alpha1.PushSecret
	PushSecretSource        *v1.Secret
	AdditionalObjects       []client.Object
	Secrets                 map[string]SecretEntry
	ExpectedSecret          *v1.Secret
	AfterSync               func(SecretStoreProvider, *v1.Secret)
	VerifyPushSecretOutcome func(ps *esv1alpha1.PushSecret, pushClient esv1beta1.SecretsClient)
}

type SecretEntry struct {
	Value string
	Tags  map[string]string
}

// SecretStoreProvider is a interface that must be implemented
// by a provider that runs the e2e test.
type SecretStoreProvider interface {
	CreateSecret(key string, val SecretEntry)
	DeleteSecret(key string)
}

// TableFuncWithExternalSecret returns the main func that runs a TestCase in a table driven test.
func TableFuncWithExternalSecret(f *Framework, prov SecretStoreProvider) func(...func(*TestCase)) {
	return func(tweaks ...func(*TestCase)) {
		// make default test case
		// and apply customization to it
		tc := makeDefaultExternalSecretTestCase(f)
		for _, tweak := range tweaks {
			tweak(tc)
		}

		// create secrets & defer delete
		var deferRemoveKeys []string
		for k, v := range tc.Secrets {
			key := k
			prov.CreateSecret(key, v)
			deferRemoveKeys = append(deferRemoveKeys, key)
		}

		defer func() {
			for _, k := range deferRemoveKeys {
				prov.DeleteSecret(k)
			}
		}()

		// create v1alpha1 external secret, if provided
		createProvidedExternalSecret(tc)

		// create additional objects
		generateAdditionalObjects(tc)

		// in case target name is empty
		if tc.ExternalSecret != nil && tc.ExternalSecret.Spec.Target.Name == "" {
			TargetSecretName = tc.ExternalSecret.ObjectMeta.Name
		}

		// wait for Kind=Secret to have the expected data
		executeAfterSync(tc, f, prov)
	}
}

func executeAfterSync(tc *TestCase, f *Framework, prov SecretStoreProvider) {
	if tc.ExpectedSecret != nil {
		secret, err := tc.Framework.WaitForSecretValue(tc.Framework.Namespace.Name, TargetSecretName, tc.ExpectedSecret)
		if err != nil {
			f.printESDebugLogs(tc.ExternalSecret.Name, tc.ExternalSecret.Namespace)
			log.Logf("Did not match. Expected: %+v, Got: %+v", tc.ExpectedSecret, secret)
		}

		Expect(err).ToNot(HaveOccurred())
		tc.AfterSync(prov, secret)
	} else {
		tc.AfterSync(prov, nil)
	}
}

func generateAdditionalObjects(tc *TestCase) {
	if tc.AdditionalObjects != nil {
		for _, obj := range tc.AdditionalObjects {
			err := tc.Framework.CRClient.Create(context.Background(), obj)
			Expect(err).ToNot(HaveOccurred())
		}
	}
}

func createProvidedExternalSecret(tc *TestCase) {
	if tc.ExternalSecretV1Alpha1 != nil {
		err := tc.Framework.CRClient.Create(context.Background(), tc.ExternalSecretV1Alpha1)
		Expect(err).ToNot(HaveOccurred())
	} else if tc.ExternalSecret != nil {
		// create v1beta1 external secret otherwise
		err := tc.Framework.CRClient.Create(context.Background(), tc.ExternalSecret)
		Expect(err).ToNot(HaveOccurred())
	}
}

// TableFuncWithPushSecret returns the main func that runs a TestCase in a table driven test for push secrets.
func TableFuncWithPushSecret(f *Framework, prov SecretStoreProvider, pushClient esv1beta1.SecretsClient) func(...func(*TestCase)) {
	return func(tweaks ...func(*TestCase)) {
		var err error

		// make default test case
		// and apply customization to it
		tc := makeDefaultPushSecretTestCase(f)
		for _, tweak := range tweaks {
			tweak(tc)
		}

		if tc.PushSecretSource != nil {
			err := tc.Framework.CRClient.Create(context.Background(), tc.PushSecretSource)
			Expect(err).ToNot(HaveOccurred())
		}

		// create v1alpha1 push secret, if provided
		if tc.PushSecret != nil {
			// create v1beta1 external secret otherwise
			err = tc.Framework.CRClient.Create(context.Background(), tc.PushSecret)
			Expect(err).ToNot(HaveOccurred())
		}

		// additional objects
		generateAdditionalObjects(tc)

		// Run verification on the secret that push secret created or not.
		tc.VerifyPushSecretOutcome(tc.PushSecret, pushClient)
	}
}

func makeDefaultExternalSecretTestCase(f *Framework) *TestCase {
	return &TestCase{
		AfterSync: func(ssp SecretStoreProvider, s *v1.Secret) {},
		Framework: f,
		ExternalSecret: &esv1beta1.ExternalSecret{
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
					Name: TargetSecretName,
				},
			},
		},
	}
}

func makeDefaultPushSecretTestCase(f *Framework) *TestCase {
	return &TestCase{
		Framework: f,
		PushSecret: &esv1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-ps",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.PushSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: time.Second * 5},
				SecretStoreRefs: []esv1alpha1.PushSecretStoreRef{
					{
						Name: f.Namespace.Name,
					},
				},
			},
		},
	}
}
