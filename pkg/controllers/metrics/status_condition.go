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

package metrics

// useDeprecatedStatusCondition toggles the legacy dual-emit behavior for the
// Ready condition of the externalsecret_status_condition and
// pushsecret_status_condition gauges. When true, both {status="True"} and
// {status="False"} series are emitted (the pre-consolidation behavior); when
// false (the default) only {status="False"} is emitted. It is set once at
// startup from the --use-deprecated-status-condition flag.
//
// Deprecated: kept only as a migration fallback. The legacy dual-emit path is
// slated for removal in v3.
var useDeprecatedStatusCondition bool

// SetUseDeprecatedStatusCondition configures whether the legacy dual-emit
// status_condition behavior is used. Call once at startup, before SetUpMetrics.
func SetUseDeprecatedStatusCondition(v bool) {
	useDeprecatedStatusCondition = v
}

// UseDeprecatedStatusCondition reports whether the legacy dual-emit
// status_condition behavior is enabled.
func UseDeprecatedStatusCondition() bool {
	return useDeprecatedStatusCondition
}
