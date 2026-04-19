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
	"context"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1api "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

	. "github.com/onsi/gomega"
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

type flakyCreateClient struct {
	client.Client
	createErrs  []error
	createCalls int
}

func (c *flakyCreateClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	callIndex := c.createCalls
	c.createCalls++
	if callIndex < len(c.createErrs) && c.createErrs[callIndex] != nil {
		return c.createErrs[callIndex]
	}
	return c.Client.Create(ctx, obj, opts...)
}

func TestCreateObjectWithRetryPollingRetriesMissingAPIResourceErrors(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	Expect(esv1.AddToScheme(scheme)).To(Succeed())

	baseClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	cl := &flakyCreateClient{
		Client: baseClient,
		createErrs: []error{
			&metav1api.NoResourceMatchError{},
		},
	}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "retry-external-secret",
			Namespace: "default",
		},
	}

	err := createObjectWithRetryPolling(context.Background(), cl, es, 5*time.Millisecond, 100*time.Millisecond)
	Expect(err).NotTo(HaveOccurred())
	Expect(cl.createCalls).To(Equal(2))

	var created esv1.ExternalSecret
	Expect(baseClient.Get(context.Background(), client.ObjectKeyFromObject(es), &created)).To(Succeed())
}

func TestCreateObjectWithRetryPollingRetriesMissingResourceEndpointErrors(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	Expect(esv1.AddToScheme(scheme)).To(Succeed())

	baseClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	cl := &flakyCreateClient{
		Client: baseClient,
		createErrs: []error{
			&apierrors.StatusError{ErrStatus: metav1.Status{
				Status:  metav1.StatusFailure,
				Message: "the server could not find the requested resource (post externalsecrets.external-secrets.io)",
				Reason:  metav1.StatusReasonNotFound,
				Code:    404,
			}},
		},
	}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "retry-endpoint-external-secret",
			Namespace: "default",
		},
	}

	err := createObjectWithRetryPolling(context.Background(), cl, es, 5*time.Millisecond, 100*time.Millisecond)
	Expect(err).NotTo(HaveOccurred())
	Expect(cl.createCalls).To(Equal(2))

	var created esv1.ExternalSecret
	Expect(baseClient.Get(context.Background(), client.ObjectKeyFromObject(es), &created)).To(Succeed())
}
