/*
Copyright © 2025 ESO Maintainer Team

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
package common

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// StatusNotUpdatedAfterSuccessfulSync validates that a successful sync does not trigger
// continuous status updates when the refresh interval is long.
func StatusNotUpdatedAfterSuccessfulSync(f *framework.Framework) (string, func(*framework.TestCase)) {
	return "[regression] should not continuously update status after a successful sync", func(tc *framework.TestCase) {
		remoteKey := f.MakeRemoteRefKey(f.Namespace.Name + "-feedback-loop-key")
		remoteValue := "feedback-loop-value"

		tc.Secrets = map[string]framework.SecretEntry{
			remoteKey: {Value: remoteValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				"value": []byte(remoteValue),
			},
		}
		tc.ExternalSecret.ObjectMeta.Name = "feedback-loop-es"
		tc.ExternalSecret.Spec.RefreshInterval = &metav1.Duration{Duration: time.Hour}
		tc.ExternalSecret.Spec.Target.Name = framework.TargetSecretName
		tc.ExternalSecret.Spec.Data = []esv1.ExternalSecretData{
			{
				SecretKey: "value",
				RemoteRef: esv1.ExternalSecretDataRemoteRef{
					Key: remoteKey,
				},
			},
		}
		tc.AfterSync = func(_ framework.SecretStoreProvider, _ *v1.Secret) {
			key := types.NamespacedName{Name: tc.ExternalSecret.Name, Namespace: tc.ExternalSecret.Namespace}
			var baseline *esv1.ExternalSecret

			Eventually(func() bool {
				current := &esv1.ExternalSecret{}
				if err := tc.Framework.CRClient.Get(GinkgoT().Context(), key, current); err != nil {
					return false
				}

				ready := getExternalSecretCondition(current.Status, esv1.ExternalSecretReady)
				if ready == nil || ready.Status != v1.ConditionTrue {
					return false
				}
				if current.Status.RefreshTime.IsZero() || current.Status.SyncedResourceVersion == "" {
					return false
				}

				if baseline == nil {
					baseline = current.DeepCopy()
					return false
				}

				if current.ResourceVersion != baseline.ResourceVersion {
					baseline = current.DeepCopy()
					return false
				}
				if !current.Status.RefreshTime.Equal(&baseline.Status.RefreshTime) {
					baseline = current.DeepCopy()
					return false
				}
				if current.Status.SyncedResourceVersion != baseline.Status.SyncedResourceVersion {
					baseline = current.DeepCopy()
					return false
				}

				return true
			}).WithTimeout(30 * time.Second).WithPolling(500 * time.Millisecond).Should(BeTrue())

			Consistently(func(g gomega.Gomega) {
				current := &esv1.ExternalSecret{}
				g.Expect(tc.Framework.CRClient.Get(GinkgoT().Context(), key, current)).To(Succeed())
				g.Expect(current.ResourceVersion).To(Equal(baseline.ResourceVersion))
				g.Expect(current.Status.RefreshTime.Equal(&baseline.Status.RefreshTime)).To(BeTrue())
				g.Expect(current.Status.SyncedResourceVersion).To(Equal(baseline.Status.SyncedResourceVersion))
			}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).Should(Succeed())
		}
	}
}

func getExternalSecretCondition(status esv1.ExternalSecretStatus, condType esv1.ExternalSecretConditionType) *esv1.ExternalSecretStatusCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}
