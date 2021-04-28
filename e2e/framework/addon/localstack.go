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
package addon

import "github.com/external-secrets/external-secrets/e2e/framework/util"

type Localstack struct {
	Addon
}

func NewLocalstack() *Localstack {
	return &Localstack{
		&HelmChart{
			Namespace:    "default",
			ReleaseName:  "localstack",
			Chart:        "localstack-charts/localstack",
			ChartVersion: "0.2.0",
			Repo: ChartRepo{
				Name: "localstack-charts",
				URL:  "https://localstack.github.io/helm-charts",
			},
			Values: []string{"/k8s/localstack.values.yaml"},
		},
	}
}

func (l *Localstack) Install() error {
	err := l.Addon.Install()
	if err != nil {
		return err
	}
	return util.WaitForURL("http://localstack.default/health")
}
