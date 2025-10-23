/*
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

package v2

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var _ = Describe("V2 Provider Metrics", Label("v2", "metrics"), func() {
	f := framework.New("v2-metrics")

	var (
		testNamespace  *corev1.Namespace
		providerCRName string
		providerName   string
		secretName     string
		externalSecret string
		fakeData       []esv1.FakeProviderData
	)

	BeforeEach(func() {
		testNamespace = SetupTestNamespace(f, "v2-metrics-")
		providerCRName = "fake-provider-cr"
		providerName = "fake-provider-conn"
		secretName = "test-secret-metrics"
		externalSecret = "test-es-metrics"

		// Create fake provider configuration
		fakeData = []esv1.FakeProviderData{
			{
				Key:   "password",
				Value: "supersecret123",
			},
			{
				Key:   "username",
				Value: "admin",
			},
		}
	})

	AfterEach(func() {
		if testNamespace != nil {
			Expect(f.CRClient.Delete(context.Background(), testNamespace)).To(Succeed())
		}
	})

	Describe("Controller Metrics", func() {
		It("should expose Provider controller metrics", func() {
			By("Creating a Fake provider CRD")
			CreateFakeProvider(f, testNamespace.Name, providerCRName, fakeData)

			By("Creating a Provider connection")
			CreateFakeProviderConnection(f, testNamespace.Name, providerName, providerCRName, testNamespace.Name)

			By("Waiting for Provider to be ready")
			WaitForProviderConnectionReady(f, testNamespace.Name, providerName, 60*time.Second)

			By("Scraping controller metrics")
			metrics, err := scrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet.(*kubernetes.Clientset), "external-secrets-system")
			Expect(err).ToNot(HaveOccurred())

			By("Verifying provider_status_condition metric exists")
			ExpectMetricExists(metrics, "provider_status_condition")

			By("Verifying provider_status_condition shows Ready=True")
			ExpectMetricValue(metrics, "provider_status_condition", map[string]string{
				"name":      providerName,
				"namespace": testNamespace.Name,
				"condition": "Ready",
				"status":    "True",
			}, 1.0)

			By("Verifying provider_reconcile_duration exists and is > 0")
			ExpectMetricGreaterThan(metrics, "provider_reconcile_duration", map[string]string{
				"name":      providerName,
				"namespace": testNamespace.Name,
			}, 0.0)
		})

		It("should expose ClusterProvider controller metrics", func() {
			clusterProviderName := "fake-cluster-provider-metrics"

			By("Creating a Fake provider CRD")
			CreateFakeProvider(f, testNamespace.Name, providerCRName, fakeData)

			By("Creating a ClusterProvider resource")
			CreateClusterProvider(f, clusterProviderName, "provider-fake.external-secrets-system.svc:8080",
				"provider.external-secrets.io/v2alpha1", "Fake", providerCRName, testNamespace.Name,
				esv1.AuthenticationScopeProviderNamespace, nil)

			By("Waiting for ClusterProvider to be ready")
			WaitForClusterProviderReady(f, clusterProviderName, 60*time.Second)

			By("Scraping controller metrics")
			metrics, err := scrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet.(*kubernetes.Clientset), "external-secrets-system")
			Expect(err).ToNot(HaveOccurred())

			By("Verifying clusterprovider_status_condition metric exists")
			ExpectMetricExists(metrics, "clusterprovider_status_condition")

			By("Verifying clusterprovider_status_condition shows Ready=True")
			ExpectMetricValue(metrics, "clusterprovider_status_condition", map[string]string{
				"name":      clusterProviderName,
				"condition": "Ready",
				"status":    "True",
			}, 1.0)

			By("Verifying clusterprovider_reconcile_duration exists and is > 0")
			ExpectMetricGreaterThan(metrics, "clusterprovider_reconcile_duration", map[string]string{
				"name": clusterProviderName,
			}, 0.0)
		})

		It("should track clientmanager cache hits and misses", func() {
			By("Creating a Fake provider CRD")
			CreateFakeProvider(f, testNamespace.Name, providerCRName, fakeData)

			By("Creating a Provider connection")
			CreateFakeProviderConnection(f, testNamespace.Name, providerName, providerCRName, testNamespace.Name)
			WaitForProviderConnectionReady(f, testNamespace.Name, providerName, 60*time.Second)

			By("Creating an ExternalSecret with multiple data entries to trigger cache hits")
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      externalSecret,
					Namespace: testNamespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: 60 * time.Second},
					SecretStoreRef: esv1.SecretStoreRef{
						Name: providerName,
						Kind: "Provider",
					},
					Target: esv1.ExternalSecretTarget{
						Name: secretName,
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "password",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "password",
							},
						},
						{
							SecretKey: "username",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "username",
							},
						},
					},
				},
			}
			err := f.CRClient.Create(context.Background(), es)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for secret to be created")
			Eventually(func() bool {
				secret := &corev1.Secret{}
				err := f.CRClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: testNamespace.Name}, secret)
				return err == nil && len(secret.Data) >= 2
			}, 60*time.Second, 1*time.Second).Should(BeTrue())

			By("Waiting a moment for metrics to be recorded")
			time.Sleep(2 * time.Second)

			By("Scraping controller metrics")
			metrics, err := scrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet.(*kubernetes.Clientset), "external-secrets-system")
			Expect(err).ToNot(HaveOccurred())

			By("Verifying clientmanager_cache_hits_total metric exists")
			ExpectMetricExists(metrics, "clientmanager_cache_hits_total")

			By("Verifying cache hits occurred within the reconcile")
			// With 2 data entries, the second Get() call should hit the cache
			value, found := getMetricValue(metrics, "clientmanager_cache_hits_total", map[string]string{
				"provider_type": "provider",
			})
			Expect(found).To(BeTrue())
			Expect(value).To(BeNumerically(">=", 1.0), "should have at least one cache hit from multiple data entries")
		})
	})

	Describe("Provider Pod Metrics", func() {
		BeforeEach(func() {
			By("Creating a Fake provider CRD")
			CreateFakeProvider(f, testNamespace.Name, providerCRName, fakeData)

			By("Creating a Provider connection")
			CreateFakeProviderConnection(f, testNamespace.Name, providerName, providerCRName, testNamespace.Name)
			WaitForProviderConnectionReady(f, testNamespace.Name, providerName, 60*time.Second)
		})

		It("should expose connection pool metrics", func() {
			By("Creating an ExternalSecret")
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      externalSecret,
					Namespace: testNamespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: 10 * time.Second},
					SecretStoreRef: esv1.SecretStoreRef{
						Name: providerName,
						Kind: "Provider",
					},
					Target: esv1.ExternalSecretTarget{
						Name: secretName,
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "password",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "password",
							},
						},
					},
				},
			}
			err := f.CRClient.Create(context.Background(), es)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for secret to be created")
			Eventually(func() bool {
				secret := &corev1.Secret{}
				err := f.CRClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: testNamespace.Name}, secret)
				return err == nil && len(secret.Data) > 0
			}, 60*time.Second, 1*time.Second).Should(BeTrue())

			By("Scraping controller metrics for pool stats")
			metrics, err := scrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet.(*kubernetes.Clientset), "external-secrets-system")
			Expect(err).ToNot(HaveOccurred())

			By("Verifying grpc_pool_misses_total exists (first connection)")
			ExpectMetricExists(metrics, "grpc_pool_misses_total")

			By("Verifying at least one cache miss occurred")
			ExpectMetricGreaterThan(metrics, "grpc_pool_misses_total", map[string]string{}, 0.0)

			By("Verifying grpc_pool_connections_total exists")
			ExpectMetricExists(metrics, "grpc_pool_connections_total")

			By("Triggering another reconcile to test cache hit")
			// Update the ExternalSecret to trigger reconciliation
			err = f.CRClient.Get(context.Background(), types.NamespacedName{Name: externalSecret, Namespace: testNamespace.Name}, es)
			Expect(err).ToNot(HaveOccurred())
			es.Annotations = map[string]string{"test": "trigger-reconcile"}
			err = f.CRClient.Update(context.Background(), es)
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(5 * time.Second)

			By("Scraping controller metrics again for pool hits")
			metrics, err = scrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet.(*kubernetes.Clientset), "external-secrets-system")
			Expect(err).ToNot(HaveOccurred())

			By("Verifying grpc_pool_hits_total incremented")
			ExpectMetricGreaterThan(metrics, "grpc_pool_hits_total", map[string]string{}, 0.0)
		})

		It("should expose gRPC client metrics", func() {
			By("Creating an ExternalSecret")
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      externalSecret,
					Namespace: testNamespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: 60 * time.Second},
					SecretStoreRef: esv1.SecretStoreRef{
						Name: providerName,
						Kind: "Provider",
					},
					Target: esv1.ExternalSecretTarget{
						Name: secretName,
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "password",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "password",
							},
						},
					},
				},
			}
			err := f.CRClient.Create(context.Background(), es)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for secret to be created")
			Eventually(func() bool {
				secret := &corev1.Secret{}
				err := f.CRClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: testNamespace.Name}, secret)
				return err == nil && len(secret.Data) > 0
			}, 60*time.Second, 1*time.Second).Should(BeTrue())

			By("Scraping controller metrics for client stats")
			metrics, err := scrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet.(*kubernetes.Clientset), "external-secrets-system")
			Expect(err).ToNot(HaveOccurred())

			By("Verifying grpc_client_requests_total exists")
			ExpectMetricExists(metrics, "grpc_client_requests_total")

			By("Verifying GetSecret requests were made successfully")
			value, found := getMetricValue(metrics, "grpc_client_requests_total", map[string]string{
				"method": "GetSecret",
				"status": "success",
			})
			Expect(found).To(BeTrue())
			Expect(value).To(BeNumerically(">=", 1.0), "should have at least one successful GetSecret call")

			By("Verifying grpc_client_request_duration_seconds exists")
			ExpectMetricExists(metrics, "grpc_client_request_duration_seconds_count")

			By("Verifying request duration was recorded")
			value, found = getMetricValue(metrics, "grpc_client_request_duration_seconds_count", map[string]string{
				"method": "GetSecret",
				"status": "success",
			})
			Expect(found).To(BeTrue())
			Expect(value).To(BeNumerically(">=", 1.0))
		})

		It("should expose gRPC server metrics", func() {
			By("Creating an ExternalSecret")
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      externalSecret,
					Namespace: testNamespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: 60 * time.Second},
					SecretStoreRef: esv1.SecretStoreRef{
						Name: providerName,
						Kind: "Provider",
					},
					Target: esv1.ExternalSecretTarget{
						Name: secretName,
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "password",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "password",
							},
						},
					},
				},
			}
			err := f.CRClient.Create(context.Background(), es)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for secret to be created")
			Eventually(func() bool {
				secret := &corev1.Secret{}
				err := f.CRClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: testNamespace.Name}, secret)
				return err == nil && len(secret.Data) > 0
			}, 60*time.Second, 1*time.Second).Should(BeTrue())

			By("Scraping provider pod metrics")
			metrics, err := scrapeProviderMetrics(context.Background(), f.KubeConfig, f.KubeClientSet.(*kubernetes.Clientset), "external-secrets-system", "fake")
			Expect(err).ToNot(HaveOccurred())

			By("Verifying grpc_server_requests_total exists")
			ExpectMetricExists(metrics, "grpc_server_requests_total")

			By("Verifying server handled GetSecret requests")
			value, found := getMetricValue(metrics, "grpc_server_requests_total", map[string]string{
				"method": "/provider.v1.SecretStoreProvider/GetSecret",
				"status": "success",
			})
			Expect(found).To(BeTrue())
			Expect(value).To(BeNumerically(">=", 1.0))

			By("Verifying grpc_server_request_duration_seconds exists")
			ExpectMetricExists(metrics, "grpc_server_request_duration_seconds_count")

			By("Verifying server request duration was recorded")
			value, found = getMetricValue(metrics, "grpc_server_request_duration_seconds_count", map[string]string{
				"method": "/provider.v1.SecretStoreProvider/GetSecret",
			})
			Expect(found).To(BeTrue())
			Expect(value).To(BeNumerically(">=", 1.0))
		})
	})

	Describe("End-to-End Metrics Workflow", func() {
		It("should track metrics through full Provider lifecycle", func() {
			By("1. Creating Provider and verifying controller metrics")
			CreateFakeProvider(f, testNamespace.Name, providerCRName, fakeData)
			CreateFakeProviderConnection(f, testNamespace.Name, providerName, providerCRName, testNamespace.Name)
			WaitForProviderConnectionReady(f, testNamespace.Name, providerName, 60*time.Second)

			controllerMetrics, err := scrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet.(*kubernetes.Clientset), "external-secrets-system")
			Expect(err).ToNot(HaveOccurred())
			ExpectMetricValue(controllerMetrics, "provider_status_condition", map[string]string{
				"name":      providerName,
				"namespace": testNamespace.Name,
				"condition": "Ready",
				"status":    "True",
			}, 1.0)

			By("2. Creating ExternalSecret with multiple data entries and verifying metrics")
			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      externalSecret,
					Namespace: testNamespace.Name,
				},
				Spec: esv1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: 10 * time.Second},
					SecretStoreRef: esv1.SecretStoreRef{
						Name: providerName,
						Kind: "Provider",
					},
					Target: esv1.ExternalSecretTarget{
						Name: secretName,
					},
					Data: []esv1.ExternalSecretData{
						{
							SecretKey: "password",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "password",
							},
						},
						{
							SecretKey: "username",
							RemoteRef: esv1.ExternalSecretDataRemoteRef{
								Key: "username",
							},
						},
					},
				},
			}
			err = f.CRClient.Create(context.Background(), es)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				secret := &corev1.Secret{}
				err := f.CRClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: testNamespace.Name}, secret)
				return err == nil && len(secret.Data) >= 2
			}, 60*time.Second, 1*time.Second).Should(BeTrue())

			// Pool and client metrics are on controller, server metrics on provider pod
			controllerMetrics, err = scrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet.(*kubernetes.Clientset), "external-secrets-system")
			Expect(err).ToNot(HaveOccurred())
			ExpectMetricGreaterThan(controllerMetrics, "grpc_pool_misses_total", map[string]string{}, 0.0)
			ExpectMetricGreaterThan(controllerMetrics, "grpc_client_requests_total", map[string]string{
				"method": "GetSecret",
				"status": "success",
			}, 0.0)
			
			providerMetrics, err := scrapeProviderMetrics(context.Background(), f.KubeConfig, f.KubeClientSet.(*kubernetes.Clientset), "external-secrets-system", "fake")
			Expect(err).ToNot(HaveOccurred())
			ExpectMetricGreaterThan(providerMetrics, "grpc_server_requests_total", map[string]string{
				"method": "/provider.v1.SecretStoreProvider/GetSecret",
				"status": "success",
			}, 0.0)

			By("3. Waiting for refresh and verifying pool hits")
			time.Sleep(15 * time.Second)

			controllerMetrics, err = scrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet.(*kubernetes.Clientset), "external-secrets-system")
			Expect(err).ToNot(HaveOccurred())
			ExpectMetricGreaterThan(controllerMetrics, "grpc_pool_hits_total", map[string]string{}, 0.0)

			// Clientmanager cache hits occur within a single reconcile when multiple data entries exist
			controllerMetrics, err = scrapeControllerMetrics(context.Background(), f.KubeConfig, f.KubeClientSet.(*kubernetes.Clientset), "external-secrets-system")
			Expect(err).ToNot(HaveOccurred())
			ExpectMetricGreaterThan(controllerMetrics, "clientmanager_cache_hits_total", map[string]string{
				"provider_type": "provider",
			}, 0.0)

			By("4. Workflow completed successfully with all metrics tracked")
		})
	})
})

