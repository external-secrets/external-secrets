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

package kubernetes

import (
	"context"
	"fmt"
	"time"

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

const (
	foreignFinalizer = "e2e.external-secrets.io/foreign"
	sourceName       = "e2e-ps-source"
	dataKey          = "value"
	pollInterval     = 100 * time.Millisecond
)

func regressionNoPushWhileDeletingPolicyNone(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[regression] should not overwrite the pushed secret while a PushSecret with deletionPolicy=None is being deleted", func(tc *framework.TestCase) {
		setupPushSecretWithForeignFinalizer(tc, esv1alpha1.PushSecretDeletionPolicyNone)
		tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
			pushedSecretHasV1 := expectPushedSecretValue(f, "v1")
			Eventually(pushedSecretHasV1, time.Minute, pollInterval).Should(Succeed())

			deleteAndWaitForDeletionTimestamp(f, ps)
			triggerReconcileWithNewSourceValue(f, ps)
			Consistently(pushedSecretHasV1, time.Second, pollInterval).Should(Succeed())

			removeForeignFinalizerAndWaitForDeletion(f, ps)
			pushedSecretHasV1(Default)
		}
	}
}

func regressionNoPushWhileDeletingPolicyDelete(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[regression] should not recreate the pushed secret while a PushSecret with deletionPolicy=Delete is being deleted", func(tc *framework.TestCase) {
		setupPushSecretWithForeignFinalizer(tc, esv1alpha1.PushSecretDeletionPolicyDelete)
		tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
			pushedSecretAbsent := expectPushedSecretAbsent(f)
			Eventually(expectPushedSecretValue(f, "v1"), time.Minute, pollInterval).Should(Succeed())

			deleteAndWaitForDeletionTimestamp(f, ps)
			Eventually(pushedSecretAbsent, time.Minute, pollInterval).Should(Succeed())
			triggerReconcileWithNewSourceValue(f, ps)
			Consistently(pushedSecretAbsent, time.Second, pollInterval).Should(Succeed())

			removeForeignFinalizerAndWaitForDeletion(f, ps)
			pushedSecretAbsent(Default)
		}
	}
}

func setupPushSecretWithForeignFinalizer(tc *framework.TestCase, policy esv1alpha1.PushSecretDeletionPolicy) {
	f := tc.Framework
	tc.PushSecretSource = &v1.Secret{
		Name: sourceName, Namespace: f.Namespace.Name,
		Data: map[string][]byte{dataKey: []byte("v1")},
	}
	tc.PushSecret.Finalizers = []string{foreignFinalizer}
	tc.PushSecret.Spec.DeletionPolicy = policy
	tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
		Secret: &esv1alpha1.PushSecretSecret{Name: sourceName},
	}
	tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{
		{
			Match: esv1alpha1.PushSecretMatch{
				SecretKey: dataKey,
				RemoteRef: esv1alpha1.PushSecretRemoteRef{RemoteKey: pushedSecretName(f), Property: dataKey},
			},
		},
	}

	// remove the foreign finalizer if the test fails halfway, otherwise namespace teardown hangs
	psKey := client.ObjectKeyFromObject(tc.PushSecret)
	DeferCleanup(func() {
		// Cannot use the ginkgo context inside DeferCleanup, it would register
		// another cleanup. Use a plain context with a short timeout instead.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		ps := &esv1alpha1.PushSecret{}
		if err := f.CRClient.Get(ctx, psKey, ps); err != nil {
			return
		}
		patch := client.MergeFrom(ps.DeepCopy())
		ps.Finalizers = nil
		_ = f.CRClient.Patch(ctx, ps, patch)
	})
}

func pushedSecretName(f *framework.Framework) string {
	return fmt.Sprintf("e2e-ps-pushed-%s", f.Namespace.Name)
}

func getPushedSecret(f *framework.Framework) (*v1.Secret, error) {
	secret := &v1.Secret{}
	err := f.CRClient.Get(GinkgoT().Context(), client.ObjectKey{Name: pushedSecretName(f), Namespace: f.Namespace.Name}, secret)
	return secret, err
}

func expectPushedSecretValue(f *framework.Framework, want string) func(Gomega) {
	return func(g Gomega) {
		secret, err := getPushedSecret(f)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(secret.Data[dataKey])).To(Equal(want))
	}
}

func expectPushedSecretAbsent(f *framework.Framework) func(Gomega) {
	return func(g Gomega) {
		_, err := getPushedSecret(f)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "expected pushed secret to be absent, got err %v", err)
	}
}

func deleteAndWaitForDeletionTimestamp(f *framework.Framework, ps *esv1alpha1.PushSecret) {
	Expect(f.CRClient.Delete(GinkgoT().Context(), ps)).To(Succeed())
	Eventually(func(g Gomega) {
		g.Expect(f.CRClient.Get(GinkgoT().Context(), client.ObjectKeyFromObject(ps), ps)).To(Succeed())
		g.Expect(ps.DeletionTimestamp).NotTo(BeNil())
		g.Expect(ps.Finalizers).To(ConsistOf(foreignFinalizer))
	}, time.Minute, pollInterval).Should(Succeed())
}

func triggerReconcileWithNewSourceValue(f *framework.Framework, ps *esv1alpha1.PushSecret) {
	source := &v1.Secret{}
	Expect(f.CRClient.Get(GinkgoT().Context(), client.ObjectKey{Name: sourceName, Namespace: f.Namespace.Name}, source)).To(Succeed())
	source.Data[dataKey] = []byte("v2")
	Expect(f.CRClient.Update(GinkgoT().Context(), source)).To(Succeed())

	patch := client.MergeFrom(ps.DeepCopy())
	ps.Annotations = map[string]string{"e2e.external-secrets.io/trigger": "reconcile"}
	Expect(f.CRClient.Patch(GinkgoT().Context(), ps, patch)).To(Succeed())
}

func removeForeignFinalizerAndWaitForDeletion(f *framework.Framework, ps *esv1alpha1.PushSecret) {
	Expect(f.CRClient.Get(GinkgoT().Context(), client.ObjectKeyFromObject(ps), ps)).To(Succeed())
	patch := client.MergeFrom(ps.DeepCopy())
	ps.Finalizers = nil
	Expect(f.CRClient.Patch(GinkgoT().Context(), ps, patch)).To(Succeed())
	Eventually(func() bool {
		return apierrors.IsNotFound(f.CRClient.Get(GinkgoT().Context(), client.ObjectKeyFromObject(ps), &esv1alpha1.PushSecret{}))
	}, time.Minute, pollInterval).Should(BeTrue())
}
