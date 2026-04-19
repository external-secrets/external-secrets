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
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	. "github.com/onsi/ginkgo/v2"
)

const clusterProviderClassCRDName = "clusterproviderclasses.external-secrets.io"

var (
	externalSecretsCRDInstallPollInterval = time.Second
	externalSecretsCRDInstallTimeout      = 5 * time.Minute
)

func installCRDs(cfg *Config) error {
	bundlePath := filepath.Join(AssetDir(), "deploy/crds/bundle.yaml")
	cmd := exec.Command("kubectl", "apply", "--server-side", "--force-conflicts", "-f", bundlePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to install eso CRDs from %s: %w: %s", bundlePath, err, string(output))
	}

	return wait.PollUntilContextTimeout(GinkgoT().Context(), externalSecretsCRDInstallPollInterval, externalSecretsCRDInstallTimeout, true, func(ctx context.Context) (bool, error) {
		var crd apiextensionsv1.CustomResourceDefinition
		err := cfg.CRClient.Get(ctx, types.NamespacedName{Name: clusterProviderClassCRDName}, &crd)
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		for _, condition := range crd.Status.Conditions {
			if condition.Type == apiextensionsv1.Established && condition.Status == apiextensionsv1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
}
