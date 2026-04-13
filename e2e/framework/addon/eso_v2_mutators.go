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

import "os"

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
		version := os.Getenv("VERSION")
		vars := []StringTuple{
			{Key: "replicaCount", Value: "1"},
			{Key: "v2.enabled", Value: "true"},
			{Key: "crds.createProvider", Value: "true"},
			{Key: "crds.createClusterProvider", Value: "true"},
			{Key: "providers.enabled", Value: "true"},
			{Key: "providerDefaults.replicaCount", Value: "1"},
			{Key: "providers.list[0].name", Value: "kubernetes"},
			{Key: "providers.list[0].type", Value: "kubernetes"},
			{Key: "providers.list[0].enabled", Value: "true"},
			{Key: "providers.list[0].replicaCount", Value: "1"},
			{Key: "providers.list[0].image.repository", Value: "ghcr.io/external-secrets/provider-kubernetes"},
			{Key: "providers.list[0].image.tag", Value: version},
			{Key: "providers.list[0].image.pullPolicy", Value: "IfNotPresent"},
		}
		for _, variable := range vars {
			setOrAppendVar(eso.HelmChart, variable)
		}
	}
}

func WithV2FakeProvider() MutationFunc {
	return func(eso *ESO) {
		version := os.Getenv("VERSION")
		vars := []StringTuple{
			{Key: "providers.enabled", Value: "true"},
			{Key: "providers.list[1].name", Value: "fake"},
			{Key: "providers.list[1].type", Value: "fake"},
			{Key: "providers.list[1].enabled", Value: "true"},
			{Key: "providers.list[1].replicaCount", Value: "1"},
			{Key: "providers.list[1].image.repository", Value: "ghcr.io/external-secrets/provider-fake"},
			{Key: "providers.list[1].image.tag", Value: version},
			{Key: "providers.list[1].image.pullPolicy", Value: "IfNotPresent"},
		}
		for _, variable := range vars {
			setOrAppendVar(eso.HelmChart, variable)
		}
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

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}
