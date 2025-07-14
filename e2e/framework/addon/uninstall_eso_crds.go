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
	"context"

	"github.com/onsi/ginkgo/v2"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func uninstallCRDs(cfg *Config) error {
	ginkgo.By("Uninstalling eso CRDs")
	var crdList apiextensionsv1.CustomResourceDefinitionList
	if err := cfg.CRClient.List(context.Background(), &crdList); err != nil {
		return err
	}

	for _, crd := range crdList.Items {
		if crd.Spec.Group != "external-secrets.io" {
			continue
		}
		err := cfg.CRClient.Delete(context.Background(), &crd, &client.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
