//go:build perf

package fanout

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	ctrlcommon "github.com/external-secrets/external-secrets/pkg/controllers/common"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret/esmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/ssmetrics"
	perf "github.com/external-secrets/external-secrets/pkg/perf"
	"github.com/external-secrets/external-secrets/runtime/testing/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	testEnv     *envtest.Environment
	k8sClient   client.Client
	cancelSuite context.CancelFunc

	fakeProvider *fake.Client

	allResults []perf.PerfResult
)

func TestPerf(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fanout Perf Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.Level(zapcore.WarnLevel)))

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:        []string{filepath.Join("..", "..", "..", "deploy", "crds")},
		ControlPlaneStartTimeout: 5 * time.Minute,
	}

	testEnv.ControlPlane.GetAPIServer().Configure().
		Set("max-requests-inflight", "1000").
		Set("max-mutating-requests-inflight", "500")

	ctx, cancel := context.WithCancel(context.Background())
	cancelSuite = cancel

	cfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())

	// Tune the REST client for high throughput.
	cfg.QPS = float32(perf.EnvOrInt("PERF_QPS", 500))
	cfg.Burst = perf.EnvOrInt("PERF_BURST", 1000)

	Expect(esv1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(esv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(genv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme.Scheme,
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

	// SecretStore controller
	Expect((&secretstore.StoreReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		Log:             ctrl.Log.WithName("controllers").WithName("SecretStore"),
		RequeueInterval: time.Hour,
	}).SetupWithManager(mgr, controller.Options{
		MaxConcurrentReconciles: perf.EnvOrInt("PERF_STORE_CONCURRENCY", 16),
		RateLimiter:             ctrlcommon.BuildRateLimiter(),
	})).To(Succeed())

	// ExternalSecret controller
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
		MaxConcurrentReconciles: perf.EnvOrInt("PERF_ES_CONCURRENCY", 16),
		RateLimiter:             ctrlcommon.BuildRateLimiter(),
	})).To(Succeed())

	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	cancelSuite()
	Expect(testEnv.Stop()).To(Succeed())

	if len(allResults) > 0 {
		Expect(perf.WriteResultsJSON(allResults, ".")).To(Succeed())
		perf.PrintResultsTable(allResults, GinkgoWriter)
	}
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
	ssmetrics.SetUpMetrics()
}
