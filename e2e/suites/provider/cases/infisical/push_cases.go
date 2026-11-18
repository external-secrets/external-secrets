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

package infisical

import (
	"encoding/json"
	"fmt"
	"time"

	//nolint
	. "github.com/onsi/ginkgo/v2"
	//nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

type pushCase func(*framework.Framework) (string, func(*framework.TestCase))

// pushSource points the PushSecret at a fresh opaque source secret.
func pushSource(tc *framework.TestCase, name string, data map[string]string) {
	raw := make(map[string][]byte, len(data))
	for k, v := range data {
		raw[k] = []byte(v)
	}
	tc.PushSecretSource = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: tc.Framework.Namespace.Name},
		Type:       v1.SecretTypeOpaque,
		Data:       raw,
	}
	tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
		Secret: &esv1alpha1.PushSecretSecret{Name: name},
	}
}

func pushData(secretKey, remoteKey, property string) esv1alpha1.PushSecretData {
	return esv1alpha1.PushSecretData{
		Match: esv1alpha1.PushSecretMatch{
			SecretKey: secretKey,
			RemoteRef: esv1alpha1.PushSecretRemoteRef{RemoteKey: remoteKey, Property: property},
		},
	}
}

func waitPushSecretReady(f *framework.Framework, ps *esv1alpha1.PushSecret) {
	Eventually(func() bool {
		got := &esv1alpha1.PushSecret{}
		Expect(f.CRClient.Get(GinkgoT().Context(), types.NamespacedName{
			Name: ps.Name, Namespace: ps.Namespace,
		}, got)).To(Succeed())
		for _, c := range got.Status.Conditions {
			if c.Type == esv1alpha1.PushSecretReady && c.Status == v1.ConditionTrue {
				return true
			}
		}
		return false
	}, time.Minute*2, time.Second*5).Should(BeTrue())
}

// requireRefreshesWithVersion waits for n later successful reconciles (Ready,
// RefreshTime advancing) while the remote version stays at want, so a stalled
// or failing controller cannot satisfy the check.
func requireRefreshesWithVersion(f *framework.Framework, prov *infisicalProvider, ps *esv1alpha1.PushSecret, key string, want, n int) {
	var last metav1.Time
	seen := 0
	Eventually(func() (int, error) {
		got := &esv1alpha1.PushSecret{}
		Expect(f.CRClient.Get(GinkgoT().Context(), types.NamespacedName{
			Name: ps.Name, Namespace: ps.Namespace,
		}, got)).To(Succeed())
		version, err := prov.remoteSecretVersion(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(version).To(Equal(want), "remote version changed for an unchanged value")
		ready := false
		for _, c := range got.Status.Conditions {
			ready = ready || (c.Type == esv1alpha1.PushSecretReady && c.Status == v1.ConditionTrue)
		}
		if last.IsZero() {
			last = got.Status.RefreshTime
		} else if ready && got.Status.RefreshTime.After(last.Time) {
			last = got.Status.RefreshTime
			seen++
		}
		return seen, nil
	}, time.Minute*2, time.Second*2).Should(BeNumerically(">=", n))
}

// eventuallyRemoteValue waits until the remote secret holds want.
func eventuallyRemoteValue(prov *infisicalProvider, key, want string) {
	Eventually(func() (string, error) {
		return prov.remoteSecretValue(key)
	}, time.Minute*2, time.Second*5).Should(Equal(want))
}

// eventuallyRemoteJSON waits until the remote secret is a JSON object of want.
func eventuallyRemoteJSON(prov *infisicalProvider, key string, want map[string]string) {
	Eventually(func() (map[string]string, error) {
		raw, err := prov.remoteSecretValue(key)
		if err != nil {
			return nil, err
		}
		got := map[string]string{}
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			return nil, fmt.Errorf("remote value %q is not a JSON object: %w", raw, err)
		}
		return got, nil
	}, time.Minute*2, time.Second*5).Should(Equal(want))
}

// pushSecretWithProperty pushes two source keys into one remote secret under
// different properties, so the second push has to merge into the first.
func pushSecretWithProperty(prov *infisicalProvider) pushCase {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "[infisical] should push source keys as properties of one JSON secret", func(tc *framework.TestCase) {
			remoteKey := fmt.Sprintf("%s-prop", f.Namespace.Name)
			pushSource(tc, remoteKey+"-src", map[string]string{"user": "alice", "pass": "s3cret"})
			tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{
				pushData("user", remoteKey, "username"),
				pushData("pass", remoteKey, "password"),
			}

			tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				DeferCleanup(func() { prov.deleteRemote(remoteKey) })
				waitPushSecretReady(f, ps)
				eventuallyRemoteJSON(prov, remoteKey, map[string]string{
					"username": "alice",
					"password": "s3cret",
				})
			}
		}
	}
}

