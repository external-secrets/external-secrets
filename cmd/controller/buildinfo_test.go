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
	"runtime"
	"testing"

	"github.com/prometheus/common/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func TestBuildInfoMetricRegistered(t *testing.T) {
	families, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range families {
		if mf.GetName() != "external_secrets_build_info" {
			continue
		}
		found = true
		require.Len(t, mf.GetMetric(), 1)
		m := mf.GetMetric()[0]
		assert.InDelta(t, 1.0, m.GetGauge().GetValue(), 0)

		labels := map[string]string{}
		for _, lp := range m.GetLabel() {
			labels[lp.GetName()] = lp.GetValue()
		}
		assert.Equal(t, version.Version, labels["version"])
		assert.Equal(t, version.GetRevision(), labels["revision"])
		assert.Equal(t, runtime.Version(), labels["goversion"])
		assert.Equal(t, runtime.GOOS, labels["goos"])
		assert.Equal(t, runtime.GOARCH, labels["goarch"])
	}
	assert.True(t, found, "external_secrets_build_info metric not registered")
}
