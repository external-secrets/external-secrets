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

package addon

import (
	"os"
	"strconv"
	"strings"
)

const (
	v2HelmNamespace   = "external-secrets-system"
	v2HelmReleaseName = "external-secrets"
)

func WithV2Namespace() MutationFunc {
	return func(eso *ESO) {
		eso.HelmChart.Namespace = v2HelmNamespace
		eso.HelmChart.ReleaseName = v2HelmReleaseName
		if !containsArg(eso.HelmChart.Args, "--create-namespace") {
			eso.HelmChart.Args = append(eso.HelmChart.Args, "--create-namespace")
		}
	}
}

func WithV2KubernetesProvider() MutationFunc {
	return func(eso *ESO) {
		ensureV2ProviderConfig(eso.HelmChart)
		setProvider(eso.HelmChart, "kubernetes", "kubernetes", "ghcr.io/external-secrets/provider-kubernetes", os.Getenv("VERSION"))
	}
}

func WithV2FakeProvider() MutationFunc {
	return func(eso *ESO) {
		ensureV2ProviderConfig(eso.HelmChart)
		setProvider(eso.HelmChart, "fake", "fake", "ghcr.io/external-secrets/provider-fake", os.Getenv("VERSION"))
	}
}

func WithV2AWSProvider() MutationFunc {
	return func(eso *ESO) {
		ensureV2ProviderConfig(eso.HelmChart)
		setProvider(eso.HelmChart, "aws", "aws", "ghcr.io/external-secrets/provider-aws", os.Getenv("VERSION"))
	}
}

func WithV2ProviderServiceAccount(providerName, serviceAccountName string) MutationFunc {
	return func(eso *ESO) {
		index := findProviderIndex(eso.HelmChart, providerName)
		if index < 0 {
			panic("provider entry must exist before overriding service account")
		}

		prefix := "providers.list[" + strconv.Itoa(index) + "].serviceAccount"
		setOrAppendVar(eso.HelmChart, StringTuple{Key: prefix + ".create", Value: "false"})
		setOrAppendVar(eso.HelmChart, StringTuple{Key: prefix + ".name", Value: serviceAccountName})
	}
}

func setOrAppendVar(chart *HelmChart, variable StringTuple) {
	for i := range chart.Vars {
		if chart.Vars[i].Key == variable.Key {
			chart.Vars[i].Value = variable.Value
			return
		}
	}
	chart.Vars = append(chart.Vars, variable)
}

func ensureV2ProviderConfig(chart *HelmChart) {
	requiredVars := []StringTuple{
		{Key: "v2.enabled", Value: "true"},
		{Key: "crds.createClusterProviderClass", Value: "true"},
		{Key: "crds.createProviderStore", Value: "true"},
		{Key: "crds.createClusterProviderStore", Value: "true"},
		{Key: "providers.enabled", Value: "true"},
	}
	for _, variable := range requiredVars {
		setOrAppendVar(chart, variable)
	}

	defaultVars := []StringTuple{
		{Key: "replicaCount", Value: "1"},
		{Key: "providerDefaults.replicaCount", Value: "1"},
	}
	for _, variable := range defaultVars {
		setVarIfMissing(chart, variable)
	}
}

func setVarIfMissing(chart *HelmChart, variable StringTuple) {
	for i := range chart.Vars {
		if chart.Vars[i].Key == variable.Key {
			return
		}
	}
	chart.Vars = append(chart.Vars, variable)
}

func setProvider(chart *HelmChart, name, providerType, imageRepository, imageTag string) {
	index := findProviderIndex(chart, name)
	if index < 0 {
		index = nextProviderIndex(chart)
	}

	prefix := "providers.list[" + strconv.Itoa(index) + "]"
	vars := []StringTuple{
		{Key: prefix + ".name", Value: name},
		{Key: prefix + ".type", Value: providerType},
		{Key: prefix + ".enabled", Value: "true"},
		{Key: prefix + ".replicaCount", Value: "1"},
		{Key: prefix + ".image.repository", Value: imageRepository},
		{Key: prefix + ".image.tag", Value: imageTag},
		{Key: prefix + ".image.pullPolicy", Value: "IfNotPresent"},
	}
	for _, variable := range vars {
		setOrAppendVar(chart, variable)
	}
}

func findProviderIndex(chart *HelmChart, name string) int {
	const prefix = "providers.list["
	const suffix = "].name"
	for _, variable := range chart.Vars {
		if !strings.HasPrefix(variable.Key, prefix) || !strings.HasSuffix(variable.Key, suffix) {
			continue
		}
		if variable.Value != name {
			continue
		}
		indexStr := strings.TrimSuffix(strings.TrimPrefix(variable.Key, prefix), suffix)
		index, err := strconv.Atoi(indexStr)
		if err == nil {
			return index
		}
	}
	return -1
}

func nextProviderIndex(chart *HelmChart) int {
	const prefix = "providers.list["
	maxIndex := -1
	for _, variable := range chart.Vars {
		if !strings.HasPrefix(variable.Key, prefix) {
			continue
		}

		remainder := strings.TrimPrefix(variable.Key, prefix)
		closingBracket := strings.Index(remainder, "]")
		if closingBracket < 0 {
			continue
		}

		index, err := strconv.Atoi(remainder[:closingBracket])
		if err != nil {
			continue
		}
		if index > maxIndex {
			maxIndex = index
		}
	}
	return maxIndex + 1
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}
