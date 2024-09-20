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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	providers "github.com/external-secrets/external-secrets/apis/providers/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/clusterexternalsecret"
	"github.com/external-secrets/external-secrets/pkg/controllers/clusterexternalsecret/cesmetrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret/esmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/pushsecret"
	"github.com/external-secrets/external-secrets/pkg/controllers/pushsecret/psmetrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/cssmetrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/ssmetrics"
	"github.com/external-secrets/external-secrets/pkg/feature"

	// To allow using gcp auth.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
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
	enableSecretsCache                    bool
	enableConfigMapsCache                 bool
	enablePartialCache                    bool
	concurrent                            int
	port                                  int
	clientQPS                             float32
	clientBurst                           int
	loglevel                              string
	zapTimeEncoding                       string
	namespace                             string
	enableClusterStoreReconciler          bool
	enableClusterExternalSecretReconciler bool
	enablePushSecretReconciler            bool
	enableFloodGate                       bool
	enableExtendedMetricLabels            bool
	storeRequeueInterval                  time.Duration
	serviceName, serviceNamespace         string
	secretName, secretNamespace           string
	crdNames                              []string
	crdRequeueInterval                    time.Duration
	certCheckInterval                     time.Duration
	certLookaheadInterval                 time.Duration
	tlsCiphers                            string
	tlsMinVersion                         string
)

const (
	errCreateController = "unable to create controller"
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = esv1beta1.AddToScheme(scheme)
	_ = esv1alpha1.AddToScheme(scheme)
	_ = providers.AddToScheme(scheme)
	_ = genv1alpha1.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)
}

