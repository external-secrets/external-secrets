// /*
// Copyright © The ESO Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

//go:build perf

package burst_es

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	ctrlcommon "github.com/external-secrets/external-secrets/pkg/controllers/common"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret/esmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	perf "github.com/external-secrets/external-secrets/pkg/perf"
	"github.com/external-secrets/external-secrets/runtime/testing/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	testEnv      *envtest.Environment
	k8sClient    client.Client
	cancelSuite  context.CancelFunc
	fakeProvider *fake.Client

	// storeRef is created per-scenario in BeforeEach.
	testNamespace string
	storeRef      esv1.SecretStoreRef

	// allResults accumulates PerfResults across all scenarios; written to JSON in AfterSuite.
	allResults []perf.PerfResult
)

func TestPerf(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Burst Perf Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.Level(zapcore.WarnLevel)))

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "deploy", "crds")},
	}
	testEnv.ControlPlane.GetAPIServer().Configure().
		Set("max-requests-inflight", "1000").
		Set("max-mutating-requests-inflight", "500")

	ctx, cancel := context.WithCancel(context.Background())
	cancelSuite = cancel

	cfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())

	// Raise client-side rate limits; defaults (5 QPS / burst 10) starve the controller at scale.
	cfg.QPS = float32(perf.EnvOrInt("PERF_QPS", 500))
	cfg.Burst = perf.EnvOrInt("PERF_BURST", 1000)

	Expect(esv1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(genv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: server.Options{BindAddress: "0"},
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{&v1.Secret{}, &v1.ConfigMap{}},
			},
		},
	})
	Expect(err).ToNot(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())

	secretClient, err := ctrlcommon.BuildManagedSecretClient(mgr, "")
	Expect(err).ToNot(HaveOccurred())

	Expect((&externalsecret.Reconciler{
		Client:                             mgr.GetClient(),
		SecretClient:                       secretClient,
		EnableSecretAPIReadOnCacheMismatch: true,
		RestConfig:                         cfg,
		Scheme:                             mgr.GetScheme(),
		Log:                                ctrl.Log.WithName("controllers").WithName("ExternalSecrets"),
		RequeueInterval:                    time.Hour,
		ClusterSecretStoreEnabled:          true,
	}).SetupWithManager(ctx, mgr, controller.Options{
		MaxConcurrentReconciles: perf.EnvOrInt("PERF_ES_CONCURRENCY", 4),
		RateLimiter:             ctrlcommon.BuildRateLimiter(),
	})).To(Succeed())

	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	cancelSuite()
	// Work around a controller-runtime race (https://github.com/kubernetes-sigs/controller-runtime/issues/1571)
	// where the manager has not yet released its API server connections by the time Stop() is called,
	// causing envtest to fail tearing down the control plane. Retrying with exponential backoff gives the
	// manager enough time to drain its connections and exit cleanly.
	sleepTime := time.Millisecond
	var err error
	for range 12 {
		if err = testEnv.Stop(); err == nil {
			break
		}
		sleepTime *= 2
		time.Sleep(sleepTime)
	}
	Expect(err).ToNot(HaveOccurred())

	if len(allResults) > 0 {
		Expect(perf.WriteResultsJSON(allResults, ".")).To(Succeed())
		perf.PrintResultsTable(allResults, GinkgoWriter)
	}
})

var _ = BeforeEach(func() {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "perf-burst-",
		},
	}
	Expect(k8sClient.Create(context.Background(), ns)).To(Succeed())
	testNamespace = ns.Name

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "perf-store",
			Namespace: testNamespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: fakeProviderSpec(),
		},
	}
	Expect(k8sClient.Create(context.Background(), store)).To(Succeed())
	storeRef = esv1.SecretStoreRef{Name: store.Name}
})

var _ = AfterEach(func() {
	Expect(k8sClient.Delete(context.Background(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	})).To(Succeed())
})

func fakeProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		AWS: &esv1.AWSProvider{
			Service: esv1.AWSServiceSecretsManager,
		},
	}
}

func init() {
	fakeProvider = fake.New()
	fakeProvider.WithGetSecret([]byte("perf-value"), nil)
	esv1.ForceRegister(fakeProvider, fakeProviderSpec(), esv1.MaintenanceStatusMaintained)

	ctrlmetrics.SetUpLabelNames(false)
	esmetrics.SetUpMetrics()
}
