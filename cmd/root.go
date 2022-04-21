/*
Copyright © 2022 ESO Maintainer Team

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
package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	// To allow using gcp auth.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/clusterexternalsecret"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
)

var (
	scheme                                = runtime.NewScheme()
	setupLog                              = ctrl.Log.WithName("setup")
	dnsName                               string
	certDir                               string
	metricsAddr                           string
	healthzAddr                           string
	controllerClass                       string
	enableLeaderElection                  bool
	concurrent                            int
	port                                  int
	loglevel                              string
	namespace                             string
	enableClusterStoreReconciler          bool
	enableClusterExternalSecretReconciler bool
	enableFloodGate                       bool
	storeRequeueInterval                  time.Duration
	serviceName, serviceNamespace         string
	secretName, secretNamespace           string
	crdRequeueInterval                    time.Duration
	certCheckInterval                     time.Duration
)

const (
	errCreateController = "unable to create controller"
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = esv1beta1.AddToScheme(scheme)
	_ = esv1alpha1.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)
}

var rootCmd = &cobra.Command{
	Use:   "external-secrets",
	Short: "operator that reconciles ExternalSecrets and SecretStores",
	Long:  `For more information visit https://external-secrets.io`,
	Run: func(cmd *cobra.Command, args []string) {
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
			ClientDisableCacheFor: []client.Object{
				// the client creates a ListWatch for all resource kinds that
				// are requested with .Get().
				// We want to avoid to cache all secrets or configmaps in memory.
				// The ES controller uses v1.PartialObjectMetadata for the secrets
				// that he owns.
				// see #721
				&v1.Secret{},
				&v1.ConfigMap{},
			},
			Namespace: namespace,
		})
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}
		if err = (&secretstore.StoreReconciler{
			Client:          mgr.GetClient(),
			Log:             ctrl.Log.WithName("controllers").WithName("SecretStore"),
			Scheme:          mgr.GetScheme(),
			ControllerClass: controllerClass,
			RequeueInterval: storeRequeueInterval,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, errCreateController, "controller", "SecretStore")
			os.Exit(1)
		}
		if enableClusterStoreReconciler {
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
		}
		if err = (&externalsecret.Reconciler{
			Client:                    mgr.GetClient(),
			Log:                       ctrl.Log.WithName("controllers").WithName("ExternalSecret"),
			Scheme:                    mgr.GetScheme(),
			ControllerClass:           controllerClass,
			RequeueInterval:           time.Hour,
			ClusterSecretStoreEnabled: enableClusterStoreReconciler,
			EnableFloodGate:           enableFloodGate,
		}).SetupWithManager(mgr, controller.Options{
			MaxConcurrentReconciles: concurrent,
		}); err != nil {
			setupLog.Error(err, errCreateController, "controller", "ExternalSecret")
			os.Exit(1)
		}
		if enableClusterExternalSecretReconciler {
			if err = (&clusterexternalsecret.Reconciler{
				Client:          mgr.GetClient(),
				Log:             ctrl.Log.WithName("controllers").WithName("ClusterExternalSecret"),
				Scheme:          mgr.GetScheme(),
				RequeueInterval: time.Hour,
			}).SetupWithManager(mgr, controller.Options{
				MaxConcurrentReconciles: concurrent,
			}); err != nil {
				setupLog.Error(err, errCreateController, "controller", "ClusterExternalSecret")
				os.Exit(1)
			}
		}
		setupLog.Info("starting manager")
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}

	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.Flags().StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	rootCmd.Flags().StringVar(&controllerClass, "controller-class", "default", "the controller is instantiated with a specific controller name and filters ES based on this property")
	rootCmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	rootCmd.Flags().IntVar(&concurrent, "concurrent", 1, "The number of concurrent ExternalSecret reconciles.")
	rootCmd.Flags().StringVar(&loglevel, "loglevel", "info", "loglevel to use, one of: debug, info, warn, error, dpanic, panic, fatal")
	rootCmd.Flags().StringVar(&namespace, "namespace", "", "watch external secrets scoped in the provided namespace only. ClusterSecretStore can be used but only work if it doesn't reference resources from other namespaces")
	rootCmd.Flags().BoolVar(&enableClusterStoreReconciler, "enable-cluster-store-reconciler", true, "Enable cluster store reconciler.")
	rootCmd.Flags().BoolVar(&enableClusterExternalSecretReconciler, "enable-cluster-external-secret-reconciler", true, "Enable cluster external secret reconciler.")
	rootCmd.Flags().DurationVar(&storeRequeueInterval, "store-requeue-interval", time.Minute*5, "Default Time duration between reconciling (Cluster)SecretStores")
	rootCmd.Flags().BoolVar(&enableFloodGate, "enable-flood-gate", true, "Enable flood gate. External secret will be reconciled only if the ClusterStore or Store have an healthy or unknown state.")
}
