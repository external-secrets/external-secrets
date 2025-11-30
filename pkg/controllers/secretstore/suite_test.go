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

package secretstore

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	ctrlcommon "github.com/external-secrets/external-secrets/pkg/controllers/common"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/cssmetrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/ssmetrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var cancel context.CancelFunc

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	log := zap.New(zap.WriteTo(GinkgoWriter))
	logf.SetLogger(log)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "deploy", "crds")},
	}

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = esapi.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = esv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: server.Options{
			BindAddress: "0", // avoid port collision when testing
		},
	})
	Expect(err).ToNot(HaveOccurred())

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	err = (&StoreReconciler{
		Client:            k8sManager.GetClient(),
		Scheme:            k8sManager.GetScheme(),
		Log:               ctrl.Log.WithName("controllers").WithName("SecretStore"),
		ControllerClass:   defaultControllerClass,
		PushSecretEnabled: true, // enable PushSecret feature for testing
	}).SetupWithManager(k8sManager, controller.Options{
		MaxConcurrentReconciles: 1,
		RateLimiter:             ctrlcommon.BuildRateLimiter(),
	})
	Expect(err).ToNot(HaveOccurred())

	// Index PushSecret status.syncedPushSecrets to find all stores that have synced a specific PushSecret.
	err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &esv1alpha1.PushSecret{}, "status.syncedPushSecrets", func(obj client.Object) []string {
		ps := obj.(*esv1alpha1.PushSecret)
		var storeNames []string
		if ps.Spec.DeletionPolicy != esv1alpha1.PushSecretDeletionPolicyDelete {
			return nil
		}
		for storeKey := range ps.Status.SyncedPushSecrets {
			if strings.Contains(storeKey, "/") {
				parts := strings.SplitN(storeKey, "/", 2)
				if len(parts) == 2 {
					storeNames = append(storeNames, parts[1])
				}
			}
		}
		return storeNames
	})
	Expect(err).ToNot(HaveOccurred())

	// Index PushSecret spec.deletionPolicy to find all PushSecrets with deletionPolicy: Delete.
	err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &esv1alpha1.PushSecret{}, "spec.deletionPolicy", func(obj client.Object) []string {
		ps := obj.(*esv1alpha1.PushSecret)
		return []string{string(ps.Spec.DeletionPolicy)}
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&ClusterStoreReconciler{
		Client:            k8sManager.GetClient(),
		Scheme:            k8sManager.GetScheme(),
		ControllerClass:   defaultControllerClass,
		Log:               ctrl.Log.WithName("controllers").WithName("ClusterSecretStore"),
		PushSecretEnabled: true, // enable PushSecret feature for testing
	}).SetupWithManager(k8sManager, controller.Options{
		MaxConcurrentReconciles: 1,
		RateLimiter:             ctrlcommon.BuildRateLimiter(),
	})
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		Expect(k8sManager.Start(ctx)).ToNot(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel() // stop manager
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func init() {
	ctrlmetrics.SetUpLabelNames(false)
	cssmetrics.SetUpMetrics()
	ssmetrics.SetUpMetrics()
}
