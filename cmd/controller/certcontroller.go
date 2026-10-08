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

// Package controller implements the various controllers for external-secrets
package controller

import (
	"os"
	"time"

	"github.com/spf13/cobra"
	admissionregistration "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	ctrlcommon "github.com/external-secrets/external-secrets/pkg/controllers/common"
	"github.com/external-secrets/external-secrets/pkg/controllers/crds"
	"github.com/external-secrets/external-secrets/pkg/controllers/webhookconfig"
	"github.com/external-secrets/external-secrets/runtime/constants"
)

var certcontrollerCmd = &cobra.Command{
	Use:   "certcontroller",
	Short: "Controller to manage certificates for external secrets CRDs and ValidatingWebhookConfigs",
	Long: `Controller to manage certificates for external secrets CRDs and ValidatingWebhookConfigs.
	For more information visit https://external-secrets.io`,
	Run: func(_ *cobra.Command, _ []string) {
		setupLogger()

		// completely disable caching of Secrets and ConfigMaps to save memory
		// see: https://github.com/external-secrets/external-secrets/issues/721
		clientCacheDisableFor := make([]client.Object, 0, 2)
		clientCacheDisableFor = append(clientCacheDisableFor, &v1.Secret{}, &v1.ConfigMap{})

		// in large clusters, the CRDs and ValidatingWebhookConfigurations can take up a lot of memory
		// see: https://github.com/external-secrets/external-secrets/pull/3588
		cacheByObject := make(map[client.Object]cache.ByObject)
		if enablePartialCache {
			// only cache ValidatingWebhookConfiguration with "external-secrets.io/component=webhook" label
			cacheByObject[&admissionregistration.ValidatingWebhookConfiguration{}] = cache.ByObject{
				Label: labels.SelectorFromSet(labels.Set{
					constants.WellKnownLabelKey: constants.WellKnownLabelValueWebhook,
				}),
			}

			// only cache CustomResourceDefinition with "external-secrets.io/component=controller" label
			cacheByObject[&apiextensions.CustomResourceDefinition{}] = cache.ByObject{
				Label: labels.SelectorFromSet(labels.Set{
					constants.WellKnownLabelKey: constants.WellKnownLabelValueController,
				}),
			}
		}

		metricsServerOpts := setupMetricServerOptions()

		mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
			Scheme:  scheme,
			Metrics: metricsServerOpts,
			WebhookServer: webhook.NewServer(webhook.Options{
				Port: 9443,
			}),
			HealthProbeBindAddress: healthzAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       "crd-certs-controller",
			Cache: cache.Options{
				ByObject: cacheByObject,
			},
			Client: client.Options{
				Cache: &client.CacheOptions{
					DisableFor: clientCacheDisableFor,
				},
			},
		})
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}

		crdctrl := crds.New(mgr.GetClient(), mgr.GetScheme(), mgr.Elected(),
			ctrl.Log.WithName("controllers").WithName("webhook-certs-updater"),
			crdRequeueInterval,
			crds.Opts{
				SvcName:         serviceName,
				SvcNamespace:    serviceNamespace,
				SecretName:      secretName,
				SecretNamespace: secretNamespace,
				Resources:       crdNames,
			})
		if err := crdctrl.SetupWithManager(mgr, ctrlcommon.BuildControllerOptions(concurrent)); err != nil {
			setupLog.Error(err, errCreateController, "controller", "CustomResourceDefinition")
			os.Exit(1)
		}

		whc := webhookconfig.New(mgr.GetClient(), mgr.GetScheme(), mgr.Elected(),
			ctrl.Log.WithName("controllers").WithName("webhook-certs-updater"),
			webhookconfig.Opts{
				SvcName:         serviceName,
				SvcNamespace:    serviceNamespace,
				SecretName:      secretName,
				SecretNamespace: secretNamespace,
				RequeueInterval: crdRequeueInterval,
			})
		if err := whc.SetupWithManager(mgr, ctrlcommon.BuildControllerOptions(concurrent)); err != nil {
			setupLog.Error(err, errCreateController, "controller", "WebhookConfig")
			os.Exit(1)
		}

		if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
			setupLog.Error(err, "unable to add cert controller healthz check")
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
	certcontrollerCmd.Flags().BoolVar(&metricsAuth, "metrics-auth", false, "Enable Kubernetes RBAC-based authentication and authorization for the metrics endpoint.")
	certcontrollerCmd.Flags().BoolVar(&metricsSecure, "metrics-secure", false, "Enable HTTPS for the metrics endpoint.")
	certcontrollerCmd.Flags().StringVar(&metricsCertDir, "metrics-cert-dir", "", "Directory containing TLS certificate and key for metrics endpoint.")
	certcontrollerCmd.Flags().StringVar(&metricsCertName, "metrics-cert-name", "tls.crt", "TLS certificate filename for metrics endpoint.")
	certcontrollerCmd.Flags().StringVar(&metricsKeyName, "metrics-key-name", "tls.key", "TLS key filename for metrics endpoint.")
	certcontrollerCmd.Flags().StringVar(&serviceName, "service-name", "external-secrets-webhook", "Webhook service name")
	certcontrollerCmd.Flags().StringVar(&serviceNamespace, "service-namespace", "default", "Webhook service namespace")
	certcontrollerCmd.Flags().StringVar(&secretName, "secret-name", "external-secrets-webhook", "Secret to store certs for webhook")
	certcontrollerCmd.Flags().StringVar(&secretNamespace, "secret-namespace", "default", "namespace of the secret to store certs")
	certcontrollerCmd.Flags().
		StringSliceVar(&crdNames, "crd-names", []string{"externalsecrets.external-secrets.io", "clustersecretstores.external-secrets.io", "secretstores.external-secrets.io"}, "CRD names reconciled by the controller")
	certcontrollerCmd.Flags().BoolVar(&enablePartialCache, "enable-partial-cache", false,
		"Enable caching of only the relevant CRDs and Webhook configurations in the Informer to improve memory efficiency")
	certcontrollerCmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	certcontrollerCmd.Flags().StringVar(&loglevel, "loglevel", "info", "loglevel to use, one of: debug, info, warn, error, dpanic, panic, fatal")
	certcontrollerCmd.Flags().StringVar(&zapTimeEncoding, "zap-time-encoding", "epoch", "Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano')")
	certcontrollerCmd.Flags().DurationVar(&crdRequeueInterval, "crd-requeue-interval", time.Minute*5, "Time duration between reconciling CRDs for new certs")
	certcontrollerCmd.Flags().StringVar(&tlsCiphers, "tls-ciphers", "", "comma separated list of tls ciphers allowed for the metrics server. "+
		"This does not apply to TLS 1.3 as the ciphers are selected automatically. "+
		"Full lists of available ciphers can be found at https://pkg.go.dev/crypto/tls#pkg-constants")
	certcontrollerCmd.Flags().StringVar(&tlsMinVersion, "tls-min-version", "", "minimum version of TLS supported for the metrics server. "+
		"If not specified, Go's default minimum version is used. Valid values: 1.0, 1.1, 1.2, 1.3")
	certcontrollerCmd.Flags().StringSliceVar(&tlsCurvePreferences, "tls-curve-preferences", nil,
		"comma separated list of TLS key exchange curves allowed for the metrics server. "+
			"Use names like X25519, CurveP256, CurveP384, CurveP521, or a decimal CurveID. "+
			"If omitted, Go defaults are used.")
	certcontrollerCmd.Flags().BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics server")
}