// pushSecretWholeSecret omits secretKey, so the provider pushes the whole
// source secret as a JSON object of its string values.
func pushSecretWholeSecret(prov *infisicalProvider) pushCase {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "[infisical] should push the whole source secret as JSON when secretKey is empty", func(tc *framework.TestCase) {
			remoteKey := fmt.Sprintf("%s-whole", f.Namespace.Name)
			src := map[string]string{"host": "db.internal", "port": "5432"}
			pushSource(tc, remoteKey+"-src", src)
			tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{pushData("", remoteKey, "")}

			tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				DeferCleanup(func() { prov.deleteRemote(remoteKey) })
				waitPushSecretReady(f, ps)
				eventuallyRemoteJSON(prov, remoteKey, src)
			}
		}
	}
}

// pushSecretUpdatesExisting changes the source value and expects the remote to
// follow, then lets further refreshes run and expects no new remote version,
// since the provider skips the write when the value is unchanged.
func pushSecretUpdatesExisting(prov *infisicalProvider) pushCase {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "[infisical] should update a changed value and leave an unchanged one alone", func(tc *framework.TestCase) {
			remoteKey := fmt.Sprintf("%s-upd", f.Namespace.Name)
			sourceName := remoteKey + "-src"
			pushSource(tc, sourceName, map[string]string{"credential": "v1"})
			tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{pushData("credential", remoteKey, "")}

			tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				DeferCleanup(func() { prov.deleteRemote(remoteKey) })
				waitPushSecretReady(f, ps)
				eventuallyRemoteValue(prov, remoteKey, "v1")

				src := &v1.Secret{}
				key := types.NamespacedName{Name: sourceName, Namespace: f.Namespace.Name}
				Expect(f.CRClient.Get(GinkgoT().Context(), key, src)).To(Succeed())
				src.Data["credential"] = []byte("v2")
				Expect(f.CRClient.Update(GinkgoT().Context(), src)).To(Succeed())
				eventuallyRemoteValue(prov, remoteKey, "v2")

				// Baseline taken after v2 landed, so an in-flight update
				// cannot move the version during the check.
				version, err := prov.remoteSecretVersion(remoteKey)
				Expect(err).ToNot(HaveOccurred())
				requireRefreshesWithVersion(f, prov, ps, remoteKey, version, 2)
			}
		}
	}
}

// pushSecretViaClusterStore pushes through a ClusterSecretStore reference.
func pushSecretViaClusterStore(prov *infisicalProvider) pushCase {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "[infisical] should push through a ClusterSecretStore", func(tc *framework.TestCase) {
			remoteKey := fmt.Sprintf("%s-cluster", f.Namespace.Name)
			pushSource(tc, remoteKey+"-src", map[string]string{"credential": "cluster-value"})
			tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{pushData("credential", remoteKey, "")}

			tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				DeferCleanup(func() { prov.deleteRemote(remoteKey) })
				waitPushSecretReady(f, ps)
				eventuallyRemoteValue(prov, remoteKey, "cluster-value")
			}
		}
	}
}

// pushSecretIfNotExists seeds one of two remote keys, then pushes both with
// updatePolicy IfNotExists: the seeded one must keep its value (SecretExists
// returns true) and the missing one must be created.
func pushSecretIfNotExists(prov *infisicalProvider) pushCase {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "[infisical] should not overwrite an existing secret with updatePolicy IfNotExists", func(tc *framework.TestCase) {
			existingKey := fmt.Sprintf("%s-keep", f.Namespace.Name)
			newKey := fmt.Sprintf("%s-fresh", f.Namespace.Name)
			DeferCleanup(func() {
				prov.deleteRemote(existingKey)
				prov.deleteRemote(newKey)
			})
			prov.CreateSecret(existingKey, framework.SecretEntry{Value: "original"})

			pushSource(tc, existingKey+"-src", map[string]string{"credential": "overwrite-attempt"})
			tc.PushSecret.Spec.UpdatePolicy = esv1alpha1.PushSecretUpdatePolicyIfNotExists
			tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{
				pushData("credential", existingKey, ""),
				pushData("credential", newKey, ""),
			}

			tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				waitPushSecretReady(f, ps)
				// The new key proves the push ran; the old one must be untouched.
				eventuallyRemoteValue(prov, newKey, "overwrite-attempt")
				Consistently(func() (string, error) {
					return prov.remoteSecretValue(existingKey)
				}, time.Second*20, time.Second*5).Should(Equal("original"))
			}
		}
	}
}
