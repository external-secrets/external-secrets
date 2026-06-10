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
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	infisicalSdk "github.com/infisical/go-sdk"
	sdkErrors "github.com/infisical/go-sdk/packages/errors"
	//nolint
	. "github.com/onsi/ginkgo/v2"
	//nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

// pushSecretValue pushes a single value through the provider's PushSecret
// implementation, then reads it back with an ExternalSecret to confirm the
// round-trip. The remote key is namespaced so parallel specs sharing the e2e
// project do not collide.
func pushSecretValue(prov *infisicalProvider) func(*framework.Framework) (string, func(*framework.TestCase)) {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "[infisical] should push a secret and read it back", func(tc *framework.TestCase) {
			sourceName := fmt.Sprintf("%s-src", f.Namespace.Name)
			remoteKey := fmt.Sprintf("%s-pushed", f.Namespace.Name)
			value := "pushed-value"

			tc.PushSecretSource = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sourceName,
					Namespace: f.Namespace.Name,
				},
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{"credential": []byte(value)},
			}
			tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
				Secret: &esv1alpha1.PushSecretSecret{Name: sourceName},
			}
			tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{
				{
					Match: esv1alpha1.PushSecretMatch{
						SecretKey: "credential",
						RemoteRef: esv1alpha1.PushSecretRemoteRef{RemoteKey: remoteKey},
					},
				},
			}

			tc.VerifyPushSecretOutcome = func(_ *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				// Remove the pushed secret from the shared project so re-runs
				// start clean, regardless of controller teardown ordering.
				DeferCleanup(func() { prov.deleteRemote(remoteKey) })

				Eventually(func() bool {
					ps := &esv1alpha1.PushSecret{}
					err := f.CRClient.Get(GinkgoT().Context(), types.NamespacedName{
						Name:      tc.PushSecret.Name,
						Namespace: tc.PushSecret.Namespace,
					}, ps)
					Expect(err).ToNot(HaveOccurred())
					for i := range ps.Status.Conditions {
						c := ps.Status.Conditions[i]
						if c.Type == esv1alpha1.PushSecretReady && c.Status == v1.ConditionTrue {
							return true
						}
					}
					return false
				}, time.Minute*2, time.Second*5).Should(BeTrue())

				// Read the pushed value back through an ExternalSecret.
				const target = "push-readback"
				es := &esv1.ExternalSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "e2e-push-es",
						Namespace: f.Namespace.Name,
					},
					Spec: esv1.ExternalSecretSpec{
						RefreshInterval: &metav1.Duration{Duration: time.Second * 5},
						SecretStoreRef:  esv1.SecretStoreRef{Name: f.Namespace.Name},
						Target:          esv1.ExternalSecretTarget{Name: target},
						Data: []esv1.ExternalSecretData{
							{
								SecretKey: target,
								RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: remoteKey},
							},
						},
					},
				}
				Expect(f.CRClient.Create(GinkgoT().Context(), es)).ToNot(HaveOccurred())

				readBack := &v1.Secret{}
				err := wait.PollUntilContextTimeout(GinkgoT().Context(), time.Second*5, time.Minute*2, true, func(ctx context.Context) (bool, error) {
					gerr := f.CRClient.Get(ctx, types.NamespacedName{Namespace: f.Namespace.Name, Name: target}, readBack)
					if apierrors.IsNotFound(gerr) {
						return false, nil
					}
					return gerr == nil, gerr
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(string(readBack.Data[target])).To(Equal(value))
			}
		}
	}
}

// pushSecretDeletesOnPolicy pushes a secret with deletionPolicy: Delete, then
// deletes the PushSecret and confirms the operator removed the secret from
// Infisical through the provider's DeleteSecret.
func pushSecretDeletesOnPolicy(prov *infisicalProvider) func(*framework.Framework) (string, func(*framework.TestCase)) {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "[infisical] should delete the remote secret when the PushSecret is deleted", func(tc *framework.TestCase) {
			sourceName := fmt.Sprintf("%s-del-src", f.Namespace.Name)
			remoteKey := fmt.Sprintf("%s-del", f.Namespace.Name)
			value := "delete-me"

			tc.PushSecretSource = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: sourceName, Namespace: f.Namespace.Name},
				Type:       v1.SecretTypeOpaque,
				Data:       map[string][]byte{"credential": []byte(value)},
			}
			tc.PushSecret.Spec.DeletionPolicy = esv1alpha1.PushSecretDeletionPolicyDelete
			tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
				Secret: &esv1alpha1.PushSecretSecret{Name: sourceName},
			}
			tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{
				{
					Match: esv1alpha1.PushSecretMatch{
						SecretKey: "credential",
						RemoteRef: esv1alpha1.PushSecretRemoteRef{RemoteKey: remoteKey},
					},
				},
			}

			tc.VerifyPushSecretOutcome = func(ps *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				// Best-effort guard in case the deletion assertion below fails.
				DeferCleanup(func() { prov.deleteRemote(remoteKey) })

				// The push lands the secret in Infisical.
				Eventually(func() error {
					_, err := prov.remoteSecretValue(remoteKey)
					return err
				}, time.Minute*2, time.Second*5).Should(Succeed())

				// Deleting the PushSecret must drive the provider's DeleteSecret
				// (deletionPolicy: Delete), removing the secret from Infisical.
				Expect(f.CRClient.Delete(GinkgoT().Context(), ps)).To(Succeed())

				// Wait until the secret is confirmed absent. Using
				// Should(Succeed()) on a helper that returns nil only on
				// "not found" avoids a false pass on transient API errors
				// (which ShouldNot(Succeed()) would also accept).
				Eventually(func() error {
					_, err := prov.remoteSecretValue(remoteKey)
					if err == nil {
						return fmt.Errorf("secret %q still exists in Infisical", remoteKey)
					}
					if isInfisicalNotFound(err) {
						return nil
					}
					return err
				}, time.Minute*2, time.Second*5).Should(Succeed())
			}
		}
	}
}

// isInfisicalNotFound reports whether err is a 404 from the Infisical API,
// meaning the secret is definitively absent rather than transiently unavailable.
func isInfisicalNotFound(err error) bool {
	var apiErr *sdkErrors.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// remoteSecretValue reads a secret from the e2e project by slug via the addon
// SDK; it returns an error when the secret is absent.
func (s *infisicalProvider) remoteSecretValue(key string) (string, error) {
	secretPath, name := secretAddress(scopePath, key)
	secret, err := s.addon.SDKClient.Secrets().Retrieve(infisicalSdk.RetrieveSecretOptions{
		ProjectSlug: s.addon.ProjectSlug,
		Environment: s.addon.EnvironmentSlug,
		SecretKey:   name,
		SecretPath:  secretPath,
	})
	if err != nil {
		return "", err
	}
	return secret.SecretValue, nil
}

// deleteRemote best-effort removes a secret from the shared e2e project via the
// addon SDK, used to clean up after a push spec.
func (s *infisicalProvider) deleteRemote(key string) {
	secretPath, name := secretAddress(scopePath, key)
	_, _ = s.addon.SDKClient.Secrets().Delete(infisicalSdk.DeleteSecretOptions{
		ProjectID:   s.addon.ProjectID,
		Environment: s.addon.EnvironmentSlug,
		SecretPath:  secretPath,
		SecretKey:   name,
	})
}
