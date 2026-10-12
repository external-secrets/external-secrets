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
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/common/version"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// buildInfoProgram is the metric namespace, producing external_secrets_build_info.
const buildInfoProgram = "external_secrets"

func init() {
	// The controller, webhook and certcontroller share this registry, so all of them expose the metric.
	metrics.Registry.MustRegister(versioncollector.NewCollector(buildInfoProgram))
}

func logBuildInfo() {
	setupLog.Info("build info",
		"version", version.Version,
		"revision", version.GetRevision(),
		"goversion", version.GoVersion,
		"goos", version.GoOS,
		"goarch", version.GoArch,
	)
}
