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
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/crds"
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

func checkCerts(certDir, dnsName string) {
	for {
		setupLog.Info("Checking certs")
		certFile := certDir + "/tls.crt"
		_, err := os.Stat(certFile)
		if err != nil {
			setupLog.Error(err, "certs not found")
			os.Exit(1)
		}
		ca, err := os.ReadFile(certDir + "/ca.crt")
		if err != nil {
			setupLog.Error(err, "error loading ca cert")
			os.Exit(1)
		}
		cert, err := os.ReadFile(certDir + "/tls.crt")
		if err != nil {
			setupLog.Error(err, "error loading server cert")
			os.Exit(1)
		}
		key, err := os.ReadFile(certDir + "/tls.key")
		if err != nil {
			setupLog.Error(err, "error loading server key")
			os.Exit(1)
		}
		ok, err := crds.ValidCert(ca, cert, key, dnsName, time.Now())
		if err != nil || !ok {
			setupLog.Error(err, "certificates are not valid!")
			os.Exit(1)
		}
		setupLog.Info("Certs valid")
		time.Sleep(time.Hour * 24)
	}
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var loglevel string
	var namespace string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&loglevel, "loglevel", "info", "loglevel to use, one of: debug, info, warn, error, dpanic, panic, fatal")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&namespace, "namespace", "", "watch external secrets scoped in the provided namespace only. ClusterSecretStore can be used but only work if it doesn't reference resources from other namespaces")
	flag.Parse()

	var lvl zapcore.Level
	err := lvl.UnmarshalText([]byte(loglevel))
	if err != nil {
		setupLog.Error(err, "error unmarshalling loglevel")
		os.Exit(1)
	}
	go checkCerts("/tmp/k8s-webhook-server/serving-certs", "host.minikube.internal")
	logger := zap.New(zap.Level(lvl))
	ctrl.SetLogger(logger)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "webhook-controller",
		Namespace:          namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
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
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
