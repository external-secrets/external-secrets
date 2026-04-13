/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type testcaseProviderStub struct {
	name string
}

func (s *testcaseProviderStub) CreateSecret(string, SecretEntry) {}

func (s *testcaseProviderStub) DeleteSecret(string) {}

func TestPrepareTestCaseUsesDefaultProviderWithoutOverride(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	defaultProvider := &testcaseProviderStub{name: "default"}
	tc := &TestCase{}

	provider := prepareTestCase(tc, defaultProvider)

	Expect(provider).To(BeIdenticalTo(defaultProvider))
}

func TestPrepareTestCaseLetsPrepareInstallProviderOverride(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	defaultProvider := &testcaseProviderStub{name: "default"}
	overrideProvider := &testcaseProviderStub{name: "override"}
	tc := &TestCase{
		Prepare: func(tc *TestCase, provider SecretStoreProvider) {
			Expect(provider).To(BeIdenticalTo(defaultProvider))
			tc.ProviderOverride = overrideProvider
		},
	}

	provider := prepareTestCase(tc, defaultProvider)

	Expect(provider).To(BeIdenticalTo(overrideProvider))
}

func TestPrepareTestCaseUsesExistingOverrideDuringPrepare(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	defaultProvider := &testcaseProviderStub{name: "default"}
	overrideProvider := &testcaseProviderStub{name: "override"}
	tc := &TestCase{
		ProviderOverride: overrideProvider,
		Prepare: func(tc *TestCase, provider SecretStoreProvider) {
			Expect(provider).To(BeIdenticalTo(overrideProvider))
		},
	}

	provider := prepareTestCase(tc, defaultProvider)

	Expect(provider).To(BeIdenticalTo(overrideProvider))
}

func TestExternalSecretTargetNameUsesExplicitTargetName(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	tc := &TestCase{
		ExternalSecret: &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "external-secret-name",
			},
			Spec: esv1.ExternalSecretSpec{
				Target: esv1.ExternalSecretTarget{
					Name: "explicit-target-name",
				},
			},
		},
	}

	Expect(externalSecretTargetName(tc)).To(Equal("explicit-target-name"))
}

func TestExternalSecretTargetNameFallsBackToExternalSecretName(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	tc := &TestCase{
		ExternalSecret: &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "external-secret-name",
			},
		},
	}

	Expect(externalSecretTargetName(tc)).To(Equal("external-secret-name"))
}
