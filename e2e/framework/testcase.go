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
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	//nolint
	"github.com/external-secrets/external-secrets-e2e/framework/log"
	"github.com/external-secrets/external-secrets-e2e/framework/util"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var TargetSecretName = "target-secret"

const (
	createObjectRetryPollInterval = 250 * time.Millisecond
	createObjectRetryTimeout      = 30 * time.Second
)

// TestCase contains the test infra to run a table driven test.
type TestCase struct {
	Framework               *Framework
	ExternalSecret          *esv1.ExternalSecret
	PushSecret              *esv1alpha1.PushSecret
	PushSecretSource        *v1.Secret
	AdditionalObjects       []client.Object
	Secrets                 map[string]SecretEntry
	ExpectedSecret          *v1.Secret
	Prepare                 func(*TestCase, SecretStoreProvider)
	Cleanup                 func()
	ProviderOverride        SecretStoreProvider
	AfterSync               func(SecretStoreProvider, *v1.Secret)
	VerifyPushSecretOutcome func(ps *esv1alpha1.PushSecret, pushClient esv1.SecretsClient)
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

		defer func() {
			if tc.Cleanup != nil {
				tc.Cleanup()
			}
		}()

		prov = prepareTestCase(tc, prov)

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

		// create additional objects
		generateAdditionalObjects(tc)

		// create v1alpha1 external secret, if provided
		createProvidedExternalSecret(tc)

		// wait for Kind=Secret to have the expected data
		executeAfterSync(tc, f, prov)
	}
}

func executeAfterSync(tc *TestCase, f *Framework, prov SecretStoreProvider) {
	if tc.ExpectedSecret != nil {
		secret, err := tc.Framework.WaitForSecretValue(tc.Framework.Namespace.Name, externalSecretTargetName(tc), tc.ExpectedSecret)
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

func externalSecretTargetName(tc *TestCase) string {
	if tc == nil || tc.ExternalSecret == nil {
		return TargetSecretName
	}
	if tc.ExternalSecret.Spec.Target.Name != "" {
		return tc.ExternalSecret.Spec.Target.Name
	}
	if tc.ExternalSecret.Name != "" {
		return tc.ExternalSecret.Name
	}
	return TargetSecretName
}

func generateAdditionalObjects(tc *TestCase) {
	if tc.AdditionalObjects != nil {
		for _, obj := range tc.AdditionalObjects {
			err := tc.Framework.CreateObjectWithRetry(obj)
			Expect(err).ToNot(HaveOccurred())
		}
	}
}

func createProvidedExternalSecret(tc *TestCase) {
	if tc.ExternalSecret == nil {
		return
	}
	err := tc.Framework.CreateObjectWithRetry(tc.ExternalSecret)
	Expect(err).ToNot(HaveOccurred())
}

// TableFuncWithPushSecret returns the main func that runs a TestCase in a table driven test for push secrets.
func TableFuncWithPushSecret(f *Framework, prov SecretStoreProvider, pushClient esv1.SecretsClient) func(...func(*TestCase)) {
	return func(tweaks ...func(*TestCase)) {
		var err error

		// make default test case
		// and apply customization to it
		tc := makeDefaultPushSecretTestCase(f)
		for _, tweak := range tweaks {
			tweak(tc)
		}

		prov = prepareTestCase(tc, prov)

		// additional objects
		generateAdditionalObjects(tc)

		if tc.PushSecretSource != nil {
			err := tc.Framework.CreateObjectWithRetry(tc.PushSecretSource)
			Expect(err).ToNot(HaveOccurred())
		}

		// create v1alpha1 push secret, if provided
		if tc.PushSecret != nil {
			// create v1beta1 external secret otherwise
			err = tc.Framework.CreateObjectWithRetry(tc.PushSecret)
			Expect(err).ToNot(HaveOccurred())
		}

		// Run verification on the secret that push secret created or not.
		tc.VerifyPushSecretOutcome(tc.PushSecret, pushClient)
	}
}

func prepareTestCase(tc *TestCase, prov SecretStoreProvider) SecretStoreProvider {
	prov = effectiveTestCaseProvider(tc, prov)
	if tc.Prepare != nil {
		tc.Prepare(tc, prov)
	}
	return effectiveTestCaseProvider(tc, prov)
}

func effectiveTestCaseProvider(tc *TestCase, prov SecretStoreProvider) SecretStoreProvider {
	if tc.ProviderOverride != nil {
		return tc.ProviderOverride
	}
	return prov
}

func makeDefaultExternalSecretTestCase(f *Framework) *TestCase {
	return &TestCase{
		AfterSync: func(ssp SecretStoreProvider, s *v1.Secret) {},
		Framework: f,
		ExternalSecret: &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "e2e-es",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1.ExternalSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: time.Second * 5},
				SecretStoreRef: esv1.SecretStoreRef{
					Name: f.Namespace.Name,
					Kind: f.DefaultSecretStoreRefKind,
				},
				Target: esv1.ExternalSecretTarget{
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
						Name:       f.Namespace.Name,
						Kind:       f.DefaultPushSecretStoreRefKind,
						APIVersion: f.DefaultPushSecretStoreRefAPIVersion,
					},
				},
			},
		},
	}
}

func (f *Framework) CreateObjectWithRetry(obj client.Object) error {
	return f.CreateObjectWithRetryContext(GinkgoT().Context(), obj)
}

func (f *Framework) CreateObjectWithRetryContext(ctx context.Context, obj client.Object) error {
	return wait.PollUntilContextTimeout(ctx, createObjectRetryPollInterval, createObjectRetryTimeout, true, func(ctx context.Context) (bool, error) {
		err := f.CRClient.Create(ctx, obj)
		switch {
		case err == nil, apierrors.IsAlreadyExists(err):
			return true, nil
		case isRetryableCreateObjectError(err):
			if f.KubeConfig != nil {
				f.refreshClients()
			}
			return false, nil
		default:
			return false, err
		}
	})
}

func createObjectWithRetryPolling(ctx context.Context, c client.Client, obj client.Object, pollInterval, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		err := c.Create(ctx, obj)
		switch {
		case err == nil, apierrors.IsAlreadyExists(err):
			return true, nil
		case isRetryableCreateObjectError(err):
			return false, nil
		default:
			return false, err
		}
	})
}

func isRetryableCreateObjectError(err error) bool {
	if util.IsMissingAPIResourceError(err) {
		return true
	}
	if apierrors.IsNotFound(err) && strings.Contains(err.Error(), "could not find the requested resource") {
		return true
	}

	if !(apierrors.IsInternalError(err) || apierrors.IsServiceUnavailable(err)) {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "failed calling webhook") &&
		(strings.Contains(msg, "connection refused") || strings.Contains(msg, "no endpoints available"))
}
