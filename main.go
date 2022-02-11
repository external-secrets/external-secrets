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

package main

import (
	"flag"
	"os"
	"time"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/crds"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	errCreateController = "unable to create controller"
	errCreateWebhook    = "unable to create webhook"
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = esv1beta1.AddToScheme(scheme)
	_ = esv1alpha1.AddToScheme(scheme)
}

func main() {
	var metricsAddr string
	var controllerClass string
	var enableLeaderElection bool
	var concurrent int
	var loglevel string
	var namespace string
	var webhook bool
	var storeRequeueInterval time.Duration
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&controllerClass, "controller-class", "default", "the controller is instantiated with a specific controller name and filters ES based on this property")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&webhook, "webhook", false, "Run as webhook") // Properly separate
	flag.IntVar(&concurrent, "concurrent", 1, "The number of concurrent ExternalSecret reconciles.")
	flag.StringVar(&loglevel, "loglevel", "info", "loglevel to use, one of: debug, info, warn, error, dpanic, panic, fatal")
	flag.StringVar(&namespace, "namespace", "", "watch external secrets scoped in the provided namespace only. ClusterSecretStore can be used but only work if it doesn't reference resources from other namespaces")
	flag.DurationVar(&storeRequeueInterval, "store-requeue-interval", time.Minute*5, "Time duration between reconciling (Cluster)SecretStores")
	flag.Parse()

	var lvl zapcore.Level
	err := lvl.UnmarshalText([]byte(loglevel))
	if err != nil {
		setupLog.Error(err, "error unmarshalling loglevel")
		os.Exit(1)
	}
	logger := zap.New(zap.Level(lvl))
	ctrl.SetLogger(logger)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "external-secrets-controller",
		Namespace:          namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	if webhook {
		crds := &crds.Reconciler{
			Client:                 mgr.GetClient(),
			Log:                    ctrl.Log.WithName("controllers").WithName("CustomResourceDefinition"),
			Scheme:                 mgr.GetScheme(),
			SvcLabels:              map[string]string{"external-secrets.io/component": "webhook"},
			SecretLabels:           map[string]string{"external-secrets.io/component": "webhook"},
			CrdResources:           []string{"externalsecrets.external-secrets.io", "clustersecretstores.external-secrets.io", "secretstores.external-secrets.io"},
			CertDir:                "/tmp/k8s-webhook-server/serving-certs",
			CAName:                 "external-secrets",
			CAOrganization:         "external-secrets",
			RestartOnSecretRefresh: true,
		}
		if err := crds.SetupWithManager(mgr, controller.Options{
			MaxConcurrentReconciles: concurrent,
		}); err != nil {
			setupLog.Error(err, errCreateController, "controller", "CustomResourceDefinition")
			os.Exit(1)
		}
		if crtsReady := crds.EnsureCertsMounted(); crtsReady {
			if err = (&esv1beta1.ExternalSecret{}).SetupWebhookWithManager(mgr); err != nil {
				setupLog.Error(err, errCreateWebhook, "webhook", "ExternalSecret-v1beta1")
				os.Exit(1)
			}
			if err = (&esv1beta1.SecretStore{}).SetupWebhookWithManager(mgr); err != nil {
				setupLog.Error(err, errCreateWebhook, "webhook", "SecretStore-v1beta1")
				os.Exit(1)
			}
			if err = (&esv1beta1.ClusterSecretStore{}).SetupWebhookWithManager(mgr); err != nil {
				setupLog.Error(err, errCreateWebhook, "webhook", "ClusterSecretStore-v1beta1")
				os.Exit(1)
			}
			if err = (&esv1alpha1.ExternalSecret{}).SetupWebhookWithManager(mgr); err != nil {
				setupLog.Error(err, errCreateWebhook, "webhook", "ExternalSecret-v1alpha1")
				os.Exit(1)
			}
			if err = (&esv1alpha1.SecretStore{}).SetupWebhookWithManager(mgr); err != nil {
				setupLog.Error(err, errCreateWebhook, "webhook", "SecretStore-v1alpha1")
				os.Exit(1)
			}
			if err = (&esv1alpha1.ClusterSecretStore{}).SetupWebhookWithManager(mgr); err != nil {
				setupLog.Error(err, errCreateWebhook, "webhook", "ClusterSecretStore-v1alpha1")
				os.Exit(1)
			}
		}
	} else {
		if err = (&secretstore.StoreReconciler{
			Client:          mgr.GetClient(),
			Log:             ctrl.Log.WithName("contllers").WithName("SecretStore"),
			Scheme:          mgr.GetScheme(),
			ControllerClass: controllerClass,
			RequeueInterval: storeRequeueInterval,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, errCreateController, "controller", "SecretStore")
			os.Exit(1)
		}
		if err = (&secretstore.ClusterStoreReconciler{
			Client:          mgr.GetClient(),
			Log:             ctrl.Log.WithName("controllers").WithName("ClusterSecretStore"),
			Scheme:          mgr.GetScheme(),
			ControllerClass: controllerClass,
			RequeueInterval: storeRequeueInterval,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, errCreateController, "controller", "ClusterSecretStore")
			os.Exit(1)
		}
		if err = (&externalsecret.Reconciler{
			Client:          mgr.GetClient(),
			Log:             ctrl.Log.WithName("controllers").WithName("ExternalSecret"),
			Scheme:          mgr.GetScheme(),
			ControllerClass: controllerClass,
			RequeueInterval: time.Hour,
		}).SetupWithManager(mgr, controller.Options{
			MaxConcurrentReconciles: concurrent,
		}); err != nil {
			setupLog.Error(err, errCreateController, "controller", "ExternalSecret")
			os.Exit(1)
		}
	}
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