var rootCmd = &cobra.Command{
	Use:   "external-secrets",
	Short: "operator that reconciles ExternalSecrets and SecretStores",
	Long:  `For more information visit https://external-secrets.io`,
	Run: func(cmd *cobra.Command, args []string) {
		var lvl zapcore.Level
		var enc zapcore.TimeEncoder
		// the client creates a ListWatch for all resource kinds that
		// are requested with .Get().
		// We want to avoid to cache all secrets or configmaps in memory.
		// The ES controller uses v1.PartialObjectMetadata for the secrets
		// that he owns.
		// see #721
		cacheList := make([]client.Object, 0)
		if !enableSecretsCache {
			cacheList = append(cacheList, &v1.Secret{})
		}
		if !enableConfigMapsCache {
			cacheList = append(cacheList, &v1.ConfigMap{})
		}
		lvlErr := lvl.UnmarshalText([]byte(loglevel))
		if lvlErr != nil {
			setupLog.Error(lvlErr, "error unmarshalling loglevel")
			os.Exit(1)
		}
		encErr := enc.UnmarshalText([]byte(zapTimeEncoding))
		if encErr != nil {
			setupLog.Error(encErr, "error unmarshalling timeEncoding")
			os.Exit(1)
		}
		opts := zap.Options{
			Level:       lvl,
			TimeEncoder: enc,
		}
		logger := zap.New(zap.UseFlagOptions(&opts))
		ctrl.SetLogger(logger)
		ctrlmetrics.SetUpLabelNames(enableExtendedMetricLabels)
		esmetrics.SetUpMetrics()
		config := ctrl.GetConfigOrDie()
		config.QPS = clientQPS
		config.Burst = clientBurst
		ctrlOpts := ctrl.Options{
			Scheme: scheme,
			Metrics: server.Options{
				BindAddress: metricsAddr,
			},
			WebhookServer: webhook.NewServer(webhook.Options{
				Port: 9443,
			}),
			Client: client.Options{
				Cache: &client.CacheOptions{
					DisableFor: cacheList,
				},
			},
			LeaderElection:   enableLeaderElection,
			LeaderElectionID: "external-secrets-controller",
		}
		if namespace != "" {
			ctrlOpts.Cache.DefaultNamespaces = map[string]cache.Config{
				namespace: {},
			}
		}
		mgr, err := ctrl.NewManager(config, ctrlOpts)
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}

		ssmetrics.SetUpMetrics()
		if err = (&secretstore.StoreReconciler{
			Client:          mgr.GetClient(),
			Log:             ctrl.Log.WithName("controllers").WithName("SecretStore"),
			Scheme:          mgr.GetScheme(),
			ControllerClass: controllerClass,
			RequeueInterval: storeRequeueInterval,
		}).SetupWithManager(mgr, controller.Options{
			MaxConcurrentReconciles: concurrent,
		}); err != nil {
			setupLog.Error(err, errCreateController, "controller", "SecretStore")
			os.Exit(1)
		}
		if enableClusterStoreReconciler {
			cssmetrics.SetUpMetrics()
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
			RestConfig:                mgr.GetConfig(),
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
		if enablePushSecretReconciler {
			psmetrics.SetUpMetrics()
			if err = (&pushsecret.Reconciler{
				Client:          mgr.GetClient(),
				Log:             ctrl.Log.WithName("controllers").WithName("PushSecret"),
				Scheme:          mgr.GetScheme(),
				ControllerClass: controllerClass,
				RequeueInterval: time.Hour,
			}).SetupWithManager(mgr); err != nil {
				setupLog.Error(err, errCreateController, "controller", "PushSecret")
				os.Exit(1)
			}
		}
		if enableClusterExternalSecretReconciler {
			cesmetrics.SetUpMetrics()

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

		fs := feature.Features()
		for _, f := range fs {
			if f.Initialize == nil {
				continue
			}
			f.Initialize()
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
	rootCmd.Flags().StringVar(&controllerClass, "controller-class", "default", "The controller is instantiated with a specific controller name and filters ES based on this property")
	rootCmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	rootCmd.Flags().IntVar(&concurrent, "concurrent", 1, "The number of concurrent reconciles.")
	rootCmd.Flags().Float32Var(&clientQPS, "client-qps", 0, "QPS configuration to be passed to rest.Client")
	rootCmd.Flags().IntVar(&clientBurst, "client-burst", 0, "Maximum Burst allowed to be passed to rest.Client")
	rootCmd.Flags().StringVar(&loglevel, "loglevel", "info", "loglevel to use, one of: debug, info, warn, error, dpanic, panic, fatal")
	rootCmd.Flags().StringVar(&zapTimeEncoding, "zap-time-encoding", "epoch", "Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano')")
	rootCmd.Flags().StringVar(&namespace, "namespace", "", "watch external secrets scoped in the provided namespace only. ClusterSecretStore can be used but only work if it doesn't reference resources from other namespaces")
	rootCmd.Flags().BoolVar(&enableClusterStoreReconciler, "enable-cluster-store-reconciler", true, "Enable cluster store reconciler.")
	rootCmd.Flags().BoolVar(&enableClusterExternalSecretReconciler, "enable-cluster-external-secret-reconciler", true, "Enable cluster external secret reconciler.")
	rootCmd.Flags().BoolVar(&enablePushSecretReconciler, "enable-push-secret-reconciler", true, "Enable push secret reconciler.")
	rootCmd.Flags().BoolVar(&enableSecretsCache, "enable-secrets-caching", false, "Enable secrets caching for external-secrets pod.")
	rootCmd.Flags().BoolVar(&enableConfigMapsCache, "enable-configmaps-caching", false, "Enable secrets caching for external-secrets pod.")
	rootCmd.Flags().DurationVar(&storeRequeueInterval, "store-requeue-interval", time.Minute*5, "Default Time duration between reconciling (Cluster)SecretStores")
	rootCmd.Flags().BoolVar(&enableFloodGate, "enable-flood-gate", true, "Enable flood gate. External secret will be reconciled only if the ClusterStore or Store have an healthy or unknown state.")
	rootCmd.Flags().BoolVar(&enableExtendedMetricLabels, "enable-extended-metric-labels", false, "Enable recommended kubernetes annotations as labels in metrics.")
	fs := feature.Features()
	for _, f := range fs {
		rootCmd.Flags().AddFlagSet(f.Flags)
	}
}
