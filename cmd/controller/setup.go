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
	"os"

	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func setupMetricServerOptions() server.Options {
	if metricsAuth && !metricsSecure {
		setupLog.Error(nil, "--metrics-auth requires --metrics-secure; bearer tokens over plaintext HTTP is not allowed")
		os.Exit(1)
	}

	// Configure metrics server options
	metricsServerOpts := server.Options{
		BindAddress: metricsAddr,
	}

	if metricsSecure {
		metricsTLSOpts, err := buildTLSConfigFuncs(tlsCiphers, tlsMinVersion, tlsCurvePreferences, enableHTTP2)
		if err != nil {
			setupLog.Error(err, "unable to configure TLS for metrics server")
			os.Exit(1)
		}

		metricsServerOpts.SecureServing = true
		metricsServerOpts.CertDir = metricsCertDir
		metricsServerOpts.CertName = metricsCertName
		metricsServerOpts.KeyName = metricsKeyName
		metricsServerOpts.TLSOpts = metricsTLSOpts

		if metricsAuth {
			metricsServerOpts.FilterProvider = filters.WithAuthenticationAndAuthorization
		}
	}
	return metricsServerOpts
}

func setupLogger() {
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
	klog.SetLogger(logger)
}
