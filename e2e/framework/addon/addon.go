/*
Copyright © 2025 ESO Maintainer Team

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
	"path/filepath"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets-e2e/framework/log"
	"github.com/external-secrets/external-secrets-e2e/framework/util"
)

var globalAddons []Addon

func init() {
	globalAddons = make([]Addon, 0)
}

type Config struct {
	// KubeConfig which was used to create the connection.
	KubeConfig *rest.Config

	// Kubernetes API clientsets
	KubeClientSet kubernetes.Interface

	// controller-runtime client for newer controllers
	CRClient crclient.Client
}

type Addon interface {
	Setup(*Config) error
	Install() error
	Logs() error
	Uninstall() error
}

func InstallGlobalAddon(addon Addon) {
	globalAddons = append(globalAddons, addon)
	cfg := &Config{}
	cfg.KubeConfig, cfg.KubeClientSet, cfg.CRClient = util.NewConfig()

	ginkgo.By("installing global addon")
	err := addon.Setup(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	err = addon.Install()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func UninstallGlobalAddons() {
	for _, addon := range globalAddons {
		ginkgo.By("uninstalling addon")
		err := addon.Uninstall()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}
}

// AssetDir returns the path to the k8s asset directory
// which holds the helm charts, vault and conjur configuration.
// It starts at the cwd, and walks its way up to the root.
// It returns /k8s as a fallback.
// When running the e2e suite locally, this should return $REPO/e2e/k8s,
// when ran in CI this returns /k8s because the tests run in a dedicated pod where
// the assets are copied into the container.
func AssetDir() string {
	// Start from current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Traverse up the directory tree looking for "k8s" directory
	for {
		k8sPath := filepath.Join(currentDir, "k8s")

		// Check if "k8s" directory exists
		if info, err := os.Stat(k8sPath); err == nil && info.IsDir() {
			return k8sPath
		}

		// Get parent directory
		parentDir := filepath.Dir(currentDir)

		// If we've reached the root directory, stop searching
		if parentDir == currentDir {
			break
		}

		currentDir = parentDir
	}
	return "/k8s"
}

func PrintLogs() {
	for _, addon := range globalAddons {
		err := addon.Logs()
		if err != nil {
			log.Logf("error fetching logs: %s", err.Error())
		}
	}
}
