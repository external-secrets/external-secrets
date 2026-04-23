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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[kubernetes] v2 metrics", Label("kubernetes", "v2", "metrics"), func() {
	f := framework.New("eso-kubernetes-v2-metrics")
	NewProvider(f)

	controllerClientMethods := []string{"GetSecret", "GetSecretMap"}
	providerClientMethods := []string{
		"/provider.v1.SecretStoreProvider/GetSecret",
		"/provider.v1.SecretStoreProvider/GetSecretMap",
	}

	hasSuccessfulClientRequest := func(metrics frameworkv2.MetricsMap, methods []string) bool {
		for _, method := range methods {
			value, found := frameworkv2.GetMetricValue(metrics, "grpc_client_requests_total", map[string]string{
				"method": method,
				"status": "success",
			})
			if found && value >= 1.0 {
				return true
			}
		}
		return false
	}

	hasSuccessfulServerRequest := func(metrics frameworkv2.MetricsMap, methods []string) bool {
		for _, method := range methods {
			value, found := frameworkv2.GetMetricValue(metrics, "grpc_server_requests_total", map[string]string{
				"method": method,
				"status": "success",
			})
			if found && value >= 1.0 {
				return true
			}
		}
		return false
	}

	BeforeEach(func() {
		if !framework.IsV2ProviderMode() {
			Skip("v2 mode only")
		}
		frameworkv2.WaitForSecretStoreReady(f, f.Namespace.Name, f.Namespace.Name, defaultV2WaitTimeout)
		frameworkv2.WaitForClusterSecretStoreReady(f, referentStoreName(f), defaultV2WaitTimeout)
	})

	It("exposes Provider and ClusterProvider controller metrics", func() {
		metrics, err := frameworkv2.ScrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet, frameworkv2.ProviderNamespace)
		Expect(err).ToNot(HaveOccurred())

		frameworkv2.ExpectMetricExists(metrics, "provider_status_condition")
		frameworkv2.ExpectMetricValue(metrics, "provider_status_condition", map[string]string{
			"name":      f.Namespace.Name,
			"namespace": f.Namespace.Name,
			"condition": "Ready",
			"status":    "True",
		}, 1.0)
		frameworkv2.ExpectMetricGreaterThan(metrics, "provider_reconcile_duration", map[string]string{
			"name":      f.Namespace.Name,
			"namespace": f.Namespace.Name,
		}, 0.0)

		frameworkv2.ExpectMetricExists(metrics, "clusterprovider_status_condition")
		frameworkv2.ExpectMetricValue(metrics, "clusterprovider_status_condition", map[string]string{
			"name":      referentStoreName(f),
			"condition": "Ready",
			"status":    "True",
		}, 1.0)
		frameworkv2.ExpectMetricGreaterThan(metrics, "clusterprovider_reconcile_duration", map[string]string{
			"name": referentStoreName(f),
		}, 0.0)
	})

	It("tracks client, server, and cache metrics during secret sync", func() {
		externalSecretName := "test-es-metrics"
		targetSecretName := "test-secret-metrics"
		tcSecretOne := fmt.Sprintf("%s-one", f.Namespace.Name)
		tcSecretTwo := fmt.Sprintf("%s-two", f.Namespace.Name)

		secretOne := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tcSecretOne,
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"foo": []byte("bar"),
			},
		}
		secretTwo := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tcSecretTwo,
				Namespace: f.Namespace.Name,
			},
			Data: map[string][]byte{
				"baz": []byte("qux"),
			},
		}
		Expect(f.CRClient.Create(context.Background(), secretOne)).To(Succeed())
		Expect(f.CRClient.Create(context.Background(), secretTwo)).To(Succeed())

		es := &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      externalSecretName,
				Namespace: f.Namespace.Name,
			},
			Spec: esv1.ExternalSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
				SecretStoreRef: esv1.SecretStoreRef{
					Name: f.Namespace.Name,
					Kind: esv1.SecretStoreKind,
				},
				Target: esv1.ExternalSecretTarget{
					Name: targetSecretName,
				},
				Data: []esv1.ExternalSecretData{
					{
						SecretKey: "one",
						RemoteRef: esv1.ExternalSecretDataRemoteRef{
							Key:      tcSecretOne,
							Property: "foo",
						},
					},
					{
						SecretKey: "two",
						RemoteRef: esv1.ExternalSecretDataRemoteRef{
							Key:      tcSecretTwo,
							Property: "baz",
						},
					},
				},
			},
		}
		Expect(f.CRClient.Create(context.Background(), es)).To(Succeed())

		Eventually(func(g Gomega) {
			var secret corev1.Secret
			g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{Name: targetSecretName, Namespace: f.Namespace.Name}, &secret)).To(Succeed())
			g.Expect(secret.Data["one"]).To(Equal([]byte("bar")))
			g.Expect(secret.Data["two"]).To(Equal([]byte("qux")))
		}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())

		Eventually(func() bool {
			metrics, err := frameworkv2.ScrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet, frameworkv2.ProviderNamespace)
			if err != nil {
				return false
			}
			return hasSuccessfulClientRequest(metrics, controllerClientMethods)
		}, defaultV2WaitTimeout, defaultV2PollInterval).Should(BeTrue())

		Eventually(func() bool {
			metrics, err := frameworkv2.ScrapeProviderMetrics(context.Background(), f.KubeConfig, f.KubeClientSet, frameworkv2.ProviderNamespace, "kubernetes")
			if err != nil {
				return false
			}
			return hasSuccessfulServerRequest(metrics, providerClientMethods)
		}, defaultV2WaitTimeout, defaultV2PollInterval).Should(BeTrue())

		Eventually(func() bool {
			metrics, err := frameworkv2.ScrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet, frameworkv2.ProviderNamespace)
			if err != nil {
				return false
			}
			value, found := frameworkv2.GetMetricValue(metrics, "clientmanager_cache_hits_total", map[string]string{
				"provider_type": "provider",
			})
			return found && value >= 1.0
		}, defaultV2WaitTimeout, defaultV2PollInterval).Should(BeTrue())
	})

	It("reuses one backend connection across many namespaced kubernetes Provider consumers", func() {
		const consumerCount = 6

		for i := 0; i < consumerCount; i++ {
			remoteKey := fmt.Sprintf("%s-operational-metric-%d", f.Namespace.Name, i)
			targetName := fmt.Sprintf("kubernetes-operational-metric-target-%d", i)
			expectedValue := fmt.Sprintf("metric-value-%d", i)

			Expect(f.CRClient.Create(context.Background(), &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      remoteKey,
					Namespace: f.Namespace.Name,
				},
				Data: map[string][]byte{
					"value": []byte(expectedValue),
				},
			})).To(Succeed())

			Expect(f.CRClient.Create(context.Background(), &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("kubernetes-operational-metric-es-%d", i),
					Namespace: f.Namespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: defaultV2RefreshInterval},
					SecretStoreRef: esv1.SecretStoreRef{
						Name: f.Namespace.Name,
						Kind: esv1.SecretStoreKind,
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

			Eventually(func(g Gomega) {
				var secret corev1.Secret
				g.Expect(f.CRClient.Get(context.Background(), types.NamespacedName{Name: targetName, Namespace: f.Namespace.Name}, &secret)).To(Succeed())
				g.Expect(secret.Data["value"]).To(Equal([]byte(expectedValue)))
			}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())
		}

		Eventually(func(g Gomega) {
			metrics, err := frameworkv2.ScrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet, frameworkv2.ProviderNamespace)
			g.Expect(err).NotTo(HaveOccurred())

			total := frameworkv2.SumMetricValues(metrics, "grpc_pool_connections_total", map[string]string{
				"address": frameworkv2.ProviderAddress("kubernetes"),
			})
			g.Expect(total).To(BeNumerically(">=", 1))
			g.Expect(total).To(BeNumerically("<=", 4), "expected bounded connection reuse for kubernetes backend")
			g.Expect(total).To(BeNumerically("<", consumerCount), "expected fewer pooled connections than consumers")
		}, defaultV2WaitTimeout, defaultV2PollInterval).Should(Succeed())
	})
})
