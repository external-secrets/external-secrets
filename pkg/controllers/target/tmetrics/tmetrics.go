// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Copyright External Secrets Inc. 2025
// All Rights Reserved

// Package tmetrics provides metrics for the Target controller.
package tmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	commonmetrics "github.com/external-secrets/external-secrets/pkg/controllers/secretstore/metrics"
)

const (
	// TargetSubsystem is the Prometheus subsystem for Target metrics.
	TargetSubsystem            = "target"
	// TargetReconcileDurationKey is the metric key for reconcile duration.
	TargetReconcileDurationKey = "reconcile_duration"
)

var gaugeVecMetrics = map[string]*prometheus.GaugeVec{}

// SetUpMetrics is called at the root to set-up the metric logic using the
// config flags provided.
func SetUpMetrics() {
	targetReconcileDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: TargetSubsystem,
		Name:      TargetReconcileDurationKey,
		Help:      "The duration time to reconcile the Secret Store",
	}, ctrlmetrics.NonConditionMetricLabelNames)

	targetCondition := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: TargetSubsystem,
		Name:      commonmetrics.StatusConditionKey,
		Help:      "The status condition of a specific Secret Store",
	}, ctrlmetrics.ConditionMetricLabelNames)

	metrics.Registry.MustRegister(targetReconcileDuration, targetCondition)

	gaugeVecMetrics = map[string]*prometheus.GaugeVec{
		TargetReconcileDurationKey:       targetReconcileDuration,
		commonmetrics.StatusConditionKey: targetCondition,
	}
}

// GetGaugeVec retrieves a GaugeVec metric by key.
func GetGaugeVec(key string) *prometheus.GaugeVec {
	return gaugeVecMetrics[key]
}

// RemoveMetrics deletes all metrics published by the resource.
func RemoveMetrics(namespace, name string) {
	for _, gaugeVecMetric := range gaugeVecMetrics {
		gaugeVecMetric.DeletePartialMatch(
			map[string]string{
				"namespace": namespace,
				"name":      name,
			},
		)
	}
}
