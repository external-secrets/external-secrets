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

package burst_ss

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
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
	ctrlcommon "github.com/external-secrets/external-secrets/pkg/controllers/common"
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

	// allResults accumulates PerfResults; written to JSON in AfterSuite.
	allResults []perf.PerfResult
)

func TestPerf(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stores Perf Suite")
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

	Expect(esv1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(esv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: server.Options{BindAddress: "0"},
	})
	Expect(err).ToNot(HaveOccurred())

	k8sClient = mgr.GetClient()

	Expect((&secretstore.StoreReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		Log:             ctrl.Log.WithName("controllers").WithName("SecretStore"),
		RequeueInterval: time.Hour,
	}).SetupWithManager(mgr, controller.Options{
		MaxConcurrentReconciles: perf.EnvOrInt("PERF_STORE_CONCURRENCY", 16),
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
	esv1.ForceRegister(fakeProvider, fakeProviderSpec(), esv1.MaintenanceStatusMaintained)

	ctrlmetrics.SetUpLabelNames(false)
	ssmetrics.SetUpMetrics()
}
