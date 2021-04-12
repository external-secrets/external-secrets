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

package externalsecret

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	ExternalSecretSubsystem = "external_secret"
	SyncCallsKey            = "sync_calls_total"
	SyncCallsErrorKey       = "sync_calls_error"
)

var (
	sync_calls_total = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      SyncCallsKey,
		Help:      "Total number of the External Secret sync calls",
	}, []string{"name", "namespace"})

	sync_calls_error = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      SyncCallsErrorKey,
		Help:      "Total number of the External Secret sync errors",
	}, []string{"name", "namespace"})
)

func init() {
	metrics.Registry.MustRegister(sync_calls_total, sync_calls_error)
}
