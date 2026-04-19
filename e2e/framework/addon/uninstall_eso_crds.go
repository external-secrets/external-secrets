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
	"strings"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
)

var (
	externalSecretsCRDDeletePollInterval = time.Second
	externalSecretsCRDDeleteTimeout      = 5 * time.Minute
)

func uninstallCRDs(cfg *Config) error {
	By("Uninstalling eso CRDs")
	crdList, err := listExternalSecretsCRDs(GinkgoT().Context(), cfg)
	if err != nil {
		return err
	}

	for _, crd := range crdList {
		err := cfg.CRClient.Delete(GinkgoT().Context(), &crd, &client.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	if len(crdList) == 0 {
		return nil
	}

	return wait.PollUntilContextTimeout(GinkgoT().Context(), externalSecretsCRDDeletePollInterval, externalSecretsCRDDeleteTimeout, true, func(ctx context.Context) (bool, error) {
		crds, err := listExternalSecretsCRDs(ctx, cfg)
		if err != nil {
			return false, err
		}
		return len(crds) == 0, nil
	})
}

func listExternalSecretsCRDs(ctx context.Context, cfg *Config) ([]apiextensionsv1.CustomResourceDefinition, error) {
	var crdList apiextensionsv1.CustomResourceDefinitionList
	if err := cfg.CRClient.List(ctx, &crdList); err != nil {
		return nil, err
	}

	crds := make([]apiextensionsv1.CustomResourceDefinition, 0, len(crdList.Items))
	for _, crd := range crdList.Items {
		if !isExternalSecretsCRDGroup(crd.Spec.Group) {
			continue
		}
		crds = append(crds, crd)
	}
	return crds, nil
}

func isExternalSecretsCRDGroup(group string) bool {
	return strings.Contains(group, "external-secrets.io")
}
