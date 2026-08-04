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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	if err != nil {
		addon.Logs() // Print logs in case installation fails
	}
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func UninstallGlobalAddons() {
	for _, addon := range globalAddons {
		ginkgo.By("uninstalling addon")
		err := addon.Uninstall()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}
}

// skipGlobalTeardownVar opts a run out of uninstalling the global addons.
// e2e/run.sh forwards it into the e2e pod; CI sets it per leg.
const skipGlobalTeardownVar = "E2E_SKIP_GLOBAL_TEARDOWN"

// SkipGlobalTeardown reports whether a suite should leave the global addons
// installed instead of uninstalling them on the way out.
//
// CI runs every leg against a kind cluster that is discarded immediately
// afterwards, so uninstalling the ESO release costs about a minute per leg and
// buys nothing. It is also the only step that can fail a leg whose specs all
// passed: the release owns the CRDs, so `helm uninstall --wait` waits for CRD
// deletion, which blocks on finalizers that the controller being removed by the
// same uninstall is no longer around to release.
//
// Off unless asked for, so a run against a cluster it does not own (notably
// `make test.managed`) keeps cleaning up exactly as before.
//
// The TEST_SUITES check is not optional. entrypoint.sh runs each suite binary in
// turn against one cluster, and the provider and generator suites both install
// an "eso" release with different values, so a suite that left its release
// behind would break the next suite's install. A multi-suite run therefore tears
// down as normal and logs why the request was refused.
func SkipGlobalTeardown() bool {
	raw, ok := os.LookupEnv(skipGlobalTeardownVar)
	if !ok || raw == "" {
		return false
	}
	skip, err := strconv.ParseBool(raw)
	if err != nil {
		// Fall back to tearing down, which is the safe answer, and say so
		// loudly. Failing here instead would unwind the whole AfterSuite, so a
		// typo would leave the cluster with neither a teardown nor its logs.
		teardownLogf("%s is not a boolean (%q), so the teardown will run: %v",
			skipGlobalTeardownVar, raw, err)
		return false
	}
	if !skip {
		return false
	}
	// Only covers suites sharing one process's cluster, which is what
	// entrypoint.sh does. Two separate single-suite runs pointed at the same
	// cluster are indistinguishable from here and would still collide.
	if suites := strings.Fields(os.Getenv("TEST_SUITES")); len(suites) > 1 {
		teardownLogf("%s ignored: suites %q share one cluster, so the global "+
			"addons have to come out between them", skipGlobalTeardownVar,
			strings.Join(suites, " "))
		return false
	}
	teardownLogf("%s set: leaving the global addons installed for the cluster to "+
		"be discarded with", skipGlobalTeardownVar)
	return true
}

// teardownLogf reports a teardown decision on stderr.
//
// Not log.Logf: that writes to GinkgoWriter, and ginkgo drops a passing node's
// writer output unless the suite runs with -v, which CI does not. These lines
// have to survive a green run, since they are the only evidence of whether the
// skip took effect.
func teardownLogf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
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
