/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

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
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/crds"
)

const (
	errCreateWebhook = "unable to create webhook"
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = esv1beta1.AddToScheme(scheme)
	_ = esv1alpha1.AddToScheme(scheme)
}

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Webhook implementation for ExternalSecrets and SecretStores.",
	Long: `Webhook implementation for ExternalSecrets and SecretStores.
	For more information visit https://external-secrets.io`,
	Run: func(cmd *cobra.Command, args []string) {
		var lvl zapcore.Level
		var enc zapcore.TimeEncoder
		c := crds.CertInfo{
			CertDir:  certDir,
			CertName: "tls.crt",
			KeyName:  "tls.key",
			CAName:   "ca.crt",
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

		err := waitForCerts(c, time.Minute*2)
		if err != nil {
			setupLog.Error(err, "unable to validate certificates")
			os.Exit(1)
		}

		ctx, cancel := context.WithCancel(context.Background())
		go func(c crds.CertInfo, dnsName string, every time.Duration) {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			ticker := time.NewTicker(every)
			for {
				select {
				case <-sigs:
					cancel()
				case <-ticker.C:
					setupLog.Info("validating certs")
					err = crds.CheckCerts(c, dnsName, time.Now().Add(certLookaheadInterval))
					if err != nil {
						setupLog.Error(err, "certs are not valid at now + lookahead, triggering shutdown", "certLookahead", certLookaheadInterval.String())
						cancel()
						return
					}
					setupLog.Info("certs are valid")
				}
			}
		}(c, dnsName, certCheckInterval)

		cipherList, err := getTLSCipherSuitesIDs(tlsCiphers)
		if err != nil {
			ctrl.Log.Error(err, "unable to fetch tls ciphers")
			os.Exit(1)
		}
		mgrTLSOptions := func(cfg *tls.Config) {
			cfg.CipherSuites = cipherList
		}
		mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
			Scheme: scheme,
			Metrics: server.Options{
				BindAddress: metricsAddr,
			},
			HealthProbeBindAddress: healthzAddr,
			WebhookServer: webhook.NewServer(webhook.Options{
				CertDir: certDir,
				Port:    port,
				TLSOpts: []func(*tls.Config){
					mgrTLSOptions,
					func(c *tls.Config) {
						c.MinVersion = tlsVersion(tlsMinVersion)
					},
				},
			}),
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

		err = mgr.AddReadyzCheck("certs", func(_ *http.Request) error {
			return crds.CheckCerts(c, dnsName, time.Now().Add(time.Hour))
		})
		if err != nil {
			setupLog.Error(err, "unable to add certs readyz check")
			os.Exit(1)
		}

		setupLog.Info("starting manager")
		if err := mgr.Start(ctx); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	},
}

// tlsVersion converts from human-readable TLS version (for example "1.1")
// to the values accepted by tls.Config (for example 0x301).
func tlsVersion(version string) uint16 {
	switch version {
	case "":
		return tls.VersionTLS10
	case "1.0":
		return tls.VersionTLS10
	case "1.1":
		return tls.VersionTLS11
	case "1.2":
		return tls.VersionTLS12
	case "1.3":
		return tls.VersionTLS13
	default:
		return tls.VersionTLS13
	}
}

// waitForCerts waits until the certificates become ready.
// If they don't become ready within a given time duration
// this function returns an error.
// certs are generated by the certcontroller.
func waitForCerts(c crds.CertInfo, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		setupLog.Info("validating certs")
		err := crds.CheckCerts(c, dnsName, time.Now().Add(time.Hour))
		if err == nil {
			return nil
		}
		if err != nil {
			setupLog.Error(err, "invalid certs. retrying...")
			<-time.After(time.Second * 10)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func getTLSCipherSuitesIDs(cipherListString string) ([]uint16, error) {
	if cipherListString == "" {
		return nil, nil
	}
	cipherList := strings.Split(cipherListString, ",")
	cipherIds := map[string]uint16{}
	for _, cs := range tls.CipherSuites() {
		cipherIds[cs.Name] = cs.ID
	}
	ret := make([]uint16, 0, len(cipherList))
	for _, c := range cipherList {
		id, ok := cipherIds[c]
		if !ok {
			return ret, fmt.Errorf("cipher %s was not found", c)
		}
		ret = append(ret, id)
	}
	return ret, nil
}

func init() {
	rootCmd.AddCommand(webhookCmd)
	webhookCmd.Flags().StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	webhookCmd.Flags().StringVar(&healthzAddr, "healthz-addr", ":8081", "The address the health endpoint binds to.")
	webhookCmd.Flags().IntVar(&port, "port", 10250, "Port number that the webhook server will serve.")
	webhookCmd.Flags().StringVar(&dnsName, "dns-name", "localhost", "DNS name to validate certificates with")
	webhookCmd.Flags().StringVar(&certDir, "cert-dir", "/tmp/k8s-webhook-server/serving-certs", "path to check for certs")
	webhookCmd.Flags().StringVar(&zapTimeEncoding, "zap-time-encoding", "epoch", "Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano')")
	webhookCmd.Flags().StringVar(&loglevel, "loglevel", "info", "loglevel to use, one of: debug, info, warn, error, dpanic, panic, fatal")
	webhookCmd.Flags().DurationVar(&certCheckInterval, "check-interval", 5*time.Minute, "certificate check interval")
	webhookCmd.Flags().DurationVar(&certLookaheadInterval, "lookahead-interval", crds.LookaheadInterval, "certificate check interval")
	// https://go.dev/blog/tls-cipher-suites explains the ciphers selection process
	webhookCmd.Flags().StringVar(&tlsCiphers, "tls-ciphers", "", "comma separated list of tls ciphers allowed."+
		" This does not apply to TLS 1.3 as the ciphers are selected automatically."+
		" The order of this list does not give preference to the ciphers, the ordering is done automatically."+
		" Full lists of available ciphers can be found at https://pkg.go.dev/crypto/tls#pkg-constants."+
		" E.g. 'TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256'")
	webhookCmd.Flags().StringVar(&tlsMinVersion, "tls-min-version", "1.2", "minimum version of TLS supported.")
}
