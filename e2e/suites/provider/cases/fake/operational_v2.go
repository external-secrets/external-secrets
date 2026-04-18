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

package fake

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	"github.com/external-secrets/external-secrets-e2e/suites/provider/cases/common"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

var _ = Describe("[fake] v2 operational", Serial, Label("fake", "v2", "operational"), func() {
	f := framework.New("eso-fake-v2-operational")
	prov := NewProviderV2(f)

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
	})

	DescribeTable("external secret operational behavior",
		framework.TableFuncWithExternalSecret(f, prov),
		Entry(common.NamespacedProviderUnavailable(f, newFakeOperationalExternalSecretHarness(f, prov), "fake-operational-unavailable", "recovered")),
		Entry(common.NamespacedProviderRestart(f, newFakeOperationalExternalSecretHarness(f, prov), "fake-operational-restart", "restarted")),
		Entry(common.ClusterProviderUnavailable(f, newFakeOperationalExternalSecretHarness(f, prov), "fake-operational-cluster-unavailable", "cluster-recovered", esv1.AuthenticationScopeManifestNamespace)),
		Entry(common.ClusterProviderRestart(f, newFakeOperationalExternalSecretHarness(f, prov), "fake-operational-cluster-restart", "cluster-restarted", esv1.AuthenticationScopeManifestNamespace)),
	)

	DescribeTable("push secret operational behavior",
		framework.TableFuncWithPushSecret(f, prov, nil),
		Entry(common.NamespacedPushSecretUnavailable(f, newFakeOperationalPushHarness(f, prov))),
		Entry(common.ClusterProviderPushUnavailable(f, newFakeOperationalPushHarness(f, prov), esv1.AuthenticationScopeManifestNamespace)),
	)

	It("reuses one backend connection across many namespaced fake Provider consumers", func() {
		const consumerCount = 10

		for i := 0; i < consumerCount; i++ {
			remoteKey := fmt.Sprintf("fake-operational-consumer-%d", i)
			targetName := fmt.Sprintf("fake-operational-consumer-target-%d", i)
			expectedValue := fmt.Sprintf("value-%d", i)

			prov.CreateSecret(remoteKey, framework.SecretEntry{
				Value: fmt.Sprintf(`{"value":"%s"}`, expectedValue),
			})

			Expect(f.CreateObjectWithRetry(&esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("fake-operational-consumer-es-%d", i),
					Namespace: f.Namespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
					SecretStoreRef: esv1.SecretStoreRef{
						Name: f.Namespace.Name,
						Kind: esv1.ProviderStoreKindStr,
					},
					Target: esv1.ExternalSecretTarget{
						Name: targetName,
					},
					Data: []esv1.ExternalSecretData{{
						SecretKey: "value",
						RemoteRef: esv1.ExternalSecretDataRemoteRef{
							Key:      remoteKey,
							Property: "value",
						},
					}},
				},
			})).To(Succeed())

			_, err := f.WaitForSecretValue(f.Namespace.Name, targetName, &corev1.Secret{
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"value": []byte(expectedValue),
				},
			})
			Expect(err).NotTo(HaveOccurred())
		}

		Eventually(func(g Gomega) {
			metrics, err := frameworkv2.ScrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet, frameworkv2.ProviderNamespace)
			g.Expect(err).NotTo(HaveOccurred())

			total := frameworkv2.SumMetricValues(metrics, "grpc_pool_connections_total", map[string]string{
				"address": frameworkv2.ProviderAddress("fake"),
			})
			g.Expect(total).To(BeNumerically(">=", 1))
			g.Expect(total).To(BeNumerically("<=", 2), "expected bounded connection reuse for one backend")
			g.Expect(total).To(BeNumerically("<", consumerCount), "expected fewer pooled connections than consumers")
		}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())
	})

	It("reuses backend connections across multiple fake Provider resources that share one backend", func() {
		const providerCount = 4

		for i := 0; i < providerCount; i++ {
			providerName := fmt.Sprintf("fake-fanout-provider-%d", i)
			remoteKey := fmt.Sprintf("fake-fanout-remote-%d", i)
			expectedValue := fmt.Sprintf("fanout-%d", i)
			targetName := fmt.Sprintf("fake-fanout-target-%d", i)

			frameworkv2.CreateProviderConnection(
				f,
				f.Namespace.Name,
				providerName,
				frameworkv2.ProviderAddress("fake"),
				fakeProviderAPIVersion,
				fakeProviderKind,
				f.Namespace.Name,
				"",
			)
			frameworkv2.WaitForProviderConnectionReady(f, f.Namespace.Name, providerName, defaultV2WaitTimeout)

			prov.CreateSecret(remoteKey, framework.SecretEntry{
				Value: fmt.Sprintf(`{"value":"%s"}`, expectedValue),
			})

			Expect(f.CreateObjectWithRetry(&esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("fake-fanout-es-%d", i),
					Namespace: f.Namespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
					SecretStoreRef: esv1.SecretStoreRef{
						Name: providerName,
						Kind: esv1.ProviderStoreKindStr,
					},
					Target: esv1.ExternalSecretTarget{
						Name: targetName,
					},
					Data: []esv1.ExternalSecretData{{
						SecretKey: "value",
						RemoteRef: esv1.ExternalSecretDataRemoteRef{
							Key:      remoteKey,
							Property: "value",
						},
					}},
				},
			})).To(Succeed())

			_, err := f.WaitForSecretValue(f.Namespace.Name, targetName, &corev1.Secret{
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"value": []byte(expectedValue),
				},
			})
			Expect(err).NotTo(HaveOccurred())
		}

		Eventually(func(g Gomega) {
			metrics, err := frameworkv2.ScrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet, frameworkv2.ProviderNamespace)
			g.Expect(err).NotTo(HaveOccurred())

			total := frameworkv2.SumMetricValues(metrics, "grpc_pool_connections_total", map[string]string{
				"address": frameworkv2.ProviderAddress("fake"),
			})
			g.Expect(total).To(BeNumerically(">=", 1))
			g.Expect(total).To(BeNumerically("<=", 2), "expected bounded connection reuse across shared backend fanout")
			g.Expect(total).To(BeNumerically("<", providerCount), "expected fewer pooled connections than Provider resources")
		}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())
	})

	It("recovers generator-backed push traffic after fake provider outage", func() {
		const (
			generatorName   = "fake-operational-generator"
			pushSecretName  = "fake-operational-generator-push"
			remoteSecretKey = "fake-operational-generator-remote"
		)

		generator := &genv1alpha1.Fake{
			TypeMeta: metav1.TypeMeta{
				APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version,
				Kind:       genv1alpha1.FakeKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      generatorName,
				Namespace: f.Namespace.Name,
			},
			Spec: genv1alpha1.FakeSpec{
				Data: map[string]string{
					"value": "before-outage",
				},
			},
		}
		Expect(f.CreateObjectWithRetry(generator)).To(Succeed())

		pushSecret := &esv1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pushSecretName,
				Namespace: f.Namespace.Name,
			},
			Spec: esv1alpha1.PushSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
				SecretStoreRefs: []esv1alpha1.PushSecretStoreRef{{
					Name:       f.Namespace.Name,
					Kind:       esv1.ProviderStoreKindStr,
					APIVersion: esv1.SchemeGroupVersion.String(),
				}},
				Selector: esv1alpha1.PushSecretSelector{
					GeneratorRef: &esv1.GeneratorRef{
						APIVersion: genv1alpha1.Group + "/" + genv1alpha1.Version,
						Kind:       genv1alpha1.FakeKind,
						Name:       generatorName,
					},
				},
				Data: []esv1alpha1.PushSecretData{{
					Match: esv1alpha1.PushSecretMatch{
						SecretKey: "value",
						RemoteRef: esv1alpha1.PushSecretRemoteRef{
							RemoteKey: remoteSecretKey,
							Property:  "value",
						},
					},
				}},
			},
		}
		Expect(f.CreateObjectWithRetry(pushSecret)).To(Succeed())

		commonWaitForPushSecretReady(f, f.Namespace.Name, pushSecretName, corev1.ConditionTrue)
		waitForPushedValueViaExternalSecret(f, esv1.SecretStoreRef{
			Name: f.Namespace.Name,
			Kind: esv1.ProviderStoreKindStr,
		}, remoteSecretKey, "before-outage")

		DeferCleanup(func() {
			frameworkv2.ScaleDeploymentBySelectorAndWait(f, fakeBackendTarget(), 1, defaultV2WaitTimeout)
			frameworkv2.WaitForProviderConnectionReady(f, f.Namespace.Name, f.Namespace.Name, defaultV2WaitTimeout)
		})
		frameworkv2.ScaleDeploymentBySelectorAndWait(f, fakeBackendTarget(), 0, defaultV2WaitTimeout)
		commonWaitForPushSecretReady(f, f.Namespace.Name, pushSecretName, corev1.ConditionFalse)

		var updated genv1alpha1.Fake
		Expect(f.CRClient.Get(context.Background(), types.NamespacedName{Name: generatorName, Namespace: f.Namespace.Name}, &updated)).To(Succeed())
		updated.Spec.Data["value"] = "after-outage"
		Expect(f.CRClient.Update(context.Background(), &updated)).To(Succeed())

		frameworkv2.ScaleDeploymentBySelectorAndWait(f, fakeBackendTarget(), 1, defaultV2WaitTimeout)
		frameworkv2.WaitForProviderConnectionReady(f, f.Namespace.Name, f.Namespace.Name, defaultV2WaitTimeout)
		commonWaitForPushSecretReady(f, f.Namespace.Name, pushSecretName, corev1.ConditionTrue)
		waitForPushedValueViaExternalSecret(f, esv1.SecretStoreRef{
			Name: f.Namespace.Name,
			Kind: esv1.ProviderStoreKindStr,
		}, remoteSecretKey, "after-outage")
	})
})
