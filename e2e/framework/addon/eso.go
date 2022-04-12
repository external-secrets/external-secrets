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

import (
	"os"

	// nolint
	. "github.com/onsi/ginkgo/v2"
)

type ESO struct {
	*HelmChart
}

func NewESO(mutators ...MutationFunc) *ESO {
	eso := &ESO{
		&HelmChart{
			Namespace:   "default",
			ReleaseName: "eso",
			Chart:       "/k8s/deploy/charts/external-secrets",
			Vars: []StringTuple{
				{
					Key:   "image.repository",
					Value: os.Getenv("IMAGE_REGISTRY"),
				},
				{
					Key:   "webhook.port",
					Value: "9443",
				},
				{
					Key:   "webhook.image.repository",
					Value: os.Getenv("IMAGE_REGISTRY"),
				},
				{
					Key:   "certController.image.repository",
					Value: os.Getenv("IMAGE_REGISTRY"),
				},
				{
					Key:   "webhook.image.tag",
					Value: os.Getenv("VERSION"),
				},
				{
					Key:   "certController.image.tag",
					Value: os.Getenv("VERSION"),
				},
				{
					Key:   "image.tag",
					Value: os.Getenv("VERSION"),
				},
				{
					Key:   "installCRDs",
					Value: "false",
				},
			},
		},
	}

	for _, f := range mutators {
		f(eso)
	}

	return eso
}

type MutationFunc func(eso *ESO)

func WithReleaseName(name string) MutationFunc {
	return func(eso *ESO) {
		eso.HelmChart.ReleaseName = name
	}
}

func WithNamespace(namespace string) MutationFunc {
	return func(eso *ESO) {
		eso.HelmChart.Namespace = namespace
	}
}

func WithNamespaceScope(namespace string) MutationFunc {
	return func(eso *ESO) {
		eso.HelmChart.Vars = append(eso.HelmChart.Vars, StringTuple{
			Key:   "scopedNamespace",
			Value: namespace,
		})
	}
}

func WithoutWebhook() MutationFunc {
	return func(eso *ESO) {
		eso.HelmChart.Vars = append(eso.HelmChart.Vars, StringTuple{
			Key:   "webhook.create",
			Value: "false",
		})
	}
}

func WithoutCertController() MutationFunc {
	return func(eso *ESO) {
		eso.HelmChart.Vars = append(eso.HelmChart.Vars, StringTuple{
			Key:   "certController.create",
			Value: "false",
		})
	}
}

func WithServiceAccount(saName string) MutationFunc {
	return func(eso *ESO) {
		eso.HelmChart.Vars = append(eso.HelmChart.Vars, []StringTuple{
			{
				Key:   "serviceAccount.create",
				Value: "false",
			},
			{
				Key:   "serviceAccount.name",
				Value: saName,
			},
		}...)
	}
}

func WithControllerClass(class string) MutationFunc {
	return func(eso *ESO) {
		eso.HelmChart.Vars = append(eso.HelmChart.Vars, StringTuple{
			Key:   "extraArgs.controller-class",
			Value: class,
		})
	}
}

func (l *ESO) Install() error {
	By("Installing eso\n")
	err := l.HelmChart.Install()
	if err != nil {
		return err
	}

	return nil
}
