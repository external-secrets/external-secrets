/*
Copyright © 2022 ESO Maintainer team

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
	admissionregistration "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/controllers/crds"
	"github.com/external-secrets/external-secrets/pkg/controllers/webhookconfig"
)

var certcontrollerCmd = &cobra.Command{
	Use:   "certcontroller",
	Short: "Controller to manage certificates for external secrets CRDs and ValidatingWebhookConfigs",
	Long: `Controller to manage certificates for external secrets CRDs and ValidatingWebhookConfigs.
	For more information visit https://external-secrets.io`,
	Run: func(cmd *cobra.Command, args []string) {
		var lvl zapcore.Level
		var enc zapcore.TimeEncoder
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

		cacheOptions := cache.Options{}
		if enablePartialCache {
			cacheOptions.ByObject = map[client.Object]cache.ByObject{
				&admissionregistration.ValidatingWebhookConfiguration{}: {
					Label: labels.SelectorFromSet(map[string]string{
						constants.WellKnownLabelKey: constants.WellKnownLabelValueWebhook,
					}),
				},
				&apiextensions.CustomResourceDefinition{}: {
					Label: labels.SelectorFromSet(map[string]string{
						constants.WellKnownLabelKey: constants.WellKnownLabelValueController,
					}),
				},
			}
		}

		mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
			Scheme: scheme,
			Metrics: server.Options{
				BindAddress: metricsAddr,
			},
			WebhookServer: webhook.NewServer(webhook.Options{
				Port: 9443,
			}),
			HealthProbeBindAddress: healthzAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       "crd-certs-controller",
			Cache:                  cacheOptions,
			Client: client.Options{
				Cache: &client.CacheOptions{
					DisableFor: []client.Object{
						// the client creates a ListWatch for all resource kinds that
						// are requested with .Get().
						// We want to avoid to cache all secrets or configmaps in memory.
						// The ES controller uses v1.PartialObjectMetadata for the secrets
						// that he owns.
						// see #721
						&v1.Secret{},
					},
				},
			},
		})
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}

		crdctrl := crds.New(mgr.GetClient(), mgr.GetScheme(), mgr.Elected(),
			ctrl.Log.WithName("controllers").WithName("webhook-certs-updater"),
			crdRequeueInterval, serviceName, serviceNamespace, secretName, secretNamespace, crdNames)
		if err := crdctrl.SetupWithManager(mgr, controller.Options{
			MaxConcurrentReconciles: concurrent,
		}); err != nil {
			setupLog.Error(err, errCreateController, "controller", "CustomResourceDefinition")
			os.Exit(1)
		}

		whc := webhookconfig.New(mgr.GetClient(), mgr.GetScheme(), mgr.Elected(),
			ctrl.Log.WithName("controllers").WithName("webhook-certs-updater"),
			serviceName, serviceNamespace,
			secretName, secretNamespace, crdRequeueInterval)
		if err := whc.SetupWithManager(mgr, controller.Options{
			MaxConcurrentReconciles: concurrent,
		}); err != nil {
			setupLog.Error(err, errCreateController, "controller", "WebhookConfig")
			os.Exit(1)
		}

		err = mgr.AddReadyzCheck("crd-inject", crdctrl.ReadyCheck)
		if err != nil {
			setupLog.Error(err, "unable to add crd readyz check")
			os.Exit(1)
		}
		err = mgr.AddReadyzCheck("validation-webhook-inject", whc.ReadyCheck)
		if err != nil {
			setupLog.Error(err, "unable to add webhook readyz check")
			os.Exit(1)
		}

		setupLog.Info("starting manager")
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(certcontrollerCmd)

	certcontrollerCmd.Flags().StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	certcontrollerCmd.Flags().StringVar(&healthzAddr, "healthz-addr", ":8081", "The address the health endpoint binds to.")
	certcontrollerCmd.Flags().StringVar(&serviceName, "service-name", "external-secrets-webhook", "Webhook service name")
	certcontrollerCmd.Flags().StringVar(&serviceNamespace, "service-namespace", "default", "Webhook service namespace")
	certcontrollerCmd.Flags().StringVar(&secretName, "secret-name", "external-secrets-webhook", "Secret to store certs for webhook")
	certcontrollerCmd.Flags().StringVar(&secretNamespace, "secret-namespace", "default", "namespace of the secret to store certs")
	certcontrollerCmd.Flags().StringSliceVar(&crdNames, "crd-names", []string{"externalsecrets.external-secrets.io", "clustersecretstores.external-secrets.io", "secretstores.external-secrets.io"}, "CRD names reconciled by the controller")
	certcontrollerCmd.Flags().BoolVar(&enablePartialCache, "enable-partial-cache", false,
		"Enable caching of only the relevant CRDs and Webhook configurations in the Informer to improve memory efficiency")
	certcontrollerCmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	certcontrollerCmd.Flags().StringVar(&loglevel, "loglevel", "info", "loglevel to use, one of: debug, info, warn, error, dpanic, panic, fatal")
	certcontrollerCmd.Flags().StringVar(&zapTimeEncoding, "zap-time-encoding", "epoch", "Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano')")
	certcontrollerCmd.Flags().DurationVar(&crdRequeueInterval, "crd-requeue-interval", time.Minute*5, "Time duration between reconciling CRDs for new certs")
}
