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
package gcp

import (

	// nolint
	. "github.com/onsi/ginkgo/v2"

	// nolint
	// . "github.com/onsi/gomega"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/framework/addon"
	"github.com/external-secrets/external-secrets/e2e/suites/provider/cases/common"
)

const (
	withPodID     = "sync secrets with pod identity"
	withSpecifcSA = "sync secrets with specificSA identity"
)

// Deploys eso to the default namespace
// that uses the service account provisioned by terraform
// to test pod-identity authentication.
var _ = Describe("[gcpmanaged] with pod identity", Label("gcp", "secretsmanager", "managed", "pod-identity"), func() {
	f := framework.New("eso-gcpmanaged")
	prov := NewFromEnv(f, f.BaseName)

	// each test case gets its own ESO instance
	BeforeEach(func() {
		f.Install(addon.NewESO(
			addon.WithControllerClass(f.BaseName),
			addon.WithServiceAccount(prov.ServiceAccountName),
			addon.WithReleaseName(f.Namespace.Name),
			addon.WithNamespace("default"),
			addon.WithoutWebhook(),
			addon.WithoutCertController(),
		))
	})

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
	)
})

// We're using a namespace scoped ESO
// that runs WITHOUT pod identity (with default sa)
// It uses a specific service account defined in the ClusterSecretStore spec
// to authenticate against cloud provider APIs.
var _ = Describe("[gcpmanaged] with service account", Label("gcp", "secretsmanager", "managed", "service-account"), func() {
	f := framework.New("eso-gcpmanaged")
	prov := NewFromEnv(f, f.BaseName)

	BeforeEach(func() {
		f.Install(addon.NewESO(
			addon.WithControllerClass(f.BaseName),
			addon.WithReleaseName(f.Namespace.Name),
			addon.WithNamespace(f.Namespace.Name),
			addon.WithoutWebhook(),
			addon.WithoutCertController(),
		))
	})

	DescribeTable("sync secrets",
		framework.TableFunc(f,
			prov),
		// uses specific sa
		framework.Compose(withSpecifcSA, f, common.JSONDataFromSync, useSpecifcSAESReference(prov)),
		framework.Compose(withSpecifcSA, f, common.JSONDataWithProperty, useSpecifcSAESReference(prov)),
		framework.Compose(withSpecifcSA, f, common.JSONDataFromSync, useSpecifcSAESReference(prov)),
		framework.Compose(withSpecifcSA, f, common.NestedJSONWithGJSON, useSpecifcSAESReference(prov)),
		framework.Compose(withSpecifcSA, f, common.JSONDataWithTemplate, useSpecifcSAESReference(prov)),
		framework.Compose(withSpecifcSA, f, common.DockerJSONConfig, useSpecifcSAESReference(prov)),
		framework.Compose(withSpecifcSA, f, common.DataPropertyDockerconfigJSON, useSpecifcSAESReference(prov)),
		framework.Compose(withSpecifcSA, f, common.SSHKeySync, useSpecifcSAESReference(prov)),
		framework.Compose(withSpecifcSA, f, common.SSHKeySyncDataProperty, useSpecifcSAESReference(prov)),
		framework.Compose(withSpecifcSA, f, common.SyncWithoutTargetName, useSpecifcSAESReference(prov)),
		framework.Compose(withSpecifcSA, f, common.JSONDataWithoutTargetName, useSpecifcSAESReference(prov)),
	)
})

func usePodIDESReference(tc *framework.TestCase) {
	tc.ExternalSecret.Spec.SecretStoreRef.Name = PodIDSecretStoreName
}

func useSpecifcSAESReference(prov *GcpProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1beta1.ClusterSecretStoreKind
		tc.ExternalSecret.Spec.SecretStoreRef.Name = prov.SAClusterSecretStoreName()
	}
}
