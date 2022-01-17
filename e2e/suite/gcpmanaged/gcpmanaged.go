/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/
package gcpmanaged

import (
	"os"

	// nolint
	. "github.com/onsi/ginkgo"
	// nolint
	. "github.com/onsi/ginkgo/extensions/table"

	// nolint
	// . "github.com/onsi/gomega"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/suite/common"
	"github.com/external-secrets/external-secrets/e2e/suite/gcp"
)

const (
	withPodID     = "sync secrets with pod identity"
	withSpecifcSA = "sync secrets with specificSA identity"
)

var _ = Describe("[gcpmanaged] ", func() {
	if os.Getenv("FOCUS") == "gcpmanaged" {
		f := framework.New("eso-gcp-managed")
		projectID := os.Getenv("GCP_PROJECT_ID")
		clusterLocation := "europe-west1-b"
		clusterName := "test-cluster"
		serviceAccountName := os.Getenv("GCP_KSA_NAME")
		serviceAccountNamespace := "default"
		prov := &gcp.GcpProvider{}
		if projectID != "" {
			prov = gcp.NewgcpProvider(f, "", projectID, clusterLocation, clusterName, serviceAccountName, serviceAccountNamespace)
		}
		DescribeTable("sync secrets",
			framework.TableFunc(f,
				prov),
			// uses pod id
			framework.Compose(withPodID, f, common.SimpleDataSync, usePodIDESReference),
			framework.Compose(withPodID, f, common.JSONDataWithProperty, usePodIDESReference),
			framework.Compose(withPodID, f, common.JSONDataFromSync, usePodIDESReference),
			framework.Compose(withPodID, f, common.NestedJSONWithGJSON, usePodIDESReference),
			framework.Compose(withPodID, f, common.JSONDataWithTemplate, usePodIDESReference),
			framework.Compose(withPodID, f, common.DockerJSONConfig, usePodIDESReference),
			framework.Compose(withPodID, f, common.DataPropertyDockerconfigJSON, usePodIDESReference),
			framework.Compose(withPodID, f, common.SSHKeySync, usePodIDESReference),
			framework.Compose(withPodID, f, common.SSHKeySyncDataProperty, usePodIDESReference),
			framework.Compose(withPodID, f, common.SyncWithoutTargetName, usePodIDESReference),
			framework.Compose(withPodID, f, common.JSONDataWithoutTargetName, usePodIDESReference),
			// uses specific sa
			framework.Compose(withSpecifcSA, f, common.JSONDataFromSync, useSpecifcSAESReference),
			framework.Compose(withSpecifcSA, f, common.JSONDataWithProperty, useSpecifcSAESReference),
			framework.Compose(withSpecifcSA, f, common.JSONDataFromSync, useSpecifcSAESReference),
			framework.Compose(withSpecifcSA, f, common.NestedJSONWithGJSON, useSpecifcSAESReference),
			framework.Compose(withSpecifcSA, f, common.JSONDataWithTemplate, useSpecifcSAESReference),
			framework.Compose(withSpecifcSA, f, common.DockerJSONConfig, useSpecifcSAESReference),
			framework.Compose(withSpecifcSA, f, common.DataPropertyDockerconfigJSON, useSpecifcSAESReference),
			framework.Compose(withSpecifcSA, f, common.SSHKeySync, useSpecifcSAESReference),
			framework.Compose(withSpecifcSA, f, common.SSHKeySyncDataProperty, useSpecifcSAESReference),
			framework.Compose(withSpecifcSA, f, common.SyncWithoutTargetName, useSpecifcSAESReference),
			framework.Compose(withSpecifcSA, f, common.JSONDataWithoutTargetName, useSpecifcSAESReference),
		)
	}
})

func usePodIDESReference(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = gcp.PodIDSecretStoreName
}

func useSpecifcSAESReference(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1alpha1.ClusterSecretStoreKind
	tc.ExternalSecret.Spec.SecretStoreRef.Name = gcp.SpecifcSASecretStoreName
}
