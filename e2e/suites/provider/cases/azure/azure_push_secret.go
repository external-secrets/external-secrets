/*
Copyright © The ESO Authors

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

package azure

import (
	"fmt"
	"time"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	// nolint
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

const (
	softDeleteInitialValue = "initial-value"
	softDeleteUpdatedValue = "updated-after-recovery"
)

var _ = Describe("[azure]", Label("azure", "keyvault", "pushsecret"), func() {
	f := framework.New("eso-azure-push")
	prov := newFromEnv(f)

	DescribeTable("recover soft-deleted secrets",
		framework.TableFuncWithPushSecret(f, prov, nil),
		framework.Compose(withStaticCredentials, f, recoverSoftDeletedPushSecret(prov), useStaticCredentialsForPush),
		framework.Compose(withNewSDK, f, recoverSoftDeletedPushSecret(prov), useNewSDKForPush),
	)
})

func recoverSoftDeletedPushSecret(prov *azureProvider) func(*framework.Framework) (string, func(*framework.TestCase)) {
	return func(f *framework.Framework) (string, func(*framework.TestCase)) {
		return "should recover a soft-deleted secret and update it on a later reconciliation", func(tc *framework.TestCase) {
			sourceName := fmt.Sprintf("%s-source", f.Namespace.Name)
			remoteKey := fmt.Sprintf("%s-soft-delete", f.Namespace.Name)

			tc.PushSecretSource = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: sourceName, Namespace: f.Namespace.Name},
				Data:       map[string][]byte{"value": []byte(softDeleteInitialValue)},
			}
			tc.PushSecret.Spec.Selector = esv1alpha1.PushSecretSelector{
				Secret: &esv1alpha1.PushSecretSecret{Name: sourceName},
			}
			tc.PushSecret.Spec.Data = []esv1alpha1.PushSecretData{
				{
					Match: esv1alpha1.PushSecretMatch{
						SecretKey: "value",
						RemoteRef: esv1alpha1.PushSecretRemoteRef{RemoteKey: remoteKey},
					},
					Metadata: &apiextensionsv1.JSON{Raw: []byte(`{"contentType":"text/plain","tags":{"e2e":"soft-delete-recovery"}}`)},
				},
			}

			tc.VerifyPushSecretOutcome = func(_ *esv1alpha1.PushSecret, _ esv1.SecretsClient) {
				verifySoftDeleteRecovery(tc, prov, remoteKey)
			}
		}
	}
}

func useStaticCredentialsForPush(tc *framework.TestCase) {
	tc.PushSecret.Spec.SecretStoreRefs = []esv1alpha1.PushSecretStoreRef{{Name: tc.Framework.Namespace.Name}}
}

func useNewSDKForPush(tc *framework.TestCase) {
	tc.PushSecret.Spec.SecretStoreRefs = []esv1alpha1.PushSecretStoreRef{{Name: tc.Framework.Namespace.Name + "-new-sdk"}}
}

func verifySoftDeleteRecovery(tc *framework.TestCase, prov *azureProvider, remoteKey string) {
	Eventually(func() error {
		secret, err := prov.client.GetSecret(GinkgoT().Context(), prov.vaultURL, remoteKey, "")
		if err != nil {
			return err
		}
		if secret.Value == nil || *secret.Value != softDeleteInitialValue {
			return fmt.Errorf("secret value = %v, want %q", secret.Value, softDeleteInitialValue)
		}
		return nil
	}, time.Minute*2, time.Second*5).Should(Succeed())

	DeferCleanup(func() {
		if _, err := prov.client.GetSecret(GinkgoT().Context(), prov.vaultURL, remoteKey, ""); err == nil {
			_, err = prov.client.DeleteSecret(GinkgoT().Context(), prov.vaultURL, remoteKey)
			Expect(err).ToNot(HaveOccurred())
		}
		Eventually(func() error {
			_, err := prov.client.GetDeletedSecret(GinkgoT().Context(), prov.vaultURL, remoteKey)
			return err
		}, time.Minute*2, time.Second*5).Should(Succeed())
		_, err := prov.client.PurgeDeletedSecret(GinkgoT().Context(), prov.vaultURL, remoteKey)
		Expect(err).ToNot(HaveOccurred())
	})

	_, err := prov.client.DeleteSecret(GinkgoT().Context(), prov.vaultURL, remoteKey)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		_, err := prov.client.GetDeletedSecret(GinkgoT().Context(), prov.vaultURL, remoteKey)
		return err
	}, time.Minute*2, time.Second*5).Should(Succeed())

	source := &corev1.Secret{}
	Expect(tc.Framework.CRClient.Get(GinkgoT().Context(), types.NamespacedName{
		Name:      tc.PushSecretSource.Name,
		Namespace: tc.PushSecretSource.Namespace,
	}, source)).To(Succeed())
	source.Data["value"] = []byte(softDeleteUpdatedValue)
	Expect(tc.Framework.CRClient.Update(GinkgoT().Context(), source)).To(Succeed())

	Eventually(func() error {
		secret, err := prov.client.GetSecret(GinkgoT().Context(), prov.vaultURL, remoteKey, "")
		if err != nil {
			return err
		}
		if secret.Value == nil || *secret.Value != softDeleteUpdatedValue {
			return fmt.Errorf("secret value = %v, want %q", secret.Value, softDeleteUpdatedValue)
		}
		if secret.ContentType == nil || *secret.ContentType != "text/plain" {
			return fmt.Errorf("secret content type = %v, want text/plain", secret.ContentType)
		}
		for key, value := range map[string]string{
			"managed-by": "external-secrets",
			"e2e":        "soft-delete-recovery",
		} {
			actual, ok := secret.Tags[key]
			if !ok || actual == nil || *actual != value {
				return fmt.Errorf("secret tag %q = %v, want %q", key, actual, value)
			}
		}
		return nil
	}, time.Minute*3, time.Second*5).Should(Succeed())
}
