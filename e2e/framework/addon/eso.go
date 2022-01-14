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
	"fmt"
	"os"

	// nolint
	. "github.com/onsi/ginkgo"
	// nolint
	. "github.com/onsi/gomega"

	// nolint
	"github.com/external-secrets/external-secrets/e2e/framework/util"
)

type ESO struct {
	Addon
}

func NewESO() *ESO {
	return &ESO{
		&HelmChart{
			Namespace:   "default",
			ReleaseName: "eso",
			Chart:       "/k8s/deploy/charts/external-secrets",
			Values:      []string{"/k8s/eso.values.yaml"},
		},
	}
}

func (l *ESO) Install() error {
	By("Installing eso\n")
	err := l.Addon.Install()
	if err != nil {
		return err
	}

	By("afterInstall eso\n")
	err = l.afterInstall()
	if err != nil {
		return err
	}

	return nil
}

func (l *ESO) afterInstall() error {
	err := gcpPreparation()
	Expect(err).NotTo(HaveOccurred())
	err = awsPreparation()
	Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return err
	}
	return nil
}

func gcpPreparation() error {
	gcpProjectID := os.Getenv("GCP_PROJECT_ID")
	gcpGSAName := os.Getenv("GCP_GSA_NAME")
	gcpKSAName := os.Getenv("GCP_KSA_NAME")
	_, kubeClientSet, _ := util.NewConfig()

	annotations := make(map[string]string)
	annotations["iam.gke.io/gcp-service-account"] = fmt.Sprintf("%s@%s.iam.gserviceaccount.com", gcpGSAName, gcpProjectID)
	_, err := util.UpdateKubeSA(gcpKSAName, kubeClientSet, "default", annotations)
	Expect(err).NotTo(HaveOccurred())

	_, err = util.UpdateKubeSA("external-secrets-e2e", kubeClientSet, "default", annotations)
	Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return err
	}

	return nil
}

func awsPreparation() error {
	return nil
}

func NewScopedESO() *ESO {
	return &ESO{
		&HelmChart{
			Namespace:   "default",
			ReleaseName: "eso-aws-sm",
			Chart:       "/k8s/deploy/charts/external-secrets",
			Values:      []string{"/k8s/eso.scoped.values.yaml"},
		},
	}
}
