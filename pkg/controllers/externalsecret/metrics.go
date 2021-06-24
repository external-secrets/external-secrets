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

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

const (
	ExternalSecretSubsystem          = "externalsecret"
	SyncCallsKey                     = "sync_calls_total"
	SyncCallsErrorKey                = "sync_calls_error"
	externalSecretStatusConditionKey = "status_condition"
)

var (
	syncCallsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      SyncCallsKey,
		Help:      "Total number of the External Secret sync calls",
	}, []string{"name", "namespace"})

	syncCallsError = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      SyncCallsErrorKey,
		Help:      "Total number of the External Secret sync errors",
	}, []string{"name", "namespace"})

	externalSecretCondition = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: ExternalSecretSubsystem,
		Name:      externalSecretStatusConditionKey,
		Help:      "The status condition of a specific External Secret",
	}, []string{"name", "namespace", "condition", "status"})
)

// updateExternalSecretCondition updates the ExternalSecret conditions.
func updateExternalSecretCondition(es *esv1alpha1.ExternalSecret, condition *esv1alpha1.ExternalSecretStatusCondition, value float64) {
	externalSecretCondition.With(prometheus.Labels{
		"name":      es.Name,
		"namespace": es.Namespace,
		"condition": string(condition.Type),
		"status":    string(condition.Status),
	}).Set(value)
}

func init() {
	metrics.Registry.MustRegister(syncCallsTotal, syncCallsError, externalSecretCondition)
}
