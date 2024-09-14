//Copyright External Secrets Inc. All Rights Reserved

package addon

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func uninstallCRDs(cfg *Config) error {
	ginkgo.By("Uninstalling eso CRDs")
	for _, crdName := range []string{
		"clusterexternalsecrets.external-secrets.io",
		"clustersecretstores.external-secrets.io",
		"externalsecrets.external-secrets.io",
		"secretstores.external-secrets.io",
	} {
		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: crdName,
			},
		}
		err := cfg.CRClient.Delete(context.Background(), crd, &client.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
