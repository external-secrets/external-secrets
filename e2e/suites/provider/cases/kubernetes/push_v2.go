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

package kubernetes

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

var _ = Describe("[kubernetes] v2 push secret", Label("kubernetes", "v2", "push-secret"), func() {
	f := framework.New("eso-kubernetes-v2-push")
	NewProvider(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
		frameworkv2.WaitForProviderConnectionReady(f, f.Namespace.Name, f.Namespace.Name, defaultV2WaitTimeout)
	})

	It("preserves source secret type, labels, and annotations when pushing to the namespaced Provider", func() {
		sourceSecret := &corev1.Secret{
			Type: corev1.SecretTypeDockerConfigJson,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-secret-metadata",
				Namespace: f.Namespace.Name,
				Labels: map[string]string{
					"team": "platform",
				},
				Annotations: map[string]string{
					"owner": "eso",
				},
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"registry.example.com":{"auth":"ZXNvOnNlY3JldA=="}}}`),
			},
		}
		Expect(f.CRClient.Create(context.Background(), sourceSecret)).To(Succeed())

		pushSecret := &esv1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pushsecret-metadata",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.PushSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
				SecretStoreRefs: []esv1alpha1.PushSecretStoreRef{
					{
						Name:       f.Namespace.Name,
						Kind:       f.DefaultPushSecretStoreRefKind,
						APIVersion: f.DefaultPushSecretStoreRefAPIVersion,
					},
				},
				Selector: esv1alpha1.PushSecretSelector{
					Secret: &esv1alpha1.PushSecretSecret{
						Name: sourceSecret.Name,
					},
				},
				Data: []esv1alpha1.PushSecretData{{
					Match: esv1alpha1.PushSecretMatch{
						SecretKey: corev1.DockerConfigJsonKey,
						RemoteRef: esv1alpha1.PushSecretRemoteRef{
							RemoteKey: "pushed-docker-secret",
							Property:  corev1.DockerConfigJsonKey,
						},
					},
				}},
			},
		}
		Expect(f.CRClient.Create(context.Background(), pushSecret)).To(Succeed())

		waitForPushSecretReady(f, pushSecret.Name)

		var pushedSecret corev1.Secret
		Eventually(func(g Gomega) {
			g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{Name: "pushed-docker-secret", Namespace: f.Namespace.Name}, &pushedSecret)).To(Succeed())
		}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())

		Expect(pushedSecret.Type).To(Equal(corev1.SecretTypeDockerConfigJson))
		Expect(pushedSecret.Labels).To(Equal(sourceSecret.Labels))
		Expect(pushedSecret.Annotations).To(Equal(sourceSecret.Annotations))
		Expect(pushedSecret.Data).To(Equal(sourceSecret.Data))
	})

	It("supports namespaced Provider refs when kind is omitted", func() {
		sourceSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-secret-implicit-kind",
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"key1": []byte("value1"),
			},
		}
		Expect(f.CRClient.Create(context.Background(), sourceSecret)).To(Succeed())

		pushSecret := &esv1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pushsecret-implicit-kind",
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.PushSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
				DeletionPolicy:  esv1alpha1.PushSecretDeletionPolicyDelete,
				SecretStoreRefs: []esv1alpha1.PushSecretStoreRef{
					{
						Name:       f.Namespace.Name,
						APIVersion: f.DefaultPushSecretStoreRefAPIVersion,
					},
				},
				Selector: esv1alpha1.PushSecretSelector{
					Secret: &esv1alpha1.PushSecretSecret{
						Name: sourceSecret.Name,
					},
				},
				Data: []esv1alpha1.PushSecretData{
					{
						Match: esv1alpha1.PushSecretMatch{
							SecretKey: "key1",
							RemoteRef: esv1alpha1.PushSecretRemoteRef{
								RemoteKey: "pushed-secret-implicit-kind",
								Property:  "key1",
							},
						},
					},
				},
			},
		}
		Expect(f.CRClient.Create(context.Background(), pushSecret)).To(Succeed())
		waitForPushSecretReady(f, pushSecret.Name)

		Eventually(func(g Gomega) {
			var pushedSecret corev1.Secret
			g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{Name: "pushed-secret-implicit-kind", Namespace: f.Namespace.Name}, &pushedSecret)).To(Succeed())
			g.Expect(string(pushedSecret.Data["key1"])).To(Equal("value1"))
		}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())

		Expect(f.CRClient.Delete(context.Background(), pushSecret)).To(Succeed())

		Eventually(func() bool {
			var pushedSecret corev1.Secret
			err := f.CRClient.Get(context.Background(), types.NamespacedName{Name: "pushed-secret-implicit-kind", Namespace: f.Namespace.Name}, &pushedSecret)
			return apierrors.IsNotFound(err)
		}, defaultV2WaitTimeout, defaultV2PollInterval).Should(BeTrue())
	})
})

func waitForPushSecretReady(f *framework.Framework, name string) {
	Eventually(func(g Gomega) {
		var ps esv1alpha1.PushSecret
		g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: f.Namespace.Name}, &ps)).To(Succeed())
		g.Expect(ps.Status.Conditions).NotTo(BeEmpty())
		ready := false
		for _, condition := range ps.Status.Conditions {
			if condition.Type == esv1alpha1.PushSecretReady && condition.Status == corev1.ConditionTrue {
				ready = true
			}
		}
		g.Expect(ready).To(BeTrue())
	}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())
}
