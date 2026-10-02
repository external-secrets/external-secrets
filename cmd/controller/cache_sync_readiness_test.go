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

package controller

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// cacheSyncTimeout is the per-call WaitForCacheSync deadline.
const cacheSyncTimeout = 2 * time.Second

var _ = Describe("CacheSyncReadiness", func() {
	// This suite verifies the behavior of the readiness probe when a
	// ClusterRoleBinding is absent, reproducing the scenario from issue #6679.
	//
	// A service account is impersonated that has no RBAC rules. The manager's
	// informer cache cannot list/watch ExternalSecrets (403), so WaitForCacheSync
	// never returns true, keeping readyz non-200.

	var (
		adminClient client.Client
		mgrCancel   context.CancelFunc
		healthBase  string
		testNS      *v1.Namespace
	)

	BeforeEach(func() {
		// ── Admin client ─────────────────────────────────────────────────────
		var err error
		adminClient, err = client.New(cfg, client.Options{Scheme: testScheme})
		Expect(err).NotTo(HaveOccurred())

		testNS = &v1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "cache-sync-test-"}}
		Expect(adminClient.Create(context.Background(), testNS)).To(Succeed())

		// ── Restricted config (no RBAC rules for this SA) ────────────────────
		restrictedCfg := rest.CopyConfig(cfg)
		restrictedCfg.Impersonate = rest.ImpersonationConfig{
			UserName: "system:serviceaccount:external-secrets:eso-restricted",
		}

		// ── Free port for the health probe server ────────────────────────────
		addr, err := freeAddr()
		Expect(err).NotTo(HaveOccurred())
		healthBase = fmt.Sprintf("http://%s", addr)

		// ── Manager with restricted credentials ──────────────────────────────
		mgr, err := ctrl.NewManager(restrictedCfg, ctrl.Options{
			Scheme: testScheme,
			Metrics: server.Options{
				BindAddress: "0",
			},
			HealthProbeBindAddress: addr,
			LeaderElection:         false,
			WebhookServer:          nil,
		})
		Expect(err).NotTo(HaveOccurred())

		// A no-op reconciler forces the cache to start an ExternalSecret
		// informer (LIST+WATCH). The restricted SA cannot execute those calls,
		// so the cache never fully syncs.
		err = ctrl.NewControllerManagedBy(mgr).
			For(&esv1.ExternalSecret{}).
			WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
			Complete(reconcile.Func(func(_ context.Context, _ reconcile.Request) (reconcile.Result, error) {
				return reconcile.Result{}, nil
			}))
		Expect(err).NotTo(HaveOccurred())

		// Wire probes via the same helper used in production.
		Expect(setupHealthChecks(mgr)).To(Succeed())

		// ── Start manager ────────────────────────────────────────────────────
		var mgrCtx context.Context
		mgrCtx, mgrCancel = context.WithCancel(context.Background())
		go func() {
			defer GinkgoRecover()
			Expect(mgr.Start(mgrCtx)).To(Succeed())
		}()

		// Wait for the HTTP server to come up.
		Eventually(func() error {
			resp, err := http.Get(healthBase + "/healthz")
			if err != nil {
				return err
			}
			resp.Body.Close()
			return nil
		}, 15*time.Second, 200*time.Millisecond).Should(Succeed(),
			"health probe server did not start in time")
	})

	AfterEach(func() {
		mgrCancel()
		if testNS != nil {
			_ = adminClient.Delete(context.Background(), testNS)
		}
	})

	It("reports liveness 200, readyz non-200, and leaves ExternalSecrets unreconciled", func() {
		// ── Liveness must always be 200 ──────────────────────────────────────
		resp, err := http.Get(healthBase + "/healthz")
		Expect(err).NotTo(HaveOccurred())
		resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK),
			"liveness probe should return 200 regardless of cache state")

		// ── ExternalSecret must stay unreconciled ─────────────────────────────
		//
		// Create via admin client (so the API call itself succeeds). The
		// manager's cache never delivered the object to the work queue, so the
		// reconciler was never invoked and the status has no conditions.
		es := &esv1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-es",
				Namespace: testNS.Name,
			},
			Spec: esv1.ExternalSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: 2 * time.Second},
				Target:          esv1.ExternalSecretTarget{Name: "test-secret"},
			},
		}
		Expect(adminClient.Create(context.Background(), es)).To(Succeed())

		Consistently(func() bool {
			got := &esv1.ExternalSecret{}
			if err := adminClient.Get(context.Background(),
				types.NamespacedName{Name: es.Name, Namespace: testNS.Name}, got); err != nil {
				return false
			}
			return len(got.Status.Conditions) == 0
		}, 10*time.Second, time.Second).Should(BeTrue(),
			"ExternalSecret should remain unreconciled while cache is not synced")

		// ── Readiness must not be 200 ─────────────────────────────────────────
		//
		// Each /readyz call blocks inside WaitForCacheSync for up to
		// cacheSyncTimeout (2 s). The poll interval is set to cacheSyncTimeout+1 s
		// so that each HTTP round-trip completes before the next poll starts,
		// avoiding concurrent requests and making the timing explicit.
		pollInterval := cacheSyncTimeout + time.Second
		Consistently(func() int {
			r, err := http.Get(healthBase + "/readyz")
			if err != nil {
				return 0
			}
			r.Body.Close()
			return r.StatusCode
		}, 10*time.Second, pollInterval).ShouldNot(Equal(http.StatusOK),
			"readiness probe should not return 200 while cache is not synced (RBAC missing)")
	})
})

// freeAddr returns a host:port string with an OS-assigned free TCP port.
// The listener is closed immediately; there is a small TOCTOU race that is
// acceptable for tests.
func freeAddr() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr, nil
}
